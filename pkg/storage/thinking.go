package storage

import (
	"encoding/json"
	"sync"
)

// ThinkingBudgetConfig defines global reasoning/thinking token budget policy.
type ThinkingBudgetConfig struct {
	Mode         string `json:"mode"`         // "passthrough", "auto", "custom", "adaptive"
	CustomBudget int    `json:"customBudget"` // token limit when mode is "custom"
	EffortLevel  string `json:"effortLevel"`  // "none", "low", "medium", "high", "max"
}

var (
	thinkingMu     sync.RWMutex
	thinkingConfig = ThinkingBudgetConfig{
		Mode:         "adaptive",
		CustomBudget: 4000,
		EffortLevel:  "medium",
	}
)

// GetThinkingBudgetConfig returns active thinking budget configuration.
func GetThinkingBudgetConfig() ThinkingBudgetConfig {
	thinkingMu.RLock()
	defer thinkingMu.RUnlock()
	return thinkingConfig
}

// SetThinkingBudgetConfig updates in-memory and database thinking budget config.
func SetThinkingBudgetConfig(db *DBStore, cfg ThinkingBudgetConfig) error {
	thinkingMu.Lock()
	defer thinkingMu.Unlock()

	thinkingConfig = cfg
	if db != nil && db.db != nil {
		bytes, err := json.Marshal(cfg)
		if err != nil {
			return err
		}
		_, err = db.db.Exec(`
			INSERT OR REPLACE INTO key_value (namespace, key, value)
			VALUES ('settings', 'thinkingBudget', ?)
		`, string(bytes))
		return err
	}
	return nil
}

// InitThinkingBudget hydrates thinking budget from SQLite on startup.
func InitThinkingBudget(db *DBStore) {
	if db == nil || db.db == nil {
		return
	}
	var val string
	err := db.db.QueryRow(`SELECT value FROM key_value WHERE namespace = 'settings' AND key = 'thinkingBudget'`).Scan(&val)
	if err == nil && val != "" {
		var cfg ThinkingBudgetConfig
		if json.Unmarshal([]byte(val), &cfg) == nil {
			thinkingMu.Lock()
			thinkingConfig = cfg
			thinkingMu.Unlock()
		}
	}
}

// ApplyThinkingBudget modifies a request's reasoning tokens or effort based on policy.
func ApplyThinkingBudget(reasoningEffort *string, maxTokens *int, thinking *interface{}) {
	cfg := GetThinkingBudgetConfig()
	switch cfg.Mode {
	case "auto":
		// Strip client reasoning effort and thinking
		if reasoningEffort != nil {
			*reasoningEffort = ""
		}
		if thinking != nil {
			*thinking = nil
		}
	case "custom":
		if maxTokens != nil && cfg.CustomBudget > 0 {
			if *maxTokens <= 0 || *maxTokens > cfg.CustomBudget {
				*maxTokens = cfg.CustomBudget
			}
		}
		if reasoningEffort != nil && cfg.EffortLevel != "" {
			*reasoningEffort = cfg.EffortLevel
		}
	case "adaptive":
		// Cap reasoning budget at CustomBudget (default 4000) to prevent runaway token burns
		capBudget := cfg.CustomBudget
		if capBudget <= 0 {
			capBudget = 4000
		}
		if thinking != nil && *thinking != nil {
			if thMap, ok := (*thinking).(map[string]interface{}); ok {
				if bVal, hasBudget := thMap["budget_tokens"]; hasBudget {
					var b int
					switch v := bVal.(type) {
					case float64:
						b = int(v)
					case int:
						b = v
					}
					if b > capBudget {
						thMap["budget_tokens"] = capBudget
					}
				}
			}
		}
		if reasoningEffort != nil && *reasoningEffort == "" {
			*reasoningEffort = cfg.EffortLevel
		}
	case "passthrough":
		fallthrough
	default:
		// Do nothing, pass through client fields untouched
	}
}
