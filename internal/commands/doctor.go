package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/config"
	"github.com/spf13/cobra"
)

func newDoctorCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Check configuration and connectivity",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("devdash %s\n\n", Version)

			issues := 0

			c, err := config.Load()
			if err != nil {
				fmt.Printf("✗ Config: %v\n", err)
				issues++
			} else {
				fmt.Printf("✓ Config directory: %s\n", c.ConfigDir)
			}

			if c != nil && c.Token != "" {
				fmt.Printf("✓ Token: present (%s)\n", c.TokenFilePath())
			} else {
				fmt.Printf("✗ Token: not found — run 'devdash login'\n")
				issues++
			}

			if c != nil && c.ProjectID != "" {
				fmt.Printf("✓ Project: %s\n", c.ProjectID)
			} else {
				fmt.Printf("○ Project: not configured — run 'devdash init'\n")
			}

			if _, err := os.Stat(config.ProjectFileName); err == nil {
				fmt.Printf("✓ %s: found\n", config.ProjectFileName)
			} else {
				fmt.Printf("○ %s: not found in current directory\n", config.ProjectFileName)
			}

			if _, err := exec.LookPath("git"); err == nil {
				fmt.Printf("✓ git: available\n")
			} else {
				fmt.Printf("✗ git: not found\n")
				issues++
			}

			if c != nil && c.Token != "" {
				fmt.Printf("\nTesting API connectivity to %s...\n", c.APIURL)
				testClient := api.New(c.APIURL, c.Token)
				_, err := testClient.Get("/projects")
				if err != nil {
					fmt.Printf("✗ API: %v\n", err)
					issues++
				} else {
					fmt.Printf("✓ API: connected\n")
				}
			}

			if issues > 0 {
				fmt.Printf("\n%d issue(s) found.\n", issues)
				os.Exit(1)
			}
			fmt.Println("\nAll checks passed.")
			return nil
		},
	}
}
