package storage

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ChaosConfig holds settings for resilience and chaos fault injection testing.
type ChaosConfig struct {
	Enabled          bool     `json:"enabled"`
	FailureRate      float64  `json:"failureRate"` // 0.0 to 1.0
	MinDelayMs       int      `json:"minDelayMs"`
	MaxDelayMs       int      `json:"maxDelayMs"`
	ErrorStatusCodes []int    `json:"errorStatusCodes"`
	Mode             string   `json:"mode"` // "parallel" | "collaborative"
	SystemPrompt     string   `json:"systemPrompt,omitempty"`
	Providers        []string `json:"providers,omitempty"`
}

var (
	chaosMu     sync.RWMutex
	globalChaos = ChaosConfig{
		Enabled:          false,
		FailureRate:      0.0,
		MinDelayMs:       0,
		MaxDelayMs:       500,
		ErrorStatusCodes: []int{429, 500, 503},
		Mode:             "parallel",
	}
)

// GetChaosConfig returns current chaos test configuration, loading from SQLite key_value table if present.
func GetChaosConfig() ChaosConfig {
	chaosMu.RLock()
	defer chaosMu.RUnlock()

	db := GetDB()
	if db != nil {
		if val, err := db.GetKeyValue("settings", "chaosConfig"); err == nil && val != "" {
			var cfg ChaosConfig
			if err := json.Unmarshal([]byte(val), &cfg); err == nil {
				return cfg
			}
		}
	}
	return globalChaos
}

// SetChaosConfig updates chaos configuration in-memory and persists it into SQLite key_value table.
func SetChaosConfig(cfg ChaosConfig) error {
	chaosMu.Lock()
	defer chaosMu.Unlock()
	globalChaos = cfg

	db := GetDB()
	if db != nil {
		data, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		return db.SetKeyValue("settings", "chaosConfig", string(data))
	}
	return nil
}

// FreeModelBudget describes a provider model offering a free quota tier.
type FreeModelBudget struct {
	Provider      string `json:"provider"`
	ModelID       string `json:"modelId"`
	DisplayName   string `json:"displayName"`
	MonthlyTokens int64  `json:"monthlyTokens"`
	CreditTokens  int64  `json:"creditTokens"`
	FreeType      string `json:"freeType"` // "monthly" | "credit" | "unlimited"
	PoolKey       string `json:"poolKey"`
	ToS           string `json:"tos"`
}

// DefaultFreeModels returns curated list of free models across providers.
func DefaultFreeModels() []FreeModelBudget {
	return []FreeModelBudget{
		{
			Provider:      "google",
			ModelID:       "gemini-2.5-flash",
			DisplayName:   "Gemini 2.5 Flash",
			MonthlyTokens: 100000000,
			FreeType:      "monthly",
			PoolKey:       "google:gemini-2.5-flash",
			ToS:           "Google AI Studio Terms",
		},
		{
			Provider:      "groq",
			ModelID:       "llama-3.3-70b-versatile",
			DisplayName:   "Llama 3.3 70B Versatile",
			MonthlyTokens: 50000000,
			FreeType:      "monthly",
			PoolKey:       "groq:llama-3.3-70b-versatile",
			ToS:           "Groq Cloud Terms",
		},
		{
			Provider:      "deepseek",
			ModelID:       "deepseek-chat",
			DisplayName:   "DeepSeek V3",
			CreditTokens:  10000000,
			FreeType:      "credit",
			PoolKey:       "deepseek:deepseek-chat",
			ToS:           "DeepSeek Platform Terms",
		},
		{
			Provider:      "openrouter",
			ModelID:       "openrouter/free",
			DisplayName:   "OpenRouter Free Tier",
			MonthlyTokens: 20000000,
			FreeType:      "monthly",
			PoolKey:       "openrouter:free",
			ToS:           "OpenRouter Terms",
		},
		{
			Provider:      "cloudflare",
			ModelID:       "@cf/meta/llama-3.1-8b-instruct",
			DisplayName:   "Llama 3.1 8B (Workers AI)",
			MonthlyTokens: 10000000,
			FreeType:      "monthly",
			PoolKey:       "cf:llama-3.1-8b",
			ToS:           "Cloudflare Workers Terms",
		},
	}
}

// ExportDatabase writes a copy of the SQLite database to the given writer.
func (s *DBStore) ExportDatabase(w io.Writer) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dbPath := filepath.Join(home, ".goroute", "goroute.db")

	f, err := os.Open(dbPath)
	if err != nil {
		return fmt.Errorf("unable to open database file: %w", err)
	}
	defer f.Close()

	_, err = io.Copy(w, f)
	return err
}

// BackupMeta describes an available snapshot.
type BackupMeta struct {
	Filename  string `json:"filename"`
	SizeBytes int64  `json:"sizeBytes"`
	CreatedAt string `json:"createdAt"`
}

// ListBackups scans ~/.goroute/backups for existing snapshots.
func ListBackups() ([]BackupMeta, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	backupDir := filepath.Join(home, ".goroute", "backups")
	_ = os.MkdirAll(backupDir, 0755)

	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return nil, err
	}

	var metas []BackupMeta
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		metas = append(metas, BackupMeta{
			Filename:  e.Name(),
			SizeBytes: info.Size(),
			CreatedAt: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	return metas, nil
}
