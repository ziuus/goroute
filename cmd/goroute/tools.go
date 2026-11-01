package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/ziuus/goroute/pkg/config"
)

func NewDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Run comprehensive diagnostic and health checks for GoRoute",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Println("🩺 Running GoRoute Health & Usability Diagnostics...")
			fmt.Println()

			// 1. Check Gateway Daemon
			fmt.Print(" [1/5] Gateway Server (127.0.0.1:8128): ")
			resp, err := http.Get("http://127.0.0.1:8128/api/status")
			if err != nil {
				fmt.Println("❌ NOT RUNNING (Connection refused)")
				fmt.Println("       Start with: 'goroute serve' or 'systemctl --user start goroute.service'")
			} else {
				defer resp.Body.Close()
				var st map[string]interface{}
				_ = json.NewDecoder(resp.Body).Decode(&st)
				mem, _ := st["memory_mb"].(float64)
				uptime, _ := st["uptime_seconds"].(float64)
				fmt.Printf("✅ ACTIVE (Memory: %.1f MB, Uptime: %.0fs)\n", mem, uptime)
			}

			// 2. Check Config Integrity
			fmt.Print(" [2/5] Configuration (~/.goroute/config.yaml): ")
			home, _ := os.UserHomeDir()
			cfgPath := filepath.Join(home, ".goroute", "config.yaml")
			cfg, err := config.Load(cfgPath)
			if err != nil {
				fmt.Printf("⚠️  MISSING or Corrupt (%v)\n", err)
			} else {
				fmt.Printf("✅ OK (%d Providers, %d Combos configured)\n", len(cfg.Providers), len(cfg.Combos))
			}

			// 3. Check Universal Models Endpoint
			fmt.Print(" [3/5] Universal OpenAI Endpoint (/v1/models): ")
			mResp, err := http.Get("http://127.0.0.1:8128/v1/models")
			if err != nil {
				fmt.Println("❌ Unreachable")
			} else {
				defer mResp.Body.Close()
				var ml struct {
					Data []interface{} `json:"data"`
				}
				_ = json.NewDecoder(mResp.Body).Decode(&ml)
				fmt.Printf("✅ OK (%d active models ready to route)\n", len(ml.Data))
			}

			// 4. Check Coding Agent Installations
			fmt.Println(" [4/5] Agent Readiness on this machine:")
			agents := []struct {
				name string
				bin  string
				path string
			}{
				{"Claude Code", "claude", filepath.Join(home, ".claude", "settings.json")},
				{"OpenCode", "opencode", filepath.Join(home, ".config", "opencode", "opencode.json")},
				{"Codex CLI", "codex", filepath.Join(home, ".codex", "config.toml")},
				{"Hermes Agent", "hermes", filepath.Join(home, ".hermes", "config.yaml")},
				{"OpenClaw", "openclaw", filepath.Join(home, ".openclaw", "openclaw.json")},
				{"Cursor / Cline", "cursor", ""},
			}
			for _, ag := range agents {
				_, binErr := exec.LookPath(ag.bin)
				binStatus := "not in PATH"
				if binErr == nil {
					binStatus = "installed"
				}
				confStatus := "not configured"
				if ag.path == "" {
					confStatus = "configured (universal)"
				} else if _, err := os.Stat(ag.path); err == nil {
					confStatus = "configured"
				}
				fmt.Printf("       • %-15s: %s, %s\n", ag.name, binStatus, confStatus)
			}

			// 5. Cooldowns & Resilience
			fmt.Print(" [5/5] Resilience & Quarantine: ")
			cResp, err := http.Get("http://127.0.0.1:8128/api/resilience/model-cooldowns")
			if err != nil {
				fmt.Println("⚠️  Status unavailable")
			} else {
				defer cResp.Body.Close()
				var cd struct {
					Count int `json:"count"`
				}
				_ = json.NewDecoder(cResp.Body).Decode(&cd)
				if cd.Count == 0 {
					fmt.Println("✅ All upstreams healthy (0 active cooldowns)")
				} else {
					fmt.Printf("⚠️  %d model targets temporarily in cooldown quarantine\n", cd.Count)
				}
			}

			fmt.Println("\n🎉 Diagnostic complete. GoRoute is fully operational!")
			return nil
		},
	}
}

