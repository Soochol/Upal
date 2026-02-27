package chat

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/soochol/upal/internal/llmutil"
	"github.com/soochol/upal/internal/skills"
	"github.com/soochol/upal/internal/upal/ports"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ChatRequest is the request body for the /api/chat endpoint.
type ChatRequest struct {
	Message  string         `json:"message"`
	Page     string         `json:"page"`
	Context  map[string]any `json:"context"`
	History  []ChatMessage  `json:"history"`
	Model    string         `json:"model"`
	Thinking bool           `json:"thinking"`
}

// ChatMessage represents a single message in the conversation history.
type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Handler serves the /api/chat SSE endpoint.
type Handler struct {
	registry    *ChatRegistry
	skills      skills.Provider
	llmResolver ports.LLMResolver
	defaultLLM  func(ctx context.Context) (adkmodel.LLM, string, error)
}

// NewHandler creates a new chat Handler.
func NewHandler(registry *ChatRegistry, skills skills.Provider, llmResolver ports.LLMResolver, defaultLLM func(ctx context.Context) (adkmodel.LLM, string, error)) *Handler {
	return &Handler{
		registry:    registry,
		skills:      skills,
		llmResolver: llmResolver,
		defaultLLM:  defaultLLM,
	}
}

// ServeHTTP handles POST /api/chat requests.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var req ChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		http.Error(w, "message is required", http.StatusBadRequest)
		return
	}
	if req.Page == "" {
		http.Error(w, "page is required", http.StatusBadRequest)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	chatTools := h.registry.Resolve(req.Page, req.Context)
	h.runChat(r.Context(), w, flusher, &req, chatTools)
}

// buildSystemPrompt constructs the system prompt from the skill registry,
// available tools, and request context.
func (h *Handler) buildSystemPrompt(req *ChatRequest, chatTools []*ChatTool) string {
	var sb strings.Builder

	base := h.skills.GetPrompt("chat-" + req.Page)
	if base == "" {
		base = "You are a helpful AI assistant for the Upal workflow platform. Respond in Korean."
	}
	sb.WriteString(base)

	if len(chatTools) > 0 {
		sb.WriteString("\n\n---\n\n## Available Tools\n\n")
		for _, t := range chatTools {
			sb.WriteString(fmt.Sprintf("- **%s**: %s\n", t.Name, t.Description))
		}
	}

	if len(req.Context) > 0 {
		contextJSON, _ := json.MarshalIndent(req.Context, "", "  ")
		sb.WriteString("\n\n---\n\n## Current Context\n\n```json\n")
		sb.Write(contextJSON)
		sb.WriteString("\n```\n")
	}

	return sb.String()
}

// resolveLLM returns the LLM and model name for the request.
// Uses llmResolver for explicit model overrides, otherwise falls back to defaultLLM.
func (h *Handler) resolveLLM(ctx context.Context, model string) (adkmodel.LLM, string, error) {
	if model != "" && h.llmResolver != nil {
		if resolved, resolvedName, err := h.llmResolver.Resolve(model); err == nil {
			return resolved, resolvedName, nil
		}
	}
	return h.defaultLLM(ctx)
}

