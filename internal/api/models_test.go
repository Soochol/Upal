package api

import (
	"testing"

	"github.com/soochol/upal/internal/config"
)

func TestIsOllama(t *testing.T) {
	cases := []struct {
		pc   config.ProviderConfig
		want bool
	}{
		{config.ProviderConfig{Type: "ollama", URL: "http://localhost:11434/v1"}, true},
		{config.ProviderConfig{Type: "ollama"}, true},  // explicit type, no URL needed
		{config.ProviderConfig{Type: "openai", URL: "http://localhost:11434/v1"}, true},
		{config.ProviderConfig{Type: "openai", URL: "http://localhost:8080/v1"}, false},
		{config.ProviderConfig{Type: "anthropic", URL: "http://localhost:11434/v1"}, false},
		{config.ProviderConfig{Type: "openai"}, false},
	}
	for _, c := range cases {
		got := isOllama(c.pc)
		if got != c.want {
			t.Errorf("isOllama(%+v) = %v, want %v", c.pc, got, c.want)
		}
	}
}
