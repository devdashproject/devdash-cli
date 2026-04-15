package commands

import (
	"fmt"
	"os"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
	"github.com/spf13/cobra"
)

var Version = "0.3.2"

// Deps holds shared dependencies injected into all commands.
type Deps struct {
	Cfg    *config.Config
	Client *api.Client
}

// requireAuth ensures the deps have a valid token and client.
func (d *Deps) requireAuth() error {
	if d.Cfg == nil {
		return fmt.Errorf("configuration not loaded")
	}
	if _, err := d.Cfg.RequireToken(); err != nil {
		return err
	}
	if d.Client == nil {
		d.Client = api.New(d.Cfg.APIURL, d.Cfg.Token)
	}
	return nil
}

// requireProject ensures we have a project ID configured.
func (d *Deps) requireProject(cmd *cobra.Command) (string, error) {
	if err := d.requireAuth(); err != nil {
		return "", err
	}
	if p, _ := cmd.Root().PersistentFlags().GetString("project"); p != "" {
		return p, nil
	}
	return d.Cfg.RequireProjectID()
}

// NewRootCmd constructs the full command tree with injected dependencies.
// In production, pass nil deps — PersistentPreRunE will load config.
// In tests, pass pre-configured deps to skip config loading.
func NewRootCmd(deps *Deps) *cobra.Command {
	if deps == nil {
		deps = &Deps{}
	}

	rootCmd := &cobra.Command{
		Use:   "devdash",
		Short: "AI-powered task tracking for developers and agents",
		Long:  "DevDash CLI — lightweight task tracking built for AI coding agents and developer workflows.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// If deps were injected (tests), skip config loading
			if deps.Cfg != nil {
				return nil
			}

			switch cmd.Name() {
			case "version", "help":
				return nil
			}

			var err error
			deps.Cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("config error: %w", err)
			}

			if deps.Cfg.Token != "" {
				deps.Client = api.New(deps.Cfg.APIURL, deps.Cfg.Token)
			}

			return nil
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().String("project", "", "Override project ID (full UUID or prefix)")

	// Register all subcommands
	rootCmd.AddCommand(newVersionCmd())
	rootCmd.AddCommand(newReadyCmd(deps))
	rootCmd.AddCommand(newShowCmd(deps))
	rootCmd.AddCommand(newCreateCmd(deps))
	rootCmd.AddCommand(newUpdateCmd(deps))
	rootCmd.AddCommand(newCloseCmd(deps))
	rootCmd.AddCommand(newListCmd(deps))
	rootCmd.AddCommand(newReportCmd(deps))
	rootCmd.AddCommand(newBlockedCmd(deps))
	rootCmd.AddCommand(newStaleCmd(deps))
	rootCmd.AddCommand(newStatsCmd(deps))
	rootCmd.AddCommand(newDepCmd(deps))
	rootCmd.AddCommand(newCommentCmd(deps))
	rootCmd.AddCommand(newCommentsCmd(deps))
	rootCmd.AddCommand(newActivityCmd(deps))
	rootCmd.AddCommand(newFindCmd(deps))
	rootCmd.AddCommand(newDeleteCmd(deps))
	rootCmd.AddCommand(newJobsCmd(deps))
	rootCmd.AddCommand(newDiagnoseCmd(deps))
	rootCmd.AddCommand(newLoginCmd(deps))
	rootCmd.AddCommand(newInitCmd(deps))
	rootCmd.AddCommand(newDoctorCmd(deps))
	rootCmd.AddCommand(newTeamCmd(deps))
	rootCmd.AddCommand(newPrimeCmd(deps))
	rootCmd.AddCommand(newTokenCmd(deps))
	rootCmd.AddCommand(newProjectCmd(deps))
	rootCmd.AddCommand(newAnalyzeCmd(deps))
	rootCmd.AddCommand(newDispatchCmd(deps))
	rootCmd.AddCommand(newScoreCmd(deps))
	rootCmd.AddCommand(newSyncCmd(deps))
	rootCmd.AddCommand(newImportCmd(deps))
	rootCmd.AddCommand(newAdminCmd(deps))
	rootCmd.AddCommand(newSelfUpdateCmd())
	rootCmd.AddCommand(newUninstallCmd())
	rootCmd.AddCommand(newAliasSetupCmd())
	rootCmd.AddCommand(newAgentSetupCmd(deps))
	rootCmd.AddCommand(newReconcileCmd(deps))

	registerHelpTopics(rootCmd)

	return rootCmd
}

// Execute creates the root command and runs it.
func Execute() {
	rootCmd := NewRootCmd(nil)
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print the version",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("devdash %s\n", Version)
		},
	}
}
