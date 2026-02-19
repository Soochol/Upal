package api

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

type Server struct {
	workflows            *WorkflowStore
	llms                 map[string]adkmodel.LLM
	sessionService       session.Service
	toolReg              *tools.Registry
	generator            *generate.Generator
	defaultGenerateModel string
	storage              storage.Storage
	db                   *db.DB
	providerConfigs      map[string]config.ProviderConfig
}

// SetDB configures the database backend. When set, workflow CRUD
// operations persist to PostgreSQL instead of in-memory only.
func (s *Server) SetDB(database *db.DB) {
	s.db = database
}

// SetProviderConfigs stores the provider configuration for model discovery.
func (s *Server) SetProviderConfigs(configs map[string]config.ProviderConfig) {
	s.providerConfigs = configs
}

func NewServer(llms map[string]adkmodel.LLM, sessionService session.Service, toolReg *tools.Registry) *Server {
	return &Server{
		workflows:      NewWorkflowStore(),
		llms:           llms,
		sessionService: sessionService,
		toolReg:        toolReg,
	}
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
			r.Post("/{name}/run", s.runWorkflow)
		})
		r.Post("/generate", s.generateWorkflow)
		r.Post("/nodes/configure", s.configureNode)
		r.Post("/upload", s.uploadFile)
		r.Get("/files", s.listFiles)
		r.Get("/models", s.listModels)
	})

	// Serve static files (frontend)
	r.Handle("/*", StaticHandler("web/dist"))

	return r
}

// SetGenerator configures the natural language workflow generator.
func (s *Server) SetGenerator(gen *generate.Generator, defaultModel string) {
	s.generator = gen
	s.defaultGenerateModel = defaultModel
}

// SetStorage configures the file storage backend.
func (s *Server) SetStorage(store storage.Storage) {
	s.storage = store
}
