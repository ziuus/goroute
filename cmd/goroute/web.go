package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"

	"github.com/spf13/cobra"
)

func NewUiCmd() *cobra.Command {
	return NewWebCmd()
}

func NewWebCmd() *cobra.Command {
	var port int
	cmd := &cobra.Command{
		Use:     "web [on|off|status]",
		Aliases: []string{"ui", "dashboard"},
		Short:   "Manage and hot-toggle the GoRoute web dashboard on the fly without restarting",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				switch args[0] {
				case "on", "enable", "start":
					return toggleWebDashboard(port, true)
				case "off", "disable", "stop":
					return toggleWebDashboard(port, false)
				case "status":
					return checkWebDashboardStatus(port)
				default:
					return fmt.Errorf("unknown action '%s'. Use 'on', 'off', or 'status'", args[0])
				}
			}

			// Default action: make sure dashboard is active, then open browser
			url := fmt.Sprintf("http://127.0.0.1:%d/", port)
			client := http.Client{Timeout: 500 * time.Millisecond}
			resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/health", port))
			if err != nil || resp.StatusCode != http.StatusOK {
				fmt.Printf("⚡ GoRoute daemon is not responding on port %d.\n", port)
				fmt.Println("   Starting background gateway daemon...")

				startCmd := exec.Command("goroute", "serve", "-p", fmt.Sprintf("%d", port))
				if err := startCmd.Start(); err != nil {
					return fmt.Errorf("failed to start daemon: %w", err)
				}
				time.Sleep(1 * time.Second)
			} else {
				resp.Body.Close()
			}

			// Ensure dashboard is enabled on the running daemon
			_ = toggleWebDashboard(port, true)

			fmt.Printf("🚀 Opening GoRoute Dashboard at %s\n", url)
			openBrowser(url)
			return nil
		},
	}

	onCmd := &cobra.Command{
		Use:   "on",
		Short: "Instantly enable the web dashboard on the running gateway without restarting",
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleWebDashboard(port, true)
		},
	}

	offCmd := &cobra.Command{
		Use:   "off",
		Short: "Instantly disable the web dashboard on the running gateway without restarting",
		RunE: func(cmd *cobra.Command, args []string) error {
			return toggleWebDashboard(port, false)
		},
	}

	statusCmd := &cobra.Command{
		Use:   "status",
		Short: "Show whether the web dashboard is currently active on the running gateway",
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkWebDashboardStatus(port)
		},
	}

	cmd.AddCommand(onCmd, offCmd, statusCmd)
	cmd.PersistentFlags().IntVarP(&port, "port", "p", 8128, "GoRoute gateway port")
	return cmd
}

func toggleWebDashboard(port int, enable bool) error {
	client := http.Client{Timeout: 1 * time.Second}
	action := "disable"
	if enable {
		action = "enable"
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/web/%s", port, action)
	resp, err := client.Post(url, "application/json", nil)
	if err != nil {
		return fmt.Errorf("failed to contact running GoRoute gateway on port %d: %w", port, err)
	}
	defer resp.Body.Close()

	var result struct {
		Status  string `json:"status"`
		Enabled bool   `json:"enabled"`
		Message string `json:"message"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if enable {
		fmt.Printf("⚡ Web Dashboard: ENABLED (available at http://127.0.0.1:%d/)\n", port)
		fmt.Println("   Gateway is unaffected and continues running with zero downtime.")
	} else {
		fmt.Printf("⚡ Web Dashboard: DISABLED (http://127.0.0.1:%d/ now returns 404)\n", port)
		fmt.Println("   AI Gateway (:8128/v1/) is unaffected and continues running with zero downtime.")
	}
	return nil
}

func checkWebDashboardStatus(port int) error {
	client := http.Client{Timeout: 1 * time.Second}
	url := fmt.Sprintf("http://127.0.0.1:%d/api/web/status", port)
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("failed to contact running GoRoute gateway on port %d: %w", port, err)
	}
	defer resp.Body.Close()

	var result struct {
		Enabled bool `json:"enabled"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&result)

	if result.Enabled {
		fmt.Printf("● Web Dashboard is ACTIVE on http://127.0.0.1:%d/\n", port)
	} else {
		fmt.Printf("○ Web Dashboard is DISABLED (Headless gateway mode on :%d)\n", port)
	}
	return nil
}
