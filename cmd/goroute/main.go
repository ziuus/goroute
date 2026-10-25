package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var Version = "0.6.0"

func NewRootCmd(version string) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "goroute",
		Short: "GoRoute: Ultra-efficient LLM gateway and multi-provider router",
		Long: `GoRoute is a high-performance, resource-efficient AI proxy (< 15MB RAM) 
providing multi-account rotation, OAuth PKCE flows, and automatic fallback cascades.`,
		Version: version,
	}

	rootCmd.AddCommand(
		NewServeCmd(),
		NewAuthCmd(),
		NewStatusCmd(),
		NewProviderCmd(),
		NewComboCmd(),
		NewModelsCmd(),
		NewCatalogCmd(),
		NewSetupCmd(),
		NewLaunchCmd(),
		NewRunCmd(),
		NewAutostartCmd(),
		NewDoctorCmd(),
		NewChatCmd(),
		NewLogsCmd(),
		NewReloadCmd(),
		NewKeysCmd(),
		NewWebCmd(),
		NewSyncCmd(),
	)
	return rootCmd
}

func main() {
	cmd := NewRootCmd(Version)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
