package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/ziuus/goroute/pkg/config"
)

type AgentSetupOptions struct {
	Provider   string   `json:"provider,omitempty"`
	Model      string   `json:"model,omitempty"`
	SmallModel string   `json:"small_model,omitempty"`
	Models     []string `json:"models,omitempty"`
	BaseURL    string   `json:"base_url,omitempty"`
	APIKey     string   `json:"api_key,omitempty"`
}

type AgentConfigurator func(home string, opts AgentSetupOptions) error

var agentConfigurators = map[string]struct {
	Desc string
	Run  AgentConfigurator
}{
	"claude": {
		Desc: "Claude Code CLI (~/.claude/settings.json)",
		Run:  setupClaude,
	},
	"codex": {
		Desc: "OpenAI Codex CLI (~/.codex/config.toml)",
		Run:  setupCodex,
	},
	"opencode": {
		Desc: "OpenCode CLI (~/.config/opencode/opencode.json)",
		Run:  setupOpenCode,
	},
	"cursor": {
		Desc: "Cursor IDE (settings and connection instructions)",
		Run:  setupCursor,
	},
	"cline": {
		Desc: "Cline Agent (~/.cline/data/secrets.json & globalState.json)",
		Run:  setupCline,
	},
	"roo": {
		Desc: "Roo Code (~/.roo/settings-import.json)",
		Run:  setupRoo,
	},
	"kilo": {
		Desc: "Kilo Code (~/.local/share/kilo/auth.json)",
		Run:  setupKilo,
	},
	"aider": {
		Desc: "Aider Assistant (~/.aider.conf.yml)",
		Run:  setupAider,
	},
	"continue": {
		Desc: "Continue.dev & cn CLI (~/.continue/config.yaml)",
		Run:  setupContinue,
	},
	"crush": {
		Desc: "Charm Crush (~/.config/crush/crush.json)",
		Run:  setupCrush,
	},
	"goose": {
		Desc: "Block Goose (~/.config/goose/config.yaml)",
		Run:  setupGoose,
	},
	"qwen": {
		Desc: "Qwen Code (~/.qwen/settings.json)",
		Run:  setupQwen,
	},
	"gemini": {
		Desc: "Google Gemini CLI (environment & command setup)",
		Run:  setupGemini,
	},
	"5dive": {
		Desc: "5dive Agent Fleet (auth profile setup)",
		Run:  setup5dive,
	},
	"hermes": {
		Desc: "Hermes Agent CLI (~/.hermes/config.yaml)",
		Run:  setupHermes,
	},
	"openclaw": {
		Desc: "OpenClaw Autonomous Gateway (~/.openclaw/openclaw.json)",
		Run:  setupOpenClaw,
	},
}

func NewSetupCmd() *cobra.Command {
	var provider, model, smallModel, modelsStr, baseURL, apiKey string

	setupCmd := &cobra.Command{
		Use:   "setup [agent|all]",
		Short: "Configure any AI coding agent to connect to GoRoute (Claude, Codex, OpenCode, Aider, etc.)",
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("could not determine user home directory: %w", err)
			}

			var modelsList []string
			if modelsStr != "" {
				for _, m := range strings.Split(modelsStr, ",") {
					m = strings.TrimSpace(m)
					if m != "" {
						modelsList = append(modelsList, m)
					}
				}
			}

			opts := AgentSetupOptions{
				Provider:   provider,
				Model:      model,
				SmallModel: smallModel,
				Models:     modelsList,
				BaseURL:    baseURL,
				APIKey:     apiKey,
			}

			if len(args) == 0 {
				fmt.Println("🤖 Available Agent Setups in GoRoute:")
				for k, v := range agentConfigurators {
					fmt.Printf("  goroute setup %-10s - %s\n", k, v.Desc)
				}
				fmt.Println("  goroute setup all        - Configure ALL supported agents in one click")
				fmt.Println("\nRun any agent directly through GoRoute:")
				fmt.Println("  goroute run <command> [args...]")
				fmt.Println("  goroute launch <agent> [args...]")
				return nil
			}

			target := strings.ToLower(args[0])

			if target == "all" {
				fmt.Println("🚀 Configuring all supported AI coding agents for GoRoute...")
				successCount := 0
				for k, v := range agentConfigurators {
					fmt.Printf("\n▶ Setting up %s...\n", k)
					if err := v.Run(home, opts); err != nil {
						fmt.Printf("  ⚠️  Failed to setup %s: %v\n", k, err)
					} else {
						successCount++
					}
				}
				fmt.Printf("\n🎉 Setup complete! %d/%d agents successfully configured.\n", successCount, len(agentConfigurators))
				return nil
			}

			cfg, exists := agentConfigurators[target]
			if !exists {
				return fmt.Errorf("unknown agent '%s'. Run 'goroute setup' to view supported agents", target)
			}

			return cfg.Run(home, opts)
		},
	}

	setupCmd.Flags().StringVarP(&provider, "provider", "p", "", "Target upstream provider (e.g. antigravity, kiro, codex, or auto)")
	setupCmd.Flags().StringVarP(&model, "model", "m", "", "Primary / default model for this agent (e.g. claude-3-7-sonnet, gemini-2.5-flash, or combo name)")
	setupCmd.Flags().StringVarP(&smallModel, "small-model", "s", "", "Small / fast / planning model for this agent (e.g. gemini-2.5-flash)")
	setupCmd.Flags().StringVar(&modelsStr, "models", "", "Comma-separated list of models/combos to expose to this agent")
	setupCmd.Flags().StringVar(&baseURL, "url", "", "Custom GoRoute base URL (default: http://127.0.0.1:8128/v1)")
	setupCmd.Flags().StringVar(&apiKey, "key", "", "API key (default: goroute)")

	return setupCmd
}

