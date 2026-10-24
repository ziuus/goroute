package sync

import (
	"crypto/aes"
	"crypto/cipher"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/scrypt"
	_ "modernc.org/sqlite"

	"github.com/ziuus/goroute/pkg/config"
)

const omniSalt = "omniroute-field-encryption-v1"

// DecryptOmniCredential decrypts an enc:v1:<iv>:<ciphertext>:<tag> string
func DecryptOmniCredential(encrypted string, secret string) (string, error) {
	if !strings.HasPrefix(encrypted, "enc:v1:") {
		return encrypted, nil
	}
	parts := strings.Split(strings.TrimPrefix(encrypted, "enc:v1:"), ":")
	if len(parts) != 3 {
		return "", fmt.Errorf("invalid encrypted format: %s", encrypted)
	}

	iv, err := hex.DecodeString(parts[0])
	if err != nil {
		return "", fmt.Errorf("invalid iv: %w", err)
	}
	ciphertext, err := hex.DecodeString(parts[1])
	if err != nil {
		return "", fmt.Errorf("invalid ciphertext: %w", err)
	}
	tag, err := hex.DecodeString(parts[2])
	if err != nil {
		return "", fmt.Errorf("invalid tag: %w", err)
	}

	key, err := scrypt.Key([]byte(secret), []byte(omniSalt), 16384, 8, 1, 32)
	if err != nil {
		return "", fmt.Errorf("scrypt derivation failed: %w", err)
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher failed: %w", err)
	}

	gcm, err := cipher.NewGCMWithNonceSize(block, len(iv))
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCMWithNonceSize failed: %w", err)
	}

	combined := append(ciphertext, tag...)
	plaintext, err := gcm.Open(nil, iv, combined, nil)
	if err != nil {
		return "", fmt.Errorf("gcm decryption failed: %w", err)
	}

	return string(plaintext), nil
}

// SyncResult details the outcome of an OmniRoute import
type SyncResult struct {
	ProvidersImported int      `json:"providers_imported"`
	AccountsImported  int      `json:"accounts_imported"`
	CombosImported    int      `json:"combos_imported"`
	Details           []string `json:"details"`
}

