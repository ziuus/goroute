package clitools

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type CliTool struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Category       string `json:"category"` // "code", "autonomous", "acp"
	Description    string `json:"description"`
	DefaultCommand string `json:"default_command"`
	DocsURL        string `json:"docs_url"`
	SettingsFile   string `json:"settings_file"`
	Detected       bool   `json:"detected"`
	Runnable       bool   `json:"runnable"`
	Configured     bool   `json:"configured"`
	Version        string `json:"version,omitempty"`
}

var Registry = map[string]CliTool{
	"claude": {
		ID:             "claude",
		Name:           "Claude Code",
		Category:       "code",
		Description:    "Anthropic Claude Code CLI — ANTHROPIC_BASE_URL points to GoRoute",
		DefaultCommand: "claude",
		DocsURL:        "https://docs.anthropic.com/en/docs/claude-code/overview",
		SettingsFile:   "~/.claude/settings.json",
	},
	"codex": {
		ID:             "codex",
		Name:           "OpenAI Codex CLI",
		Category:       "code",
		Description:    "OpenAI Codex CLI — OpenAI-compatible base URL targets GoRoute",
		DefaultCommand: "codex",
		DocsURL:        "https://github.com/openai/codex",
		SettingsFile:   "~/.codex/config.toml",
	},
	"opencode": {
		ID:             "opencode",
		Name:           "OpenCode CLI",
		Category:       "code",
		Description:    "OpenCode Terminal & TUI coding agent with GoRoute provider engine",
		DefaultCommand: "opencode",
		DocsURL:        "https://github.com/opencode-ai/opencode",
		SettingsFile:   "~/.config/opencode/opencode.json",
	},
	"openclaw": {
		ID:             "openclaw",
		Name:           "Open Claw",
		Category:       "autonomous",
		Description:    "OpenClaw autonomous agent with universal tool calling and ACP support",
		DefaultCommand: "openclaw",
		DocsURL:        "https://github.com/openclaw/openclaw",
		SettingsFile:   "~/.openclaw/config.json",
	},
	"hermes": {
		ID:             "hermes",
		Name:           "Hermes Agent",
		Category:       "autonomous",
		Description:    "NousResearch Hermes Agent — multi-model terminal reasoning & delegation",
		DefaultCommand: "hermes",
		DocsURL:        "https://github.com/NousResearch/Hermes-Agent",
		SettingsFile:   "~/.hermes/config.yaml",
	},
	"goose": {
		ID:             "goose",
		Name:           "Goose",
		Category:       "autonomous",
		Description:    "Block Goose autonomous developer agent CLI with ACP support",
		DefaultCommand: "goose",
		DocsURL:        "https://github.com/block/goose",
		SettingsFile:   "~/.config/goose/config.yaml",
	},
	"prime": {
		ID:             "prime",
		Name:           "Prime Agent",
		Category:       "autonomous",
		Description:    "Self-improving RLM coding harness with OpenAI-compatible provider support",
		DefaultCommand: "prime",
		DocsURL:        "https://github.com/prime-agent/prime",
		SettingsFile:   "~/.prime/config.json",
	},
	"cursor": {
		ID:             "cursor",
		Name:           "Cursor",
		Category:       "code",
		Description:    "Cursor IDE with custom OpenAI base URL configured to GoRoute",
		DefaultCommand: "cursor",
		DocsURL:        "https://cursor.com",
		SettingsFile:   "~/.cursor/settings.json",
	},
	"cline": {
		ID:             "cline",
		Name:           "Cline",
		Category:       "code",
		Description:    "Autonomous coding agent in VS Code / VSCodium",
		DefaultCommand: "cline",
		DocsURL:        "https://github.com/cline/cline",
		SettingsFile:   "~/.cline/config.json",
	},
	"roo": {
		ID:             "roo",
		Name:           "Roo Code",
		Category:       "code",
		Description:    "Roo Code multi-role AI agent for VS Code",
		DefaultCommand: "roo",
		DocsURL:        "https://github.com/RooVetGit/Roo-Code",
		SettingsFile:   "~/.roo/config.json",
	},
	"aider": {
		ID:             "aider",
		Name:           "Aider",
		Category:       "code",
		Description:    "AI pair programming in your terminal with Git integration",
		DefaultCommand: "aider",
		DocsURL:        "https://aider.chat",
		SettingsFile:   "~/.aider.conf.yml",
	},
	"continue": {
		ID:             "continue",
		Name:           "Continue",
		Category:       "code",
		Description:    "Open-source AI code assistant for VS Code and JetBrains",
		DefaultCommand: "continue",
		DocsURL:        "https://continue.dev",
		SettingsFile:   "~/.continue/config.json",
	},
	"qwen": {
		ID:             "qwen",
		Name:           "Qwen Code",
		Category:       "code",
		Description:    "Alibaba Qwen Code CLI terminal assistant",
		DefaultCommand: "qwen-code",
		DocsURL:        "https://github.com/QwenLM/Qwen",
		SettingsFile:   "~/.qwen/config.json",
	},
	"droid": {
		ID:             "droid",
		Name:           "Droid (Factory)",
		Category:       "code",
		Description:    "Factory Droid CLI agent with custom model routing",
		DefaultCommand: "droid",
		DocsURL:        "https://factory.ai",
		SettingsFile:   "~/.factory/settings.json",
	},
	"kilo": {
		ID:             "kilo",
		Name:           "Kilo Code",
		Category:       "code",
		Description:    "Kilo terminal agent with GoRoute proxy backend",
		DefaultCommand: "kilocode",
		DocsURL:        "https://kilo.ai",
		SettingsFile:   "~/.local/share/kilo/auth.json",
	},
	"devin": {
		ID:             "devin",
		Name:           "Devin CLI",
		Category:       "autonomous",
		Description:    "Cognition Devin terminal agent with custom endpoint",
		DefaultCommand: "devin",
		DocsURL:        "https://devin.ai",
		SettingsFile:   "~/.config/devin/config.json",
	},
	"grok-build": {
		ID:             "grok-build",
		Name:           "Grok Build",
		Category:       "code",
		Description:    "xAI Grok Build terminal developer CLI",
		DefaultCommand: "grok-build",
		DocsURL:        "https://x.ai",
		SettingsFile:   "~/.config/grok-build/config.json",
	},
	"windsurf": {
		ID:             "windsurf",
		Name:           "Windsurf",
		Category:       "code",
		Description:    "Codeium Windsurf IDE with custom model routing via GoRoute",
		DefaultCommand: "windsurf",
		DocsURL:        "https://codeium.com/windsurf",
		SettingsFile:   "~/.codeium/windsurf/settings.json",
	},
	"zcode": {
		ID:             "zcode",
		Name:           "ZCode (GLM Coding Plan)",
		Category:       "code",
		Description:    "Local ZCode app-server backend with GoRoute upstream",
		DefaultCommand: "zcode",
		DocsURL:        "https://zcode.z.ai",
		SettingsFile:   "~/.zcode/config.json",
	},
	"copilot": {
		ID:             "copilot",
		Name:           "GitHub Copilot",
		Category:       "code",
		Description:    "GitHub Copilot Chat with COPILOT_PROVIDER_BASE_URL support",
		DefaultCommand: "copilot",
		DocsURL:        "https://code.visualstudio.com/docs/copilot/overview",
		SettingsFile:   "~/.config/github-copilot/hosts.json",
	},
	"kiro": {
		ID:             "kiro",
		Name:           "Kiro AI",
		Category:       "code",
		Description:    "Amazon Kiro — AI-powered IDE with MITM intercept",
		DefaultCommand: "kiro",
		DocsURL:        "https://kiro.aws",
		SettingsFile:   "~/.kiro/config.json",
	},
	"custom": {
		ID:             "custom",
		Name:           "Custom CLI",
		Category:       "code",
		Description:    "Generic OpenAI-compatible CLI or SDK configuration generator",
		DefaultCommand: "",
		DocsURL:        "https://goroute.dev/docs/cli-tools",
		SettingsFile:   "~/.goroute/custom-tools.json",
	},
	"forge": {
		ID:             "forge",
		Name:           "ForgeCode",
		Category:       "code",
		Description:    "ForgeCode coding agent CLI — custom provider via .forge.toml",
		DefaultCommand: "forge",
		DocsURL:        "https://github.com/antinomyhq/forge",
		SettingsFile:   "~/.forge.toml",
	},
	"jcode": {
		ID:             "jcode",
		Name:           "jcode",
		Category:       "code",
		Description:    "jcode terminal coding agent — OpenAI-compatible CLI",
		DefaultCommand: "jcode",
		DocsURL:        "https://github.com/1jehuang/jcode",
		SettingsFile:   "~/.jcode/config.json",
	},
	"codewhale": {
		ID:             "codewhale",
		Name:           "CodeWhale",
		Category:       "code",
		Description:    "CodeWhale — Rust-based coding agent CLI with OPENAI_BASE_URL support",
		DefaultCommand: "codewhale",
		DocsURL:        "https://github.com/Hmbown/CodeWhale",
		SettingsFile:   "~/.codewhale/config.toml",
	},
	"smelt": {
		ID:             "smelt",
		Name:           "Smelt",
		Category:       "code",
		Description:    "Smelt coding agent CLI — OpenAI-compatible agent",
		DefaultCommand: "smelt",
		DocsURL:        "https://github.com/leonardcser/smelt",
		SettingsFile:   "~/.smelt/config.json",
	},
	"pi": {
		ID:             "pi",
		Name:           "Pi",
		Category:       "code",
		Description:    "Pi coding agent CLI — lightweight terminal AI",
		DefaultCommand: "pi",
		DocsURL:        "https://github.com/badlogic/pi",
		SettingsFile:   "~/.pi/config.json",
	},
	"crush": {
		ID:             "crush",
		Name:           "Crush",
		Category:       "code",
		Description:    "Crush coding agent CLI — terminal AI agent by Charm",
		DefaultCommand: "crush",
		DocsURL:        "https://github.com/charmbracelet/crush",
		SettingsFile:   "~/.config/crush/crush.json",
	},
	"zed": {
		ID:             "zed",
		Name:           "Zed",
		Category:       "code",
		Description:    "Zed code editor — built-in agent through an OpenAI-compatible provider",
		DefaultCommand: "zed",
		DocsURL:        "https://zed.dev/docs/ai/llm-providers#openai-api-compatible",
		SettingsFile:   "~/.config/zed/settings.json",
	},
	"interpreter": {
		ID:             "interpreter",
		Name:           "Open Interpreter",
		Category:       "autonomous",
		Description:    "Open Interpreter — autonomous coding agent CLI with --api_base flag",
		DefaultCommand: "interpreter",
		DocsURL:        "https://docs.openinterpreter.com/",
		SettingsFile:   "~/.open-interpreter/config.yaml",
	},
	"omp": {
		ID:             "omp",
		Name:           "Oh My Pi",
		Category:       "autonomous",
		Description:    "Oh My Pi terminal coding agent via GoRoute",
		DefaultCommand: "omp",
		DocsURL:        "https://github.com/can1357/oh-my-pi",
		SettingsFile:   "~/.omp/config.json",
	},
	"letta": {
		ID:             "letta",
		Name:           "Letta CLI",
		Category:       "autonomous",
		Description:    "Letta CLI — AI agent with persistent memory and tool use",
		DefaultCommand: "letta",
		DocsURL:        "https://docs.letta.com",
		SettingsFile:   "~/.letta/config.json",
	},
	"warp": {
		ID:             "warp",
		Name:           "Warp AI",
		Category:       "autonomous",
		Description:    "Warp AI terminal — BYOK desktop app with partial base URL support",
		DefaultCommand: "warp",
		DocsURL:        "https://docs.warp.dev/",
		SettingsFile:   "~/.warp/settings.json",
	},
	"deepseek-tui": {
		ID:             "deepseek-tui",
		Name:           "DeepSeek TUI",
		Category:       "code",
		Description:    "DeepSeek TUI terminal coding client",
		DefaultCommand: "deepseek",
		DocsURL:        "https://github.com/deepseek-ai",
		SettingsFile:   "~/.config/deepseek-tui/config.toml",
	},
}

