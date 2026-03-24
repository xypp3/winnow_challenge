package edge

import (
	"encoding/json"
	"errors"
	"fmt"
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

type ManifestState struct {
	ETag    string               `json:"etag"`
	Status  string               `json:"status"` // pending|complete
	Items   map[string]ItemState `json:"items"`
	Updated time.Time            `json:"updated"`
}

type State struct {
	CurrentETag string                   `json:"current_etag"`
	Manifests   map[string]ManifestState `json:"manifests"`
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
			Manifests: make(map[string]ManifestState),
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
	return s.state.CurrentETag
}

func (s *Store) ManifestStatus() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if m, ok := s.state.Manifests[s.state.CurrentETag]; ok {
		return m.Status
	}
	return ""
}

func (s *Store) Snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Store) Item(key string) (ItemState, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.state.Manifests[s.state.CurrentETag]
	if !ok {
		return ItemState{}, false
	}
	item, ok := m.Items[key]
	return item, ok
}

func (s *Store) UpdateItem(key string, item ItemState) error {
	return s.update(func(st *State) error {
		m, ok := st.Manifests[st.CurrentETag]
		if !ok {
			return fmt.Errorf("manifest %s not found", st.CurrentETag)
		}
		if m.Items == nil {
			m.Items = make(map[string]ItemState)
		}
		m.Items[key] = item
		m.Updated = time.Now().UTC()
		st.Manifests[st.CurrentETag] = m
		return nil
	})
}

func (s *Store) UpdateManifestETag(etag string) error {
	return s.update(func(st *State) error {
		st.CurrentETag = etag
		st.Manifests[etag] = ManifestState{
			ETag:    etag,
			Status:  "pending",
			Items:   make(map[string]ItemState),
			Updated: time.Now().UTC(),
		}
		return nil
	})
}

func (s *Store) UpdateManifestStatus(status string) error {
	return s.update(func(st *State) error {
		m, ok := st.Manifests[st.CurrentETag]
		if !ok {
			return fmt.Errorf("manifest %s not found", st.CurrentETag)
		}
		m.Status = status
		m.Updated = time.Now().UTC()
		st.Manifests[st.CurrentETag] = m
		return nil
	})
}

func (s *Store) UpdateItemStatus(key string, status string) error {
	return s.update(func(st *State) error {
		m, ok := st.Manifests[st.CurrentETag]
		if !ok {
			return fmt.Errorf("manifest %s not found", st.CurrentETag)
		}
		item, ok := m.Items[key]
		if !ok {
			return fmt.Errorf("item %s not found", key)
		}
		item.Status = status
		m.Items[key] = item
		m.Updated = time.Now().UTC()
		st.Manifests[st.CurrentETag] = m
		return nil
	})
}

func (s *Store) UpdateCurrentEtag(etag string) error {
	return s.update(func(st *State) error {
		st.CurrentETag = etag
		return nil
	})
}

func (s *Store) update(apply func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := apply(&s.state); err != nil {
		return err
	}
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
	if len(raw) == 0 {
		return s.saveLocked()
	}

	var st State
	if err := json.Unmarshal(raw, &st); err != nil {
		return err
	}
	if st.Manifests == nil {
		st.Manifests = make(map[string]ManifestState)
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
