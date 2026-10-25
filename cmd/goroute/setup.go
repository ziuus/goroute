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
	var model, smallModel, modelsStr, baseURL, apiKey string

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

	setupCmd.Flags().StringVarP(&model, "model", "m", "", "Primary / default model for this agent (e.g. claude-3-7-sonnet, gemini-2.5-flash, or combo name)")
	setupCmd.Flags().StringVarP(&smallModel, "small-model", "s", "", "Small / fast / planning model for this agent (e.g. gemini-2.5-flash)")
	setupCmd.Flags().StringVar(&modelsStr, "models", "", "Comma-separated list of models/combos to expose to this agent")
	setupCmd.Flags().StringVar(&baseURL, "url", "", "Custom GoRoute base URL (default: http://127.0.0.1:8128/v1)")
	setupCmd.Flags().StringVar(&apiKey, "key", "", "API key (default: goroute)")

	return setupCmd
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

	data["customBaseUrl"] = targetURL

	env, ok := data["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
	}
	env["ANTHROPIC_BASE_URL"] = targetURL
	env["ANTHROPIC_API_KEY"] = targetKey
	if opts.Model != "" {
		data["model"] = opts.Model
		env["ANTHROPIC_MODEL"] = opts.Model
		env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = opts.Model
	}
	data["env"] = env

	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(settingsPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Claude Code!")
	fmt.Printf("   Updated settings at: %s\n", settingsPath)
	if opts.Model != "" {
		fmt.Printf("   Active model set to: %s\n", opts.Model)
	}
	fmt.Println("   Preserved all existing hooks, plugins, and custom configurations.")
	fmt.Println("\nLaunch with:")
	fmt.Println("  claude")
	return nil
}

func setupCodex(home string, opts AgentSetupOptions) error {
	codexDir := filepath.Join(home, ".codex")
	_ = os.MkdirAll(codexDir, 0700)
	profilePath := filepath.Join(codexDir, "goroute.config.toml")
	configPath := filepath.Join(codexDir, "config.toml")

	modelName := "claude-3-7-sonnet"
	if opts.Model != "" {
		modelName = opts.Model
	}
	targetURL := "http://127.0.0.1:8128/v1"
	if opts.BaseURL != "" {
		targetURL = opts.BaseURL
	}

	tomlContent := fmt.Sprintf(`# GoRoute Configuration for OpenAI Codex CLI
model = "%s"
model_provider = "goroute"
model_context_window = 131072
model_auto_compact_token_limit = 112000
tool_output_token_limit = 32768

[model_providers.goroute]
name = "GoRoute"
base_url = "%s"
env_key = "OPENAI_API_KEY"
wire_api = "responses"
requires_openai_auth = false
`, modelName, targetURL)

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

	fmt.Println("✅ Configured OpenAI Codex CLI!")
	fmt.Printf("   Wrote profile to: %s\n", profilePath)
	fmt.Printf("   Active model: %s\n", modelName)
	fmt.Println("\nLaunch with:")
	fmt.Println("  goroute run codex")
	fmt.Println("  OR: codex --profile goroute")
	return nil
}

