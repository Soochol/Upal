package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoad_ValidYAML(t *testing.T) {
	content := `
server:
  host: "127.0.0.1"
  port: 9090

database:
  url: "postgres://user:pass@localhost:5432/testdb"

providers:
  ollama:
    type: "openai"
    url: "http://localhost:11434/v1"
    api_key: "test-key"
  openai:
    type: "openai"
    url: "https://api.openai.com/v1"
    api_key: "sk-abc123"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	// Server
	if cfg.Server.Host != "127.0.0.1" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "127.0.0.1")
	}
	if cfg.Server.Port != 9090 {
		t.Errorf("Server.Port = %d, want %d", cfg.Server.Port, 9090)
	}

	// Database
	if cfg.Database.URL != "postgres://user:pass@localhost:5432/testdb" {
		t.Errorf("Database.URL = %q, want postgres://user:pass@localhost:5432/testdb", cfg.Database.URL)
	}

	// Providers
	if len(cfg.Providers) != 2 {
		t.Fatalf("len(Providers) = %d, want 2", len(cfg.Providers))
	}

	ollama, ok := cfg.Providers["ollama"]
	if !ok {
		t.Fatal("expected provider 'ollama' not found")
	}
	if ollama.Type != "openai" {
		t.Errorf("ollama.Type = %q, want %q", ollama.Type, "openai")
	}
	if ollama.URL != "http://localhost:11434/v1" {
		t.Errorf("ollama.URL = %q, want %q", ollama.URL, "http://localhost:11434/v1")
	}
	if ollama.APIKey != "test-key" {
		t.Errorf("ollama.APIKey = %q, want %q", ollama.APIKey, "test-key")
	}

	openai, ok := cfg.Providers["openai"]
	if !ok {
		t.Fatal("expected provider 'openai' not found")
	}
	if openai.APIKey != "sk-abc123" {
		t.Errorf("openai.APIKey = %q, want %q", openai.APIKey, "sk-abc123")
	}
}

func TestLoad_EmptyProviders(t *testing.T) {
	content := `
server:
  host: "0.0.0.0"
  port: 8080

providers: {}
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Providers == nil {
		t.Fatal("Providers should not be nil")
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("len(Providers) = %d, want 0", len(cfg.Providers))
	}
}

func TestLoad_FileNotFound(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("Load() should return error for nonexistent file")
	}
}

func TestLoad_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	// A YAML mapping value where the key "server" expects a nested map
	// but gets an invalid indentation / structure that can't unmarshal into Config.
	badYAML := "server:\n\t- not valid\n  port: oops"
	if err := os.WriteFile(path, []byte(badYAML), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("Load() should return error for invalid YAML")
	}
}

func TestLoad_PartialConfig(t *testing.T) {
	// Only server section; other fields should get defaults.
	content := `
server:
  port: 3000
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.Server.Port != 3000 {
		t.Errorf("Server.Port = %d, want 3000", cfg.Server.Port)
	}
	// Host should retain the default since we unmarshal onto defaults.
	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q (default)", cfg.Server.Host, "0.0.0.0")
	}
	// Providers should be non-nil even when omitted from YAML.
	if cfg.Providers == nil {
		t.Fatal("Providers should not be nil when omitted from YAML")
	}
}

func TestLoadDefault_NoFile(t *testing.T) {
	// Run from a temp directory where config.yaml does not exist.
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	dir := t.TempDir()
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() returned error: %v", err)
	}

	if cfg.Server.Host != "0.0.0.0" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "0.0.0.0")
	}
	if cfg.Server.Port != 8080 {
		t.Errorf("Server.Port = %d, want 8080", cfg.Server.Port)
	}
	if cfg.Providers == nil {
		t.Fatal("Providers should not be nil")
	}
	if len(cfg.Providers) != 0 {
		t.Errorf("len(Providers) = %d, want 0", len(cfg.Providers))
	}
}

func TestLoadDefault_WithFile(t *testing.T) {
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	dir := t.TempDir()
	content := `
server:
  host: "10.0.0.1"
  port: 4000
`
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadDefault()
	if err != nil {
		t.Fatalf("LoadDefault() returned error: %v", err)
	}

	if cfg.Server.Host != "10.0.0.1" {
		t.Errorf("Server.Host = %q, want %q", cfg.Server.Host, "10.0.0.1")
	}
	if cfg.Server.Port != 4000 {
		t.Errorf("Server.Port = %d, want 4000", cfg.Server.Port)
	}
}
