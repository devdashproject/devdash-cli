package commands

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/devdashproject/devdash-cli/internal/auth"
	"github.com/devdashproject/devdash-cli/internal/config"
	"github.com/spf13/cobra"
)

func newLoginCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "login",
		Short: "Authenticate with DevDash",
		Long: `Authenticate with DevDash using an OAuth browser flow.

Starts a local HTTP callback server, generates a one-time nonce, and opens
your default browser to the DevDash auth page. Once you approve access the
token is saved to the CLI config file automatically.

Pass --no-browser to print the auth URL instead of launching a browser
(useful for SSH sessions or headless environments). The command will wait
up to 120 seconds for the browser callback before timing out.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if d.Cfg == nil {
				var err error
				d.Cfg, err = config.Load()
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

			authURL := fmt.Sprintf("%s/api/auth/cli-token?port=%d&nonce=%s", d.Cfg.APIURL, port, nonce)

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
				if err := d.Cfg.SaveToken(result.Token); err != nil {
					return fmt.Errorf("failed to save token: %w", err)
				}
				fmt.Println("Authentication successful! Token saved.")
				return nil
			case <-time.After(120 * time.Second):
				return fmt.Errorf("authentication timed out after 120 seconds")
			}
		},
	}
	cmd.Flags().Bool("no-browser", false, "Skip automatic browser launch")
	return cmd
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
