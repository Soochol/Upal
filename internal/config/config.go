package config

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/soochol/upal/internal/upal"
	"gopkg.in/yaml.v3"
)

// Config holds the top-level application configuration.
type Config struct {
	Server    ServerConfig              `yaml:"server"`
	Database  DatabaseConfig            `yaml:"database"`
	Auth      AuthConfig                `yaml:"auth"`
	Providers map[string]ProviderConfig `yaml:"providers"`
	Scheduler upal.ConcurrencyLimits    `yaml:"scheduler"`
	Runs      RunsConfig                `yaml:"runs"`
	Generator GeneratorConfig           `yaml:"generator"`
}

type AuthConfig struct {
	Google    OAuthProviderConfig `yaml:"google"`
	GitHub    OAuthProviderConfig `yaml:"github"`
	JWTSecret string             `yaml:"jwt_secret"`
}

type OAuthProviderConfig struct {
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"`
}

func (c OAuthProviderConfig) IsConfigured() bool {
	return c.ClientID != "" && c.ClientSecret != ""
}

// ServerConfig holds HTTP server settings.
type ServerConfig struct {
	Host          string   `yaml:"host"`
	Port          int      `yaml:"port"`
	UploadMaxSize int64    `yaml:"upload_max_size"`
	CORSOrigins   []string `yaml:"cors_origins"`
}

// RunsConfig holds run manager settings.
type RunsConfig struct {
	TTL time.Duration `yaml:"ttl"`
}

// GeneratorConfig holds generation-related timeouts.
type GeneratorConfig struct {
	ThumbnailTimeout time.Duration `yaml:"thumbnail_timeout"`
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
			Host:          "0.0.0.0",
			Port:          8080,
			UploadMaxSize: 50 << 20, // 50 MB
		},
		Database:  DatabaseConfig{},
		Providers: map[string]ProviderConfig{},
		Runs: RunsConfig{
			TTL: 15 * time.Minute,
		},
		Generator: GeneratorConfig{
			ThumbnailTimeout: 60 * time.Second,
		},
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
			cfg = defaults()
		} else {
			return nil, err
		}
	}
	applyEnvOverrides(cfg)
	return cfg, nil
}

// applyEnvOverrides loads .env (if present) and overrides sensitive config
// fields from environment variables.
//
// Supported variables:
//   - DATABASE_URL         → cfg.Database.URL
//   - {PROVIDER}_API_KEY   → cfg.Providers[provider].APIKey
//     (provider name uppercased, hyphens replaced with underscores)
func applyEnvOverrides(cfg *Config) {
	_ = godotenv.Load() // ignore error — .env is optional

	if v := os.Getenv("DATABASE_URL"); v != "" {
		cfg.Database.URL = v
	}

	for name, pc := range cfg.Providers {
		envKey := strings.ToUpper(strings.ReplaceAll(name, "-", "_")) + "_API_KEY"
		if v := os.Getenv(envKey); v != "" {
			pc.APIKey = v
			cfg.Providers[name] = pc
		}
	}

	if v := os.Getenv("CORS_ORIGINS"); v != "" {
		cfg.Server.CORSOrigins = strings.Split(v, ",")
	}

	if v := os.Getenv("GOOGLE_CLIENT_ID"); v != "" {
		cfg.Auth.Google.ClientID = v
	}
	if v := os.Getenv("GOOGLE_CLIENT_SECRET"); v != "" {
		cfg.Auth.Google.ClientSecret = v
	}
	if v := os.Getenv("GITHUB_CLIENT_ID"); v != "" {
		cfg.Auth.GitHub.ClientID = v
	}
	if v := os.Getenv("GITHUB_CLIENT_SECRET"); v != "" {
		cfg.Auth.GitHub.ClientSecret = v
	}
	if v := os.Getenv("JWT_SECRET"); v != "" {
		cfg.Auth.JWTSecret = v
	}
}
