package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/jasonmassey/devdash-cli-go/internal/auth"
	"github.com/jasonmassey/devdash-cli-go/internal/config"
	"github.com/spf13/cobra"
)

func init() {
	loginCmd.Flags().Bool("no-browser", false, "Skip automatic browser launch")
	rootCmd.AddCommand(loginCmd)
}

var loginCmd = &cobra.Command{
	Use:   "login",
	Short: "Authenticate with DevDash",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config (may not have token yet)
		if cfg == nil {
			var err error
			cfg, err = config.Load()
			if err != nil {
				return err
			}
		}

		nonce, err := auth.GenerateNonce()
		if err != nil {
			return err
		}

		port, resultCh, cleanup, err := auth.StartCallbackServer(nonce)
		if err != nil {
			return err
		}
		defer cleanup()

		authURL := fmt.Sprintf("%s/api/auth/cli-token?port=%d&nonce=%s", cfg.APIURL, port, nonce)

		noBrowser, _ := cmd.Flags().GetBool("no-browser")
		if noBrowser {
			fmt.Printf("Open this URL in your browser:\n%s\n", authURL)
		} else {
			fmt.Println("Opening browser for authentication...")
			if err := openBrowser(authURL); err != nil {
				fmt.Printf("Could not open browser. Open this URL manually:\n%s\n", authURL)
			}
		}

		fmt.Println("Waiting for authentication (timeout: 120s)...")

		select {
		case result := <-resultCh:
			if result.Error != nil {
				return fmt.Errorf("authentication failed: %w", result.Error)
			}
			if err := cfg.SaveToken(result.Token); err != nil {
				return fmt.Errorf("failed to save token: %w", err)
			}
			fmt.Println("Authentication successful! Token saved.")
			return nil
		case <-time.After(120 * time.Second):
			return fmt.Errorf("authentication timed out after 120 seconds")
		}
	},
}

func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