type ResolvedModelInfo struct {
	FullID       string
	ProviderID   string
	ModelName    string
	DisplayName  string
	ProviderName string
	IsCombo      bool
	ContextLen   int
	OutputLen    int
}

func getProviderDisplayName(pID string) string {
	switch strings.ToLower(pID) {
	case "antigravity", "agy":
		return "Antigravity (Google Cloud Vertex AI)"
	case "kiro":
		return "Kiro (AWS Bedrock CodeWhisperer)"
	case "codex":
		return "Codex (OpenAI ChatGPT Core)"
	case "openai":
		return "OpenAI Native API"
	case "mistral":
		return "Mistral AI"
	case "agentrouter":
		return "AgentRouter Gateway"
	case "tokenrouter":
		return "TokenRouter API"
	case "xkiro":
		return "XKiro API"
	case "combo":
		return "GoRoute Failover Combo"
	default:
		if pID == "" || pID == "goroute" || pID == "auto" {
			return "GoRoute Auto Affinity"
		}
		return fmt.Sprintf("%s Provider", strings.ToUpper(pID))
	}
}

func ResolveModel(home string, modelInput string, prefProvider string, fallbackDefault string) ResolvedModelInfo {
	if modelInput == "" {
		modelInput = fallbackDefault
	}
	modelInput = strings.TrimSpace(modelInput)

	// Strip common client prefixes
	for _, p := range []string{"goroute/", "gr/", "omniroute/"} {
		if strings.HasPrefix(modelInput, p) {
			modelInput = strings.TrimPrefix(modelInput, p)
			break
		}
	}

	var provID, modelName string
	if strings.Contains(modelInput, "/") {
		parts := strings.SplitN(modelInput, "/", 2)
		provID = parts[0]
		modelName = parts[1]
	} else {
		modelName = modelInput
	}

	if prefProvider != "" && prefProvider != "auto" && provID == "" {
		provID = prefProvider
	}

	// Check if this model is a configured combo
	cfgPath := filepath.Join(home, ".goroute", "config.yaml")
	if cfg, err := config.Load(cfgPath); err == nil && cfg != nil {
		for _, combo := range cfg.Combos {
			if strings.EqualFold(combo.Name, modelName) {
				return ResolvedModelInfo{
					FullID:       combo.Name,
					ProviderID:   "combo",
					ModelName:    combo.Name,
					DisplayName:  fmt.Sprintf("%s (Failover Combo)", combo.Name),
					ProviderName: fmt.Sprintf("GoRoute Combo (%s)", combo.Strategy),
					IsCombo:      true,
					ContextLen:   200000,
					OutputLen:    8192,
				}
			}
		}
	}
	for _, cName := range []string{"frontier", "claude-models", "claude", "auto", "deepseek", "claude-opus"} {
		if strings.EqualFold(cName, modelName) {
			return ResolvedModelInfo{
				FullID:       cName,
				ProviderID:   "combo",
				ModelName:    cName,
				DisplayName:  fmt.Sprintf("%s (Failover Combo)", cName),
				ProviderName: "GoRoute Failover Combo",
				IsCombo:      true,
				ContextLen:   200000,
				OutputLen:    8192,
			}
		}
	}

	// Resolve based on model affinity if no provider is designated
	if provID == "" || provID == "auto" {
		lower := strings.ToLower(modelName)
		if strings.Contains(lower, "gemini") {
			provID = "antigravity"
		} else if strings.Contains(lower, "claude") || strings.Contains(lower, "sonnet") || strings.Contains(lower, "opus") || strings.Contains(lower, "haiku") {
			provID = "kiro"
		} else if strings.Contains(lower, "gpt") || strings.Contains(lower, "o1") || strings.Contains(lower, "o3") || strings.Contains(lower, "codex") {
			provID = "codex"
		} else if strings.Contains(lower, "deepseek") || strings.Contains(lower, "qwen") {
			provID = "kiro"
		} else {
			provID = "goroute"
		}
	}

	fullID := modelName
	if provID != "goroute" && provID != "" {
		fullID = fmt.Sprintf("%s/%s", provID, modelName)
	}

	ctxLen := 128000
	if strings.Contains(modelName, "gemini") {
		ctxLen = 1048576
	} else if strings.Contains(modelName, "claude") {
		ctxLen = 200000
	}

	return ResolvedModelInfo{
		FullID:       fullID,
		ProviderID:   provID,
		ModelName:    modelName,
		DisplayName:  fmt.Sprintf("%s (%s)", modelName, provID),
		ProviderName: getProviderDisplayName(provID),
		IsCombo:      false,
		ContextLen:   ctxLen,
		OutputLen:    8192,
	}
}

