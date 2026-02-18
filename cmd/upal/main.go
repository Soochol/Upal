package main

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/soochol/upal/internal/api"
	"github.com/soochol/upal/internal/config"
	"github.com/soochol/upal/internal/engine"
	"github.com/soochol/upal/internal/nodes"
	"github.com/soochol/upal/internal/provider"
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
		p := provider.NewOpenAIProvider(name, pc.URL, pc.APIKey)
		providerReg.Register(p)
	}

	toolReg := tools.NewRegistry()
	runner := engine.NewRunner(eventBus, sessions)

	executors := map[engine.NodeType]engine.NodeExecutorInterface{
		engine.NodeTypeInput:  &nodes.InputNode{},
		engine.NodeTypeAgent:  nodes.NewAgentNode(providerReg, toolReg, eventBus),
		engine.NodeTypeTool:   nodes.NewToolNode(toolReg),
		engine.NodeTypeOutput: &nodes.OutputNode{},
	}

	srv := api.NewServer(eventBus, sessions, runner, executors)
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	slog.Info("starting upal server", "addr", addr)
	if err := http.ListenAndServe(addr, srv.Handler()); err != nil {
		slog.Error("server error", "err", err)
		os.Exit(1)
	}
}
