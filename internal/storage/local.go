package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/soochol/upal/internal/upal"
)

// LocalStorage stores files on the local filesystem.
type LocalStorage struct {
	baseDir string
	mu      sync.RWMutex
	files   map[string]*FileInfo
}

func NewLocalStorage(baseDir string) (*LocalStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("create storage dir: %w", err)
	}
	return &LocalStorage{
		baseDir: baseDir,
		files:   make(map[string]*FileInfo),
	}, nil
}

func (s *LocalStorage) Save(_ context.Context, filename string, contentType string, reader io.Reader) (*FileInfo, error) {
	id := upal.GenerateID("file")
	ext := filepath.Ext(filename)
	storedName := id + ext
	fullPath := filepath.Join(s.baseDir, storedName)

	f, err := os.Create(fullPath)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}
	defer f.Close()

	n, err := io.Copy(f, reader)
	if err != nil {
		os.Remove(fullPath)
		return nil, fmt.Errorf("write file: %w", err)
	}

	info := &FileInfo{
		ID:          id,
		Filename:    filename,
		ContentType: contentType,
		Size:        n,
		Path:        storedName,
		CreatedAt:   time.Now(),
	}

	s.mu.Lock()
	s.files[id] = info
	s.mu.Unlock()

	return info, nil
}

func (s *LocalStorage) Get(_ context.Context, id string) (*FileInfo, io.ReadCloser, error) {
	s.mu.RLock()
	info, ok := s.files[id]
	s.mu.RUnlock()
	if !ok {
		return nil, nil, fmt.Errorf("file not found %s: %w", id, ErrNotFound)
	}

	fullPath := filepath.Join(s.baseDir, info.Path)
	f, err := os.Open(fullPath)
	if err != nil {
		return nil, nil, fmt.Errorf("open file: %w", err)
	}
	return info, f, nil
}

func (s *LocalStorage) Delete(_ context.Context, id string) error {
	s.mu.Lock()
	info, ok := s.files[id]
	if ok {
		delete(s.files, id)
	}
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("file not found %s: %w", id, ErrNotFound)
	}

	fullPath := filepath.Join(s.baseDir, info.Path)
	return os.Remove(fullPath)
}

func (s *LocalStorage) UpdateInfo(_ context.Context, info *FileInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.files[info.ID]; !ok {
		return fmt.Errorf("file not found: %s", info.ID)
	}
	s.files[info.ID] = info
	return nil
}

func (s *LocalStorage) List(_ context.Context) ([]FileInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]FileInfo, 0, len(s.files))
	for _, info := range s.files {
		result = append(result, *info)
	}
	return result, nil
}
