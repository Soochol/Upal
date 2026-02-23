package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"

	"github.com/soochol/upal/internal/agents"
	"github.com/soochol/upal/internal/generate"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/repository"
	"github.com/soochol/upal/internal/services"
	"github.com/soochol/upal/internal/upal"
	"google.golang.org/adk/session"
)

// spyWorkflowRepository wraps a MemoryRepository and records Create calls.
type spyWorkflowRepository struct {
	repository.WorkflowRepository
	mu      sync.Mutex
	created []*upal.WorkflowDefinition
}

func (s *spyWorkflowRepository) Create(ctx context.Context, wf *upal.WorkflowDefinition) error {
	s.mu.Lock()
	s.created = append(s.created, wf)
	s.mu.Unlock()
	return s.WorkflowRepository.Create(ctx, wf)
}

func (s *spyWorkflowRepository) CreatedWorkflows() []*upal.WorkflowDefinition {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]*upal.WorkflowDefinition, len(s.created))
	copy(out, s.created)
	return out
}

// openAICompatResponse builds a minimal OpenAI-style chat completion JSON response.
func openAICompatResponse(content string) map[string]any {
	return map[string]any{
		"choices": []map[string]any{
			{
				"message":       map[string]any{"role": "assistant", "content": content},
				"finish_reason": "stop",
			},
		},
	}
}

// noopSkills satisfies the unexported generate.skillProvider interface with no-op responses.
// generate.New() accepts any type implementing Get and GetPrompt — structural typing in Go.
type noopSkills struct{}

func (noopSkills) Get(name string) string      { return "" }
func (noopSkills) GetPrompt(name string) string { return "" }

// TestGeneratePipelineHandlerSavesWorkflows verifies that bundle.Workflows returned
// by GeneratePipelineBundle (Phase 2) are saved to the repo via repo.Create.
func TestGeneratePipelineHandlerSavesWorkflows(t *testing.T) {
	// --- Phase 1 response: a pipeline bundle with one workflow spec ---
	phase1Bundle := map[string]any{
		"pipeline": map[string]any{
			"id":          "pipe-1",
			"name":        "test-pipeline",
			"description": "A test pipeline",
			"stages": []map[string]any{
				{
					"id":   "stage-1",
					"name": "Summarise",
					"type": "workflow",
					"config": map[string]any{
						"workflow_name": "phase2-workflow",
					},
				},
			},
		},
		"workflow_specs": []map[string]any{
			{"name": "phase2-workflow", "description": "A workflow generated in Phase 2"},
		},
	}

	// --- Phase 2 response: a valid workflow JSON ---
	phase2Workflow := upal.WorkflowDefinition{
		Name:    "llm-chosen-name", // will be overridden to "phase2-workflow" by the generator
		Version: 1,
		Nodes: []upal.NodeDefinition{
			{ID: "inp", Type: upal.NodeTypeInput, Config: map[string]any{}},
			{ID: "agt", Type: upal.NodeTypeAgent, Config: map[string]any{
				"model":  "openai/gpt-4o",
				"prompt": "Do something with {{inp}}",
			}},
			{ID: "out", Type: upal.NodeTypeOutput, Config: map[string]any{}},
		},
		Edges: []upal.EdgeDefinition{
			{From: "inp", To: "agt"},
			{From: "agt", To: "out"},
		},
	}

	// Fake LLM HTTP server: first call → phase 1 bundle, subsequent → phase 2 workflow.
	callCount := 0
	fakeLLM := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var content string
		if callCount == 1 {
			b, _ := json.Marshal(phase1Bundle)
			content = string(b)
		} else {
			b, _ := json.Marshal(phase2Workflow)
			content = string(b)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(openAICompatResponse(content))
	}))
	defer fakeLLM.Close()

	// Wire up the spy repo.
	spy := &spyWorkflowRepository{WorkflowRepository: repository.NewMemory()}

	// Build the workflow service (uses spy repo internally for lookups).
	sessionSvc := session.InMemoryService()
	wfSvc := services.NewWorkflowService(spy, nil, sessionSvc, nil, agents.DefaultRegistry(), "")

	// Build the server with the spy repo.
	srv := NewServer(nil, wfSvc, spy, nil)

	// Wire up a pipeline service (required by generatePipeline to call pipelineSvc.List).
	pipelineRepo := repository.NewMemoryPipelineRepository()
	pipelineRunRepo := repository.NewMemoryPipelineRunRepository()
	pipelineSvc := services.NewPipelineService(pipelineRepo, pipelineRunRepo)
	srv.SetPipelineService(pipelineSvc)

	// Wire up the generator pointing at the fake LLM.
	// noopSkills prevents nil-pointer panics in GeneratePipelineThumbnail which
	// calls g.skills.GetPrompt before issuing any LLM request.
	llm := upalmodel.NewOpenAILLM("test-key", upalmodel.WithOpenAIBaseURL(fakeLLM.URL))
	gen := generate.New(llm, "gpt-4o", noopSkills{}, nil, nil)
	srv.SetGenerator(gen, "gpt-4o")

	// Issue the generate-pipeline request.
	reqBody, _ := json.Marshal(GeneratePipelineRequest{
		Description: "a pipeline that summarises articles",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/generate-pipeline", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d — body: %s", w.Code, w.Body.String())
	}

	// The handler must have called repo.Create for the Phase 2 workflow.
	created := spy.CreatedWorkflows()
	if len(created) == 0 {
		t.Fatal("expected repo.Create to be called for Phase 2 workflow, but no workflows were saved")
	}

	found := false
	for _, wf := range created {
		if wf.Name == "phase2-workflow" {
			found = true
			break
		}
	}
	if !found {
		names := make([]string, len(created))
		for i, wf := range created {
			names[i] = wf.Name
		}
		t.Errorf("expected workflow %q to be saved, got: %v", "phase2-workflow", names)
	}
}
