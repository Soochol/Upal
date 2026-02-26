package api

import (
	"fmt"

	upalmodel "github.com/soochol/upal/internal/model"
	"github.com/soochol/upal/internal/upal"
	adkmodel "google.golang.org/adk/model"
	"google.golang.org/genai"
)

// ConfigChatMsg represents a single message in the configuration chat history.
type ConfigChatMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// resolveLLM returns the LLM and model name to use for a configuration request.
// If requestModel is provided and resolvable, it overrides the generator default.
func (s *Server) resolveLLM(requestModel string) (adkmodel.LLM, string) {
	llm := s.generator.LLM()
	modelName := s.generator.Model()
	if requestModel != "" && s.llmResolver != nil {
		if resolved, resolvedName, err := s.llmResolver.Resolve(requestModel); err == nil {
			llm = resolved
			modelName = resolvedName
		}
	}
	return llm, modelName
}

// buildChatHistory converts ConfigChatMsg slices into genai.Content for LLM requests.
func buildChatHistory(history []ConfigChatMsg) []*genai.Content {
	var contents []*genai.Content
	for _, h := range history {
		switch h.Role {
		case "user":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleUser))
		case "assistant":
			contents = append(contents, genai.NewContentFromText(h.Content, genai.RoleModel))
		}
	}
	return contents
}

// appendModelCatalog appends available model information to a system prompt so the
// LLM can reference real model IDs in its configuration output.
func (s *Server) appendModelCatalog(sysPrompt string, modelName string) string {
	allModels := upalmodel.KnownModelsGrouped(s.providerConfigs)
	if len(allModels) == 0 {
		return sysPrompt
	}

	sysPrompt += fmt.Sprintf("\n\nAvailable models (use in \"model\" field):\nDefault model: %q\n", modelName)

	var textModels, imageModels []upal.ModelInfo
	for _, m := range allModels {
		switch m.Category {
		case upal.ModelCategoryText:
			textModels = append(textModels, m)
		case upal.ModelCategoryImage:
			imageModels = append(imageModels, m)
		}
	}
	if len(textModels) > 0 {
		sysPrompt += "\nText/reasoning models:\n"
		for _, m := range textModels {
			sysPrompt += fmt.Sprintf("- %q [%s] — %s\n", m.ID, m.Tier, m.Hint)
		}
	}
	if len(imageModels) > 0 {
		sysPrompt += "\nImage generation models:\n"
		for _, m := range imageModels {
			sysPrompt += fmt.Sprintf("- %q — %s\n", m.ID, m.Hint)
		}
	}
	return sysPrompt
}
