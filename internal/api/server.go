package api

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	a2apkg "github.com/soochol/upal/internal/a2a"
	"github.com/soochol/upal/internal/a2atypes"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/storage"
)

type Server struct {
	eventBus             *engine.EventBus
	sessions             *engine.SessionManager
	workflows            *WorkflowStore
	runner               *engine.Runner
	a2aRunner            *engine.A2ARunner
	executors            map[engine.NodeType]engine.NodeExecutorInterface
	generator            *generate.Generator
	defaultGenerateModel string
	storage              storage.Storage
	db                   *db.DB
}

// SetDB configures the database backend. When set, workflow CRUD
// operations persist to PostgreSQL instead of in-memory only.
func (s *Server) SetDB(database *db.DB) {
	s.db = database
}

func NewServer(eventBus *engine.EventBus, sessions *engine.SessionManager, runner *engine.Runner, a2aRunner *engine.A2ARunner, executors map[engine.NodeType]engine.NodeExecutorInterface) *Server {
	return &Server{
		eventBus:  eventBus,
		sessions:  sessions,
		workflows: NewWorkflowStore(),
		runner:    runner,
		a2aRunner: a2aRunner,
		executors: executors,
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
		r.Post("/upload", s.uploadFile)
		r.Get("/files", s.listFiles)
	})

	// A2A protocol endpoints — dynamic node handlers
	r.Route("/a2a", func(r chi.Router) {
		r.Get("/agent-card", s.aggregateAgentCard)
		r.Post("/nodes/{nodeID}", s.handleA2ANode)
		r.Get("/nodes/{nodeID}/agent-card", s.handleA2ANodeCard)
	})

	// Serve static files (frontend)
	r.Handle("/*", StaticHandler("web/dist"))

	return r
}

func (s *Server) aggregateAgentCard(w http.ResponseWriter, r *http.Request) {
	card := a2atypes.AgentCard{
		Name:               "upal",
		Description:        "Upal visual AI workflow platform",
		URL:                getBaseURL(r) + "/a2a",
		Capabilities:       a2atypes.Capabilities{},
		DefaultInputModes:  []string{"text/plain"},
		DefaultOutputModes: []string{"text/plain"},
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func (s *Server) handleA2ANode(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")

	var nodeDef *engine.NodeDefinition
	for _, wf := range s.workflows.List() {
		for i, n := range wf.Nodes {
			if n.ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		if nodeDef != nil {
			break
		}
	}
	if nodeDef == nil {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &a2atypes.JSONRPCError{Code: -32001, Message: fmt.Sprintf("node %q not registered", nodeID)},
		})
		return
	}

	executor, ok := s.executors[nodeDef.Type]
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(a2atypes.JSONRPCResponse{
			JSONRPC: "2.0",
			Error:   &a2atypes.JSONRPCError{Code: -32001, Message: fmt.Sprintf("no executor for type %q", nodeDef.Type)},
		})
		return
	}

	handler := a2apkg.NewNodeHandler(executor, nodeDef)
	handler.ServeHTTP(w, r)
}

func (s *Server) handleA2ANodeCard(w http.ResponseWriter, r *http.Request) {
	nodeID := chi.URLParam(r, "nodeID")

	var nodeDef *engine.NodeDefinition
	for _, wf := range s.workflows.List() {
		for i, n := range wf.Nodes {
			if n.ID == nodeID {
				nodeDef = &wf.Nodes[i]
				break
			}
		}
		if nodeDef != nil {
			break
		}
	}
	if nodeDef == nil {
		http.Error(w, "node not found", http.StatusNotFound)
		return
	}

	card := a2apkg.AgentCardFromNodeDef(nodeDef, getBaseURL(r))
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(card)
}

func getBaseURL(r *http.Request) string {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return fmt.Sprintf("%s://%s", scheme, r.Host)
}