func getAllGoRouteModels(home string) map[string]string {
	models := map[string]string{
		// Antigravity (Google Cloud Vertex) models
		"antigravity/gemini-2.5-flash":         "Gemini 2.5 Flash (Antigravity)",
		"antigravity/gemini-3.8-flash-high":    "Gemini 3.8 Flash High (Antigravity)",
		"antigravity/gemini-2.5-pro":           "Gemini 2.5 Pro (Antigravity)",
		"antigravity/gemini-3-flash":           "Gemini 3 Flash (Antigravity)",
		"antigravity/gemini-3.1-flash-lite":    "Gemini 3.1 Flash Lite (Antigravity)",
		"antigravity/claude-3-7-sonnet":        "Claude 3.7 Sonnet (Antigravity)",
		"antigravity/claude-3-5-sonnet":        "Claude 3.5 Sonnet (Antigravity)",
		"antigravity/claude-sonnet-4-6":        "Claude Sonnet 4.6 (Antigravity)",
		"antigravity/claude-opus-4-6-thinking": "Claude Opus 4.6 Thinking (Antigravity)",
		"antigravity/gpt-oss-120b-medium":      "GPT-OSS 120B (Antigravity)",

		// Kiro (AWS Bedrock) models
		"kiro/claude-3-7-sonnet": "Claude 3.7 Sonnet (Kiro)",
		"kiro/claude-sonnet-4.5": "Claude Sonnet 4.5 (Kiro)",
		"kiro/claude-haiku-4.5":  "Claude Haiku 4.5 (Kiro)",
		"kiro/deepseek-3.2":      "DeepSeek 3.2 (Kiro)",
		"kiro/qwen3-coder-next":  "Qwen3 Coder Next (Kiro)",
		"kiro/gpt-5.6-luna":      "GPT-5.6 Luna (Kiro)",

		// Codex (OpenAI) models
		"codex/gpt-5.6-sol":      "GPT-5.6 Sol (Codex)",
		"codex/gpt-5.6-sol-high": "GPT-5.6 Sol High (Codex)",
		"codex/o3-mini":          "o3-mini (Codex)",
		"codex/o1":               "o1 (Codex)",
		"codex/gpt-4o":           "GPT-4o (Codex)",

		// Curated un-prefixed auto aliases
		"claude-3-7-sonnet":     "Claude 3.7 Sonnet (Auto Affinity)",
		"claude-3-5-sonnet":     "Claude 3.5 Sonnet (Auto Affinity)",
		"gemini-2.5-flash":      "Gemini 2.5 Flash (Auto Affinity)",
		"gemini-2.5-pro":        "Gemini 2.5 Pro (Auto Affinity)",
		"gemini-3.8-flash-high": "Gemini 3.8 Flash High (Auto Affinity)",
		"gpt-4o":                "GPT-4o (Auto Affinity)",
	}

	cfgPath := filepath.Join(home, ".goroute", "config.yaml")
	if cfg, err := config.Load(cfgPath); err == nil && cfg != nil {
		for _, combo := range cfg.Combos {
			models[combo.Name] = fmt.Sprintf("%s (Failover Combo)", combo.Name)
		}
		for _, p := range cfg.Providers {
			if p.Type == "openai" {
				models[p.ID+"/default"] = fmt.Sprintf("%s (OpenAI-compatible)", p.ID)
			}
		}
	}
	return models
}

