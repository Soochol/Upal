package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	_ "github.com/lib/pq" // PostgreSQL driver

	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/db"
	"github.com/soochol/upal/internal/generate"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/storage"
	"github.com/soochol/upal/internal/tools"
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
	var defaultLLM adkmodel.LLM
	var defaultModelName string

	for name, pc := range cfg.Providers {
		switch pc.Type {
		case "anthropic":
			llms[name] = upalmodel.NewAnthropicLLM(pc.APIKey)
			if defaultLLM == nil {
				defaultLLM = llms[name]
				defaultModelName = "claude-sonnet-4-20250514"
			}
		case "gemini":
			// Gemini's OpenAI-compatible endpoint requires /v1beta/openai suffix
			geminiURL := strings.TrimRight(pc.URL, "/") + "/v1beta/openai"
			llms[name] = upalmodel.NewOpenAILLM(pc.APIKey,
				upalmodel.WithOpenAIBaseURL(geminiURL),
				upalmodel.WithOpenAIName(name))
			if defaultLLM == nil {
				defaultLLM = llms[name]
				defaultModelName = "gemini-2.0-flash"
			}
		default:
			llms[name] = upalmodel.NewOpenAILLM(pc.APIKey,
				upalmodel.WithOpenAIBaseURL(pc.URL),
				upalmodel.WithOpenAIName(name))
			if defaultLLM == nil {
				defaultLLM = llms[name]
				defaultModelName = "gpt-4o"
			}
		}
	}

	toolReg := tools.NewRegistry()
	sessionService := session.InMemoryService()

	srv := api.NewServer(llms, sessionService, toolReg)

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
	if defaultLLM != nil {
		gen := generate.New(defaultLLM, defaultModelName)
		srv.SetGenerator(gen, defaultModelName)
	}
	srv.SetProviderConfigs(cfg.Providers)

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
