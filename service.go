package edge

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Service struct {
	cfg            Config
	client         *Client
	state          *Store
	publisher      Publisher
	httpClient     *http.Client
	downloadClient *http.Client
}

func New(cfg Config, manifestClient *Client, stateStore *Store, publisher Publisher, client *http.Client) *Service {
	return &Service{
		cfg:            cfg,
		client:         manifestClient,
		state:          stateStore,
		publisher:      publisher,
		httpClient:     client,
		downloadClient: client,
	}
}

func (s *Service) Run(ctx context.Context) error {
	ticker := time.NewTicker(s.cfg.PollInterval)
	defer ticker.Stop()

	for {
		jobs, isNew, err := s.pollManifest(ctx)
		if err != nil {
			slog.Error("polling failed", "error", err)
		} else if isNew && len(jobs) > 0 {
			if err := s.runWorkers(ctx, jobs); err != nil {
				slog.Error("worker pool failed", "error", err)
			}
		} else {
			slog.Info("no work items")
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) pollManifest(ctx context.Context) ([]Job, bool, error) {
	currentETag := s.state.ManifestETag()
	manifestStatus := s.state.ManifestStatus()
	resp, err := s.client.Fetch(ctx, currentETag)
	if err != nil {
		return nil, false, fmt.Errorf("fetch manifest: %w", err)
	}

	switch resp.StatusCode {
	case http.StatusNotModified:
		if manifestStatus == "complete" {
			slog.Info("manifest unchanged")
			return nil, false, nil
		}
		slog.Info("manifest unchanged but pending; rebuilding jobs")
		jobs := s.buildJobsStore()
		if len(jobs) == 0 {
			if err := s.state.UpdateManifestStatus("complete"); err != nil {
				slog.Error("failed to mark manifest complete", "error", err)
			}
			return nil, false, nil
		}
		return jobs, true, nil
	case http.StatusOK:
		if resp.ETag == "" {
			slog.Warn("manifest missing ETag; proceeding but persistence may be noisy")
		} else {
			if err := s.state.UpdateManifestETag(resp.ETag); err != nil {
				return nil, false, fmt.Errorf("save manifest etag: %w", err)
			}
		}
		jobs := s.buildJobsResponse(resp.Manifest)
		if len(jobs) == 0 {
			if err := s.state.UpdateManifestStatus("complete"); err != nil {
				slog.Error("failed to mark manifest complete", "error", err)
			}
		}
		return jobs, true, nil
	default:
		return nil, false, fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
}

func (s *Service) buildJobsStore() []Job {
	pending := make([]Job, 0)
	for key, st := range s.state.Snapshot().Items {
		if st.Status == "complete" {
			continue
		}
		parts := strings.SplitN(key, "/", 2)
		if len(parts) != 2 {
			slog.Warn("invalid key format in state", "key", key)
			continue
		}
		if st.URI == "" {
			slog.Warn("missing URI for pending item; skipping", "key", key)
			continue
		}
		item := ContentItem{
			Name: parts[1],
			URI:  st.URI,
			ETag: st.ETag,
		}
		pending = append(pending, Job{
			ContentType: parts[0],
			Item:        item,
			Attempt:     1,
		})
	}
	if len(pending) == 0 {
		slog.Info("no pending items in state")
	}
	return pending
}

func (s *Service) buildJobsResponse(manifestData map[string]ManifestItem) []Job {
	pending := make([]Job, 0)

	for contentType, m := range manifestData {
		if m.Unavailable {
			slog.Warn("content type unavailable", "type", contentType)
			continue
		}
		for _, item := range m.Items {
			if item.Unavailable {
				slog.Warn("content item unavailable", "type", contentType, "name", item.Name)
				continue
			}
			key := jobKey(contentType, item.Name)
			if existing, ok := s.state.Item(key); ok && existing.ETag != "" && existing.ETag == item.ETag && existing.Status == "complete" {
				slog.Info("item already up to date", "key", key)
				continue
			}
			_ = s.state.UpdateItem(key, ItemState{
				ETag:        item.ETag,
				Status:      "pending",
				URI:         item.URI,
				LastUpdated: time.Now().UTC(),
			})
			pending = append(pending, Job{ContentType: contentType, Item: item, Attempt: 1})
		}
	}

	if len(pending) == 0 {
		slog.Info("no new or changed items")
		return nil
	}

	return pending
}

type Job struct {
	ContentType string
	Item        ContentItem
	Attempt     int
}

type jobResult struct {
	Job Job
	Err error
}

func (s *Service) runWorkers(ctx context.Context, initial []Job) error {
	workerCount := s.cfg.WorkerCount
	if workerCount <= 0 {
		workerCount = 1
	}
	maxAttempts := s.cfg.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}

	jobsCh := make(chan Job)
	resultsCh := make(chan jobResult)

	var wg sync.WaitGroup
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for job := range jobsCh {
				downloadCtx, cancel := context.WithTimeout(ctx, s.cfg.DownloadTimeout)
				err := s.downloadItem(downloadCtx, job)
				cancel()
				resultsCh <- jobResult{Job: job, Err: err}
			}
		}()
	}

	go func() {
		wg.Wait()
		close(resultsCh)
	}()

	pending := append([]Job(nil), initial...)
	inFlight := 0

	for len(pending) > 0 || inFlight > 0 {
		var next Job
		var sendCh chan Job
		if len(pending) > 0 {
			next = pending[0]
			sendCh = jobsCh
		}

		select {
		case <-ctx.Done():
			close(jobsCh)
			// Drain any in-flight results.
			for range resultsCh {
			}
			return ctx.Err()
		case sendCh <- next:
			pending = pending[1:]
			inFlight++
		case res, ok := <-resultsCh:
			if !ok {
				continue
			}
			inFlight--
			key := jobKey(res.Job.ContentType, res.Job.Item.Name)
			if res.Err == nil {
				slog.Info("downloaded item", "key", key, "attempt", res.Job.Attempt)
			} else if errors.Is(res.Err, context.DeadlineExceeded) && res.Job.Attempt < maxAttempts {
				retry := res.Job
				retry.Attempt++
				pending = append(pending, retry)
				slog.Warn("download timed out; retrying", "key", key, "attempt", retry.Attempt)
			} else {
				slog.Error("download failed", "key", key, "attempt", res.Job.Attempt, "error", res.Err)
			}
		}
	}

	close(jobsCh)
	for range resultsCh {
	}
	if err := s.state.UpdateManifestStatus("complete"); err != nil {
		slog.Error("failed to mark manifest complete", "error", err)
	}
	return nil
}

