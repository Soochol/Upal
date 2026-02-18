package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/soochol/upal/internal/engine"
)

type Server struct {
	eventBus  *engine.EventBus
	sessions  *engine.SessionManager
	workflows *WorkflowStore
}

func NewServer(eventBus *engine.EventBus, sessions *engine.SessionManager) *Server {
	return &Server{eventBus: eventBus, sessions: sessions, workflows: NewWorkflowStore()}
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE"},
		AllowedHeaders:   []string{"Content-Type"},
		AllowCredentials: true,
	}))
	r.Route("/api", func(r chi.Router) {
		r.Route("/workflows", func(r chi.Router) {
			r.Post("/", s.createWorkflow)
			r.Get("/", s.listWorkflows)
			r.Get("/{name}", s.getWorkflow)
			r.Put("/{name}", s.updateWorkflow)
			r.Delete("/{name}", s.deleteWorkflow)
		})
	})

	// Serve static files (frontend)
	r.Handle("/*", StaticHandler("web/dist"))

	return r
}