func NewChatCmd() *cobra.Command {
	var model string
	cmd := &cobra.Command{
		Use:   "chat",
		Short: "Interactive chat with any model or combo directly in your terminal",
		RunE: func(cmd *cobra.Command, args []string) error {
			if model == "" {
				model = "gemini-flash"
			}
			fmt.Printf("💬 Interactive GoRoute Terminal Chat (Model: %s)\n", model)
			fmt.Println("   Type your message and press Enter. Type 'exit' or 'quit' to end.")
			fmt.Println()

			scanner := bufio.NewScanner(os.Stdin)
			var history []map[string]string

			for {
				fmt.Print("👤 You: ")
				if !scanner.Scan() {
					break
				}
				userInput := strings.TrimSpace(scanner.Text())
				if userInput == "" {
					continue
				}
				if userInput == "exit" || userInput == "quit" {
					fmt.Println("Bye!")
					break
				}

				history = append(history, map[string]string{"role": "user", "content": userInput})

				reqBody := map[string]interface{}{
					"model":    model,
					"messages": history,
					"stream":   true,
				}
				b, _ := json.Marshal(reqBody)

				httpReq, err := http.NewRequest("POST", "http://127.0.0.1:8128/v1/chat/completions", bytes.NewReader(b))
				if err != nil {
					return err
				}
				httpReq.Header.Set("Content-Type", "application/json")
				httpReq.Header.Set("Authorization", "Bearer goroute")

				resp, err := http.DefaultClient.Do(httpReq)
				if err != nil {
					fmt.Printf("❌ Error connecting to gateway: %v\n", err)
					continue
				}

				if resp.StatusCode != 200 {
					raw, _ := io.ReadAll(resp.Body)
					resp.Body.Close()
					fmt.Printf("❌ HTTP %d: %s\n", resp.StatusCode, string(raw))
					continue
				}

				fmt.Print("🤖 AI: ")
				streamScanner := bufio.NewScanner(resp.Body)
				var fullAssistantReply strings.Builder

				for streamScanner.Scan() {
					line := streamScanner.Text()
					if strings.HasPrefix(line, "data: ") {
						data := strings.TrimPrefix(line, "data: ")
						if data == "[DONE]" {
							break
						}
						var chunk struct {
							Choices []struct {
								Delta struct {
									Content string `json:"content"`
								} `json:"delta"`
							} `json:"choices"`
						}
						if err := json.Unmarshal([]byte(data), &chunk); err == nil && len(chunk.Choices) > 0 {
							c := chunk.Choices[0].Delta.Content
							if c != "" {
								fmt.Print(c)
								fullAssistantReply.WriteString(c)
							}
						}
					}
				}
				resp.Body.Close()
				fmt.Println()
				history = append(history, map[string]string{"role": "assistant", "content": fullAssistantReply.String()})
			}
			return nil
		},
	}
	cmd.Flags().StringVarP(&model, "model", "m", "gemini-flash", "Model or combo to chat with")
	return cmd
}

type GatewayLogEntry struct {
	ID              string    `json:"id"`
	Timestamp       time.Time `json:"timestamp"`
	Endpoint        string    `json:"endpoint"`
	Model           string    `json:"model"`
	Provider        string    `json:"provider,omitempty"`
	StatusCode      int       `json:"status_code"`
	DurationMs      int64     `json:"duration_ms"`
	Tokens          int       `json:"tokens,omitempty"`
	InputTokens     int       `json:"input_tokens,omitempty"`
	OutputTokens    int       `json:"output_tokens,omitempty"`
	Error           string    `json:"error,omitempty"`
	PromptPreview   string    `json:"prompt_preview,omitempty"`
	ResponsePreview string    `json:"response_preview,omitempty"`
}

