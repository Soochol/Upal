package generate

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/soochol/upal/internal/llmutil"
	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// nodeDescPlaceholder is the generic value set by old code — treat as missing.
const nodeDescPlaceholder = "AI model processing step"

// BackfillWorkflowDescriptions generates Korean one-sentence descriptions for
// workflows that have an empty Description field. Returns the updated workflows.
// Errors per workflow are logged and skipped (best-effort).
func (g *Generator) BackfillWorkflowDescriptions(ctx context.Context, workflows []*upal.WorkflowDefinition) []*upal.WorkflowDefinition {
	var updated []*upal.WorkflowDefinition
	for _, wf := range workflows {
		if wf.Description != "" {
			continue
		}
		desc, err := g.generateWorkflowDescription(ctx, wf)
		if err != nil {
			log.Printf("backfill: workflow %q description failed: %v", wf.Name, err)
			continue
		}
		wf.Description = desc
		updated = append(updated, wf)
	}
	return updated
}

// generateWorkflowDescription asks the LLM for a one-sentence Korean description
// of a workflow based on its name and node structure.
func (g *Generator) generateWorkflowDescription(ctx context.Context, wf *upal.WorkflowDefinition) (string, error) {
	userPrompt := buildWorkflowDescribePrompt(wf)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(g.skills.GetPrompt("workflow-describe"), genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		},
	}

	ctx = upalmodel.WithEffort(ctx, "low")

	descCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(descCtx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate description: %w", err)
		}
		resp = r
	}

	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}

	desc := strings.TrimSpace(llmutil.ExtractText(resp))
	if desc == "" {
		return "", fmt.Errorf("LLM returned empty description")
	}
	return desc, nil
}

// buildWorkflowDescribePrompt builds the user message for workflow description generation.
func buildWorkflowDescribePrompt(wf *upal.WorkflowDefinition) string {
	var lines []string
	lines = append(lines, fmt.Sprintf("워크플로우 이름: %s", wf.Name))
	lines = append(lines, "노드:")
	for _, n := range wf.Nodes {
		label, _ := n.Config["label"].(string)
		desc, _ := n.Config["description"].(string)
		if label == "" {
			label = string(n.Type)
		}
		entry := fmt.Sprintf("  - [%s] %s", n.Type, label)
		if desc != "" {
			entry += ": " + desc
		}
		lines = append(lines, entry)
	}
	return strings.Join(lines, "\n")
}

// BackfillStageDescriptions fills in missing stage descriptions using a
// rule-based approach (no LLM needed). Returns true if any stage was updated.
func BackfillStageDescriptions(pipeline *upal.Pipeline) bool {
	updated := false
	for i := range pipeline.Stages {
		stage := &pipeline.Stages[i]
		if stage.Description != "" {
			continue
		}
		stage.Description = defaultStageDescription(stage)
		updated = true
	}
	return updated
}

// BackfillNodeDescriptions fills in missing or placeholder node descriptions within
// each workflow. Agent nodes use LLM; input/output nodes use rule-based text.
// Returns the workflows that had at least one node updated.
func (g *Generator) BackfillNodeDescriptions(ctx context.Context, workflows []*upal.WorkflowDefinition) []*upal.WorkflowDefinition {
	var updated []*upal.WorkflowDefinition
	for _, wf := range workflows {
		changed := false
		for i := range wf.Nodes {
			node := &wf.Nodes[i]
			desc, _ := node.Config["description"].(string)
			if desc != "" && desc != nodeDescPlaceholder {
				continue
			}
			var newDesc string
			var err error
			if node.Type == upal.NodeTypeAgent {
				newDesc, err = g.generateNodeDescription(ctx, node)
				if err != nil {
					log.Printf("backfill: node %q description failed: %v", node.ID, err)
					continue
				}
			} else {
				newDesc = defaultNodeDescription(node)
			}
			node.Config["description"] = newDesc
			changed = true
		}
		if changed {
			updated = append(updated, wf)
		}
	}
	return updated
}

// generateNodeDescription asks the LLM for a one-sentence Korean description of an agent node.
func (g *Generator) generateNodeDescription(ctx context.Context, node *upal.NodeDefinition) (string, error) {
	label, _ := node.Config["label"].(string)
	prompt, _ := node.Config["prompt"].(string)
	if len(prompt) > 200 {
		prompt = prompt[:200] + "…"
	}
	userPrompt := fmt.Sprintf("노드 라벨: %s\n프롬프트 (일부): %s", label, prompt)

	req := &adkmodel.LLMRequest{
		Model: g.model,
		Config: &genai.GenerateContentConfig{
			SystemInstruction: genai.NewContentFromText(g.skills.GetPrompt("node-describe"), genai.RoleUser),
		},
		Contents: []*genai.Content{
			genai.NewContentFromText(userPrompt, genai.RoleUser),
		},
	}

	ctx = upalmodel.WithEffort(ctx, "low")
	descCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	var resp *adkmodel.LLMResponse
	for r, err := range g.llm.GenerateContent(descCtx, req, false) {
		if err != nil {
			return "", fmt.Errorf("generate node description: %w", err)
		}
		resp = r
	}
	if resp == nil || resp.Content == nil {
		return "", fmt.Errorf("empty response from LLM")
	}
	desc := strings.TrimSpace(llmutil.ExtractText(resp))
	if desc == "" {
		return "", fmt.Errorf("LLM returned empty description")
	}
	return desc, nil
}

// defaultNodeDescription returns a rule-based Korean description for input/output nodes.
func defaultNodeDescription(node *upal.NodeDefinition) string {
	label, _ := node.Config["label"].(string)
	if label == "" {
		label = node.ID
	}
	switch node.Type {
	case upal.NodeTypeInput:
		return label + " 입력을 수집합니다."
	case upal.NodeTypeOutput:
		return label + " 결과를 표시합니다."
	default:
		return label + " 처리를 수행합니다."
	}
}

// defaultStageDescription returns a Korean description based on stage type and config.
func defaultStageDescription(stage *upal.Stage) string {
	switch stage.Type {
	case "workflow":
		if stage.Config.WorkflowName != "" {
			return stage.Config.WorkflowName + " 워크플로우를 실행합니다."
		}
		return "워크플로우를 실행합니다."
	case "approval":
		return "담당자 승인을 대기합니다."
	case "notification":
		return "알림을 전송합니다."
	case "schedule":
		return "예약된 시간에 파이프라인을 실행합니다."
	case "trigger":
		return "외부 트리거 이벤트를 대기합니다."
	case "transform":
		return "데이터를 변환하여 다음 스테이지로 전달합니다."
	default:
		return stage.Name + " 스테이지를 처리합니다."
	}
}
