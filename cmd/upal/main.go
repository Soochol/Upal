package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	upalcrypto "github.com/soochol/upal/internal/crypto"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	runpub "github.com/soochol/upal/internal/services/run"
	"github.com/soochol/upal/internal/services/scheduler"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/adk/session"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
	fmt.Println("upal v0.2.0")
	fmt.Println("Usage: upal serve")
}

func serve() {
	cfg, err := config.LoadDefault()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	llms := make(map[string]adkmodel.LLM)
	providerTypes := make(map[string]string) // name → type

	for name, pc := range cfg.Providers {
		llm, ok := upalmodel.BuildLLM(name, pc)
		if !ok {
			slog.Warn("unknown provider type, skipping", "name", name, "type", pc.Type)
			continue
		}
		llms[name] = llm
		providerTypes[name] = pc.Type
	}

	// Pick default LLM with deterministic priority order.
	// claude-code first (no API key needed), then anthropic, gemini, others.
	var defaultLLM adkmodel.LLM
	var defaultModelName string
	defaultPriority := []struct {
		typ   string
		model string
	}{
		{"claude-code", "sonnet"},
		{"anthropic", "claude-sonnet-4-6"},
		{"gemini", "gemini-2.0-flash"},
		{"openai", "gpt-4o"},
	}
	for _, p := range defaultPriority {
		for name, typ := range providerTypes {
			if typ == p.typ {
				defaultLLM = llms[name]
				defaultModelName = p.model
				break
			}
		}
		if defaultLLM != nil {
			break
		}
	}
	// Fallback: pick any remaining provider.
	if defaultLLM == nil {
		for name := range llms {
			defaultLLM = llms[name]
			defaultModelName = "gpt-4o"
			break
		}
	}

	dataDir := "data"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		slog.Error("failed to create data directory", "err", err)
		os.Exit(1)
	}

	outputDir := "outputs"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		slog.Error("failed to create outputs directory", "err", err)
		os.Exit(1)
	}

	toolReg := tools.NewRegistry()
	toolReg.RegisterNative(tools.WebSearch)
	toolReg.Register(&tools.HTTPRequestTool{})
	toolReg.Register(&tools.PythonExecTool{})
	toolReg.Register(&tools.GetWebpageTool{})
	toolReg.Register(&tools.RSSFeedTool{})
	toolReg.Register(tools.NewContentStoreTool(filepath.Join(dataDir, "content_store.json")))
	toolReg.Register(tools.NewPublishTool(filepath.Join(dataDir, "published")))
	toolReg.Register(&tools.VideoMergeTool{OutputDir: outputDir})
	toolReg.Register(&tools.RemotionRenderTool{OutputDir: outputDir})
	sessionService := session.InMemoryService()

	// Optional: Connect to PostgreSQL if database URL is configured.
	var database *db.DB
	if cfg.Database.URL != "" {
		d, err := db.New(context.Background(), cfg.Database.URL)
		if err != nil {
			slog.Warn("database unavailable, using in-memory storage", "err", err)
		} else {
			database = d
			defer database.Close()
			if err := database.Migrate(context.Background()); err != nil {
				slog.Error("database migration failed", "err", err)
				os.Exit(1)
			}
			slog.Info("database connected", "url", cfg.Database.URL)
		}
	}

	// Create auth service (requires database for user storage).
	var authSvc *services.AuthService
	if database != nil {
		baseURL := fmt.Sprintf("http://%s:%d", cfg.Server.Host, cfg.Server.Port)
		authSvc = services.NewAuthService(database, cfg.Auth, baseURL)
	}

	// Create workflow repository (in-memory, or persistent if DB is available).
	memRepo := repository.NewMemory()
	var repo repository.WorkflowRepository = memRepo
	if database != nil {
		repo = repository.NewPersistent(memRepo, database)
	}

	// Create run history repository (in-memory, or persistent if DB is available).
	memRunRepo := repository.NewMemoryRunRepository()
	var runRepo repository.RunRepository = memRunRepo
	if database != nil {
		runRepo = repository.NewPersistentRunRepository(memRunRepo, database)
	}

	// Create schedule repository (in-memory, or persistent if DB is available).
	memScheduleRepo := repository.NewMemoryScheduleRepository()
	var scheduleRepo repository.ScheduleRepository = memScheduleRepo
	if database != nil {
		scheduleRepo = repository.NewPersistentScheduleRepository(memScheduleRepo, database)
	}

	// Create trigger repository (in-memory, or persistent if DB is available).
	memTriggerRepo := repository.NewMemoryTriggerRepository()
	var triggerRepo repository.TriggerRepository = memTriggerRepo
	if database != nil {
		triggerRepo = repository.NewPersistentTriggerRepository(memTriggerRepo, database)
	}

	// Skills registry — created early so prompts are available to services.
	skillReg := skills.New()

	// Create LLM resolver for "provider/model" → LLM mapping.
	resolver := llmutil.NewMapResolver(llms, defaultLLM, defaultModelName)

	// Create WorkflowService for execution orchestration.
	nodeReg := agents.DefaultRegistry()
	workflowSvc := services.NewWorkflowService(repo, llms, sessionService, toolReg, nodeReg, outputDir, skillReg.GetPrompt("html-layout"), resolver)
	runHistorySvc := services.NewRunHistoryService(runRepo)
	runHistorySvc.CleanupOrphanedRuns(context.Background())

	// Create concurrency limiter from config (with defaults).
	concurrencyLimits := cfg.Scheduler
	if concurrencyLimits.GlobalMax <= 0 {
		concurrencyLimits.GlobalMax = 10
	}
	if concurrencyLimits.PerWorkflow <= 0 {
		concurrencyLimits.PerWorkflow = 3
	}
	limiter := services.NewConcurrencyLimiter(concurrencyLimits)

	// Create retry executor and scheduler service.
	retryExecutor := services.NewRetryExecutor(workflowSvc, runHistorySvc)
	schedulerSvc := scheduler.NewSchedulerService(
		scheduleRepo, workflowSvc, retryExecutor, limiter, runHistorySvc,
	)

	// Start the scheduler (loads existing schedules from repo).
	if err := schedulerSvc.Start(context.Background()); err != nil {
		slog.Error("scheduler start failed", "err", err)
		os.Exit(1)
	}
	defer schedulerSvc.Stop()

	srv := api.NewServer(llms, workflowSvc, repo, toolReg)
	srv.SetRunHistoryService(runHistorySvc)
	srv.SetSchedulerService(schedulerSvc)
	srv.SetConcurrencyLimiter(limiter)
	srv.SetRetryExecutor(retryExecutor)
	srv.SetTriggerRepository(triggerRepo)
	if authSvc != nil {
		srv.SetAuthService(authSvc)
	}

	// Connection management (persistent if DB is available).
	memConnRepo := repository.NewMemoryConnectionRepository()
	var connRepo repository.ConnectionRepository = memConnRepo
	if database != nil {
		connRepo = repository.NewPersistentConnectionRepository(memConnRepo, database)
	}
	enc, _ := upalcrypto.NewEncryptor(nil)
	connSvc := services.NewConnectionService(connRepo, enc)
	srv.SetConnectionService(connSvc)

	// AI provider management (persistent if DB is available).
	memAIProviderRepo := repository.NewMemoryAIProviderRepository()
	var aiProviderRepo repository.AIProviderRepository = memAIProviderRepo
	if database != nil {
		aiProviderRepo = repository.NewPersistentAIProviderRepository(memAIProviderRepo, database)
	}
	aiProviderSvc := services.NewAIProviderService(aiProviderRepo, enc)
	srv.SetAIProviderService(aiProviderSvc)

	// Merge DB-registered AI providers into LLM pool.
	if dbProviders, err := aiProviderSvc.ListAll(context.Background()); err == nil {
		for _, p := range dbProviders {
			// Skip if already configured from config.yaml
			if _, exists := llms[p.Name]; exists {
				continue
			}
			pc := config.ProviderConfig{
				Type:   p.Type,
				APIKey: p.APIKey,
				URL:    upalmodel.DefaultURLForType(p.Type),
			}
			if llm, ok := upalmodel.BuildLLM(p.Name, pc); ok {
				llms[p.Name] = llm
				providerTypes[p.Name] = p.Type
			} else {
				slog.Warn("failed to build LLM from DB provider, skipping", "name", p.Name, "type", p.Type)
			}
		}
		// Override default LLM if a DB provider is marked as default.
		for _, p := range dbProviders {
			if p.IsDefault && string(p.Category) == "llm" {
				if llm, ok := llms[p.Name]; ok {
					defaultLLM = llm
					if modelName, ok := upalmodel.FirstModelForType(p.Type); ok {
						defaultModelName = modelName
					}
				}
			}
		}
		// Update resolver with potentially new defaults.
		resolver = llmutil.NewMapResolver(llms, defaultLLM, defaultModelName)
	}

	// Build effective provider configs by merging config.yaml + DB providers.
	effectiveProviders := make(map[string]config.ProviderConfig, len(cfg.Providers))
	for k, v := range cfg.Providers {
		effectiveProviders[k] = v
	}
	for name, typ := range providerTypes {
		if _, exists := effectiveProviders[name]; !exists {
			effectiveProviders[name] = config.ProviderConfig{Type: typ}
		}
	}

	// Publish channel management.
	publishChannelRepo := repository.NewMemoryPublishChannelRepository()
	srv.SetPublishChannelRepo(publishChannelRepo)

	// Notification sender registry.
	senderReg := notify.NewSenderRegistry()
	senderReg.Register(&notify.TelegramSender{})
	senderReg.Register(&notify.SlackSender{})
	senderReg.Register(&notify.SMTPSender{})

	// Execution registry for pause/resume (pipeline stage approval).
	execReg := services.NewExecutionRegistry()
	srv.SetExecutionRegistry(execReg)

	// Run manager for background execution with event buffering.
	runManager := services.NewRunManager(cfg.Runs.TTL)
	defer runManager.Stop()
	srv.SetRunManager(runManager)

	// Generation manager for background LLM generation (workflow, pipeline).
	genManager := services.NewGenerationManager(cfg.Runs.TTL)
	defer genManager.Stop()
	srv.SetGenerationManager(genManager)

	// RunPublisher bridges workflow execution into RunManager + RunHistoryService.
	publisher := runpub.NewRunPublisher(workflowSvc, runManager, runHistorySvc, execReg)
	srv.SetRunPublisher(publisher)

	// Pipeline
	memPipelineRepo := repository.NewMemoryPipelineRepository()
	var pipelineRepo repository.PipelineRepository = memPipelineRepo
	if database != nil {
		pipelineRepo = repository.NewPersistentPipelineRepository(memPipelineRepo, database)
	}

	memPipelineRunRepo := repository.NewMemoryPipelineRunRepository()
	var pipelineRunRepo repository.PipelineRunRepository = memPipelineRunRepo
	if database != nil {
		pipelineRunRepo = repository.NewPersistentPipelineRunRepository(memPipelineRunRepo, database)
	}

	pipelineSvc := services.NewPipelineService(pipelineRepo, pipelineRunRepo)
	pipelineRunner := services.NewPipelineRunner(pipelineRunRepo)
	pipelineRunner.RegisterExecutor(services.NewWorkflowStageExecutor(workflowSvc))
	pipelineRunner.RegisterExecutor(services.NewApprovalStageExecutor(senderReg, connSvc))
	pipelineRunner.RegisterExecutor(services.NewNotificationStageExecutor(senderReg, connSvc))
	pipelineRunner.RegisterExecutor(&services.TransformStageExecutor{})
	pipelineRunner.RegisterExecutor(services.NewCollectStageExecutor(resolver, skillReg, toolReg))
	pipelineRunner.RegisterExecutor(services.NewPassthroughStageExecutor("schedule"))
	pipelineRunner.RegisterExecutor(services.NewPassthroughStageExecutor("trigger"))
	srv.SetPipelineService(pipelineSvc)
	srv.SetPipelineRunner(pipelineRunner)
	schedulerSvc.SetPipelineRunner(pipelineRunner)
	schedulerSvc.SetPipelineService(pipelineSvc)

	// Content media pipeline
	memContentSessionRepo := repository.NewMemoryContentSessionRepository()
	memSourceFetchRepo := repository.NewMemorySourceFetchRepository()
	memLLMAnalysisRepo := repository.NewMemoryLLMAnalysisRepository()
	memPublishedRepo := repository.NewMemoryPublishedContentRepository()
	memSurgeRepo := repository.NewMemorySurgeEventRepository()
	memWFResultRepo := repository.NewMemoryWorkflowResultRepository()

	var contentSessionRepo repository.ContentSessionRepository = memContentSessionRepo
	var sourceFetchRepo repository.SourceFetchRepository = memSourceFetchRepo
	var llmAnalysisRepo repository.LLMAnalysisRepository = memLLMAnalysisRepo
	var publishedRepo repository.PublishedContentRepository = memPublishedRepo
	var surgeRepo repository.SurgeEventRepository = memSurgeRepo
	var wfResultRepo repository.WorkflowResultRepository = memWFResultRepo

	if database != nil {
		contentSessionRepo = repository.NewPersistentContentSessionRepository(memContentSessionRepo, database)
		sourceFetchRepo = repository.NewPersistentSourceFetchRepository(memSourceFetchRepo, database)
		llmAnalysisRepo = repository.NewPersistentLLMAnalysisRepository(memLLMAnalysisRepo, database)
		publishedRepo = repository.NewPersistentPublishedContentRepository(memPublishedRepo, database)
		surgeRepo = repository.NewPersistentSurgeEventRepository(memSurgeRepo, database)
		wfResultRepo = repository.NewPersistentWorkflowResultRepository(memWFResultRepo, database)
	}

	contentSvc := services.NewContentSessionService(
		contentSessionRepo, sourceFetchRepo, llmAnalysisRepo, publishedRepo, surgeRepo, wfResultRepo,
	)
	contentSvc.SetPipelineRepository(pipelineRepo)
	srv.SetContentSessionService(contentSvc)

	srv.SetSkills(skillReg)

	// New Session/Run services (coexist with old pipeline/content-session services).
	memSessionRepo := repository.NewMemorySessionRepository()
	var sessionRepo repository.SessionRepository = memSessionRepo
	memSessionRunRepo := repository.NewMemorySessionRunRepository()
	var sessionRunRepo repository.SessionRunRepository = memSessionRunRepo
	memWFRunRepo := repository.NewMemoryWorkflowRunRepository()
	var wfRunRepo repository.WorkflowRunRepository = memWFRunRepo

	if database != nil {
		sessionRepo = repository.NewPersistentSessionRepository(memSessionRepo, database)
		sessionRunRepo = repository.NewPersistentSessionRunRepository(memSessionRunRepo, database)
		wfRunRepo = repository.NewPersistentWorkflowRunRepository(memWFRunRepo, database)
	}

	sessionSvc := services.NewSessionService(sessionRepo)
	runSvc := services.NewRunService(sessionRunRepo, sessionRepo, sourceFetchRepo, llmAnalysisRepo, publishedRepo, surgeRepo, wfRunRepo)
	srv.SetSessionService(sessionSvc)
	srv.SetRunService(runSvc)

	// Wire content collector for actual source fetching and workflow execution.
	var collector *services.ContentCollector
	if defaultLLM != nil {
		collector = services.NewContentCollector(
			contentSvc,
			services.NewCollectStageExecutor(resolver, skillReg, toolReg),
			workflowSvc,
			repo,
			pipelineRepo,
			resolver,
			skillReg,
			runHistorySvc,
		)
		collector.SetSessionService(sessionSvc)
		collector.SetRunService(runSvc)
		srv.SetContentCollector(collector)
		schedulerSvc.SetContentCollector(collector)
	}

	// Configure natural language workflow generator if any provider is available.
	if defaultLLM != nil {
		var toolInfos []upal.ToolSummary
		for _, t := range toolReg.AllTools() {
			toolInfos = append(toolInfos, upal.ToolSummary{Name: t.Name, Description: t.Description})
		}
		allModels := upalmodel.KnownModelsGrouped(effectiveProviders)
		var modelOpts []upal.ModelSummary
		for _, m := range allModels {
			modelOpts = append(modelOpts, upal.ModelSummary{
				ID:       m.ID,
				Category: string(m.Category),
				Tier:     string(m.Tier),
				Hint:     m.Hint,
			})
		}
		gen := generate.New(defaultLLM, defaultModelName, skillReg, toolInfos, modelOpts)
		gen.SetLLMResolver(resolver)
		gen.SetDefaultLLMFunc(func(ctx context.Context) (adkmodel.LLM, string, error) {
			providers, err := aiProviderSvc.ListAll(ctx)
			if err != nil {
				return nil, "", err
			}
			for _, p := range providers {
				if p.IsDefault && p.Category == upal.AICategoryLLM {
					pc := config.ProviderConfig{
						Type:   p.Type,
						APIKey: p.APIKey,
						URL:    upalmodel.DefaultURLForType(p.Type),
					}
					built, ok := upalmodel.BuildLLM(p.Name, pc)
					if !ok {
						return nil, "", fmt.Errorf("failed to build LLM for provider %s", p.Name)
					}
					modelName := p.Model
					if modelName == "" {
						modelName, _ = upalmodel.FirstModelForType(p.Type)
					}
					return built, modelName, nil
				}
			}
			return nil, "", fmt.Errorf("no default LLM provider configured")
		})
		srv.SetGenerator(gen, defaultModelName)
		if collector != nil {
			collector.SetGenerator(gen)
		}
	}
	srv.SetProviderConfigs(effectiveProviders)
	srv.SetServerConfig(cfg.Server, cfg.Generator)

	// Enable A2A protocol endpoints.
	a2aURL := fmt.Sprintf("http://localhost:%d", cfg.Server.Port)
	srv.SetA2ABaseURL(a2aURL)
	slog.Info("A2A enabled", "card", a2aURL+"/.well-known/agent-card.json")

	// Configure file storage
	store, err := storage.NewLocalStorage("./uploads")
	if err != nil {
		slog.Error("storage error", "err", err)
		os.Exit(1)
	}
	srv.SetStorage(store)
	nodeReg.Register(agents.NewAssetNodeBuilder(store))

	// Backfill missing descriptions for existing workflows and pipeline stages.
	// Runs in the background on startup so it doesn't block the server.
	// Uses "default" user context — only processes unowned data.
	// TODO: iterate over all users when multi-tenant backfill is needed.
	if defaultLLM != nil {
		gen := srv.Generator()
		go func() {
			ctx := upal.WithUserID(context.Background(), "default")
			wfs, _ := repo.List(ctx)
			for _, wf := range gen.BackfillWorkflowDescriptions(ctx, wfs) {
				if err := repo.Update(ctx, wf.Name, wf); err != nil {
					slog.Warn("backfill: workflow update failed", "name", wf.Name, "err", err)
				}
			}
			for _, wf := range gen.BackfillNodeDescriptions(ctx, wfs) {
				if err := repo.Update(ctx, wf.Name, wf); err != nil {
					slog.Warn("backfill: node descriptions update failed", "name", wf.Name, "err", err)
				}
			}
			pipes, _ := pipelineSvc.List(ctx)
			for _, p := range pipes {
				if generate.BackfillStageDescriptions(p) {
					if err := pipelineSvc.Update(ctx, p); err != nil {
						slog.Warn("backfill: pipeline update failed", "id", p.ID, "err", err)
					}
				}
			}
		}()
	}

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting upal server", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