func printLogEntry(l GatewayLogEntry, detailed bool) {
	timeStr := l.Timestamp.Format("15:04:05")
	isError := l.StatusCode >= 400 || l.Error != ""

	colorReset := "\033[0m"
	colorGreen := "\033[32m"
	colorRed := "\033[31;1m"
	colorYellow := "\033[33m"
	colorCyan := "\033[36m"
	colorDim := "\033[90m"
	colorMagenta := "\033[35m"

	statusColor := colorGreen
	statusText := fmt.Sprintf("%d OK", l.StatusCode)
	if l.StatusCode >= 500 {
		statusColor = colorRed
		statusText = fmt.Sprintf("%d ERR", l.StatusCode)
	} else if l.StatusCode >= 400 {
		statusColor = colorRed
		statusText = fmt.Sprintf("%d ERR", l.StatusCode)
	} else if l.StatusCode >= 300 {
		statusColor = colorYellow
		statusText = fmt.Sprintf("%d REDIR", l.StatusCode)
	}

	modelStr := l.Model
	if modelStr == "" {
		modelStr = "default"
	}

	endpointStr := l.Endpoint
	if endpointStr == "" {
		endpointStr = "/v1/chat/completions"
	}

	if detailed {
		badge := "●"
		if isError {
			badge = "✖"
		}
		tokenInfo := ""
		if l.Tokens > 0 || l.InputTokens > 0 || l.OutputTokens > 0 {
			tokenInfo = fmt.Sprintf(" | %stokens:%s %d (%din / %dout)", colorDim, colorReset, l.Tokens, l.InputTokens, l.OutputTokens)
		}
		providerInfo := ""
		if l.Provider != "" {
			providerInfo = fmt.Sprintf(" %s(%s)%s", colorMagenta, l.Provider, colorReset)
		}

		fmt.Printf("%s%s [%s]%s %s%-8s%s %sPOST %s%s\n",
			statusColor, badge, timeStr, colorReset,
			statusColor, statusText, colorReset,
			colorDim, endpointStr, colorReset)
		fmt.Printf("  • %sModel:%s    %s%s%s%s | %slatency:%s %dms%s\n",
			colorDim, colorReset, colorCyan, modelStr, colorReset, providerInfo,
			colorDim, colorReset, l.DurationMs, tokenInfo)

		if isError {
			errMsg := l.Error
			if errMsg == "" {
				errMsg = fmt.Sprintf("HTTP %d upstream error", l.StatusCode)
			}
			fmt.Printf("  • %s✖ Error:%s    %s%s%s\n", colorRed, colorReset, colorRed, errMsg, colorReset)
		}
		if l.PromptPreview != "" {
			fmt.Printf("  • %s▶ Prompt:%s   \"%s\"\n", colorDim, colorReset, l.PromptPreview)
		}
		if l.ResponsePreview != "" && !isError {
			fmt.Printf("  • %s◀ Response:%s \"%s\"\n", colorDim, colorReset, l.ResponsePreview)
		}
		if l.PromptPreview == "" && l.ResponsePreview == "" && !isError {
			fmt.Printf("  • %sInfo:%s       %s(No prompt/response captured for synthetic probe or benchmark)%s\n", colorDim, colorReset, colorDim, colorReset)
		}
		fmt.Println()
	} else {
		tokenStr := "-"
		if l.Tokens > 0 {
			tokenStr = fmt.Sprintf("%d", l.Tokens)
		} else if l.OutputTokens > 0 {
			tokenStr = fmt.Sprintf("%d", l.OutputTokens)
		}

		detailSnippet := ""
		if isError && l.Error != "" {
			detailSnippet = fmt.Sprintf("%s✖ %s%s", colorRed, l.Error, colorReset)
		} else if l.ResponsePreview != "" {
			detailSnippet = fmt.Sprintf("%s◀ %s%s", colorDim, l.ResponsePreview, colorReset)
		} else if l.PromptPreview != "" {
			detailSnippet = fmt.Sprintf("%s▶ %s%s", colorDim, l.PromptPreview, colorReset)
		}

		fmt.Printf("%-10s | %s%-8s%s | %-22s | %-7s | %-7s | %s\n",
			timeStr,
			statusColor, statusText, colorReset,
			modelStr,
			fmt.Sprintf("%dms", l.DurationMs),
			tokenStr,
			detailSnippet,
		)
	}
}