func ListTools() []CliTool {
	home, _ := os.UserHomeDir()
	var list []CliTool
	var mu sync.Mutex
	var wg sync.WaitGroup

	for id, tool := range Registry {
		wg.Add(1)
		go func(id string, tool CliTool) {
			defer wg.Done()
			t := tool
			// 1. Detect binary in PATH
			binName := t.DefaultCommand
			if binName != "" {
				if p, err := exec.LookPath(binName); err == nil && p != "" {
					t.Detected = true
					t.Runnable = true
					// Skip version probe on GUI apps that might hang without display
					if binName != "antigravity" && binName != "cursor" && binName != "windsurf" && binName != "zed" {
						ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
						cmd := exec.CommandContext(ctx, binName, "--version")
						if out, err := cmd.Output(); err == nil {
							t.Version = strings.TrimSpace(string(out))
							if len(t.Version) > 40 {
								t.Version = t.Version[:40]
							}
						}
						cancel()
					}
				}
			}

			// 2. Check settings file
			expandedPath := strings.Replace(t.SettingsFile, "~", home, 1)
			if fi, err := os.Stat(expandedPath); err == nil && !fi.IsDir() {
				t.Detected = true
				// Check if file mentions 8128 (GoRoute port)
				data, _ := os.ReadFile(expandedPath)
				if strings.Contains(string(data), "8128") || strings.Contains(string(data), "goroute") {
					t.Configured = true
				}
			}

			mu.Lock()
			list = append(list, t)
			mu.Unlock()
			_ = id
		}(id, tool)
	}

	wg.Wait()
	return list
}

