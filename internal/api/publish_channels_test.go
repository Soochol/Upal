package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/upal"
)

func newTestServerWithChannels() *Server {
	srv := newTestServer()
	srv.SetPublishChannelRepo(repository.NewMemoryPublishChannelRepository())
	return srv
}

func TestPublishChannel_CRUD(t *testing.T) {
	srv := newTestServerWithChannels()

	// Create
	body := `{"name":"My Blog","type":"wordpress"}`
	req := httptest.NewRequest("POST", "/api/publish-channels", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201; body: %s", w.Code, w.Body.String())
	}
	var created upal.PublishChannel
	json.Unmarshal(w.Body.Bytes(), &created)
	if created.Name != "My Blog" || created.Type != "wordpress" || created.ID == "" {
		t.Fatalf("create response: %+v", created)
	}

	// List
	req = httptest.NewRequest("GET", "/api/publish-channels", nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("list: got %d, want 200", w.Code)
	}
	var list []upal.PublishChannel
	json.Unmarshal(w.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("list: got %d items, want 1", len(list))
	}

	// Get
	req = httptest.NewRequest("GET", "/api/publish-channels/"+created.ID, nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("get: got %d, want 200", w.Code)
	}

	// Update
	body = `{"name":"Updated Blog","type":"medium"}`
	req = httptest.NewRequest("PUT", "/api/publish-channels/"+created.ID, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("update: got %d, want 200; body: %s", w.Code, w.Body.String())
	}
	var updated upal.PublishChannel
	json.Unmarshal(w.Body.Bytes(), &updated)
	if updated.Name != "Updated Blog" {
		t.Fatalf("update: got name %q, want Updated Blog", updated.Name)
	}

	// Delete
	req = httptest.NewRequest("DELETE", "/api/publish-channels/"+created.ID, nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204", w.Code)
	}

	// Verify deleted
	req = httptest.NewRequest("GET", "/api/publish-channels/"+created.ID, nil)
	w = httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("get after delete: got %d, want 404", w.Code)
	}
}

func TestPublishChannel_CreateValidation(t *testing.T) {
	srv := newTestServerWithChannels()

	tests := []struct {
		name string
		body string
		code int
	}{
		{"missing name", `{"type":"wordpress"}`, http.StatusBadRequest},
		{"missing type", `{"name":"My Blog"}`, http.StatusBadRequest},
		{"invalid type", `{"name":"My Blog","type":"instagram"}`, http.StatusBadRequest},
		{"empty body", `{}`, http.StatusBadRequest},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/api/publish-channels", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			srv.Handler().ServeHTTP(w, req)
			if w.Code != tt.code {
				t.Errorf("got %d, want %d; body: %s", w.Code, tt.code, w.Body.String())
			}
		})
	}
}
