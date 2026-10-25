package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

type ServerConfig struct {
	Port             int    `yaml:"port" json:"port"`
	Host             string `yaml:"host" json:"host"`
	StorageDir       string `yaml:"storage_dir" json:"storage_dir"`
	DisableDashboard bool   `yaml:"disable_dashboard,omitempty" json:"disable_dashboard,omitempty"`
}

type AccountConfig struct {
	ID           string    `yaml:"id" json:"id"`
	Pool         string    `yaml:"pool,omitempty" json:"pool,omitempty"`
	Email        string    `yaml:"email,omitempty" json:"email,omitempty"`
	RefreshToken string    `yaml:"refresh_token,omitempty" json:"refresh_token,omitempty"`
	AccessToken  string    `yaml:"access_token,omitempty" json:"access_token,omitempty"`
	ExpiresAt    time.Time `yaml:"expires_at,omitempty" json:"expires_at,omitempty"`
	ProjectID    string    `yaml:"project_id,omitempty" json:"project_id,omitempty"`
	ProfileARN   string    `yaml:"profile_arn,omitempty" json:"profile_arn,omitempty"`
	APIKey       string    `yaml:"api_key,omitempty" json:"api_key,omitempty"`
	AccountID    string    `yaml:"account_id,omitempty" json:"account_id,omitempty"`
	Status       string    `yaml:"status,omitempty" json:"status,omitempty"`
	Disabled     bool      `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Weight       float64   `yaml:"weight,omitempty" json:"weight,omitempty"`
}

type ProviderConfig struct {
	ID           string            `yaml:"id"`
	Type         string            `yaml:"type"` // "openai", "antigravity", "kiro"
	BaseURL      string            `yaml:"base_url,omitempty"`
	APIKey       string            `yaml:"api_key,omitempty"`
	RefreshToken string            `yaml:"refresh_token,omitempty"`
	ProfileARN   string            `yaml:"profile_arn,omitempty"`
	Strategy     string            `yaml:"strategy,omitempty"` // "priority", "round_robin", "balanced"
	Headers      map[string]string `yaml:"headers,omitempty"`
	Accounts     []AccountConfig   `yaml:"accounts,omitempty"`
}

type ComboTarget struct {
	ProviderID   string  `yaml:"provider_id" json:"provider_id"`
	Model        string  `yaml:"model" json:"model"`
	QuotaPercent float64 `yaml:"quota_percent,omitempty" json:"quota_percent,omitempty"`
}

type ComboConfig struct {
	Name     string        `yaml:"name" json:"name"`
	Strategy string        `yaml:"strategy" json:"strategy"` // "failover", "round_robin"
	Targets  []ComboTarget `yaml:"targets" json:"targets"`
}

type VirtualKey struct {
	ID              string    `yaml:"id" json:"id"`
	Name            string    `yaml:"name" json:"name"`
	Key             string    `yaml:"key" json:"key"` // e.g. "sk-goroute-..."
	IsActive        bool      `yaml:"is_active" json:"is_active"`
	AllowedModels   []string  `yaml:"allowed_models,omitempty" json:"allowed_models,omitempty"`
	MaxRPM          int       `yaml:"max_rpm,omitempty" json:"max_rpm,omitempty"`
	TotalTokensUsed int       `yaml:"total_tokens_used,omitempty" json:"total_tokens_used,omitempty"`
	CreatedAt       time.Time `yaml:"created_at" json:"created_at"`
}

type Config struct {
	Server    ServerConfig     `yaml:"server"`
	Providers []ProviderConfig `yaml:"providers"`
	Combos    []ComboConfig    `yaml:"combos"`
	Keys      []VirtualKey     `yaml:"keys,omitempty" json:"keys,omitempty"`
}

func DefaultConfig() *Config {
	home, _ := os.UserHomeDir()
	return &Config{
		Server: ServerConfig{
			Port:       8128,
			Host:       "127.0.0.1",
			StorageDir: filepath.Join(home, ".goroute"),
		},
		Providers: []ProviderConfig{},
		Combos:    []ComboConfig{},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	cfg := DefaultConfig()
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse yaml: %w", err)
	}

	return cfg, nil
}

func (c *Config) Save(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("create config directory: %w", err)
	}

	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("serialize yaml: %w", err)
	}

	return os.WriteFile(path, data, 0600)
}