// SyncFromOmniRoute reads ~/.omniroute/storage.sqlite and merges accounts into GoRoute config
func SyncFromOmniRoute(cfgPath string) (*SyncResult, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	omniDir := filepath.Join(home, ".omniroute")
	omniDBPath := filepath.Join(omniDir, "storage.sqlite")
	omniEnvPath := filepath.Join(omniDir, ".env")

	if _, err := os.Stat(omniDBPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("omniroute database not found at %s", omniDBPath)
	}

	// Read secret
	var secret string
	if envBytes, err := os.ReadFile(omniEnvPath); err == nil {
		for _, line := range strings.Split(string(envBytes), "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "STORAGE_ENCRYPTION_KEY=") {
				secret = strings.Trim(strings.TrimPrefix(line, "STORAGE_ENCRYPTION_KEY="), "\"' ")
				break
			}
		}
	}

	db, err := sql.Open("sqlite", "file:"+omniDBPath+"?mode=ro")
	if err != nil {
		return nil, fmt.Errorf("failed to open omniroute db: %w", err)
	}
	defer db.Close()

	if cfgPath == "" {
		cfgPath = filepath.Join(home, ".goroute", "config.yaml")
	}
	cfg, err := config.Load(cfgPath)
	if err != nil {
		cfg = config.DefaultConfig()
	}

	res := &SyncResult{Details: []string{}}

	// 1. Sync Provider Connections
	rows, err := db.Query("SELECT id, provider, auth_type, priority, is_active, refresh_token, access_token, api_key, provider_specific_data FROM provider_connections WHERE is_active = 1")
	if err != nil {
		return nil, fmt.Errorf("failed to query provider_connections: %w", err)
	}
	defer rows.Close()

	agyAccounts := make([]config.AccountConfig, 0)
	kiroAccounts := make([]config.AccountConfig, 0)

	for rows.Next() {
		var id, provider, authType string
		var priority, isActive int
		var encRefresh, encAccess, encAPIKey, specData sql.NullString

		if err := rows.Scan(&id, &provider, &authType, &priority, &isActive, &encRefresh, &encAccess, &encAPIKey, &specData); err != nil {
			continue
		}

		refresh, _ := DecryptOmniCredential(encRefresh.String, secret)
		access, _ := DecryptOmniCredential(encAccess.String, secret)
		apiKey, _ := DecryptOmniCredential(encAPIKey.String, secret)

		var specMap map[string]interface{}
		if specData.Valid && specData.String != "" {
			_ = json.Unmarshal([]byte(specData.String), &specMap)
		}

		switch provider {
		case "agy", "antigravity":
			projID := ""
			if p, ok := specMap["projectId"].(string); ok {
				projID = p
			}
			accID := fmt.Sprintf("agy-acc-%d", len(agyAccounts)+1)
			agyAccounts = append(agyAccounts, config.AccountConfig{
				ID:           accID,
				Pool:         "pool1",
				RefreshToken: refresh,
				AccessToken:  access,
				AccountID:    projID,
				Weight:       1,
			})
			res.AccountsImported++
			res.Details = append(res.Details, fmt.Sprintf("Imported Antigravity account %s (project: %s)", accID, projID))

		case "kiro":
			profileARN := ""
			if p, ok := specMap["profileArn"].(string); ok {
				profileARN = p
			}
			if profileARN == "" {
				profileARN = "arn:aws:codewhisperer:us-east-1:699475941385:profile/EHGA3GRVQMUK"
			}
			accID := fmt.Sprintf("kiro-acc-%d", len(kiroAccounts)+1)
			kiroAccounts = append(kiroAccounts, config.AccountConfig{
				ID:           accID,
				Pool:         "pool1",
				RefreshToken: refresh,
				AccessToken:  access,
				ProfileARN:   profileARN,
				Weight:       1,
			})
			res.AccountsImported++
			res.Details = append(res.Details, fmt.Sprintf("Imported Kiro account %s", accID))

		case "agentrouter":
			addOrUpdateProvider(cfg, config.ProviderConfig{
				ID:      "agentrouter",
				Type:    "openai",
				BaseURL: "https://agentrouter.org/v1",
				APIKey:  apiKey,
			})
			res.ProvidersImported++
			res.Details = append(res.Details, "Imported AgentRouter provider")

		case "xkiro":
			addOrUpdateProvider(cfg, config.ProviderConfig{
				ID:      "xkiro",
				Type:    "openai",
				BaseURL: "https://api.xkiro.com/v1",
				APIKey:  apiKey,
			})
			res.ProvidersImported++
			res.Details = append(res.Details, "Imported xKiro provider")

		default:
			if strings.HasPrefix(provider, "openai-compatible") {
				bURL := "https://api.apinex.bond/v1"
				if u, ok := specMap["baseUrl"].(string); ok && u != "" {
					bURL = u
				}
				addOrUpdateProvider(cfg, config.ProviderConfig{
					ID:      "openai-compatible",
					Type:    "openai",
					BaseURL: bURL,
					APIKey:  apiKey,
				})
				res.ProvidersImported++
				res.Details = append(res.Details, fmt.Sprintf("Imported OpenAI-compatible provider (%s)", bURL))
			}
		}
	}

	// Update Antigravity pool with all discovered accounts
	if len(agyAccounts) > 0 {
		addOrUpdateProvider(cfg, config.ProviderConfig{
			ID:           "antigravity",
			Type:         "antigravity",
			RefreshToken: agyAccounts[0].RefreshToken,
			Accounts:     agyAccounts,
			Strategy:     "priority",
		})
		res.ProvidersImported++
	}

	// Update Kiro pool
	if len(kiroAccounts) > 0 {
		addOrUpdateProvider(cfg, config.ProviderConfig{
			ID:           "kiro",
			Type:         "kiro",
			RefreshToken: kiroAccounts[0].RefreshToken,
			ProfileARN:   kiroAccounts[0].ProfileARN,
			Accounts:     kiroAccounts,
			Strategy:     "priority",
		})
		res.ProvidersImported++
	}

	// 2. Sync Combos
	comboRows, err := db.Query("SELECT name, data FROM combos")
	if err == nil {
		defer comboRows.Close()
		for comboRows.Next() {
			var name, data string
			if err := comboRows.Scan(&name, &data); err != nil {
				continue
			}
			var cData struct {
				Models []struct {
					Model      string `json:"model"`
					ProviderID string `json:"providerId"`
				} `json:"models"`
			}
			if err := json.Unmarshal([]byte(data), &cData); err == nil && len(cData.Models) > 0 {
				// Build a set of known configured provider IDs
				knownProviders := make(map[string]bool)
				for _, p := range cfg.Providers {
					knownProviders[p.ID] = true
				}

				targets := make([]config.ComboTarget, 0, len(cData.Models))
				for _, m := range cData.Models {
					prov := m.ProviderID
					if prov == "agy" {
						prov = "antigravity"
					} else if prov == "kr" {
						prov = "kiro"
					}
					// Only include targets pointing to existing, configured providers
					if !knownProviders[prov] {
						continue
					}
					modelName := m.Model
					if strings.Contains(modelName, "/") {
						parts := strings.SplitN(modelName, "/", 2)
						modelName = parts[1]
					}
					targets = append(targets, config.ComboTarget{
						ProviderID: prov,
						Model:      modelName,
					})
				}
				if len(targets) > 0 {
					addOrUpdateCombo(cfg, config.ComboConfig{
						Name:     name,
						Strategy: "failover",
						Targets:  targets,
					})
					res.CombosImported++
					res.Details = append(res.Details, fmt.Sprintf("Imported combo %s (%d targets)", name, len(targets)))
				}
			}
		}
	}

	// Save back to config.yaml
	if err := cfg.Save(cfgPath); err != nil {
		return nil, fmt.Errorf("failed to save config: %w", err)
	}

	return res, nil
}

func addOrUpdateProvider(cfg *config.Config, newP config.ProviderConfig) {
	for i, p := range cfg.Providers {
		if p.ID == newP.ID {
			cfg.Providers[i] = newP
			return
		}
	}
	cfg.Providers = append(cfg.Providers, newP)
}

func addOrUpdateCombo(cfg *config.Config, newC config.ComboConfig) {
	for i, c := range cfg.Combos {
		if c.Name == newC.Name {
			cfg.Combos[i] = newC
			return
		}
	}
	cfg.Combos = append(cfg.Combos, newC)
}
