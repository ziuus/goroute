package storage

import (
	"encoding/json"
	"sync"
)

// ModelAliasMap holds built-in and user-configured model alias translations.
type ModelAliasMap struct {
	BuiltIn map[string]string `json:"builtIn"`
	Custom  map[string]string `json:"custom"`
	All     map[string]string `json:"all"`
}

var (
	builtInModelAliases = map[string]string{
		// Gemini legacy -> current
		"gemini-pro":                   "gemini-2.5-pro",
		"gemini-pro-vision":            "gemini-2.5-pro",
		"gemini-1.5-pro":               "gemini-2.5-pro",
		"gemini-1.5-flash":             "gemini-2.5-flash",
		"gemini-1.0-pro":               "gemini-2.5-pro",
		"gemini-2.0-flash":             "gemini-2.5-flash",
		"gemini-2.0-flash-lite":        "gemini-3.1-flash-lite",
		"gemini-3.1-flash-lite-preview": "gemini-3.1-flash-lite",
		"gemini-3-pro-high":            "gemini-3-1-pro-high",
		"gemini-3-pro-low":             "gemini-3.1-pro-low",
		"gemma-4":                      "gemini-3.1-flash-lite",

		// Claude legacy -> current
		"claude-3-opus-20240229":   "claude-opus-4-8",
		"claude-3-sonnet-20240229": "claude-sonnet-4-6",
		"claude-3-haiku-20240307":  "claude-haiku-4-5-20251001",
		"claude-3-5-sonnet-latest": "claude-sonnet-4-6",
		"claude-3-5-haiku-latest":  "claude-haiku-4-5-20251001",
		"claude-frontier":          "frontier",

		// OpenAI aliases
		"gpt-4":          "gpt-4o",
		"gpt-4-turbo":    "gpt-4o",
		"gpt-3.5-turbo":  "gpt-4o-mini",

		// DeepSeek snapshot rotations
		"deepseek-v4-flash-free": "deepseek-v4.1-flash-free",
		"deepseek-coder":         "deepseek-chat",

		// Mistral
		"mistral-large": "mistral-large-latest",
		"mistral-small": "mistral-small-latest",
		"codestral":     "codestral-latest",
	}

	aliasMu       sync.RWMutex
	customAliases = make(map[string]string)
)

// InitModelAliases loads custom aliases from SQLite key_value table.
func InitModelAliases(db *DBStore) {
	if db == nil || db.db == nil {
		return
	}
	aliasMu.Lock()
	defer aliasMu.Unlock()

	var val string
	err := db.db.QueryRow(`SELECT value FROM key_value WHERE namespace = 'settings' AND key = 'modelAliases'`).Scan(&val)
	if err == nil && val != "" {
		var m map[string]string
		if json.Unmarshal([]byte(val), &m) == nil {
			customAliases = m
		}
	}
}

// ResolveModelAlias transforms a model ID according to custom, then built-in aliases.
func ResolveModelAlias(model string) string {
	aliasMu.RLock()
	defer aliasMu.RUnlock()

	if target, exists := customAliases[model]; exists && target != "" {
		return target
	}
	if target, exists := builtInModelAliases[model]; exists && target != "" {
		return target
	}
	return model
}

// GetModelAliasesSnapshot returns complete maps for GET /api/settings/model-aliases.
func GetModelAliasesSnapshot() ModelAliasMap {
	aliasMu.RLock()
	defer aliasMu.RUnlock()

	all := make(map[string]string)
	for k, v := range builtInModelAliases {
		all[k] = v
	}
	for k, v := range customAliases {
		all[k] = v
	}

	customCopy := make(map[string]string)
	for k, v := range customAliases {
		customCopy[k] = v
	}

	return ModelAliasMap{
		BuiltIn: builtInModelAliases,
		Custom:  customCopy,
		All:     all,
	}
}

// SetCustomAliases replaces user custom aliases and saves to SQLite.
func SetCustomAliases(db *DBStore, aliases map[string]string) error {
	aliasMu.Lock()
	defer aliasMu.Unlock()

	customAliases = aliases
	if db != nil {
		bytes, err := json.Marshal(aliases)
		if err != nil {
			return err
		}
		_, err = db.db.Exec(`
			INSERT OR REPLACE INTO key_value (namespace, key, value)
			VALUES ('settings', 'modelAliases', ?)
		`, string(bytes))
		return err
	}
	return nil
}

// AddCustomAlias adds or updates a single model alias.
func AddCustomAlias(db *DBStore, from, to string) error {
	aliasMu.Lock()
	defer aliasMu.Unlock()

	customAliases[from] = to
	if db != nil {
		bytes, err := json.Marshal(customAliases)
		if err != nil {
			return err
		}
		_, err = db.db.Exec(`
			INSERT OR REPLACE INTO key_value (namespace, key, value)
			VALUES ('settings', 'modelAliases', ?)
		`, string(bytes))
		return err
	}
	return nil
}

// RemoveCustomAlias deletes a single custom alias.
func RemoveCustomAlias(db *DBStore, from string) error {
	aliasMu.Lock()
	defer aliasMu.Unlock()

	delete(customAliases, from)
	if db != nil {
		bytes, err := json.Marshal(customAliases)
		if err != nil {
			return err
		}
		_, err = db.db.Exec(`
			INSERT OR REPLACE INTO key_value (namespace, key, value)
			VALUES ('settings', 'modelAliases', ?)
		`, string(bytes))
		return err
	}
	return nil
}
