package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/soochol/upal/internal/a2aclient"
	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/generate"
	"github.com/soochol/upal/internal/nodes"
	"github.com/soochol/upal/internal/provider"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
)

func main() {
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		serve()
		return
	}
	fmt.Println("upal v0.1.0")
	fmt.Println("Usage: upal serve")
}

func serve() {
	cfg, err := config.LoadDefault()
	if err != nil {
		slog.Error("config error", "err", err)
		os.Exit(1)
	}

	eventBus := engine.NewEventBus()
	sessions := engine.NewSessionManager()

	providerReg := provider.NewRegistry()
	for name, pc := range cfg.Providers {
		var p provider.Provider
		switch pc.Type {
		case "anthropic":
			p = provider.NewAnthropicProvider(name, pc.URL, pc.APIKey)
		case "gemini":
			p = provider.NewGeminiProvider(name, pc.URL, pc.APIKey)
		default:
			p = provider.NewOpenAIProvider(name, pc.URL, pc.APIKey)
		}
		providerReg.Register(p)
	}

	toolReg := tools.NewRegistry()
	runner := engine.NewRunner(eventBus, sessions)

	a2aClient := a2aclient.NewClient(http.DefaultClient)
	a2aRunner := engine.NewA2ARunner(eventBus, sessions, a2aClient)

	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:    &nodes.InputNode{},
		engine.NodeTypeAgent:    nodes.NewAgentNode(providerReg, toolReg, eventBus),
		engine.NodeTypeTool:     nodes.NewToolNode(toolReg),
		engine.NodeTypeOutput:   &nodes.OutputNode{},
		engine.NodeTypeExternal: nodes.NewExternalNode(a2aClient),
	}

	srv := api.NewServer(eventBus, sessions, runner, a2aRunner, executors)

	// Optional: Connect to PostgreSQL if database URL is configured.
	if cfg.Database.URL != "" {
		database, err := db.New(context.Background(), cfg.Database.URL)
		if err != nil {
			slog.Warn("database unavailable, using in-memory storage", "err", err)
		} else {
			defer database.Close()
			if err := database.Migrate(context.Background()); err != nil {
				slog.Error("database migration failed", "err", err)
				os.Exit(1)
			}
			srv.SetDB(database)
			slog.Info("database connected", "url", cfg.Database.URL)
		}
	}

	// Configure natural language workflow generator if any provider is available.
	gen := generate.New(providerReg)
	var defaultModel string
	for name, pc := range cfg.Providers {
		switch pc.Type {
		case "gemini":
			defaultModel = name + "/gemini-2.0-flash"
		case "anthropic":
			defaultModel = name + "/claude-sonnet-4-20250514"
		default:
			defaultModel = name + "/gpt-4o"
		}
		break
	}
	srv.SetGenerator(gen, defaultModel)

	// Configure file storage
	store, err := storage.NewLocalStorage("./uploads")
	if err != nil {
		slog.Error("storage error", "err", err)
		os.Exit(1)
	}
	srv.SetStorage(store)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting upal server", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
