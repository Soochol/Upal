package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	upalcrypto "github.com/soochol/upal/internal/crypto"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/generate"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/notify"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	runpub "github.com/soochol/upal/internal/services/run"
	scheduler "github.com/soochol/upal/internal/services/scheduler"
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
		switch pc.Type {
		case "anthropic":
			llms[name] = upalmodel.NewAnthropicLLM(pc.APIKey)
		case "gemini":
			geminiURL := strings.TrimRight(pc.URL, "/") + "/v1beta/openai"
			llms[name] = upalmodel.NewOpenAILLM(pc.APIKey,
				upalmodel.WithOpenAIBaseURL(geminiURL),
				upalmodel.WithOpenAIName(name))
		case "claude-code":
			llms[name] = upalmodel.NewClaudeCodeLLM()
		case "gemini-image":
			llms[name] = upalmodel.NewGeminiImageLLM(pc.APIKey)
		case "zimage":
			llms[name] = upalmodel.NewZImageLLM(pc.URL)
		default:
			llms[name] = upalmodel.NewOpenAILLM(pc.APIKey,
				upalmodel.WithOpenAIBaseURL(pc.URL),
				upalmodel.WithOpenAIName(name))
		}
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

	toolReg := tools.NewRegistry()
	toolReg.RegisterNative(tools.WebSearch)
	toolReg.Register(&tools.HTTPRequestTool{})
	toolReg.Register(&tools.PythonExecTool{})
	toolReg.Register(&tools.GetWebpageTool{})
	toolReg.Register(&tools.RSSFeedTool{})
	toolReg.Register(tools.NewContentStoreTool(filepath.Join(dataDir, "content_store.json")))
	toolReg.Register(tools.NewPublishTool(filepath.Join(dataDir, "published")))
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

	// Create WorkflowService for execution orchestration.
	nodeReg := agents.DefaultRegistry()
	workflowSvc := services.NewWorkflowService(repo, llms, sessionService, toolReg, nodeReg)
	runHistorySvc := services.NewRunHistoryService(runRepo)
	runHistorySvc.CleanupOrphanedRuns(context.Background())

	// Create concurrency limiter from config (with defaults).
	concurrencyLimits := upal.ConcurrencyLimits{
		GlobalMax:   cfg.Scheduler.GlobalMax,
		PerWorkflow: cfg.Scheduler.PerWorkflow,
	}
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

	// Connection management (persistent if DB is available).
	memConnRepo := repository.NewMemoryConnectionRepository()
	var connRepo repository.ConnectionRepository = memConnRepo
	if database != nil {
		connRepo = repository.NewPersistentConnectionRepository(memConnRepo, database)
	}
	connEnc, _ := upalcrypto.NewEncryptor(nil)
	connSvc := services.NewConnectionService(connRepo, connEnc)
	srv.SetConnectionService(connSvc)

	// Notification sender registry.
	senderReg := notify.NewSenderRegistry()
	senderReg.Register(&notify.TelegramSender{})
	senderReg.Register(&notify.SlackSender{})
	senderReg.Register(&notify.SMTPSender{})

	// Execution registry for pause/resume (pipeline stage approval).
	execReg := services.NewExecutionRegistry()
	srv.SetExecutionRegistry(execReg)

	// Run manager for background execution with event buffering.
	runManager := services.NewRunManager(15 * time.Minute)
	srv.SetRunManager(runManager)

	// RunPublisher bridges workflow execution into RunManager + RunHistoryService.
	publisher := runpub.NewRunPublisher(workflowSvc, runManager, runHistorySvc)
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
	pipelineRunner.RegisterExecutor(services.NewPassthroughStageExecutor("schedule"))
	pipelineRunner.RegisterExecutor(services.NewPassthroughStageExecutor("trigger"))
	srv.SetPipelineService(pipelineSvc)
	srv.SetPipelineRunner(pipelineRunner)
	schedulerSvc.SetPipelineRunner(pipelineRunner)
	schedulerSvc.SetPipelineService(pipelineSvc)

	// Configure natural language workflow generator if any provider is available.
	skillReg := skills.New()
	srv.SetSkills(skillReg)
	if defaultLLM != nil {
		var toolNames []string
		for _, t := range toolReg.AllTools() {
			toolNames = append(toolNames, t.Name)
		}
		allModels := api.KnownModelsGrouped(cfg.Providers)
		var modelOpts []generate.ModelOption
		for _, m := range allModels {
			modelOpts = append(modelOpts, generate.ModelOption{
				ID:       m.ID,
				Category: string(m.Category),
				Tier:     string(m.Tier),
				Hint:     m.Hint,
			})
		}
		gen := generate.New(defaultLLM, defaultModelName, skillReg, toolNames, modelOpts)
		srv.SetGenerator(gen, defaultModelName)
	}
	srv.SetProviderConfigs(cfg.Providers)

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
	if defaultLLM != nil {
		gen := srv.Generator()
		go func() {
			ctx := context.Background()
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