func setupClaude(home string, opts AgentSetupOptions) error {
	claudeDir := filepath.Join(home, ".claude")
	_ = os.MkdirAll(claudeDir, 0700)
	settingsPath := filepath.Join(claudeDir, "settings.json")

	data := make(map[string]interface{})
	if content, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(content, &data)
	}

	targetURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		targetURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}
	targetKey := "goroute"
	if opts.APIKey != "" {
		targetKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "frontier")

	data["customBaseUrl"] = targetURL

	env, ok := data["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
	}
	env["ANTHROPIC_BASE_URL"] = targetURL
	env["ANTHROPIC_API_KEY"] = targetKey
	data["model"] = primary.FullID
	env["ANTHROPIC_MODEL"] = primary.FullID
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = primary.FullID
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = primary.FullID
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = primary.FullID
	delete(env, "ANTHROPIC_DEFAULT_FABLE_MODEL")
	delete(env, "CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY")
	data["env"] = env

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, bytes, 0600); err != nil {
		return err
	}

	// Generate profile directories under ~/.claude/profiles/ for 1-click launch
	profilesDir := filepath.Join(claudeDir, "profiles")
	_ = os.MkdirAll(profilesDir, 0700)
	namedProfiles := []struct {
		Name     string
		Model    string
		Provider string
	}{
		{"kiro", "kiro/claude-3-7-sonnet", "Kiro"},
		{"antigravity", "antigravity/claude-sonnet-4-6", "Antigravity"},
		{"claude-models", "claude-models", "GoRoute Combo"},
		{"gemini", "antigravity/gemini-2.5-flash", "Antigravity"},
	}
	for _, p := range namedProfiles {
		pDir := filepath.Join(profilesDir, p.Name)
		_ = os.MkdirAll(pDir, 0700)
		pSettings := map[string]interface{}{
			"$schema": "https://json.schemastore.org/claude-code-settings.json",
			"model":   p.Model,
			"env": map[string]interface{}{
				"ANTHROPIC_BASE_URL":                         targetURL,
				"ANTHROPIC_API_KEY":                          targetKey,
				"ANTHROPIC_MODEL":                            p.Model,
				"CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY": "1",
			},
		}
		if pBytes, err := json.MarshalIndent(pSettings, "", "  "); err == nil {
			_ = os.WriteFile(filepath.Join(pDir, "settings.json"), pBytes, 0600)
		}
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Claude Code CLI")
	fmt.Println("======================================================================")
	fmt.Printf("  Gateway Base URL:   %s\n", targetURL)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	fmt.Printf("  Settings File:      %s\n", settingsPath)
	fmt.Printf("  Named Profiles:     %s (kiro, antigravity, claude-models, gemini)\n", profilesDir)
	fmt.Println("\nLaunch with:")
	fmt.Println("  claude")
	fmt.Printf("  OR: CLAUDE_CONFIG_DIR=%s/antigravity claude\n", profilesDir)
	return nil
}

func setupCodex(home string, opts AgentSetupOptions) error {
	codexDir := filepath.Join(home, ".codex")
	_ = os.MkdirAll(codexDir, 0700)
	profilePath := filepath.Join(codexDir, "goroute.config.toml")
	configPath := filepath.Join(codexDir, "config.toml")

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	targetURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		targetURL = opts.BaseURL
	}

	tomlContent := fmt.Sprintf(`# GoRoute Configuration for OpenAI Codex CLI
model = "%s"
model_provider = "goroute"
model_context_window = %d
model_auto_compact_token_limit = 112000
tool_output_token_limit = 32768

[model_providers.goroute]
name = "GoRoute"
base_url = "%s"
env_key = "OPENAI_API_KEY"
wire_api = "responses"
requires_openai_auth = false
`, primary.FullID, primary.ContextLen, targetURL)

	if err := os.WriteFile(profilePath, []byte(tomlContent), 0600); err != nil {
		return err
	}

	// Also write or append to config.toml if it doesn't have goroute
	existing, _ := os.ReadFile(configPath)
	if !strings.Contains(string(existing), "[model_providers.goroute]") {
		f, err := os.OpenFile(configPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0600)
		if err == nil {
			_, _ = f.WriteString("\n" + tomlContent)
			f.Close()
		}
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: OpenAI Codex CLI")
	fmt.Println("======================================================================")
	fmt.Printf("  Gateway Base URL:   %s\n", targetURL)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	fmt.Printf("  Profile Path:       %s\n", profilePath)
	fmt.Println("\nLaunch with:")
	fmt.Println("  goroute run codex")
	fmt.Println("  OR: codex --profile goroute")
	return nil
}

