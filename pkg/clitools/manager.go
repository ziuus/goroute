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

	"github.com/ziuus/goroute/pkg/storage"
)

type ToolDetection struct {
	Installed   bool   `json:"installed"`
	Runnable    bool   `json:"runnable"`
	Version     string `json:"version,omitempty"`
	Command     string `json:"command,omitempty"`
	CommandPath string `json:"commandPath,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ToolConfigStatus struct {
	Status           string `json:"status"` // "configured", "not_configured", "partial"
	Endpoint         string `json:"endpoint,omitempty"`
	LastConfiguredAt *int64 `json:"lastConfiguredAt,omitempty"`
}

type ToolBatchStatus struct {
	Detection ToolDetection    `json:"detection"`
	Config    ToolConfigStatus `json:"config"`
	Error     string           `json:"error,omitempty"`
}

type ToolBatchStatusMap map[string]ToolBatchStatus

type BackupEntry struct {
	BackupID  string `json:"backupId"`
	Tool      string `json:"tool"`
	Path      string `json:"path"`
	Filename  string `json:"filename"`
	CreatedAt string `json:"createdAt"`
	SizeBytes int64  `json:"sizeBytes"`
}

type ApplyRequest struct {
	ToolID  string `json:"toolId"`
	BaseURL string `json:"baseUrl,omitempty"`
	APIKey  string `json:"apiKey,omitempty"`
	Model   string `json:"model,omitempty"`
	DryRun  bool   `json:"dryRun,omitempty"`
}

func getBackupDir(toolId string) string {
	home, _ := os.UserHomeDir()
	dir := filepath.Join(home, ".goroute", "backups", toolId)
	os.MkdirAll(dir, 0755)
	return dir
}

// BackupConfigFile saves a timestamped copy of the config file.
func BackupConfigFile(toolId, filePath string) (*BackupEntry, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, nil
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	backupDir := getBackupDir(toolId)
	backupName := fmt.Sprintf("%s_%d.bak", filepath.Base(filePath), time.Now().Unix())
	targetPath := filepath.Join(backupDir, backupName)

	if err := os.WriteFile(targetPath, data, 0600); err != nil {
		return nil, err
	}

	fi, _ := os.Stat(targetPath)
	var size int64
	if fi != nil {
		size = fi.Size()
	}

	return &BackupEntry{
		BackupID:  backupName,
		Tool:      toolId,
		Path:      targetPath,
		Filename:  backupName,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
		SizeBytes: size,
	}, nil
}

// ListBackups returns all backups for a given tool (or all tools if empty).
func ListBackups(toolId string) ([]BackupEntry, error) {
	home, _ := os.UserHomeDir()
	baseDir := filepath.Join(home, ".goroute", "backups")

	var list []BackupEntry
	var toolDirs []string

	if toolId != "" {
		toolDirs = append(toolDirs, filepath.Join(baseDir, toolId))
	} else {
		entries, err := os.ReadDir(baseDir)
		if err == nil {
			for _, e := range entries {
				if e.IsDir() {
					toolDirs = append(toolDirs, filepath.Join(baseDir, e.Name()))
				}
			}
		}
	}

	for _, dir := range toolDirs {
		t := filepath.Base(dir)
		files, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		for _, f := range files {
			if strings.HasSuffix(f.Name(), ".bak") {
				info, _ := f.Info()
				var size int64
				var modTime time.Time
				if info != nil {
					size = info.Size()
					modTime = info.ModTime()
				}
				list = append(list, BackupEntry{
					BackupID:  f.Name(),
					Tool:      t,
					Path:      filepath.Join(dir, f.Name()),
					Filename:  f.Name(),
					CreatedAt: modTime.UTC().Format(time.RFC3339),
					SizeBytes: size,
				})
			}
		}
	}

	return list, nil
}

// RestoreBackup restores a backup file over the active config file.
func RestoreBackup(toolId, backupId string) error {
	home, _ := os.UserHomeDir()
	backupPath := filepath.Join(home, ".goroute", "backups", toolId, backupId)
	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup %s: %w", backupId, err)
	}

	tool, exists := Registry[toolId]
	if !exists {
		return fmt.Errorf("unknown tool %s", toolId)
	}

	targetPath := strings.Replace(tool.SettingsFile, "~", home, 1)
	os.MkdirAll(filepath.Dir(targetPath), 0755)
	return os.WriteFile(targetPath, data, 0644)
}

// DeleteBackup deletes a specific backup file.
func DeleteBackup(toolId, backupId string) error {
	home, _ := os.UserHomeDir()
	backupPath := filepath.Join(home, ".goroute", "backups", toolId, backupId)
	return os.Remove(backupPath)
}

// GetAllStatuses scans all registered CLI tools in parallel and checks runtime and config.
func GetAllStatuses() (ToolBatchStatusMap, error) {
	home, _ := os.UserHomeDir()
	statuses := make(ToolBatchStatusMap)
	var mu sync.Mutex

	var lastConfiguredMap map[string]int64
	db := storage.GetDB()
	if db != nil {
		lastConfiguredMap, _ = db.GetAllCliToolLastConfigured()
	}

	var wg sync.WaitGroup
	for id, tool := range Registry {
		wg.Add(1)
		go func(id string, tool CliTool) {
			defer wg.Done()

			var det ToolDetection
			binName := tool.DefaultCommand
			if binName != "" {
				if p, err := exec.LookPath(binName); err == nil && p != "" {
					det.Installed = true
					det.Runnable = true
					det.Command = binName
					det.CommandPath = p

					// Version probe with timeout (skip GUI apps)
					if binName != "antigravity" && binName != "cursor" && binName != "windsurf" && binName != "zed" {
						ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
						cmd := exec.CommandContext(ctx, binName, "--version")
						if out, err := cmd.Output(); err == nil {
							v := strings.TrimSpace(string(out))
							if len(v) > 40 {
								v = v[:40]
							}
							det.Version = v
						}
						cancel()
					}
				}
			}

			cfgStatus := ToolConfigStatus{
				Status: "not_configured",
			}

			expandedPath := strings.Replace(tool.SettingsFile, "~", home, 1)
			if fi, err := os.Stat(expandedPath); err == nil && !fi.IsDir() {
				det.Installed = true // config exists, mark as installed
				data, _ := os.ReadFile(expandedPath)
				content := string(data)
				if strings.Contains(content, "8128") || strings.Contains(content, "goroute") {
					cfgStatus.Status = "configured"
					if strings.Contains(content, "8128/v1") {
						cfgStatus.Endpoint = "http://127.0.0.1:8128/v1"
					} else {
						cfgStatus.Endpoint = "http://127.0.0.1:8128"
					}
				} else {
					cfgStatus.Status = "partial"
				}
			}

			if lastConfiguredMap != nil {
				if ts, ok := lastConfiguredMap[id]; ok {
					cfgStatus.LastConfiguredAt = &ts
				}
			}

			mu.Lock()
			statuses[id] = ToolBatchStatus{
				Detection: det,
				Config:    cfgStatus,
			}
			mu.Unlock()
		}(id, tool)
	}

	wg.Wait()
	return statuses, nil
}

// DetectTool probes a single tool.
func DetectTool(toolId string) (*ToolBatchStatus, error) {
	all, err := GetAllStatuses()
	if err != nil {
		return nil, err
	}
	if st, ok := all[toolId]; ok {
		return &st, nil
	}
	return nil, fmt.Errorf("unknown tool %s", toolId)
}

// ApplyTool executes configuration with automated backup and state tracking.
func ApplyTool(req ApplyRequest) (map[string]interface{}, error) {
	tool, exists := Registry[req.ToolID]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", req.ToolID)
	}

	home, _ := os.UserHomeDir()
	configPath := strings.Replace(tool.SettingsFile, "~", home, 1)

	// Dry run preview
	if req.DryRun {
		return map[string]interface{}{
			"success":    true,
			"dryRun":     true,
			"toolId":     req.ToolID,
			"configPath": configPath,
			"message":    fmt.Sprintf("Would configure %s at %s", tool.Name, configPath),
		}, nil
	}

	// 1. Ensure backup
	backup, _ := BackupConfigFile(req.ToolID, configPath)

	// 2. Run configure tool
	msg, err := ConfigureTool(req.ToolID)
	if err != nil {
		return nil, fmt.Errorf("failed to configure %s: %w", req.ToolID, err)
	}

	// 3. Persist state in SQLite
	now := time.Now().Unix()
	db := storage.GetDB()
	if db != nil {
		endpoint := req.BaseURL
		if endpoint == "" {
			endpoint = "http://127.0.0.1:8128/v1"
		}
		_ = db.SaveCliToolState(req.ToolID, now, configPath, endpoint)
	}

	res := map[string]interface{}{
		"success":    true,
		"message":    msg,
		"toolId":     req.ToolID,
		"configPath": configPath,
	}
	if backup != nil {
		res["backup"] = backup
	}
	return res, nil
}

// ReadSettingsFile reads tool config file as JSON or raw content.
func ReadSettingsFile(toolId string) (map[string]interface{}, error) {
	tool, exists := Registry[toolId]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", toolId)
	}
	home, _ := os.UserHomeDir()
	path := strings.Replace(tool.SettingsFile, "~", home, 1)

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(data, &res); err != nil {
		// return as raw content if not JSON
		return map[string]interface{}{
			"raw": string(data),
		}, nil
	}
	return res, nil
}

// ToolConfigDetail holds complete inspection information for a CLI agent.
type ToolConfigDetail struct {
	ToolID       string        `json:"tool_id"`
	Name         string        `json:"name"`
	Category     string        `json:"category"`
	DocsURL      string        `json:"docs_url"`
	SettingsFile string        `json:"settings_file"`
	Exists       bool          `json:"exists"`
	Configured   bool          `json:"configured"`
	Detected     bool          `json:"detected"`
	Runnable     bool          `json:"runnable"`
	Version      string        `json:"version,omitempty"`
	Content      string        `json:"content"`
	IsJSON       bool          `json:"is_json"`
	Backups      []BackupEntry `json:"backups"`
}

// GetToolConfigDetail retrieves the live configuration file, detection status, and backup history.
func GetToolConfigDetail(toolId string) (*ToolConfigDetail, error) {
	tool, exists := Registry[toolId]
	if !exists {
		return nil, fmt.Errorf("unknown tool: %s", toolId)
	}

	home, _ := os.UserHomeDir()
	path := strings.Replace(tool.SettingsFile, "~", home, 1)

	detail := &ToolConfigDetail{
		ToolID:       toolId,
		Name:         tool.Name,
		Category:     tool.Category,
		DocsURL:      tool.DocsURL,
		SettingsFile: path,
	}

	// Probe binary
	if tool.DefaultCommand != "" {
		if p, err := exec.LookPath(tool.DefaultCommand); err == nil && p != "" {
			detail.Detected = true
			detail.Runnable = true
			if tool.DefaultCommand != "antigravity" && tool.DefaultCommand != "cursor" && tool.DefaultCommand != "windsurf" && tool.DefaultCommand != "zed" {
				ctx, cancel := context.WithTimeout(context.Background(), 250*time.Millisecond)
				cmd := exec.CommandContext(ctx, tool.DefaultCommand, "--version")
				if out, err := cmd.Output(); err == nil {
					v := strings.TrimSpace(string(out))
					if len(v) > 40 {
						v = v[:40]
					}
					detail.Version = v
				}
				cancel()
			}
		}
	}

	// Read file if exists
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		detail.Exists = true
		detail.Detected = true
		if data, err := os.ReadFile(path); err == nil {
			detail.Content = string(data)
			if strings.Contains(detail.Content, "8128") || strings.Contains(detail.Content, "goroute") {
				detail.Configured = true
			}
			var js interface{}
			if json.Unmarshal(data, &js) == nil {
				detail.IsJSON = true
			}
		}
	}

	// Backups
	backups, _ := ListBackups(toolId)
	if backups == nil {
		backups = []BackupEntry{}
	}
	detail.Backups = backups

	return detail, nil
}

// SaveToolConfig writes new raw content to the agent's settings file with an automatic backup.
func SaveToolConfig(toolId string, content string) error {
	tool, exists := Registry[toolId]
	if !exists {
		return fmt.Errorf("unknown tool: %s", toolId)
	}

	home, _ := os.UserHomeDir()
	path := strings.Replace(tool.SettingsFile, "~", home, 1)

	// Backup existing file if present
	_, _ = BackupConfigFile(toolId, path)

	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		return fmt.Errorf("failed to write settings file: %w", err)
	}

	// Update DB state
	now := time.Now().Unix()
	db := storage.GetDB()
	if db != nil {
		if strings.Contains(content, "8128") || strings.Contains(content, "goroute") {
			_ = db.SaveCliToolState(toolId, now, path, "http://127.0.0.1:8128/v1")
		}
	}

	return nil
}

// DisconnectTool removes GoRoute configurations from the agent's settings file and restores defaults.
func DisconnectTool(toolId string) error {
	tool, exists := Registry[toolId]
	if !exists {
		return fmt.Errorf("unknown tool: %s", toolId)
	}

	home, _ := os.UserHomeDir()
	path := strings.Replace(tool.SettingsFile, "~", home, 1)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		// Nothing to disconnect
		db := storage.GetDB()
		if db != nil {
			_ = db.DeleteCliToolState(toolId)
		}
		return nil
	}

	// Always backup before modifying or disconnecting
	_, _ = BackupConfigFile(toolId, path)

	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	content := string(data)
	if !strings.Contains(content, "8128") && !strings.Contains(content, "goroute") {
		// Not currently pointing to GoRoute
		db := storage.GetDB()
		if db != nil {
			_ = db.DeleteCliToolState(toolId)
		}
		return nil
	}

	switch toolId {
	case "claude":
		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err == nil && settings != nil {
			if env, ok := settings["env"].(map[string]interface{}); ok {
				delete(env, "ANTHROPIC_BASE_URL")
				delete(env, "ANTHROPIC_AUTH_TOKEN")
				delete(env, "ANTHROPIC_DEFAULT_SONNET_MODEL")
				delete(env, "ANTHROPIC_DEFAULT_OPUS_MODEL")
				delete(env, "ANTHROPIC_DEFAULT_HAIKU_MODEL")
				if len(env) == 0 {
					delete(settings, "env")
				}
			}
			out, _ := json.MarshalIndent(settings, "", "  ")
			_ = os.WriteFile(path, out, 0644)
		}

	case "opencode":
		var settings map[string]interface{}
		if err := json.Unmarshal(data, &settings); err == nil && settings != nil {
			if prov, ok := settings["provider"].(map[string]interface{}); ok {
				delete(prov, "goroute")
				if len(prov) == 0 {
					delete(settings, "provider")
				}
			}
			if mod, ok := settings["model"].(string); ok && strings.HasPrefix(mod, "goroute/") {
				delete(settings, "model")
			}
			if sm, ok := settings["small_model"].(string); ok && strings.HasPrefix(sm, "goroute/") {
				delete(settings, "small_model")
			}
			out, _ := json.MarshalIndent(settings, "", "  ")
			_ = os.WriteFile(path, out, 0644)
		}

	default:
		// Generic JSON or line-based cleanup
		var js map[string]interface{}
		if err := json.Unmarshal(data, &js); err == nil && js != nil {
			// Remove GoRoute keys
			delete(js, "goroute")
			// If file was solely generated for GoRoute proxying, rename to .bak or reset
			if len(js) == 0 || (len(js) <= 3 && strings.Contains(content, "8128")) {
				_ = os.Remove(path)
			} else {
				out, _ := json.MarshalIndent(js, "", "  ")
				_ = os.WriteFile(path, out, 0644)
			}
		} else {
			// Non-JSON file (e.g. TOML/YAML)
			var cleanLines []string
			for _, line := range strings.Split(content, "\n") {
				if strings.Contains(line, "8128") || strings.Contains(line, "goroute") {
					continue
				}
				cleanLines = append(cleanLines, line)
			}
			cleanContent := strings.TrimSpace(strings.Join(cleanLines, "\n"))
			if cleanContent == "" {
				_ = os.Remove(path)
			} else {
				_ = os.WriteFile(path, []byte(cleanContent+"\n"), 0644)
			}
		}
	}

	db := storage.GetDB()
	if db != nil {
		_ = db.DeleteCliToolState(toolId)
	}

	return nil
}

