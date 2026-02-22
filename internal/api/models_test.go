package api

import (
	"testing"

	"github.com/soochol/upal/internal/config"
)

func TestIsOllama(t *testing.T) {
	cases := []struct {
		name string
		pc   config.ProviderConfig
		want bool
	}{
		{"explicit ollama type", config.ProviderConfig{Type: "ollama", URL: "http://localhost:11434/v1"}, true},
		{"openai type with port 11434", config.ProviderConfig{Type: "openai", URL: "http://localhost:11434/v1"}, true},
		{"openai type with other port", config.ProviderConfig{Type: "openai", URL: "http://localhost:8080/v1"}, false},
		{"anthropic type with 11434 in URL", config.ProviderConfig{Type: "anthropic", URL: "http://localhost:11434/v1"}, false},
		{"ollama type without URL", config.ProviderConfig{Type: "ollama"}, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := isOllama(c.pc)
			if got != c.want {
				t.Errorf("isOllama(%+v) = %v, want %v", c.pc, got, c.want)
			}
		})
	}
}
