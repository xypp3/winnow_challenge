package edge

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ItemState struct {
	ETag        string    `json:"etag"`
	Path        string    `json:"path"`
	LastUpdated time.Time `json:"last_updated"`
	Status      string    `json:"status"` // pending|complete
	URI         string    `json:"uri"`
}

type State struct {
	ManifestETag   string               `json:"manifest_etag"`
	ManifestStatus string               `json:"manifest_status"` // pending|complete
	Items          map[string]ItemState `json:"items"`
}

type Store struct {
	path  string
	mu    sync.Mutex
	state State
}

func NewStore(path string) (*Store, error) {
	s := &Store{
		path: path,
		state: State{
			Items: make(map[string]ItemState),
		},
	}
	if err := s.load(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) ManifestETag() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.ManifestETag
}

func (s *Store) ManifestStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state.ManifestStatus
}

func (s *Store) Snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	// Copy maps to avoid callers mutating internal state.
	items := make(map[string]ItemState, len(s.state.Items))
	for k, v := range s.state.Items {
		items[k] = v
	}
	return State{
		ManifestETag:   s.state.ManifestETag,
		ManifestStatus: s.state.ManifestStatus,
		Items:          items,
	}
}

func (s *Store) Item(key string) (ItemState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	item, ok := s.state.Items[key]
	return item, ok
}

func (s *Store) UpdateItem(key string, item ItemState) error {
	return s.update(func(st *State) {
		st.Items[key] = item
	})
}

func (s *Store) UpdateManifestETag(etag string) error {
	return s.update(func(st *State) {
		st.ManifestETag = etag
		st.ManifestStatus = "pending"
		st.Items = make(map[string]ItemState)
	})
}

func (s *Store) UpdateManifestStatus(status string) error {
	return s.update(func(st *State) {
		st.ManifestStatus = status
	})
}

// TODO: FIX htis terrible funciton
func (s *Store) UpdateItemStatus(key string, status string) error {
	return s.update(func(st *State) {
		item := s.state.Items[key]
		item.Status = status
		s.state.Items[key] = item
	})
}

func (s *Store) update(apply func(*State)) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	apply(&s.state)
	return s.saveLocked()
}

func (s *Store) load() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := ensureDir(filepath.Dir(s.path)); err != nil {
		return err
	}

	raw, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return s.saveLocked()
	}
	if err != nil {
		return err
	}

	var st State
	if err := json.Unmarshal(raw, &st); err != nil {
		return err
	}
	if st.Items == nil {
		st.Items = make(map[string]ItemState)
	}
	s.state = st
	return nil
}

func (s *Store) saveLocked() error {
	data, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return err
	}

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func ensureDir(path string) error {
	return os.MkdirAll(path, 0o755)
}