func ConfigureTool(id string) (string, error) {
	home, _ := os.UserHomeDir()
	switch id {
	case "claude":
		return "Configured Claude Code (~/.claude/settings.json)", setupClaude(home)
	case "codex":
		return "Configured Codex CLI (~/.codex/config.toml)", setupCodex(home)
	case "opencode":
		return "Configured OpenCode (~/.config/opencode/opencode.json)", setupOpenCode(home)
	case "openclaw":
		return "Configured OpenClaw (~/.openclaw/config.json)", setupOpenClaw(home)
	case "hermes":
		return "Configured Hermes Agent (~/.hermes/config.yaml)", setupHermes(home)
	case "goose":
		return "Configured Goose (~/.config/goose/config.yaml)", setupGoose(home)
	case "cursor":
		return "Configured Cursor settings", setupCursor(home)
	case "cline":
		return "Configured Cline settings", setupCline(home)
	case "roo":
		return "Configured Roo Code settings", setupRoo(home)
	case "aider":
		return "Configured Aider (~/.aider.conf.yml)", setupAider(home)
	case "continue":
		return "Configured Continue (~/.continue/config.json)", setupContinue(home)
	case "qwen":
		return "Configured Qwen Code (~/.qwen/config.json)", setupQwen(home)
	case "droid":
		return "Configured Droid (~/.factory/settings.json)", setupDroid(home)
	case "kilo":
		return "Configured Kilo Code (~/.local/share/kilo/auth.json)", setupKilo(home)
	case "devin":
		return "Configured Devin (~/.config/devin/config.json)", setupDevin(home)
	case "grok-build":
		return "Configured Grok Build (~/.config/grok-build/config.json)", setupGrokBuild(home)
	case "windsurf":
		return "Configured Windsurf (~/.codeium/windsurf/settings.json)", setupWindsurf(home)
	case "zcode":
		return "Configured ZCode (~/.zcode/config.json)", setupZCode(home)
	case "copilot":
		return "Configured Copilot (~/.config/github-copilot/hosts.json)", setupCopilot(home)
	case "kiro":
		return "Configured Kiro (~/.kiro/config.json)", setupKiro(home)
	case "custom":
		return "Configured Custom CLI (~/.goroute/custom-tools.json)", setupCustom(home)
	case "forge":
		return "Configured ForgeCode (~/.forge.toml)", setupForge(home)
	case "jcode":
		return "Configured jcode (~/.jcode/config.json)", setupJCode(home)
	case "codewhale":
		return "Configured CodeWhale (~/.codewhale/config.toml)", setupCodeWhale(home)
	case "smelt":
		return "Configured Smelt (~/.smelt/config.json)", setupSmelt(home)
	case "pi":
		return "Configured Pi (~/.pi/config.json)", setupPi(home)
	case "crush":
		return "Configured Crush (~/.config/crush/crush.json)", setupCrush(home)
	case "zed":
		return "Configured Zed (~/.config/zed/settings.json)", setupZed(home)
	case "interpreter":
		return "Configured Open Interpreter (~/.open-interpreter/config.yaml)", setupInterpreter(home)
	case "omp":
		return "Configured Oh My Pi (~/.omp/config.json)", setupOmp(home)
	case "letta":
		return "Configured Letta CLI (~/.letta/config.json)", setupLetta(home)
	case "warp":
		return "Configured Warp AI (~/.warp/settings.json)", setupWarp(home)
	case "deepseek-tui":
		return "Configured DeepSeek TUI (~/.config/deepseek-tui/config.toml)", setupDeepSeekTUI(home)
	default:
		return "", fmt.Errorf("agent %s not supported for automated configuration", id)
	}
}

