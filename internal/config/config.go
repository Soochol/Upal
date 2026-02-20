package config

import (
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds the top-level application configuration.
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Database  DatabaseConfig            `yaml:"database"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Scheduler SchedulerConfig           `yaml:"scheduler"`
}

// SchedulerConfig holds settings for the workflow scheduler.
type SchedulerConfig struct {
	GlobalMax   int `yaml:"global_max"`   // max concurrent runs system-wide (default: 10)
	PerWorkflow int `yaml:"per_workflow"` // max concurrent runs per workflow (default: 3)
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

// DatabaseConfig holds database connection settings.
type DatabaseConfig struct {
	URL string `yaml:"url"`
}

// ProviderConfig holds AI provider settings.
type ProviderConfig struct {
	Type   string `yaml:"type"`    // e.g. "openai"
	URL    string `yaml:"url"`     // base URL
	APIKey string `yaml:"api_key"` // API key
}

// defaults returns a Config populated with sensible default values.
func defaults() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "0.0.0.0",
			Port: 8080,
		},
		Database: DatabaseConfig{},
		Providers: map[string]ProviderConfig{},
	}
}

// Load reads a YAML configuration file at path and returns a Config.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	cfg := defaults()
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config file: %w", err)
	}

	// Ensure Providers map is never nil even if YAML has "providers: {}" or omits it.
	if cfg.Providers == nil {
		cfg.Providers = map[string]ProviderConfig{}
	}

	return cfg, nil
}

// LoadDefault tries to load "config.yaml" from the current directory.
// If the file does not exist, it returns sensible defaults.
// Any other error (e.g. permission denied, malformed YAML) is returned.
func LoadDefault() (*Config, error) {
	cfg, err := Load("config.yaml")
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return defaults(), nil
		}
		return nil, err
	}
	return cfg, nil
}
