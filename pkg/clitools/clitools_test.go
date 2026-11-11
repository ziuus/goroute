package clitools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ziuus/goroute/pkg/storage"
)

func TestGetAndSaveAndDisconnectToolConfig(t *testing.T) {
	tempHome := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	defer func() {
		if db := storage.GetDB(); db != nil {
			_ = db.Close()
		}
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	// 1. Initial state for Claude Code: file doesn't exist yet
	detail, err := GetToolConfigDetail("claude")
	if err != nil {
		t.Fatalf("GetToolConfigDetail failed: %v", err)
	}
	if detail.Exists {
		t.Errorf("expected claude settings to not exist initially")
	}

	// 2. Configure Claude
	claudeDir := filepath.Join(tempHome, ".claude")
	os.MkdirAll(claudeDir, 0755)
	settingsFile := filepath.Join(claudeDir, "settings.json")
	initialJSON := `{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8128",
    "ANTHROPIC_AUTH_TOKEN": "goroute",
    "MY_CUSTOM_VAR": "keep_me"
  }
}`
	if err := os.WriteFile(settingsFile, []byte(initialJSON), 0644); err != nil {
		t.Fatalf("failed to write settings file: %v", err)
	}

	detail, err = GetToolConfigDetail("claude")
	if err != nil {
		t.Fatalf("GetToolConfigDetail failed: %v", err)
	}
	if !detail.Exists {
		t.Errorf("expected claude settings to exist")
	}
	if !detail.Configured {
		t.Errorf("expected claude to be marked configured")
	}

	// 3. Save new config through SaveToolConfig
	updatedJSON := `{
  "env": {
    "ANTHROPIC_BASE_URL": "http://127.0.0.1:8128/v1",
    "ANTHROPIC_AUTH_TOKEN": "custom-key",
    "MY_CUSTOM_VAR": "keep_me"
  }
}`
	if err := SaveToolConfig("claude", updatedJSON); err != nil {
		t.Fatalf("SaveToolConfig failed: %v", err)
	}

	// Verify backup was made
	backups, err := ListBackups("claude")
	if err != nil || len(backups) == 0 {
		t.Errorf("expected at least 1 backup to be created, got %d", len(backups))
	}

	// 4. Disconnect Claude
	if err := DisconnectTool("claude"); err != nil {
		t.Fatalf("DisconnectTool failed: %v", err)
	}

	detail, err = GetToolConfigDetail("claude")
	if err != nil {
		t.Fatalf("GetToolConfigDetail after disconnect failed: %v", err)
	}
	if detail.Configured {
		t.Errorf("expected claude to no longer be configured after disconnect")
	}

	// Verify MY_CUSTOM_VAR was preserved
	var parsed map[string]interface{}
	json.Unmarshal([]byte(detail.Content), &parsed)
	env, ok := parsed["env"].(map[string]interface{})
	if !ok || env["MY_CUSTOM_VAR"] != "keep_me" {
		t.Errorf("expected MY_CUSTOM_VAR to be preserved, got %v", detail.Content)
	}
	if _, hasBase := env["ANTHROPIC_BASE_URL"]; hasBase {
		t.Errorf("expected ANTHROPIC_BASE_URL to be deleted, got %v", detail.Content)
	}
}

func TestOpenHandsConfigurationAndDryRun(t *testing.T) {
	tempHome := t.TempDir()
	origHome := os.Getenv("HOME")
	origUserProfile := os.Getenv("USERPROFILE")
	os.Setenv("HOME", tempHome)
	os.Setenv("USERPROFILE", tempHome)
	defer func() {
		if db := storage.GetDB(); db != nil {
			_ = db.Close()
		}
		os.Setenv("HOME", origHome)
		os.Setenv("USERPROFILE", origUserProfile)
	}()

	// 1. Verify registered
	tool, exists := Registry["openhands"]
	if !exists {
		t.Fatalf("expected openhands to be registered in Registry")
	}
	if tool.SettingsFile != "~/.openhands/config.toml" {
		t.Errorf("expected settings file ~/.openhands/config.toml, got %s", tool.SettingsFile)
	}

	// 2. Test DryRun via ApplyTool
	res, err := ApplyTool(ApplyRequest{
		ToolID: "openhands",
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("dry run failed: %v", err)
	}
	if res["dryRun"] != true {
		t.Errorf("expected dryRun=true in result: %v", res)
	}

	// Config file shouldn't exist after dry run
	expectedPath := filepath.Join(tempHome, ".openhands", "config.toml")
	if _, err := os.Stat(expectedPath); !os.IsNotExist(err) {
		t.Errorf("expected config file to not exist after dry run")
	}

	// 3. Execute actual configuration
	msg, err := ConfigureTool("openhands")
	if err != nil {
		t.Fatalf("ConfigureTool failed: %v", err)
	}
	if msg == "" {
		t.Errorf("expected non-empty success message")
	}

	data, err := os.ReadFile(expectedPath)
	if err != nil {
		t.Fatalf("failed to read created config file: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "8128/v1") {
		t.Errorf("expected config to target GoRoute port 8128/v1, got:\n%s", content)
	}

	// 4. Verify detection via GetToolConfigDetail
	detail, err := GetToolConfigDetail("openhands")
	if err != nil {
		t.Fatalf("GetToolConfigDetail failed: %v", err)
	}
	if !detail.Exists || !detail.Configured {
		t.Errorf("expected openhands to be detected as existing and configured: %+v", detail)
	}
}