func NewLogsCmd() *cobra.Command {
	var (
		limit    int
		follow   bool
		detailed bool
		filter   string
		port     int
	)

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View recent gateway request traffic, errors, and live responses",
		Long: `Stream or inspect gateway traffic in real-time or historical logs.

Examples:
  goroute logs                # Show last 20 requests in compact table
  goroute logs -d             # Detailed view showing prompts, responses, errors, tokens
  goroute logs -f             # Stream live requests in real-time as they arrive
  goroute logs -f -d          # Stream live requests with full response and error details
  goroute logs --filter error # Show only requests with errors or HTTP 4xx/5xx`,
		RunE: func(cmd *cobra.Command, args []string) error {
			baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

			if follow {
				streamURL := fmt.Sprintf("%s/api/logs/stream", baseURL)
				req, err := http.NewRequestWithContext(cmd.Context(), "GET", streamURL, nil)
				if err != nil {
					return err
				}
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					return fmt.Errorf("could not connect to gateway log stream on %s: %w", streamURL, err)
				}
				defer resp.Body.Close()

				if resp.StatusCode != http.StatusOK {
					return fmt.Errorf("gateway stream returned HTTP %d", resp.StatusCode)
				}

				fmt.Printf("⚡ Streaming live GoRoute requests from %s (Ctrl+C to quit)\n", streamURL)
				if !detailed {
					fmt.Printf("%-10s | %-8s | %-22s | %-7s | %-7s | %s\n", "TIME", "STATUS", "MODEL", "LATENCY", "TOKENS", "DETAILS")
					fmt.Println(strings.Repeat("-", 92))
				} else {
					fmt.Println(strings.Repeat("-", 60))
				}

				scanner := bufio.NewScanner(resp.Body)
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if !strings.HasPrefix(line, "data: ") {
						continue
					}
					payload := strings.TrimPrefix(line, "data: ")
					if payload == "" {
						continue
					}
					var entry GatewayLogEntry
					if err := json.Unmarshal([]byte(payload), &entry); err != nil {
						continue
					}

					if filter == "error" && entry.StatusCode < 400 && entry.Error == "" {
						continue
					}

					printLogEntry(entry, detailed)
				}
				return scanner.Err()
			}

			// Historical / snapshot mode
			logsURL := fmt.Sprintf("%s/api/logs", baseURL)
			resp, err := http.Get(logsURL)
			if err != nil {
				return fmt.Errorf("gateway not running on %s: %w", baseURL, err)
			}
			defer resp.Body.Close()

			var res struct {
				Logs []GatewayLogEntry `json:"logs"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return err
			}

			var logs []GatewayLogEntry
			for _, l := range res.Logs {
				if filter == "error" && l.StatusCode < 400 && l.Error == "" {
					continue
				}
				logs = append(logs, l)
			}

			if len(logs) == 0 {
				fmt.Println("No matching request logs found.")
				return nil
			}

			startIdx := 0
			if len(logs) > limit {
				startIdx = len(logs) - limit
			}

			if !detailed {
				fmt.Printf("%-10s | %-8s | %-22s | %-7s | %-7s | %s\n", "TIME", "STATUS", "MODEL", "LATENCY", "TOKENS", "DETAILS")
				fmt.Println(strings.Repeat("-", 92))
			}

			for i := startIdx; i < len(logs); i++ {
				printLogEntry(logs[i], detailed)
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&limit, "lines", "n", 20, "Number of recent log lines to display")
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "Stream logs continuously in real-time")
	cmd.Flags().BoolVarP(&detailed, "detailed", "d", false, "Display rich request details (prompts, responses, errors, tokens)")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter logs by level (e.g. 'error')")
	cmd.Flags().IntVarP(&port, "port", "p", 8128, "GoRoute gateway port")
	return cmd
}

func NewReloadCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "reload",
		Short: "Hot-reload configuration without restarting the gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Post("http://127.0.0.1:8128/api/reload", "application/json", nil)
			if err != nil {
				return fmt.Errorf("could not connect to gateway: %w", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				fmt.Println("✅ GoRoute configuration reloaded successfully across all providers and combos.")
			} else {
				raw, _ := io.ReadAll(resp.Body)
				fmt.Printf("⚠️  Reload returned HTTP %d: %s\n", resp.StatusCode, string(raw))
			}
			return nil
		},
	}
}

func NewKeysCmd() *cobra.Command {
	keysCmd := &cobra.Command{
		Use:   "keys",
		Short: "Manage Virtual API Keys with rate limits and model restrictions",
	}

	listCmd := &cobra.Command{
		Use:   "list",
		Short: "List all active virtual keys",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := http.Get("http://127.0.0.1:8128/api/keys")
			if err != nil {
				return fmt.Errorf("gateway not running: %w", err)
			}
			defer resp.Body.Close()
			var result struct {
				Keys []struct {
					ID        string  `json:"id"`
					Name      string  `json:"name"`
					Key       string  `json:"key"`
					RateLimit int     `json:"rate_limit"`
					BudgetCap float64 `json:"budget_cap"`
					IsActive  bool    `json:"is_active"`
				} `json:"keys"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				return err
			}
			if len(result.Keys) == 0 {
				fmt.Println("No virtual keys found. Create one with: goroute keys create <name>")
				return nil
			}
			fmt.Printf("%-24s | %-16s | %-40s | %-8s\n", "KEY ID", "NAME", "VIRTUAL API KEY", "STATUS")
			fmt.Println(strings.Repeat("-", 96))
			for _, k := range result.Keys {
				status := "ACTIVE"
				if !k.IsActive {
					status = "REVOKED"
				}
				fmt.Printf("%-24s | %-16s | %-40s | %-8s\n", k.ID, k.Name, k.Key, status)
			}
			return nil
		},
	}

	var budget float64
	var rateLimit int
	var modelsList string

	createCmd := &cobra.Command{
		Use:   "create <name>",
		Short: "Create a new virtual key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			name := args[0]
			var models []string
			if modelsList != "" {
				for _, m := range strings.Split(modelsList, ",") {
					if trimmed := strings.TrimSpace(m); trimmed != "" {
						models = append(models, trimmed)
					}
				}
			}
			payload := map[string]interface{}{
				"name":       name,
				"budget_cap": budget,
				"rate_limit": rateLimit,
				"models":     models,
			}
			b, _ := json.Marshal(payload)
			resp, err := http.Post("http://127.0.0.1:8128/api/keys", "application/json", bytes.NewReader(b))
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			var res struct {
				Key struct {
					ID  string `json:"id"`
					Key string `json:"key"`
				} `json:"key"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
				return err
			}
			fmt.Println("✅ Virtual Key created successfully!")
			fmt.Printf("   Name: %s\n", name)
			fmt.Printf("   ID:   %s\n", res.Key.ID)
			fmt.Printf("   Key:  %s\n", res.Key.Key)
			return nil
		},
	}
	createCmd.Flags().Float64Var(&budget, "budget", 0, "Budget cap in USD")
	createCmd.Flags().IntVar(&rateLimit, "rate-limit", 60, "RPM rate limit")
	createCmd.Flags().StringVar(&modelsList, "models", "", "Comma-separated list of allowed models")

	revokeCmd := &cobra.Command{
		Use:   "revoke <key-id>",
		Short: "Revoke a virtual key",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			keyID := args[0]
			req, _ := http.NewRequest("DELETE", "http://127.0.0.1:8128/api/keys/"+keyID, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				fmt.Printf("✅ Key '%s' revoked successfully.\n", keyID)
			} else {
				fmt.Printf("⚠️  Revoke returned HTTP %d\n", resp.StatusCode)
			}
			return nil
		},
	}

	keysCmd.AddCommand(listCmd, createCmd, revokeCmd)
	return keysCmd
}