func setupOpenCode(home string, opts AgentSetupOptions) error {
	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")
	small := ResolveModel(home, opts.SmallModel, opts.Provider, "gemini-2.5-flash")

	// 1. Register gr and goroute credentials in ~/.local/share/opencode/auth.json
	authDir := filepath.Join(home, ".local", "share", "opencode")
	_ = os.MkdirAll(authDir, 0700)
	authPath := filepath.Join(authDir, "auth.json")
	authExisting := make(map[string]interface{})
	if raw, err := os.ReadFile(authPath); err == nil {
		_ = json.Unmarshal(raw, &authExisting)
	}
	authExisting["gr"] = map[string]interface{}{
		"type": "api",
		"key":  apiKey,
	}
	authExisting["goroute"] = map[string]interface{}{
		"type": "api",
		"key":  apiKey,
	}
	if aBytes, err := json.MarshalIndent(authExisting, "", "  "); err == nil {
		_ = os.WriteFile(authPath, aBytes, 0600)
	}

	// 2. Discover all models from GoRoute or use requested models
	allModels := getAllGoRouteModels(home)
	modelsMap := make(map[string]interface{})

	if len(opts.Models) > 0 {
		for _, m := range opts.Models {
			m = strings.TrimSpace(m)
			if m == "" {
				continue
			}
			resolved := ResolveModel(home, m, opts.Provider, m)
			modelsMap[resolved.FullID] = map[string]interface{}{
				"name": resolved.DisplayName,
				"limit": map[string]interface{}{
					"context": resolved.ContextLen,
					"output":  resolved.OutputLen,
				},
			}
		}
	} else {
		for modelID, modelName := range allModels {
			ctxLen := 128000
			if strings.Contains(modelID, "gemini") {
				ctxLen = 1048576
			} else if strings.Contains(modelID, "claude") {
				ctxLen = 200000
			}
			modelsMap[modelID] = map[string]interface{}{
				"name": modelName,
				"limit": map[string]interface{}{
					"context": ctxLen,
					"output":  8192,
				},
			}
		}
	}

	// Ensure primary and small models are present in the models map
	modelsMap[primary.FullID] = map[string]interface{}{
		"name": primary.DisplayName,
		"limit": map[string]interface{}{
			"context": primary.ContextLen,
			"output":  primary.OutputLen,
		},
	}
	modelsMap[small.FullID] = map[string]interface{}{
		"name": small.DisplayName,
		"limit": map[string]interface{}{
			"context": small.ContextLen,
			"output":  small.OutputLen,
		},
	}

	// 3. Configure ~/.config/opencode/opencode.json
	opencodeDir := filepath.Join(home, ".config", "opencode")
	_ = os.MkdirAll(opencodeDir, 0700)
	configPath := filepath.Join(opencodeDir, "opencode.json")

	existing := make(map[string]interface{})
	if raw, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(raw, &existing)
	}

	providers, ok := existing["provider"].(map[string]interface{})
	if !ok || providers == nil {
		providers = make(map[string]interface{})
	}

	grProvider := map[string]interface{}{
		"name": "GoRoute",
		"npm":  "@ai-sdk/openai-compatible",
		"options": map[string]interface{}{
			"baseURL": baseURL,
			"apiKey":  apiKey,
		},
		"models": modelsMap,
	}

	providers["gr"] = grProvider
	providers["goroute"] = grProvider

	existing["provider"] = providers
	existing["model"] = "goroute/" + primary.FullID
	existing["small_model"] = "goroute/" + small.FullID
	existing["agent"] = map[string]interface{}{
		"build": map[string]interface{}{
			"model": "goroute/" + primary.FullID,
		},
		"plan": map[string]interface{}{
			"model": "goroute/" + primary.FullID,
		},
	}

	bytes, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(configPath, bytes, 0600); err != nil {
		return err
	}

	// 4. Also sync to opencode.jsonc if present
	jsoncPath := filepath.Join(opencodeDir, "opencode.jsonc")
	if raw, err := os.ReadFile(jsoncPath); err == nil {
		var existingJsonc map[string]interface{}
		if err := json.Unmarshal(raw, &existingJsonc); err == nil {
			p, ok := existingJsonc["provider"].(map[string]interface{})
			if !ok || p == nil {
				p = make(map[string]interface{})
			}
			p["gr"] = grProvider
			p["goroute"] = grProvider
			existingJsonc["provider"] = p
			existingJsonc["model"] = "goroute/" + primary.FullID
			existingJsonc["small_model"] = "goroute/" + small.FullID
			if jBytes, err := json.MarshalIndent(existingJsonc, "", "  "); err == nil {
				_ = os.WriteFile(jsoncPath, jBytes, 0600)
			}
		}
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: OpenCode CLI")
	fmt.Println("======================================================================")
	fmt.Printf("  Gateway Base URL:   %s\n", baseURL)
	fmt.Printf("  Primary Model:      goroute/%s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	fmt.Printf("  Fast/Small Model:   goroute/%s\n", small.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", small.ProviderName)
	fmt.Printf("  OpenCode Provider:  \"goroute\" (and \"gr\" alias)\n")
	fmt.Printf("  Config File:        %s\n", configPath)
	fmt.Printf("  Auth Secrets:       %s\n", authPath)
	fmt.Printf("\n  Configured %d GoRoute models across providers:\n", len(modelsMap))
	count := 0
	for m := range modelsMap {
		if count < 8 {
			fmt.Printf("  • goroute/%s\n", m)
			count++
		}
	}
	if len(modelsMap) > 8 {
		fmt.Printf("  ... and %d more models\n", len(modelsMap)-8)
	}
	fmt.Println("\nLaunch with:")
	fmt.Printf("  opencode\n")
	fmt.Printf("  OR: opencode -m goroute/%s\n", primary.FullID)
	fmt.Println("  OR: goroute run opencode")
	return nil
}

