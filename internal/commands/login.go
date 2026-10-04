package commands

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
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

Starts a local HTTP callback server, generates a one-time nonce and proof key,
and opens your default browser to the DevDash auth page. Once you approve
access, the CLI exchanges the one-time code and saves the token.

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
			if err := api.ValidateEndpoint(d.Cfg.APIURL); err != nil {
				return fmt.Errorf("invalid API endpoint: %w", err)
			}

			nonce, err := auth.GenerateNonce()
			if err != nil {
				return err
			}
			verifier, challenge, err := auth.GeneratePKCE()
			if err != nil {
				return err
			}

			port, resultCh, cleanup, err := auth.StartCallbackServer(nonce)
			if err != nil {
				return err
			}
			defer cleanup()

			authURL, err := cliAuthURL(d.Cfg.APIURL, port, nonce, challenge)
			if err != nil {
				return err
			}
			redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", port)

			noBrowser, _ := cmd.Flags().GetBool("no-browser")
			if noBrowser {
				fmt.Printf("Open this URL in your browser:\n%s\n", authURL.String())
			} else {
				fmt.Println("Opening browser for authentication...")
				if err := openBrowser(authURL.String()); err != nil {
					fmt.Printf("Could not open browser. Open this URL manually:\n%s\n", authURL.String())
				}
			}

			fmt.Println("Waiting for authentication (timeout: 120s)...")
			fmt.Println("(No browser on this machine? Press Ctrl+C and use: devdash login --token=dd_...)")

			select {
			case result := <-resultCh:
				if result.Error != nil {
					return fmt.Errorf("authentication failed: %w", result.Error)
				}
				token, err := exchangeLoginCode(d.Cfg.APIURL, result.Code, verifier, redirectURI, nonce)
				if err != nil {
					return fmt.Errorf("authentication failed: %w", err)
				}
				if err := d.Cfg.SaveToken(token); err != nil {
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

func cliAuthURL(apiURL string, port int, nonce, challenge string) (*url.URL, error) {
	if err := api.ValidateEndpoint(apiURL); err != nil {
		return nil, fmt.Errorf("invalid API endpoint: %w", err)
	}
	authURL, err := url.Parse(strings.TrimRight(apiURL, "/") + "/api/auth/cli-token")
	if err != nil {
		return nil, fmt.Errorf("invalid API endpoint: %w", err)
	}
	query := authURL.Query()
	query.Set("port", fmt.Sprint(port))
	query.Set("nonce", nonce)
	query.Set("code_challenge", challenge)
	authURL.RawQuery = query.Encode()
	return authURL, nil
}

// exchangeLoginCode sends the proof directly to the trusted API endpoint.
// The API client validates the endpoint and guards redirects before sending it.
func exchangeLoginCode(apiURL, code, verifier, redirectURI, nonce string) (string, error) {
	client := api.New(apiURL, "", Version)
	data, err := client.Post("/auth/cli/exchange", map[string]string{
		"code":          code,
		"code_verifier": verifier,
		"redirect_uri":  redirectURI,
		"nonce":         nonce,
	})
	if err != nil {
		var apiErr *api.APIError
		if errors.As(err, &apiErr) {
			return "", fmt.Errorf("code exchange rejected (HTTP %d); nothing was saved", apiErr.StatusCode)
		}
		return "", fmt.Errorf("code exchange failed; nothing was saved: %w", err)
	}
	var response struct {
		Token string `json:"token"`
	}
	if err := json.Unmarshal(data, &response); err != nil || !exchangedTokenPattern.MatchString(response.Token) {
		return "", fmt.Errorf("invalid code exchange response; nothing was saved")
	}
	return response.Token, nil
}

var exchangedTokenPattern = regexp.MustCompile(`^dd_[0-9a-f]{64}$`)

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

var tokenPattern = regexp.MustCompile(`dd_[A-Za-z0-9]{6,}`)

// maskSecrets hides anything that looks like an API token before echoing user input.
func maskSecrets(s string) string {
	return tokenPattern.ReplaceAllString(s, "dd_****")
}