func (s *Service) downloadItem(ctx context.Context, job Job) error {
	contentType := job.ContentType
	item := job.Item
	key := jobKey(contentType, item.Name)
	existing, _ := s.state.Item(key)

	// If manifest ETag matches what we already have, skip.
	if existing.ETag != "" && existing.ETag == item.ETag && existing.Status == "complete" {
		slog.Info("item already up to date", "key", key)
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, item.URI, nil)
	if err != nil {
		return err
	}
	if existing.ETag != "" {
		req.Header.Set("If-None-Match", existing.ETag)
	}

	res, err := s.downloadClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()

	switch res.StatusCode {
	case http.StatusOK:
		targetPath := filepath.Join(s.cfg.DataDir, contentType, item.Name)
		if err := saveFile(res.Body, targetPath); err != nil {
			return err
		}
		return s.markItemDownloaded(key, item, res.Header.Get("ETag"), targetPath)
	case http.StatusNotModified:
		slog.Info("item not modified", "key", key)
		return nil
	default:
		body, _ := io.ReadAll(io.LimitReader(res.Body, 512))
		return fmt.Errorf("unexpected status %d downloading %s: %s", res.StatusCode, key, string(body))
	}
}

func saveFile(body io.Reader, target string) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}

	tmp := target + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, body); err != nil {
		f.Close()
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		return err
	}
	return nil
}

func jobKey(contentType, name string) string {
	return fmt.Sprintf("%s/%s", contentType, name)
}

func (s *Service) markItemDownloaded(key string, item ContentItem, headerETag, targetPath string) error {
	stateETag := headerETag
	if stateETag == "" {
		stateETag = item.ETag
	}
	newState := ItemState{
		ETag:        stateETag,
		Path:        targetPath,
		URI:         item.URI,
		LastUpdated: time.Now().UTC(),
		Status:      "complete",
	}
	if err := s.state.UpdateItem(key, newState); err != nil {
		return err
	}
	s.publisher.Publish("ADDED", key)
	return nil
}
