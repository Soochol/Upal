package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

// writeJSON encodes v as JSON and writes it with 200 OK.
func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}

// writeJSONStatus encodes v as JSON and writes it with the given status code.
func writeJSONStatus(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// decodeJSON reads JSON from the request body into dst.
// Returns false and writes a 400 error if decoding fails.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if err := json.NewDecoder(r.Body).Decode(dst); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return false
	}
	return true
}

// orEmpty returns the slice as-is if non-nil, or an empty slice of the same type.
// This ensures JSON encoding produces [] instead of null.
func orEmpty[T any](s []T) []T {
	if s == nil {
		return []T{}
	}
	return s
}

// writeServiceError maps domain errors to appropriate HTTP status codes.
// Handles ErrNotFound, ErrAlreadyArchived, ErrNotArchived, ErrMustBeArchived,
// ErrInvalidStatus, and falls back to the given defaultStatus.
func writeServiceError(w http.ResponseWriter, err error, defaultStatus int) {
	switch {
	case errors.Is(err, repository.ErrNotFound):
		http.Error(w, err.Error(), http.StatusNotFound)
	case errors.Is(err, upal.ErrAlreadyArchived):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, upal.ErrNotArchived):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, upal.ErrMustBeArchived):
		http.Error(w, err.Error(), http.StatusConflict)
	case errors.Is(err, upal.ErrInvalidStatus):
		http.Error(w, err.Error(), http.StatusBadRequest)
	default:
		http.Error(w, err.Error(), defaultStatus)
	}
}
