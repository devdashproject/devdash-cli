package commands

import (
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"time"

	"github.com/devdashproject/devdash-cli/internal/auth"
	"github.com/devdashproject/devdash-cli/internal/config"
	"github.com/spf13/cobra"
)

// buildCLITokenURL builds the dev-dash cli-token auth URL. When provider is
// non-empty it's passed through as ?provider=; the server whitelists it
// server-side (only "github" switches to GitHub login — anything else falls
// back to Google), so an unknown value is harmless.
func buildCLITokenURL(apiURL string, port int, nonce, provider string) string {
	authURL := fmt.Sprintf("%s/api/auth/cli-token?port=%d&nonce=%s", apiURL, port, nonce)
	if provider != "" {
		authURL += "&provider=" + url.QueryEscape(provider)
	}
	return authURL
}

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
up to 120 seconds for the browser callback before timing out.

By default authentication uses Google. Pass --provider=github to sign in
with GitHub instead (for GitHub-first orgs).`,
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

			provider, _ := cmd.Flags().GetString("provider")
			authURL := buildCLITokenURL(d.Cfg.APIURL, port, nonce, provider)

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
				printLoginBreadcrumbs()
				return nil
			case <-time.After(120 * time.Second):
				return fmt.Errorf("authentication timed out after 120 seconds")
			}
		},
	}
	cmd.Flags().Bool("no-browser", false, "Skip automatic browser launch")
	cmd.Flags().String("provider", "", "OAuth provider for login: 'github' (default: google)")
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

func printLoginBreadcrumbs() {
	cwd := mustGetwd()

	// Home/root or not in git repo
	if isHomeOrRoot(cwd) || !isInsideGitRepo() {
		fmt.Println("\nNavigate to your project's top level directory and run `devdash link` to get started.")
		printAliasSetupOffer()
		return
	}

	repoRoot, err := gitRepoRoot()
	if err != nil {
		fmt.Println("\nNavigate to your project's top level directory and run `devdash link` to get started.")
		printAliasSetupOffer()
		return
	}

	// In git repo but not at root
	if cwd != repoRoot {
		fmt.Printf("\nNavigate to your repo's top level directory (%s) and run `devdash link` to get started.\n", repoRoot)
		printAliasSetupOffer()
		return
	}

	// At git root, check if linked
	if _, err := os.Stat(config.ProjectFileName); err == nil {
		fmt.Println("\nThis repo is already linked. Run `devdash ready` to see open issues.")
		printAliasSetupOffer()
		return
	}

	// At git root, not yet linked
	fmt.Println("\nRun `devdash link` to connect this repo to a devdash project.")
	printAliasSetupOffer()
}

func printAliasSetupOffer() {
	fmt.Println()
	fmt.Println("Want to make devdash easier to type? Run `devdash alias-setup` to add a 'dd' shortcut.")
}
