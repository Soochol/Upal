package api

import (
	"encoding/json"
	"net/http"
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

	info, err := s.storage.Save(r.Context(), header.Filename, contentType, file)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
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

