package commands

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strings"
	"time"

	"github.com/devdashproject/devdash-cli/internal/api"
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

Pass --no-browser to print the auth URL instead of launching a browser.
The browser must still run on this machine: it calls back to localhost.
The command waits up to 120 seconds for the callback before timing out.

No browser (CI, SSH, coding agents)? Use an existing API token instead:
  devdash login --token=dd_...              Verify and save the token
  echo "$TOKEN" | devdash login --with-token  Same, read from stdin
  export DEVDASH_TOKEN=dd_...               Use it without saving (no login needed)
Create a token with 'devdash token create' on a logged-in machine, or in
the web app under Settings. See 'devdash help auth'.`,
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) > 0 {
				return fmt.Errorf("unexpected argument %q — to log in with an API token use: devdash login --token=dd_... (or pipe it to: devdash login --with-token)", maskSecrets(args[0]))
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if d.Cfg == nil {
				var err error
				d.Cfg, err = config.Load()
				if err != nil {
					return err
				}
			}

			token, _ := cmd.Flags().GetString("token")
			withToken, _ := cmd.Flags().GetBool("with-token")
			if cmd.Flags().Changed("token") && strings.TrimSpace(token) == "" {
				return fmt.Errorf("--token needs a value, e.g. devdash login --token=<your dd_ token>")
			}
			if withToken {
				if token != "" {
					return fmt.Errorf("use either --token or --with-token, not both")
				}
				var err error
				token, err = readTokenFromStdin(cmd.InOrStdin())
				if err != nil {
					return err
				}
			}
			if token = strings.TrimSpace(token); token != "" {
				return loginWithToken(d.Cfg, token)
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
			fmt.Println("(No browser on this machine? Press Ctrl+C and use: devdash login --token=dd_...)")

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
	cmd.Flags().String("token", "", "Save an existing API token instead of using the browser (verified first)")
	cmd.Flags().Bool("with-token", false, "Read an API token from stdin instead of using the browser")
	return cmd
}

// loginWithToken verifies a token against the API, then saves it.
func loginWithToken(cfg *config.Config, token string) error {
	client := api.New(cfg.APIURL, token, Version)
	if _, err := client.Get("/projects"); err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) && (apiErr.StatusCode == 401 || apiErr.StatusCode == 403) {
			return fmt.Errorf("token rejected by %s (HTTP %d); nothing was saved. Check the token, or create a new one with 'devdash token create' or in the web app under Settings", cfg.APIURL, apiErr.StatusCode)
		}
		return fmt.Errorf("could not verify token against %s; nothing was saved: %w", cfg.APIURL, err)
	}
	if err := cfg.SaveToken(token); err != nil {
		return fmt.Errorf("failed to save token: %w", err)
	}
	fmt.Printf("Logged in (token verified). Token saved to %s\n", cfg.TokenFilePath())
	if os.Getenv(config.TokenEnvVar) != "" {
		fmt.Printf("Note: %s is set and takes precedence over the saved token.\n", config.TokenEnvVar)
	}
	printLoginBreadcrumbs()
	return nil
}

// readTokenFromStdin reads one line. On a terminal it prompts and hides input.
func readTokenFromStdin(in io.Reader) (string, error) {
	if f, ok := in.(*os.File); ok && isTerminal(f) {
		fmt.Fprint(os.Stderr, "Paste API token: ")
		if restore := disableEcho(f); restore != nil {
			defer func() {
				restore()
				fmt.Fprintln(os.Stderr)
			}()
		}
	}
	line, err := bufio.NewReader(in).ReadString('\n')
	if err != nil && err != io.EOF {
		return "", fmt.Errorf("failed to read token from stdin: %w", err)
	}
	token := strings.TrimSpace(line)
	if token == "" {
		return "", fmt.Errorf("no token read from stdin (usage: echo \"$TOKEN\" | devdash login --with-token)")
	}
	return token, nil
}

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

// disableEcho turns off terminal echo via stty; returns a restore func, or nil if unsupported.
func disableEcho(f *os.File) func() {
	if runtime.GOOS == "windows" {
		return nil
	}
	off := exec.Command("stty", "-echo")
	off.Stdin = f
	if off.Run() != nil {
		return nil
	}
	return func() {
		on := exec.Command("stty", "echo")
		on.Stdin = f
		_ = on.Run()
	}
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
	if !samePath(cwd, repoRoot) {
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

var tokenPattern = regexp.MustCompile(`dd_[A-Za-z0-9]{6,}`)

// maskSecrets hides anything that looks like an API token before echoing user input.
func maskSecrets(s string) string {
	return tokenPattern.ReplaceAllString(s, "dd_****")
}
