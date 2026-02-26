package api

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	runpub "github.com/soochol/upal/internal/services/run"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal/ports"
	adkmodel "google.golang.org/adk/model"
)

type Server struct {
	workflowSvc          ports.WorkflowExecutor
	runHistorySvc        ports.RunHistoryPort
	schedulerSvc         ports.SchedulerPort
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
	retryExecutor        ports.RetryExecutor
	connectionSvc        ports.ConnectionPort
	executionReg         ports.ExecutionRegistryPort
	runManager           ports.RunManagerPort
	runPublisher         *runpub.RunPublisher
	pipelineSvc          ports.PipelineServicePort
	pipelineRunner       ports.PipelineRunner
	contentSvc           ports.ContentSessionPort
	collector            *services.ContentCollector
	publishChannelRepo   repository.PublishChannelRepository
	generationManager    *services.GenerationManager
	aiProviderSvc        *services.AIProviderService
	authSvc              *services.AuthService
	thumbnailTimeout     time.Duration
	uploadMaxSize        int64
}

func (s *Server) SetProviderConfigs(configs map[string]config.ProviderConfig) {
	s.providerConfigs = configs
}

func NewServer(llms map[string]adkmodel.LLM, workflowSvc ports.WorkflowExecutor, repo repository.WorkflowRepository, toolReg *tools.Registry) *Server {
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
		AllowOriginFunc:  func(r *http.Request, origin string) bool { return true },
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
		AllowedHeaders:   []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))
	r.Use(AuthMiddleware(s.authSvc))
	r.Route("/api", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			r.Get("/login/{provider}", s.authLogin)
			r.Get("/callback/{provider}", s.authCallback)
			r.Post("/refresh", s.authRefresh)
			r.Post("/logout", s.authLogout)
			r.Get("/me", s.authMe)
		})
		r.Route("/workflows", func(r chi.Router) {
			r.Post("/", s.createWorkflow)
			r.Get("/", s.listWorkflows)
			r.Post("/suggest-name", s.suggestWorkflowName)
			r.Get("/{name}", s.getWorkflow)
			r.Put("/{name}", s.updateWorkflow)
			r.Delete("/{name}", s.deleteWorkflow)
			r.Post("/{name}/run", s.runWorkflow)
			r.Post("/{name}/thumbnail", s.generateWorkflowThumbnail)
			r.Get("/{name}/runs", s.listWorkflowRuns)
			r.Get("/{name}/triggers", s.listTriggers)
		})
		r.Route("/runs", func(r chi.Router) {
			r.Get("/", s.listRuns)
			r.Get("/{id}", s.getRun)
			r.Get("/{id}/events", s.streamRunEvents)
			r.Post("/{id}/nodes/{nodeId}/resume", s.resumeNode)
		})
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
			r.Post("/{id}/runs/{runId}/approve", s.approvePipelineRun)
			r.Post("/{id}/runs/{runId}/reject", s.rejectPipelineRun)
			r.Get("/{id}/triggers", s.listPipelineTriggers)
			r.Post("/{id}/thumbnail", s.generatePipelineThumbnail)
			r.Post("/{id}/configure", s.configurePipeline)
			if s.contentSvc != nil {
				r.Post("/{id}/collect", s.collectPipeline)
			}
		})
		if s.contentSvc != nil {
			r.Route("/content-sessions", func(r chi.Router) {
				r.Get("/", s.listContentSessions)
				r.Post("/", s.createDraftSession)
				r.Get("/{id}", s.getContentSession)
				r.Patch("/{id}", s.patchContentSession)
				r.Delete("/{id}", s.deleteContentSession)
				r.Patch("/{id}/settings", s.patchSessionSettings)
				r.Post("/{id}/collect", s.collectSession)
				r.Post("/{id}/run", s.runSessionInstance)
				r.Post("/{id}/activate", s.activateSession)
				r.Post("/{id}/deactivate", s.deactivateSession)
				r.Post("/{id}/produce", s.produceContentSession)
				r.Post("/{id}/retry-analyze", s.retryAnalyze)
				r.Post("/{id}/generate-workflow", s.generateAngleWorkflow)
				r.Get("/{id}/sources", s.listSessionSources)
				r.Get("/{id}/analysis", s.getSessionAnalysis)
				r.Patch("/{id}/analysis", s.patchSessionAnalysis)
				r.Post("/{id}/publish", s.publishContentSession)
				r.Post("/{id}/reject-result", s.rejectWorkflowResult)
			r.Post("/{id}/configure", s.configureSession)
			})
			r.Route("/published", func(r chi.Router) {
				r.Get("/", s.listPublished)
			})
			r.Route("/surges", func(r chi.Router) {
				r.Get("/", s.listSurges)
				r.Post("/{id}/dismiss", s.dismissSurge)
				r.Post("/{id}/create-session", s.createSessionFromSurge)
			})
		}
		r.Post("/hooks/{id}", s.handleWebhook)
		r.Post("/generate", s.generateWorkflow)
		r.Get("/generate/{id}", s.getGeneration)
		r.Post("/generate-pipeline", s.generatePipeline)
		r.Post("/generate/backfill", s.backfillDescriptions)
		r.Post("/nodes/configure", s.configureNode)
		r.Post("/upload", s.uploadFile)
		r.Get("/files", s.listFiles)
		r.Get("/files/{id}/serve", s.serveFile)
		r.Delete("/files/{id}", s.deleteFile)
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
		if s.publishChannelRepo != nil {
			r.Route("/publish-channels", func(r chi.Router) {
				r.Post("/", s.createPublishChannel)
				r.Get("/", s.listPublishChannels)
				r.Get("/{id}", s.getPublishChannel)
				r.Put("/{id}", s.updatePublishChannel)
				r.Delete("/{id}", s.deletePublishChannel)
			})
		}
		if s.aiProviderSvc != nil {
			r.Route("/ai-providers", func(r chi.Router) {
				r.Post("/", s.createAIProvider)
				r.Get("/", s.listAIProviders)
				r.Get("/defaults", s.getAIProviderDefaults)
				r.Put("/{id}", s.updateAIProvider)
				r.Delete("/{id}", s.deleteAIProvider)
				r.Put("/{id}/default", s.setAIProviderDefault)
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

func (s *Server) SetGenerator(gen *generate.Generator, defaultModel string) {
	s.generator = gen
	s.defaultGenerateModel = defaultModel
}

func (s *Server) Generator() *generate.Generator {
	return s.generator
}

func (s *Server) SetStorage(store storage.Storage)               { s.storage = store }
func (s *Server) SetSkills(provider skills.Provider)              { s.skills = provider }
func (s *Server) SetRunHistoryService(svc ports.RunHistoryPort)   { s.runHistorySvc = svc }
func (s *Server) SetSchedulerService(svc ports.SchedulerPort)     { s.schedulerSvc = svc }
func (s *Server) SetConcurrencyLimiter(limiter *services.ConcurrencyLimiter) { s.limiter = limiter }
func (s *Server) SetRetryExecutor(executor ports.RetryExecutor)   { s.retryExecutor = executor }
func (s *Server) SetTriggerRepository(repo repository.TriggerRepository) { s.triggerRepo = repo }
func (s *Server) SetConnectionService(svc ports.ConnectionPort)   { s.connectionSvc = svc }
func (s *Server) SetPublishChannelRepo(repo repository.PublishChannelRepository) { s.publishChannelRepo = repo }
func (s *Server) SetExecutionRegistry(reg ports.ExecutionRegistryPort) { s.executionReg = reg }
func (s *Server) SetRunManager(rm ports.RunManagerPort)           { s.runManager = rm }
func (s *Server) SetRunPublisher(pub *runpub.RunPublisher)        { s.runPublisher = pub }
func (s *Server) SetPipelineService(svc ports.PipelineServicePort) { s.pipelineSvc = svc }
func (s *Server) SetPipelineRunner(runner ports.PipelineRunner)   { s.pipelineRunner = runner }
func (s *Server) SetContentSessionService(svc ports.ContentSessionPort) { s.contentSvc = svc }
func (s *Server) SetContentCollector(c *services.ContentCollector) { s.collector = c }
func (s *Server) SetGenerationManager(gm *services.GenerationManager) { s.generationManager = gm }
func (s *Server) SetAIProviderService(svc *services.AIProviderService) { s.aiProviderSvc = svc }
func (s *Server) SetAuthService(svc *services.AuthService)             { s.authSvc = svc }

func (s *Server) SetServerConfig(cfg config.ServerConfig, genCfg config.GeneratorConfig) {
	s.thumbnailTimeout = genCfg.ThumbnailTimeout
	s.uploadMaxSize = cfg.UploadMaxSize
}

// SetA2ABaseURL enables A2A protocol endpoints.
// The URL is used in the AgentCard to advertise the invoke endpoint.
func (s *Server) SetA2ABaseURL(url string) {
	s.a2aBaseURL = url
}

func (s *Server) thumbnailTimeoutOrDefault() time.Duration {
	if s.thumbnailTimeout > 0 {
		return s.thumbnailTimeout
	}
	return 60 * time.Second
}

func (s *Server) uploadMaxSizeOrDefault() int64 {
	if s.uploadMaxSize > 0 {
		return s.uploadMaxSize
	}
	return 50 << 20
}

func (s *Server) listAvailableTools(w http.ResponseWriter, r *http.Request) {
	var result []tools.ToolInfo
	if s.toolReg != nil {
		result = s.toolReg.AllTools()
	}
	writeJSON(w, result)
}

