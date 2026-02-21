package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
	adkmodel "google.golang.org/adk/model"
)

type Server struct {
	workflowSvc          *services.WorkflowService
	runHistorySvc        *services.RunHistoryService
	schedulerSvc         *services.SchedulerService
	limiter              *services.ConcurrencyLimiter
	repo                 repository.WorkflowRepository
	triggerRepo          repository.TriggerRepository
	llms                 map[string]adkmodel.LLM
	toolReg              *tools.Registry
	generator            *generate.Generator
	defaultGenerateModel string
	storage              storage.Storage
	providerConfigs      map[string]config.ProviderConfig
	skills               skills.Provider
	a2aBaseURL           string
	retryExecutor        *services.RetryExecutor
	connectionSvc        *services.ConnectionService
	executionReg         *services.ExecutionRegistry
	runManager           *services.RunManager
	pipelineSvc          *services.PipelineService
	pipelineRunner       *services.PipelineRunner
}

// SetProviderConfigs stores the provider configuration for model discovery.
func (s *Server) SetProviderConfigs(configs map[string]config.ProviderConfig) {
	s.providerConfigs = configs
}

func NewServer(llms map[string]adkmodel.LLM, workflowSvc *services.WorkflowService, repo repository.WorkflowRepository, toolReg *tools.Registry) *Server {
	return &Server{
		workflowSvc: workflowSvc,
		repo:        repo,
		llms:        llms,
		toolReg:     toolReg,
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
			r.Post("/suggest-name", s.suggestWorkflowName)
			r.Get("/{name}", s.getWorkflow)
			r.Put("/{name}", s.updateWorkflow)
			r.Delete("/{name}", s.deleteWorkflow)
			r.Post("/{name}/run", s.runWorkflow)
			r.Get("/{name}/runs", s.listWorkflowRuns)
			r.Get("/{name}/triggers", s.listTriggers)
		})
		r.Route("/runs", func(r chi.Router) {
			r.Get("/", s.listRuns)
			r.Get("/{id}", s.getRun)
			r.Get("/{id}/events", s.streamRunEvents)
			r.Post("/{id}/nodes/{nodeId}/resume", s.resumeNode)
		})
		r.Route("/schedules", func(r chi.Router) {
			r.Post("/", s.createSchedule)
			r.Get("/", s.listSchedules)
			r.Get("/{id}", s.getSchedule)
			r.Put("/{id}", s.updateSchedule)
			r.Delete("/{id}", s.deleteSchedule)
			r.Post("/{id}/pause", s.pauseSchedule)
			r.Post("/{id}/resume", s.resumeSchedule)
			r.Post("/{id}/trigger", s.triggerSchedule)
		})
		r.Get("/scheduler/stats", s.getSchedulerStats)
		r.Route("/triggers", func(r chi.Router) {
			r.Post("/", s.createTrigger)
			r.Delete("/{id}", s.deleteTrigger)
		})
		r.Route("/pipelines", func(r chi.Router) {
			r.Post("/", s.createPipeline)
			r.Get("/", s.listPipelines)
			r.Get("/{id}", s.getPipeline)
			r.Put("/{id}", s.updatePipeline)
			r.Delete("/{id}", s.deletePipeline)
			r.Post("/{id}/start", s.startPipeline)
			r.Get("/{id}/runs", s.listPipelineRuns)
			r.Post("/{id}/stages/{stageId}/approve", s.approvePipelineStage)
			r.Post("/{id}/stages/{stageId}/reject", s.rejectPipelineStage)
		})
		r.Post("/hooks/{id}", s.handleWebhook)
		r.Post("/generate", s.generateWorkflow)
		r.Post("/nodes/configure", s.configureNode)
		r.Post("/upload", s.uploadFile)
		r.Get("/files", s.listFiles)
		r.Get("/models", s.listModels)
		r.Get("/tools", s.listAvailableTools)
		if s.connectionSvc != nil {
			r.Route("/connections", func(r chi.Router) {
				r.Post("/", s.createConnection)
				r.Get("/", s.listConnections)
				r.Get("/{id}", s.getConnection)
				r.Put("/{id}", s.updateConnection)
				r.Delete("/{id}", s.deleteConnection)
			})
		}
	})

	// A2A protocol endpoints (agent card + JSON-RPC).
	if s.a2aBaseURL != "" {
		s.setupA2ARoutes(r)
	}

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

// SetSkills configures the skill provider for LLM-guided node configuration.
func (s *Server) SetSkills(provider skills.Provider) {
	s.skills = provider
}

// SetRunHistoryService configures the run history service.
func (s *Server) SetRunHistoryService(svc *services.RunHistoryService) {
	s.runHistorySvc = svc
}

// SetSchedulerService configures the scheduler service.
func (s *Server) SetSchedulerService(svc *services.SchedulerService) {
	s.schedulerSvc = svc
}

// SetConcurrencyLimiter configures the concurrency limiter.
func (s *Server) SetConcurrencyLimiter(limiter *services.ConcurrencyLimiter) {
	s.limiter = limiter
}

// SetRetryExecutor configures the retry executor.
func (s *Server) SetRetryExecutor(executor *services.RetryExecutor) {
	s.retryExecutor = executor
}

// SetTriggerRepository configures the trigger repository.
func (s *Server) SetTriggerRepository(repo repository.TriggerRepository) {
	s.triggerRepo = repo
}

// SetConnectionService configures the connection management service.
func (s *Server) SetConnectionService(svc *services.ConnectionService) {
	s.connectionSvc = svc
}

// SetExecutionRegistry configures the execution registry for pause/resume.
func (s *Server) SetExecutionRegistry(reg *services.ExecutionRegistry) {
	s.executionReg = reg
}

// SetRunManager configures the run manager for background execution.
func (s *Server) SetRunManager(rm *services.RunManager) {
	s.runManager = rm
}

// SetPipelineService configures the pipeline management service.
func (s *Server) SetPipelineService(svc *services.PipelineService) {
	s.pipelineSvc = svc
}

// SetPipelineRunner configures the pipeline runner for stage execution.
func (s *Server) SetPipelineRunner(runner *services.PipelineRunner) {
	s.pipelineRunner = runner
}

// SetA2ABaseURL enables A2A protocol endpoints on the server.
// The URL is used in the AgentCard to advertise the invoke endpoint.
func (s *Server) SetA2ABaseURL(url string) {
	s.a2aBaseURL = url
}

// listAvailableTools returns all tools available for agent nodes.
func (s *Server) listAvailableTools(w http.ResponseWriter, r *http.Request) {
	type toolInfo struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}

	var result []toolInfo
	if s.toolReg != nil {
		for _, t := range s.toolReg.AllTools() {
			result = append(result, toolInfo{Name: t.Name, Description: t.Description})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// resolvedModel holds a resolved LLM and its model name.
type resolvedModel struct {
	llm   adkmodel.LLM
	model string
}

// resolveModel parses a "provider/model" string and returns the matching LLM.
func (s *Server) resolveModel(modelID string) (resolvedModel, bool) {
	parts := strings.SplitN(modelID, "/", 2)
	if len(parts) != 2 {
		return resolvedModel{}, false
	}
	provider, model := parts[0], parts[1]
	llm, ok := s.llms[provider]
	if !ok {
		return resolvedModel{}, false
	}
	return resolvedModel{llm: llm, model: model}, true
}
