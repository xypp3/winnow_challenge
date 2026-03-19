package edge

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
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
		if err := s.pollOnce(ctx); err != nil {
			slog.Error("polling failed", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

func (s *Service) pollOnce(ctx context.Context) error {
	currentETag := s.state.ManifestETag()
	resp, err := s.client.Fetch(ctx, currentETag)
	if err != nil {
		return fmt.Errorf("fetch manifest: %w", err)
	}

	if resp.StatusCode == http.StatusNotModified {
		slog.Info("manifest unchanged")
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}

	if resp.ETag == "" {
		slog.Warn("manifest missing ETag; proceeding but persistence may be noisy")
	}

	if err := s.processManifest(ctx, resp.Manifest); err != nil {
		return err
	}

	if resp.ETag != "" {
		if err := s.state.UpdateManifestETag(resp.ETag); err != nil {
			return fmt.Errorf("save manifest etag: %w", err)
		}
	}

	return nil
}

func (s *Service) processManifest(ctx context.Context, manifestData map[string]ManifestItem) error {
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
			if err := s.downloadItem(ctx, contentType, item); err != nil {
				return fmt.Errorf("download %s/%s: %w", contentType, item.Name, err)
			}
		}
	}
	return nil
}

func (s *Service) downloadItem(ctx context.Context, contentType string, item ContentItem) error {
	key := fmt.Sprintf("%s/%s", contentType, item.Name)
	existing, _ := s.state.Item(key)

	// If manifest ETag matches what we already have, skip.
	if existing.ETag != "" && existing.ETag == item.ETag {
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
		newState := ItemState{
			ETag:        res.Header.Get("ETag"),
			Path:        targetPath,
			LastUpdated: time.Now().UTC(),
		}
		if newState.ETag == "" {
			newState.ETag = item.ETag
		}
		if err := s.state.UpdateItem(key, newState); err != nil {
			return err
		}
		s.publisher.Publish("ADDED", key)
		return nil
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