func setupClaude(home string) error {
	claudeDir := filepath.Join(home, ".claude")
	os.MkdirAll(claudeDir, 0755)
	settingsFile := filepath.Join(claudeDir, "settings.json")

	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsFile); err == nil {
		json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	env, ok := settings["env"].(map[string]interface{})
	if !ok {
		env = make(map[string]interface{})
		settings["env"] = env
	}

	env["ANTHROPIC_BASE_URL"] = "http://127.0.0.1:8128"
	env["ANTHROPIC_AUTH_TOKEN"] = "goroute"
	env["ANTHROPIC_DEFAULT_SONNET_MODEL"] = "claude-sonnet-4.5"
	env["ANTHROPIC_DEFAULT_OPUS_MODEL"] = "claude-opus-5.5"
	env["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = "claude-haiku-4.5"
	env["ANTHROPIC_DEFAULT_FABLE_MODEL"] = "claude-gemini-3.1-pro"
	env["CLAUDE_CODE_ENABLE_GATEWAY_MODEL_DISCOVERY"] = "1"
	env["CLAUDE_CODE_AUTO_COMPACT_WINDOW"] = "200000"
	settings["customBaseUrl"] = "http://127.0.0.1:8128"

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsFile, data, 0644)
}

func setupCodex(home string) error {
	codexDir := filepath.Join(home, ".codex")
	os.MkdirAll(codexDir, 0755)
	cfgFile := filepath.Join(codexDir, "config.toml")

	tomlContent := `model = "gpt"
model_provider = "goroute"

[model_providers.goroute]
name = "GoRoute Universal Gateway"
base_url = "http://127.0.0.1:8128/v1"
wire_specification = "openai"
requires_api_key = true
`
	return os.WriteFile(cfgFile, []byte(tomlContent), 0644)
}

