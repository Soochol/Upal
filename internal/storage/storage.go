package storage

import (
	"context"
	"io"
	"time"
)

// FileInfo describes a stored file.
type FileInfo struct {
	ID            string    `json:"id"`
	Filename      string    `json:"filename"`
	ContentType   string    `json:"content_type"`
	Size          int64     `json:"size"`
	Path          string    `json:"path"`
	CreatedAt     time.Time `json:"created_at"`
	ExtractedText string    `json:"extracted_text,omitempty"`
	PreviewText   string    `json:"preview_text,omitempty"`
}

// Storage is the interface for file persistence backends.
type Storage interface {
	// Save stores a file and returns its metadata.
	Save(ctx context.Context, filename string, contentType string, reader io.Reader) (*FileInfo, error)
	// Get retrieves a file by ID.
	Get(ctx context.Context, id string) (*FileInfo, io.ReadCloser, error)
	// Delete removes a file by ID.
	Delete(ctx context.Context, id string) error
	// List returns all stored files.
	List(ctx context.Context) ([]FileInfo, error)
	// UpdateInfo stores updated metadata (e.g. after text extraction).
	UpdateInfo(ctx context.Context, info *FileInfo) error
}
