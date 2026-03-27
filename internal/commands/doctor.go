package commands

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/jasonmassey/devdash-cli-go/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(doctorCmd)
}

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check configuration and connectivity",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("devdash %s\n\n", Version)

		issues := 0

		// Check config
		c, err := config.Load()
		if err != nil {
			fmt.Printf("✗ Config: %v\n", err)
			issues++
		} else {
			fmt.Printf("✓ Config directory: %s\n", c.ConfigDir)
		}

		// Check token
		if c != nil && c.Token != "" {
			fmt.Printf("✓ Token: present (%s)\n", c.TokenFilePath())
		} else {
			fmt.Printf("✗ Token: not found — run 'devdash login'\n")
			issues++
		}

		// Check project
		if c != nil && c.ProjectID != "" {
			fmt.Printf("✓ Project: %s\n", c.ProjectID)
		} else {
			fmt.Printf("○ Project: not configured — run 'devdash init'\n")
		}

		// Check .devdash file
		if _, err := os.Stat(config.ProjectFileName); err == nil {
			fmt.Printf("✓ %s: found\n", config.ProjectFileName)
		} else {
			fmt.Printf("○ %s: not found in current directory\n", config.ProjectFileName)
		}

		// Check git
		if _, err := exec.LookPath("git"); err == nil {
			fmt.Printf("✓ git: available\n")
		} else {
			fmt.Printf("✗ git: not found\n")
			issues++
		}

		// Test API connectivity
		if c != nil && c.Token != "" {
			fmt.Printf("\nTesting API connectivity to %s...\n", c.APIURL)
			testClient := apiClientFromConfig(c)
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
