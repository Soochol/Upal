package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/soochol/upal/internal/extract"
	"github.com/soochol/upal/internal/storage"
)

const maxUploadSize = 50 << 20 // 50MB

func (s *Server) uploadFile(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "file storage not configured", http.StatusServiceUnavailable)
		return
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		http.Error(w, "file too large (max 50MB)", http.StatusBadRequest)
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "missing file field", http.StatusBadRequest)
		return
	}
	defer file.Close()

	contentType := header.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Buffer body to allow reading twice (save + extract)
	body, err := io.ReadAll(file)
	if err != nil {
		http.Error(w, "read file", http.StatusInternalServerError)
		return
	}

	info, err := s.storage.Save(r.Context(), header.Filename, contentType, bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Best-effort text extraction — never fail the upload
	if extracted, _ := extract.Extract(contentType, bytes.NewReader(body)); extracted != "" {
		info.ExtractedText = extracted
		preview := []rune(extracted)
		if len(preview) > 300 {
			preview = preview[:300]
		}
		info.PreviewText = string(preview)
		_ = s.storage.UpdateInfo(r.Context(), info)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(info)
}

func (s *Server) listFiles(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "file storage not configured", http.StatusServiceUnavailable)
		return
	}

	files, err := s.storage.List(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "file storage not configured", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	info, rc, err := s.storage.Get(r.Context(), id)
	if err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	defer rc.Close()

	escaped := strings.ReplaceAll(info.Filename, `"`, `\"`)
	w.Header().Set("Content-Type", info.ContentType)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, escaped))
	if _, err := io.Copy(w, rc); err != nil {
		slog.Warn("serveFile: copy interrupted", "id", id, "err", err)
	}
}

func (s *Server) deleteFile(w http.ResponseWriter, r *http.Request) {
	if s.storage == nil {
		http.Error(w, "file storage not configured", http.StatusServiceUnavailable)
		return
	}
	id := chi.URLParam(r, "id")
	if err := s.storage.Delete(r.Context(), id); err != nil {
		if errors.Is(err, storage.ErrNotFound) {
			http.Error(w, "not found", http.StatusNotFound)
		} else {
			http.Error(w, "internal server error", http.StatusInternalServerError)
		}
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