// runChat executes the multi-turn LLM tool call loop and streams SSE events.
func (h *Handler) runChat(ctx context.Context, w http.ResponseWriter, flusher http.Flusher, req *ChatRequest, chatTools []*ChatTool) {
	sse := &sseWriter{w: w}

	llm, modelName, err := h.resolveLLM(ctx, req.Model)
	if err != nil {
		slog.Error("chat: failed to resolve LLM", "model", req.Model, "error", err)
		sse.write("error", map[string]any{"error": err.Error()})
		flusher.Flush()
		return
	}

	slog.Info("chat: request", "page", req.Page, "model", modelName, "tools", len(chatTools))

	sysPrompt := h.buildSystemPrompt(req, chatTools)

	genCfg := &genai.GenerateContentConfig{
		SystemInstruction: genai.NewContentFromText(sysPrompt, genai.RoleUser),
	}

	funcDecls := ToFunctionDeclarations(chatTools)
	if len(funcDecls) > 0 {
		genCfg.Tools = []*genai.Tool{{FunctionDeclarations: funcDecls}}
	}

	contents := buildContents(req.History, req.Message)

	// Inject chat context so tools can access page-specific data.
	ctx = WithChatContext(ctx, req.Context)

	for turn := 0; turn < 10; turn++ {
		llmReq := &adkmodel.LLMRequest{
			Model:    modelName,
			Config:   genCfg,
			Contents: contents,
		}

		turnCtx, turnCancel := context.WithTimeout(ctx, 2*time.Minute)
		var resp *adkmodel.LLMResponse
		for r, err := range llm.GenerateContent(turnCtx, llmReq, false) {
			if err != nil {
				turnCancel()
				slog.Error("chat: LLM generation failed", "turn", turn, "error", err)
				sse.write("error", map[string]any{"error": err.Error()})
				flusher.Flush()
				return
			}
			resp = r
		}
		turnCancel()

		if resp == nil || resp.Content == nil {
			sse.write("error", map[string]any{"error": "empty response"})
			flusher.Flush()
			return
		}

		// Check for tool calls in the response.
		var toolCalls []*genai.FunctionCall
		for _, p := range resp.Content.Parts {
			if p.FunctionCall != nil {
				toolCalls = append(toolCalls, p.FunctionCall)
			}
		}

		if len(toolCalls) == 0 {
			// Final text response — stream and finish.
			text := llmutil.ExtractText(resp)
			sse.write("text_delta", map[string]any{"text": text})
			sse.write("done", map[string]any{"content": text})
			flusher.Flush()
			return
		}

		// Execute tool calls and continue to the next turn.
		contents = append(contents, resp.Content)
		toolResults := make([]*genai.Part, 0, len(toolCalls))

		for _, fc := range toolCalls {
			sse.write("tool_call", map[string]any{
				"id":   fc.ID,
				"name": fc.Name,
				"args": fc.Args,
			})
			flusher.Flush()

			result, execErr := h.registry.ExecuteToolCall(ctx, fc.Name, fc.Args)
			success := execErr == nil

			if execErr != nil {
				slog.Warn("chat: tool execution failed", "tool", fc.Name, "error", execErr)
			}

			var resultData any
			if execErr != nil {
				resultData = map[string]any{"error": execErr.Error()}
			} else {
				resultData = result
			}

			sse.write("tool_result", map[string]any{
				"id":      fc.ID,
				"name":    fc.Name,
				"success": success,
				"result":  resultData,
			})
			flusher.Flush()

			responseMap := map[string]any{}
			if execErr != nil {
				responseMap["error"] = execErr.Error()
			} else {
				responseMap["result"] = result
			}
			toolResults = append(toolResults, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     fc.Name,
					Response: responseMap,
				},
			})
		}

		contents = append(contents, &genai.Content{Role: genai.RoleUser, Parts: toolResults})
	}

	sse.write("error", map[string]any{"error": "exceeded maximum turns"})
	flusher.Flush()
}

// sseWriter tracks event IDs for SSE reconnection support.
type sseWriter struct {
	w  http.ResponseWriter
	id int
}

func (s *sseWriter) write(event string, data any) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		slog.Error("chat: failed to marshal SSE data", "event", event, "error", err)
		fmt.Fprintf(s.w, "id: %d\nevent: error\ndata: {\"error\":\"internal marshal error\"}\n\n", s.id)
		s.id++
		return
	}
	fmt.Fprintf(s.w, "id: %d\nevent: %s\ndata: %s\n\n", s.id, event, jsonData)
	s.id++
}

// buildContents converts chat history and the current message into genai Contents.
func buildContents(history []ChatMessage, message string) []*genai.Content {
	contents := make([]*genai.Content, 0, len(history)+1)
	for _, msg := range history {
		if msg.Role == "assistant" {
			contents = append(contents, genai.NewContentFromText(msg.Content, genai.RoleModel))
		} else {
			contents = append(contents, genai.NewContentFromText(msg.Content, genai.RoleUser))
		}
	}
	contents = append(contents, genai.NewContentFromText(message, genai.RoleUser))
	return contents
}
