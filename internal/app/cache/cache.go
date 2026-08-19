package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	indexName = "index.json"
	blobDir   = "blobs"
)

type entry struct {
	File    string    `json:"file"`
	Size    int64     `json:"size"`
	Fetched time.Time `json:"fetched"`
}

type Store struct {
	dir string

	mu    sync.Mutex
	index map[string]entry
}

func Open(dir string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(dir, blobDir), 0o755); err != nil {
		return nil, err
	}
	s := &Store{dir: dir, index: map[string]entry{}}
	raw, err := os.ReadFile(filepath.Join(dir, indexName))
	if err == nil {
		var idx map[string]entry
		if json.Unmarshal(raw, &idx) == nil {
			s.index = idx
		}
	}
	s.pruneOrphans()
	return s, nil
}

// pruneOrphans drops index entries whose blob file is missing, which can
// happen if the process crashed between writing the blob and the index.
func (s *Store) pruneOrphans() {
	var dirty bool
	for key, e := range s.index {
		if _, err := os.Stat(filepath.Join(s.dir, blobDir, e.File)); err != nil {
			delete(s.index, key)
			dirty = true
		}
	}
	if dirty {
		if raw, err := json.MarshalIndent(s.index, "", "  "); err == nil {
			_ = writeFileAtomic(filepath.Join(s.dir, indexName), raw)
		}
	}
}

func (s *Store) Dir() string { return s.dir }

func (s *Store) Get(key string) ([]byte, time.Time, bool) {
	if s == nil {
		return nil, time.Time{}, false
	}
	s.mu.Lock()
	e, ok := s.index[key]
	s.mu.Unlock()
	if !ok {
		return nil, time.Time{}, false
	}
	data, err := os.ReadFile(filepath.Join(s.dir, blobDir, e.File))
	if err != nil {
		return nil, time.Time{}, false
	}
	return data, e.Fetched, true
}

func (s *Store) Put(key string, data []byte) error {
	if s == nil {
		return nil
	}
	name := blobName(key)
	if err := writeFileAtomic(filepath.Join(s.dir, blobDir, name), data); err != nil {
		return err
	}

	s.mu.Lock()
	s.index[key] = entry{File: name, Size: int64(len(data)), Fetched: time.Now()}
	raw, err := json.MarshalIndent(s.index, "", "  ")
	s.mu.Unlock()
	if err != nil {
		return err
	}
	return writeFileAtomic(filepath.Join(s.dir, indexName), raw)
}

func (s *Store) Age(key string) (time.Duration, bool) {
	if s == nil {
		return 0, false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.index[key]
	if !ok {
		return 0, false
	}
	return time.Since(e.Fetched), true
}

func (s *Store) Newest() time.Time {
	if s == nil {
		return time.Time{}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	var newest time.Time
	for _, e := range s.index {
		if e.Fetched.After(newest) {
			newest = e.Fetched
		}
	}
	return newest
}

func (s *Store) Stats() (files int, bytes int64) {
	if s == nil {
		return 0, 0
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, e := range s.index {
		files++
		bytes += e.Size
	}
	return files, bytes
}

func (s *Store) Clear() error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	s.index = map[string]entry{}
	s.mu.Unlock()
	if err := os.RemoveAll(filepath.Join(s.dir, blobDir)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(s.dir, blobDir), 0o755); err != nil {
		return err
	}
	return os.Remove(filepath.Join(s.dir, indexName))
}

func blobName(key string) string {
	name := strings.TrimLeft(key, "/")
	name = strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9':
			return r
		case r == '.', r == '-', r == '_':
			return r
		}
		return '_'
	}, name)
	sum := sha256.Sum256([]byte(key))
	suffix := hex.EncodeToString(sum[:])[:8]
	if name == "" {
		return suffix
	}
	return name + "-" + suffix
}

func writeFileAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".tmp-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), path)
}
