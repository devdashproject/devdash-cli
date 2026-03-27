package commands

import (
	"fmt"
	"os"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/config"
	"github.com/spf13/cobra"
)

const Version = "0.3.0"

var (
	cfg    *config.Config
	client *api.Client
)

// rootCmd is the base command.
var rootCmd = &cobra.Command{
	Use:   "devdash",
	Short: "AI-powered task tracking for developers and agents",
	Long:  "DevDash CLI — lightweight task tracking built for AI coding agents and developer workflows.",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Skip config loading for commands that don't need it
		switch cmd.Name() {
		case "version", "help":
			return nil
		}

		var err error
		cfg, err = config.Load()
		if err != nil {
			fmt.Fprintf(os.Stderr, "config error: %v\n", err)
			os.Exit(api.ExitConfig)
		}

		// Only create API client if we have a token
		if cfg.Token != "" {
			client = api.New(cfg.APIURL, cfg.Token)
		}

		return nil
	},
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.PersistentFlags().String("project", "", "Override project ID (full UUID or prefix)")

	// Register subcommands
	rootCmd.AddCommand(versionCmd)

	// Override default help to support topic subcommands
	RegisterHelpTopics()
}

// requireAuth ensures we have a valid token and API client.
func requireAuth() error {
	if cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}
	if _, err := cfg.RequireToken(); err != nil {
		return err
	}
	if client == nil {
		client = api.New(cfg.APIURL, cfg.Token)
	}
	return nil
}

// requireProject ensures we have a project ID configured.
func requireProject() (string, error) {
	if err := requireAuth(); err != nil {
		return "", err
	}

	// Check --project flag first
	if p, _ := rootCmd.PersistentFlags().GetString("project"); p != "" {
		return p, nil
	}

	return cfg.RequireProjectID()
}

// apiClientFromConfig creates a new API client from a config (used by doctor).
func apiClientFromConfig(c *config.Config) *api.Client {
	return api.New(c.APIURL, c.Token)
}

// Version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("devdash %s\n", Version)
	},
}