func getAllGoRouteModels(home string) map[string]string {
	models := map[string]string{
		"agy/gemini-flash-3-8":      "Gemini 3.8 Flash High (Antigravity)",
		"agy/claude-opus-4-6":       "Claude Opus 4.6 Thinking (Antigravity)",
		"agy/claude-sonnet-4-6":     "Claude Sonnet 4.6 (Antigravity)",
		"agy/gemini-pro-agent":      "Gemini Pro Agent (Antigravity)",
		"agy/gemini-3-flash":        "Gemini 3 Flash (Antigravity)",
		"agy/gemini-3.1-flash-lite": "Gemini 3.1 Flash Lite (Antigravity)",
		"agy/gpt-oss-120b-medium":   "GPT-OSS 120B Medium (Antigravity)",
		"claude-3-7-sonnet":         "Claude 3.7 Sonnet (Auto/Failover)",
		"claude-3-5-sonnet":         "Claude 3.5 Sonnet",
		"claude-sonnet-4-6":         "Claude Sonnet 4.6 (Live Native)",
		"claude-opus-4-6-thinking":  "Claude Opus 4.6 (Thinking)",
		"gemini-2.5-pro":            "Gemini 2.5 Pro (Deep Reasoning)",
		"gemini-2.5-flash":          "Gemini 2.5 Flash (Sub-Second)",
		"gemini-3.8-flash-high":     "Gemini 3.8 Flash High",
		"gemini-3-flash":            "Gemini 3 Flash",
		"gemini-3.1-flash-lite":     "Gemini 3.1 Flash Lite",
		"gpt-oss-120b-medium":       "GPT-OSS 120B (Open Weights)",
	}

	cfgPath := filepath.Join(home, ".goroute", "config.yaml")
	if cfg, err := config.Load(cfgPath); err == nil {
		for _, combo := range cfg.Combos {
			if _, exists := models[combo.Name]; !exists {
				models[combo.Name] = fmt.Sprintf("%s (GoRoute Combo)", combo.Name)
			}
		}
		for _, p := range cfg.Providers {
			if p.Type == "openai" {
				models[p.ID+"/default"] = fmt.Sprintf("%s (Upstream Provider)", p.ID)
			}
		}
	}
	return models
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

	primaryModel := "claude-3-7-sonnet"
	if opts.Model != "" {
		primaryModel = opts.Model
	}
	smallModel := "gemini-2.5-flash"
	if opts.SmallModel != "" {
		smallModel = opts.SmallModel
	}

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
			label := m
			if knownLabel, ok := allModels[m]; ok {
				label = knownLabel
			} else {
				label = fmt.Sprintf("%s (GoRoute)", m)
			}
			modelsMap[m] = map[string]interface{}{"name": label}
		}
		if _, ok := modelsMap[primaryModel]; !ok {
			modelsMap[primaryModel] = map[string]interface{}{"name": primaryModel + " (Primary)"}
		}
		if _, ok := modelsMap[smallModel]; !ok {
			modelsMap[smallModel] = map[string]interface{}{"name": smallModel + " (Fast)"}
		}
	} else {
		for modelID, modelName := range allModels {
			modelsMap[modelID] = map[string]interface{}{
				"name": modelName,
			}
		}
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
	existing["model"] = "gr/" + primaryModel
	existing["small_model"] = "gr/" + smallModel
	existing["agent"] = map[string]interface{}{
		"build": map[string]interface{}{
			"model": "gr/" + primaryModel,
		},
		"plan": map[string]interface{}{
			"model": "gr/" + primaryModel,
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
			existingJsonc["model"] = "gr/" + primaryModel
			existingJsonc["small_model"] = "gr/" + smallModel
			if jBytes, err := json.MarshalIndent(existingJsonc, "", "  "); err == nil {
				_ = os.WriteFile(jsoncPath, jBytes, 0600)
			}
		}
	}

	fmt.Println("✅ Configured OpenCode!")
	fmt.Printf("   Configured %d GoRoute models across providers:\n", len(modelsMap))
	for m := range modelsMap {
		fmt.Printf("   • gr/%s\n", m)
	}
	fmt.Printf("   Wrote configuration to: %s\n", configPath)
	fmt.Printf("   Authenticated provider key in: %s\n", authPath)
	fmt.Printf("   Active model set to: gr/%s\n", primaryModel)
	fmt.Printf("   Small model set to:  gr/%s\n", smallModel)
	fmt.Println("   Uniform format: gr/agy/gemini-flash-3-8, gr/agy/claude-opus-4-6, etc.")
	fmt.Println("   Launch with: opencode (or goroute run opencode)")
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

	if _, err := os.Stat(cursorDir); err == nil {
		existing := make(map[string]interface{})
		if raw, err := os.ReadFile(settingsPath); err == nil {
			_ = json.Unmarshal(raw, &existing)
		}
		existing["cursor.openaiBaseUrl"] = baseURL
		existing["cursor.openaiApiKey"] = apiKey
		if opts.Model != "" {
			existing["cursor.model"] = opts.Model
		}
		if bytes, err := json.MarshalIndent(existing, "", "  "); err == nil {
			_ = os.WriteFile(settingsPath, bytes, 0600)
			fmt.Printf("✅ Updated Cursor settings at: %s\n", settingsPath)
		}
	}

	fmt.Println("✅ Cursor IDE Configuration Details:")
	fmt.Println("  1. Open Cursor Settings (Ctrl+,) → Models")
	fmt.Printf("  2. Turn on 'Override OpenAI Base URL' and enter: %s\n", baseURL)
	fmt.Printf("  3. Set 'OpenAI API Key' to: %s\n", apiKey)
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}
	fmt.Printf("  4. Add model: %s\n", model)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}
	planModel := model
	if opts.SmallModel != "" {
		planModel = opts.SmallModel
	}

	secrets := map[string]string{"openAiApiKey": apiKey}
	sBytes, _ := json.MarshalIndent(secrets, "", "  ")
	_ = os.WriteFile(secretsPath, sBytes, 0600)

	state := map[string]interface{}{
		"actModeApiProvider":    "openai",
		"planModeApiProvider":   "openai",
		"openAiBaseUrl":         baseURL,
		"openAiModelId":         model,
		"planModeOpenAiModelId": planModel,
	}
	stBytes, _ := json.MarshalIndent(state, "", "  ")
	_ = os.WriteFile(statePath, stBytes, 0600)

	fmt.Println("✅ Configured Cline CLI & Standalone Mode!")
	fmt.Printf("   Wrote secrets to: %s\n", secretsPath)
	fmt.Printf("   Active model set to: %s\n", model)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	importDoc := map[string]interface{}{
		"providerProfiles": map[string]interface{}{
			"currentApiConfigName": "GoRoute",
			"apiConfigs": map[string]interface{}{
				"GoRoute": map[string]interface{}{
					"apiProvider":   "openai",
					"openAiBaseUrl": baseURL,
					"openAiApiKey":  apiKey,
					"openAiModelId": model,
				},
			},
		},
	}
	bytes, _ := json.MarshalIndent(importDoc, "", "  ")
	if err := os.WriteFile(importPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("✅ Generated Roo Code Import Document!")
	fmt.Printf("   Wrote file to: %s (Active model: %s)\n", importPath, model)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	auth := map[string]interface{}{
		"openai-compatible": map[string]string{
			"apiKey":  apiKey,
			"baseUrl": baseURL,
			"model":   model,
		},
	}
	bytes, _ := json.MarshalIndent(auth, "", "  ")
	if err := os.WriteFile(authPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Kilo Code!")
	fmt.Printf("   Wrote configuration to: %s (Active model: %s)\n", authPath, model)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	content := fmt.Sprintf("openai-api-base: %s\nopenai-api-key: %s\nmodel: openai/%s\n", baseURL, apiKey, model)
	if err := os.WriteFile(aiderConf, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Aider!")
	fmt.Printf("   Wrote configuration to: %s (Model: %s)\n", aiderConf, model)
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
	primaryModel := "claude-3-7-sonnet"
	if opts.Model != "" {
		primaryModel = opts.Model
	}
	smallModel := "gemini-2.5-pro"
	if opts.SmallModel != "" {
		smallModel = opts.SmallModel
	}

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
`, primaryModel, primaryModel, baseURL, apiKey, smallModel, smallModel, baseURL, apiKey)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Continue (continue.dev & cn CLI)!")
	fmt.Printf("   Wrote configuration to: %s\n", confPath)
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
	primaryModel := "claude-3-7-sonnet"
	if opts.Model != "" {
		primaryModel = opts.Model
	}

	content := fmt.Sprintf(`{
  "providers": {
    "goroute": {
      "type": "openai-compat",
      "base_url": "%s",
      "api_key": "%s",
      "models": [
        {"id": "%s", "name": "GoRoute %s"},
        {"id": "gemini-2.5-pro", "name": "GoRoute Gemini 2.5 Pro"}
      ]
    }
  }
}
`, baseURL, apiKey, primaryModel, primaryModel)

	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Charm Crush!")
	fmt.Printf("   Wrote configuration to: %s\n", confPath)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	content := fmt.Sprintf("GOOSE_PROVIDER: openai\nOPENAI_HOST: %s\nGOOSE_MODEL: %s\n", baseURL, model)
	if err := os.WriteFile(confPath, []byte(content), 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Goose!")
	fmt.Printf("   Wrote configuration to: %s (Model: %s)\n", confPath, model)
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
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	data := map[string]string{
		"baseUrl": baseURL,
		"apiKey":  apiKey,
		"model":   model,
	}
	bytes, _ := json.MarshalIndent(data, "", "  ")
	if err := os.WriteFile(confPath, bytes, 0600); err != nil {
		return err
	}

	fmt.Println("✅ Configured Qwen Code!")
	fmt.Printf("   Wrote configuration to: %s (Model: %s)\n", confPath, model)
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

	fmt.Println("✅ Google Gemini CLI Configuration:")
	fmt.Printf("   Set the following environment variables in your shell:\n")
	fmt.Printf("     export GOOGLE_GEMINI_BASE_URL=\"%s\"\n", baseURL)
	fmt.Printf("     export GEMINI_API_KEY=\"%s\"\n", apiKey)
	return nil
}

func setup5dive(home string, opts AgentSetupOptions) error {
	baseURL := "http://127.0.0.1:8128"
	if opts.BaseURL != "" {
		baseURL = strings.TrimSuffix(opts.BaseURL, "/v1")
	}
	model := "claude-3-7-sonnet"
	if opts.Model != "" {
		model = opts.Model
	}

	fmt.Println("✅ 5dive Agent Fleet Configuration:")
	fmt.Println("   To connect your 5dive agent fleet to GoRoute, run:")
	fmt.Printf("     5dive agent auth set claude \\\n")
	fmt.Printf("       --provider=openai \\\n")
	fmt.Printf("       --base-url=%s \\\n", baseURL)
	fmt.Printf("       --auth-profile=goroute \\\n")
	fmt.Printf("       --model=%s\n", model)
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
	model := "gemini-flash"
	if opts.Model != "" {
		model = opts.Model
	}

	gorouteBlock := fmt.Sprintf("providers:\n  goroute:\n    base_url: %s\n    api_key: %s\n    default_model: %s\n", baseURL, apiKey, model)
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

	fmt.Println("✅ Configured Hermes Agent!")
	fmt.Printf("   Wrote provider configuration to: %s\n", cfgFile)
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

	model := "gemini-flash"
	if opts.Model != "" {
		model = opts.Model
	}

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
	models["goroute/"+model] = map[string]interface{}{}
	defaults["model"] = map[string]interface{}{
		"primary": "goroute/" + model,
	}

	out, err := json.MarshalIndent(root, "", "    ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(cfgFile, out, 0644); err != nil {
		return err
	}

	fmt.Println("✅ Configured OpenClaw Autonomous Gateway!")
	fmt.Printf("   Cleaned and updated: %s (Primary: goroute/%s)\n", cfgFile, model)
	return nil
}

