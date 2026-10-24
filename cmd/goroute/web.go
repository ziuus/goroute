package main

import (
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
		Use:   "web",
		Aliases: []string{"ui", "dashboard"},
		Short: "Open the GoRoute web dashboard in your default browser",
		RunE: func(cmd *cobra.Command, args []string) error {
			url := fmt.Sprintf("http://127.0.0.1:%d/", port)

			// Check if daemon is active
			client := http.Client{Timeout: 500 * time.Millisecond}
			resp, err := client.Get(fmt.Sprintf("http://127.0.0.1:%d/api/topology", port))
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

			fmt.Printf("🚀 Opening GoRoute Dashboard at %s\n", url)
			openBrowser(url)
			return nil
		},
	}
	cmd.Flags().IntVarP(&port, "port", "p", 8128, "GoRoute gateway port")
	return cmd
}