func setupCursor(home string, opts AgentSetupOptions) error {
	cursorDir := filepath.Join(home, ".config", "Cursor", "User")
	settingsPath := filepath.Join(cursorDir, "settings.json")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	if _, err := os.Stat(cursorDir); err == nil {
		existing := make(map[string]interface{})
		if raw, err := os.ReadFile(settingsPath); err == nil {
			_ = json.Unmarshal(raw, &existing)
		}
		existing["cursor.openaiBaseUrl"] = baseURL
		existing["cursor.openaiApiKey"] = apiKey
		existing["cursor.model"] = primary.FullID
		if bytes, err := json.MarshalIndent(existing, "", "  "); err == nil {
			_ = os.WriteFile(settingsPath, bytes, 0600)
			fmt.Printf("✅ Updated Cursor settings at: %s\n", settingsPath)
		}
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Cursor IDE")
	fmt.Println("======================================================================")
	fmt.Println("  1. Open Cursor Settings (Ctrl+,) → Models")
	fmt.Printf("  2. Turn on 'Override OpenAI Base URL' and enter: %s\n", baseURL)
	fmt.Printf("  3. Set 'OpenAI API Key' to: %s\n", apiKey)
	fmt.Printf("  4. Add model: %s\n", primary.FullID)
	fmt.Printf("     └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupCline(home string, opts AgentSetupOptions) error {
	clineDataDir := filepath.Join(home, ".cline", "data")
	_ = os.MkdirAll(clineDataDir, 0700)

	secretsPath := filepath.Join(clineDataDir, "secrets.json")
	statePath := filepath.Join(clineDataDir, "globalState.json")

	baseURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		baseURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")
	plan := primary
	if opts.SmallModel != "" {
		plan = ResolveModel(home, opts.SmallModel, opts.Provider, "gemini-2.5-flash")
	}

	secrets := map[string]string{"openAiApiKey": apiKey}
	sBytes, _ := json.MarshalIndent(secrets, "", "  ")
	_ = os.WriteFile(secretsPath, sBytes, 0600)

	state := map[string]interface{}{
		"actModeApiProvider":    "openai",
		"planModeApiProvider":   "openai",
		"openAiBaseUrl":         baseURL,
		"openAiModelId":         primary.FullID,
		"planModeOpenAiModelId": plan.FullID,
	}
	stBytes, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile(statePath, stBytes, 0600)

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Cline CLI & Standalone")
	fmt.Println("======================================================================")
	fmt.Printf("  Gateway Base URL:   %s\n", baseURL)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	fmt.Printf("  Plan Model:         %s\n", plan.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", plan.ProviderName)
	fmt.Printf("  Secrets Path:       %s\n", secretsPath)
	return nil
}

func setupRoo(home string, opts AgentSetupOptions) error {
	rooDir := filepath.Join(home, ".roo")
	_ = os.MkdirAll(rooDir, 0700)
	importPath := filepath.Join(rooDir, "settings-import.json")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	importDoc := map[string]interface{}{
		"providerProfiles": map[string]interface{}{
			"currentApiConfigName": "GoRoute",
			"apiConfigs": map[string]interface{}{
				"GoRoute": map[string]interface{}{
					"apiProvider":   "openai",
					"openAiBaseUrl": baseURL,
					"openAiApiKey":  apiKey,
					"openAiModelId": primary.FullID,
				},
			},
		},
	}
	bytes, _ := json.MarshalIndent(importDoc, "", "  ")
	if err := os.WriteFile(importPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Roo Code")
	fmt.Println("======================================================================")
	fmt.Printf("  Import Document:    %s\n", importPath)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupKilo(home string, opts AgentSetupOptions) error {
	kiloDir := filepath.Join(home, ".local", "share", "kilo")
	_ = os.MkdirAll(kiloDir, 0700)
	authPath := filepath.Join(kiloDir, "auth.json")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	auth := map[string]interface{}{
		"openai-compatible": map[string]string{
			"apiKey":  apiKey,
			"baseUrl": baseURL,
			"model":   primary.FullID,
		},
	}
	bytes, _ := json.MarshalIndent(auth, "", "  ")
	if err := os.WriteFile(authPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Kilo Code")
	fmt.Println("======================================================================")
	fmt.Printf("  Config Path:        %s\n", authPath)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupAider(home string, opts AgentSetupOptions) error {
	aiderConf := filepath.Join(home, ".aider.conf.yml")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	content := fmt.Sprintf("openai-api-base: %s\nopenai-api-key: %s\nmodel: openai/%s\n", baseURL, apiKey, primary.FullID)
	if err := os.WriteFile(aiderConf, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Aider")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", aiderConf)
	fmt.Printf("  Active Model:       openai/%s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupContinue(home string, opts AgentSetupOptions) error {
	contDir := filepath.Join(home, ".continue")
	_ = os.MkdirAll(contDir, 0700)
	confPath := filepath.Join(contDir, "config.yaml")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")
	small := ResolveModel(home, opts.SmallModel, opts.Provider, "gemini-2.5-flash")

	content := fmt.Sprintf(`name: GoRoute Config
version: "1.0"
schema: v1
models:
  - name: "GoRoute: %s"
    provider: openai
    model: %s
    apiBase: %s
    apiKey: %s
    roles: [chat, edit, apply]
  - name: "GoRoute: %s"
    provider: openai
    model: %s
    apiBase: %s
    apiKey: %s
    roles: [chat, edit, apply]
`, primary.FullID, primary.FullID, baseURL, apiKey, small.FullID, small.FullID, baseURL, apiKey)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Continue (continue.dev & cn CLI)")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", confPath)
	fmt.Printf("  Primary Model:      %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	fmt.Printf("  Fast/Small Model:   %s\n", small.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", small.ProviderName)
	return nil
}

func setupCrush(home string, opts AgentSetupOptions) error {
	crushDir := filepath.Join(home, ".config", "crush")
	_ = os.MkdirAll(crushDir, 0700)
	confPath := filepath.Join(crushDir, "crush.json")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	content := fmt.Sprintf(`{
  "providers": {
    "goroute": {
      "type": "openai-compat",
      "base_url": "%s",
      "api_key": "%s",
      "models": [
        {"id": "%s", "name": "GoRoute %s"}
      ]
    }
  }
}
`, baseURL, apiKey, primary.FullID, primary.DisplayName)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Charm Crush")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", confPath)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupGoose(home string, opts AgentSetupOptions) error {
	gooseDir := filepath.Join(home, ".config", "goose")
	_ = os.MkdirAll(gooseDir, 0700)
	confPath := filepath.Join(gooseDir, "config.yaml")

	baseURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		baseURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	content := fmt.Sprintf("GOOSE_PROVIDER: openai\nOPENAI_HOST: %s\nGOOSE_MODEL: %s\n", baseURL, primary.FullID)
	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Goose")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", confPath)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupQwen(home string, opts AgentSetupOptions) error {
	qwenDir := filepath.Join(home, ".qwen")
	_ = os.MkdirAll(qwenDir, 0700)
	confPath := filepath.Join(qwenDir, "settings.json")

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	data := map[string]string{
		"baseUrl": baseURL,
		"apiKey":  apiKey,
		"model":   primary.FullID,
	}
	bytes, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(confPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Qwen Code")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", confPath)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupGemini(home string, opts AgentSetupOptions) error {
	baseURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		baseURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Google Gemini CLI")
	fmt.Println("======================================================================")
	fmt.Printf("  Set the following environment variables in your shell:\n")
	fmt.Printf("    export GOOGLE_GEMINI_BASE_URL=\"%s\"\n", baseURL)
	fmt.Printf("    export GEMINI_API_KEY=\"%s\"\n", apiKey)
	return nil
}

func setup5dive(home string, opts AgentSetupOptions) error {
	baseURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		baseURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "claude-3-7-sonnet")

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: 5dive Agent Fleet")
	fmt.Println("======================================================================")
	fmt.Println("  To connect your 5dive agent fleet to GoRoute, run:")
	fmt.Printf("    5dive agent auth set claude \\\n")
	fmt.Printf("      --provider=openai \\\n")
	fmt.Printf("      --base-url=%s \\\n", baseURL)
	fmt.Printf("      --auth-profile=goroute \\\n")
	fmt.Printf("      --model=%s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupHermes(home string, opts AgentSetupOptions) error {
	dir := filepath.Join(home, ".hermes")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfgFile := filepath.Join(dir, "config.yaml")
	content := ""
	if b, err := os.ReadFile(cfgFile); err == nil {
		content = string(b)
	}

	baseURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		baseURL = opts.BaseURL
	}
	apiKey := "goroute"
	if opts.APIKey != "" {
		apiKey = opts.APIKey
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "gemini-2.5-flash")

	gorouteBlock := fmt.Sprintf("providers:\n  goroute:\n    base_url: %s\n    api_key: %s\n    default_model: %s\n", baseURL, apiKey, primary.FullID)
	if strings.Contains(content, "providers:") {
		if !strings.Contains(content, "goroute:") {
			content = strings.Replace(content, "providers:", gorouteBlock, 1)
		}
	} else {
		content = gorouteBlock + content
	}

	if err := os.WriteFile(cfgFile, []byte(content), 0644); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: Hermes Agent")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", cfgFile)
	fmt.Printf("  Active Model:       %s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

func setupOpenClaw(home string, opts AgentSetupOptions) error {
	dir := filepath.Join(home, ".openclaw")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	cfgFile := filepath.Join(dir, "openclaw.json")
	var root map[string]interface{}
	if b, err := os.ReadFile(cfgFile); err == nil {
		_ = json.Unmarshal(b, &root)
	}
	if root == nil {
		root = make(map[string]interface{})
	}

	// 1. Clean invalid keys that cause schema validation failure
	if meta, ok := root["meta"].(map[string]interface{}); ok {
		delete(meta, "lastTouchedAt")
	}
	if gw, ok := root["gateway"].(map[string]interface{}); ok {
		if ts, ok := gw["tailscale"].(map[string]interface{}); ok {
			delete(ts, "resetOnExit")
		}
		if nodes, ok := gw["nodes"].(map[string]interface{}); ok {
			delete(nodes, "denyCommands")
		}
	}

	primary := ResolveModel(home, opts.Model, opts.Provider, "gemini-2.5-flash")

	// 2. Configure agents defaults
	agents, ok := root["agents"].(map[string]interface{})
	if !ok {
		agents = make(map[string]interface{})
		root["agents"] = agents
	}
	defaults, ok := agents["defaults"].(map[string]interface{})
	if !ok {
		defaults = make(map[string]interface{})
		agents["defaults"] = defaults
	}
	models, ok := defaults["models"].(map[string]interface{})
	if !ok {
		models = make(map[string]interface{})
		defaults["models"] = models
	}
	models["goroute/"+primary.FullID] = map[string]interface{}{}
	defaults["model"] = map[string]interface{}{
		"primary": "goroute/" + primary.FullID,
	}

	out, err := json.MarshalIndent(root, "", "    ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgFile, out, 0644); err != nil {
		return err
	}

	fmt.Println("======================================================================")
	fmt.Println("  GoRoute Agent Configuration: OpenClaw Autonomous Gateway")
	fmt.Println("======================================================================")
	fmt.Printf("  Config File:        %s\n", cfgFile)
	fmt.Printf("  Active Model:       goroute/%s\n", primary.FullID)
	fmt.Printf("    └─ Upstream Provider: %s\n", primary.ProviderName)
	return nil
}