func setupOpenCode(home string) error {
	ocDir := filepath.Join(home, ".config", "opencode")
	os.MkdirAll(ocDir, 0755)
	cfgFile := filepath.Join(ocDir, "opencode.json")

	cfg := map[string]interface{}{
		"$schema": "https://opencode.ai/config.json",
		"provider": map[string]interface{}{
			"goroute": map[string]interface{}{
				"name":    "GoRoute Universal Gateway",
				"options": map[string]interface{}{"baseURL": "http://127.0.0.1:8128/v1"},
			},
		},
		"model": "goroute/claude-sonnet-4.5",
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgFile, out, 0644)
}

func setupOpenClaw(home string) error {
	dir := filepath.Join(home, ".openclaw")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")

	cfg := map[string]interface{}{
		"gateway": map[string]interface{}{
			"baseURL": "http://127.0.0.1:8128/v1",
			"apiKey":  "goroute",
		},
		"defaults": map[string]interface{}{
			"model": "claude-sonnet-4.5",
		},
	}
	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cfgFile, out, 0644)
}

func setupHermes(home string) error {
	dir := filepath.Join(home, ".hermes")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `api_base: "http://127.0.0.1:8128/v1"
api_key: "goroute"
default_model: "claude-sonnet-4.5"
temperature: 0.7
max_tokens: 4096
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupGoose(home string) error {
	dir := filepath.Join(home, ".config", "goose")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.yaml")

	content := `provider: openai
base_url: http://127.0.0.1:8128/v1
api_key: goroute
model: claude-sonnet-4.5
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCursor(home string) error {
	dir := filepath.Join(home, ".cursor")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "settings.json")
	content := `{
  "openai.baseUrl": "http://127.0.0.1:8128/v1",
  "openai.apiKey": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCline(home string) error {
	dir := filepath.Join(home, ".cline")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "apiProvider": "openai-compatible",
  "openAiBaseUrl": "http://127.0.0.1:8128/v1",
  "openAiApiKey": "goroute",
  "openAiModelId": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupRoo(home string) error {
	dir := filepath.Join(home, ".roo")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "apiProvider": "openai-compatible",
  "openAiBaseUrl": "http://127.0.0.1:8128/v1",
  "openAiApiKey": "goroute",
  "openAiModelId": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupAider(home string) error {
	cfgFile := filepath.Join(home, ".aider.conf.yml")
	content := `openai-api-base: http://127.0.0.1:8128/v1
openai-api-key: goroute
model: openai/claude-sonnet-4.5
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupContinue(home string) error {
	dir := filepath.Join(home, ".continue")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	cfg := map[string]interface{}{
		"models": []map[string]interface{}{
			{
				"title":    "GoRoute Claude Sonnet",
				"provider": "openai",
				"model":    "claude-sonnet-4.5",
				"apiBase":  "http://127.0.0.1:8128/v1",
				"apiKey":   "goroute",
			},
		},
	}
	data, _ := json.MarshalIndent(cfg, "", "  ")
	return os.WriteFile(cfgFile, data, 0644)
}

func setupQwen(home string) error {
	dir := filepath.Join(home, ".qwen")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "apiBase": "http://127.0.0.1:8128/v1",
  "apiKey": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupDroid(home string) error {
	dir := filepath.Join(home, ".factory")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "settings.json")
	content := `{
  "customEndpoint": "http://127.0.0.1:8128/v1",
  "apiKey": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupKilo(home string) error {
	dir := filepath.Join(home, ".local", "share", "kilo")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "auth.json")
	content := `{
  "api_endpoint": "http://127.0.0.1:8128/v1",
  "token": "goroute",
  "default_model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupDevin(home string) error {
	dir := filepath.Join(home, ".config", "devin")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "endpoint": "http://127.0.0.1:8128/v1",
  "auth_token": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupGrokBuild(home string) error {
	dir := filepath.Join(home, ".config", "grok-build")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "default": "goroute",
  "model": {
    "base_url": "http://127.0.0.1:8128/v1",
    "api_backend": "chat_completions",
    "api_key": "goroute"
  }
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupWindsurf(home string) error {
	dir := filepath.Join(home, ".codeium", "windsurf")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "settings.json")
	content := `{
  "custom_openai_endpoint": "http://127.0.0.1:8128/v1",
  "api_key": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupZCode(home string) error {
	dir := filepath.Join(home, ".zcode")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "endpoint": "http://127.0.0.1:8128/v1",
  "apiKey": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCopilot(home string) error {
	dir := filepath.Join(home, ".config", "github-copilot")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "hosts.json")
	content := `{
  "github.com": {
    "user": "goroute",
    "oauth_token": "goroute",
    "enterprise_endpoint": "http://127.0.0.1:8128/v1"
  }
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupKiro(home string) error {
	dir := filepath.Join(home, ".kiro")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "custom_endpoint": "http://127.0.0.1:8128/v1",
  "api_key": "goroute",
  "active_model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCustom(home string) error {
	dir := filepath.Join(home, ".goroute")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "custom-tools.json")
	content := `{
  "openai_base_url": "http://127.0.0.1:8128/v1",
  "anthropic_base_url": "http://127.0.0.1:8128",
  "api_key": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupForge(home string) error {
	cfgFile := filepath.Join(home, ".forge.toml")
	content := `[provider]
name = "goroute"
base_url = "http://127.0.0.1:8128/v1"
api_key = "goroute"
default_model = "claude-sonnet-4.5"
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupJCode(home string) error {
	dir := filepath.Join(home, ".jcode")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "base_url": "http://127.0.0.1:8128/v1",
  "api_key": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCodeWhale(home string) error {
	dir := filepath.Join(home, ".codewhale")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.toml")
	content := `[server]
base_url = "http://127.0.0.1:8128/v1"
api_key = "goroute"
model = "claude-sonnet-4.5"
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupSmelt(home string) error {
	dir := filepath.Join(home, ".smelt")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "endpoint": "http://127.0.0.1:8128/v1",
  "key": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupPi(home string) error {
	dir := filepath.Join(home, ".pi")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "openai_endpoint": "http://127.0.0.1:8128/v1",
  "api_key": "goroute",
  "default_model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupCrush(home string) error {
	dir := filepath.Join(home, ".config", "crush")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "crush.json")
	content := `{
  "endpoint": "http://127.0.0.1:8128/v1",
  "api_key": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupZed(home string) error {
	dir := filepath.Join(home, ".config", "zed")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "settings.json")
	content := `{
  "language_models": {
    "openai": {
      "api_url": "http://127.0.0.1:8128/v1",
      "available_models": ["claude-sonnet-4.5", "claude-opus-5.5"]
    }
  }
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupInterpreter(home string) error {
	dir := filepath.Join(home, ".open-interpreter")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.yaml")
	content := `api_base: http://127.0.0.1:8128/v1
api_key: goroute
model: claude-sonnet-4.5
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupOmp(home string) error {
	dir := filepath.Join(home, ".omp")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "base_url": "http://127.0.0.1:8128/v1",
  "api_key": "goroute",
  "model": "claude-sonnet-4.5"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupLetta(home string) error {
	dir := filepath.Join(home, ".letta")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.json")
	content := `{
  "openai_endpoint": "http://127.0.0.1:8128/v1",
  "auth_token": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupWarp(home string) error {
	dir := filepath.Join(home, ".warp")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "settings.json")
	content := `{
  "ai_gateway": "http://127.0.0.1:8128/v1",
  "api_key": "goroute"
}`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}

func setupDeepSeekTUI(home string) error {
	dir := filepath.Join(home, ".config", "deepseek-tui")
	os.MkdirAll(dir, 0755)
	cfgFile := filepath.Join(dir, "config.toml")
	content := `base_url = "http://127.0.0.1:8128/v1"
api_key = "goroute"
model = "deepseek-chat"
`
	return os.WriteFile(cfgFile, []byte(content), 0644)
}


