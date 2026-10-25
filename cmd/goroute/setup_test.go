package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAgentSetups(t *testing.T) {
	tempHome := t.TempDir()

	for name, cfg := range agentConfigurators {
		t.Run(name, func(t *testing.T) {
			err := cfg.Run(tempHome, AgentSetupOptions{})
			if err != nil {
				t.Fatalf("setup for %s failed: %v", name, err)
			}
		})
	}

	// Verify key configuration files were written
	expectedFiles := []string{
		filepath.Join(tempHome, ".claude", "settings.json"),
		filepath.Join(tempHome, ".codex", "goroute.config.toml"),
		filepath.Join(tempHome, ".config", "opencode", "opencode.json"),
		filepath.Join(tempHome, ".cline", "data", "secrets.json"),
		filepath.Join(tempHome, ".local", "share", "kilo", "auth.json"),
		filepath.Join(tempHome, ".aider.conf.yml"),
		filepath.Join(tempHome, ".continue", "config.yaml"),
		filepath.Join(tempHome, ".config", "crush", "crush.json"),
		filepath.Join(tempHome, ".config", "goose", "config.yaml"),
		filepath.Join(tempHome, ".qwen", "settings.json"),
		filepath.Join(tempHome, ".roo", "settings-import.json"),
		filepath.Join(tempHome, ".hermes", "config.yaml"),
		filepath.Join(tempHome, ".openclaw", "openclaw.json"),
	}

	for _, file := range expectedFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			t.Errorf("expected config file was not created: %s", file)
		}
	}
}

func TestAgentSetupWithOptions(t *testing.T) {
	tempHome := t.TempDir()

	opts := AgentSetupOptions{
		Model:      "gemini-2.5-flash",
		SmallModel: "gemini-3.1-flash-lite",
		Models:     []string{"gemini-2.5-flash", "gemini-3.1-flash-lite", "deepseek"},
	}

	if err := setupOpenCode(tempHome, opts); err != nil {
		t.Fatalf("setupOpenCode with custom options failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(tempHome, ".config", "opencode", "opencode.json"))
	if err != nil {
		t.Fatalf("failed to read opencode.json: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "gr/gemini-2.5-flash") {
		t.Errorf("expected primary model gr/gemini-2.5-flash in config, got: %s", content)
	}
	if !strings.Contains(content, "gr/gemini-3.1-flash-lite") {
		t.Errorf("expected small model gr/gemini-3.1-flash-lite in config, got: %s", content)
	}

	claudeOpts := AgentSetupOptions{
		Model: "gemini-3.8-flash-high",
	}
	if err := setupClaude(tempHome, claudeOpts); err != nil {
		t.Fatalf("setupClaude with custom options failed: %v", err)
	}
	cData, err := os.ReadFile(filepath.Join(tempHome, ".claude", "settings.json"))
	if err != nil {
		t.Fatalf("failed to read claude settings.json: %v", err)
	}
	if !strings.Contains(string(cData), "gemini-3.8-flash-high") {
		t.Errorf("expected claude settings to contain gemini-3.8-flash-high, got: %s", string(cData))
	}
}
