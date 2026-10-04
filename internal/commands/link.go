package commands

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/devdashproject/devdash-cli/internal/config"
	"github.com/devdashproject/devdash-cli/internal/resolve"
	"github.com/spf13/cobra"
)

func newLinkCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "link [project]",
		Short: "Link a git repo to a devdash project",
		Long: `Link a git repository to a devdash project.

Non-interactive (scripts, agents): name the project, by ID, ID prefix,
or exact name. Nothing is prompted:
  devdash link 47eb046a              Link the repo root
  devdash link "My Project" --here   Link only the current directory
  devdash --project=47eb046a link    Same as passing the project

Interactive: with no project given, detects the GitHub remote and matches
it against your projects. If nothing matches, you pick a project or create
a new one. If input runs out before you choose, link fails rather than
guessing.

Writes a .devdash file at the repository root (or the current directory
with --here). It records the project ID and close_gate ("push": close
issues after git push). If .devdash already exists, link leaves it alone.`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}

			cwd := mustGetwd()

			// Guard: don't run from home or root
			if isHomeOrRoot(cwd) {
				fmt.Println("devdash link connects a git repo to an existing devdash project. If you're")
				fmt.Println("linking a repo, navigate to your project's top level directory and try again.")
				fmt.Println()
				fmt.Println("Working without a repo? No need to link! Just make sure to let your AI agent know")
				fmt.Println("which devdash project you'd like to work in.")
				return nil
			}

			// Detect git repo
			repoRoot, err := gitRepoRoot()
			if err != nil {
				fmt.Println("No git repository detected.")
				fmt.Println()
				fmt.Println("devdash link connects a git repo to an existing devdash project. If you're")
				fmt.Println("linking a repo, navigate to your project's top level directory and try again.")
				fmt.Println()
				fmt.Println("Working without a repo? No need to link! Just make sure to let your AI agent know")
				fmt.Println("which devdash project you'd like to work in.")
				return nil
			}

			target := ""
			if len(args) > 0 {
				target = args[0]
			} else if p, _ := cmd.Root().PersistentFlags().GetString("project"); p != "" {
				target = p
			}
			here, _ := cmd.Flags().GetBool("here")
			// One reader for all prompts: separate scanners would each buffer
			// ahead and lose piped answers meant for later prompts.
			in := bufio.NewReader(cmd.InOrStdin())

			writeDir := repoRoot
			if here {
				writeDir = cwd
			} else if target == "" && !samePath(cwd, repoRoot) {
				// Scope selection: interactive only, when not at the repo root
				if repoName := detectGitRepo(); repoName != "" {
					fmt.Printf("Detected git repo: github.com/%s  (root: %s)\n", repoName, repoRoot)
				} else {
					fmt.Printf("Detected git repo at %s (no GitHub remote)\n", repoRoot)
				}
				fmt.Printf("Current directory:  %s\n\n", cwd)
				fmt.Println("Link the whole repo or just this directory?")
				fmt.Printf("  1. Whole repo  %s\n", repoRoot)
				fmt.Printf("  2. This directory  %s\n\n", cwd)
				fmt.Print("Select [1-2]: ")

				choice, err := readChoiceFrom(in, 2)
				if err != nil {
					return err
				}
				if choice == 2 {
					writeDir = cwd
				}
				fmt.Println()
			}

			// Check if already linked
			devdashPath := filepath.Join(writeDir, config.ProjectFileName)
			if _, err := os.Stat(devdashPath); err == nil {
				fmt.Println("This directory is already linked. Run `devdash ready` to see open issues.")
				return nil
			}

			// Fetch projects
			data, err := d.Client.Get("/projects")
			if err != nil {
				return fmt.Errorf("failed to fetch projects: %w", err)
			}

			var projects []api.Project
			if err := json.Unmarshal(data, &projects); err != nil {
				return fmt.Errorf("invalid projects response: %w", err)
			}

			repoName := detectGitRepo()
			var matched *api.Project
			if target != "" {
				p, err := resolve.ProjectInList(target, projects)
				if err != nil {
					return err
				}
				matched = &p
			} else if repoName != "" {
				// Try auto-match on the GitHub remote
				for i, p := range projects {
					if strings.EqualFold(p.GithubRepo, repoName) {
						matched = &projects[i]
						break
					}
				}
			}

			var projectID string
			if matched != nil {
				if target != "" {
					fmt.Printf("Linking to project: %s (%s)\n", matched.Name, matched.ID)
				} else {
					fmt.Printf("Found a matching project: %s\n\n", matched.Name)
				}
				projectID = matched.ID
			} else {
				// No match, show list
				fmt.Println("No matching devdash project found.")
				fmt.Println()
				for i, p := range projects {
					repo := ""
					if p.GithubRepo != "" {
						repo = fmt.Sprintf(" (%s)", p.GithubRepo)
					}
					fmt.Printf("  %d. %s%s\n", i+1, p.Name, repo)
				}
				fmt.Printf("  %d. Create new project\n\n", len(projects)+1)
				fmt.Print("Select [1-" + fmt.Sprintf("%d", len(projects)+1) + "]: ")

				choice, err := readChoiceFrom(in, len(projects)+1)
				if err != nil {
					return err
				}
				if choice <= len(projects) {
					projectID = projects[choice-1].ID
					fmt.Printf("Linked to \"%s\".\n", projects[choice-1].Name)
				} else {
					// Create new
					defaultName := filepath.Base(repoRoot)
					if writeDir != repoRoot {
						defaultName = filepath.Base(writeDir)
					}
					fmt.Printf("\nProject name [%s]: ", defaultName)
					name := readLineFrom(in, defaultName)

					reqBody := map[string]string{"name": name}
					if repoName != "" {
						reqBody["githubRepo"] = repoName
					}

					data, err := d.Client.Post("/projects", reqBody)
					if err != nil {
						return fmt.Errorf("failed to create project: %w", err)
					}

					var newProject api.Project
					if err := json.Unmarshal(data, &newProject); err != nil {
						return fmt.Errorf("invalid project response: %w", err)
					}
					projectID = newProject.ID
					fmt.Printf("Created \"%s\" and linked it to this repo.\n", newProject.Name)
				}
			}

			// The repository file may be committed and shared. Keep the
			// credentialed API endpoint in user-owned settings or DD_API_URL.
			pf := config.ProjectFile{
				ProjectID: projectID,
				CloseGate: config.DefaultCloseGate,
			}
			if d.Cfg.FrontendURL != config.DefaultFrontendURL {
				pf.FrontendURL = d.Cfg.FrontendURL
			}

			data, _ = json.MarshalIndent(pf, "", "  ")
			if err := os.WriteFile(devdashPath, append(data, '\n'), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", devdashPath, err)
			}

			fmt.Printf("Wrote %s\n", devdashPath)
			fmt.Println()
			fmt.Println("Next: run `devdash agent-setup` to configure your AI agent, or `devdash create` to add your first issue.")
			return nil
		},
	}
	cmd.Flags().Bool("here", false, "Link the current directory instead of the repo root")
	return cmd
}

// samePath compares directories after resolving symlinks (e.g. /tmp vs /private/tmp).
func samePath(a, b string) bool {
	if ra, err := filepath.EvalSymlinks(a); err == nil {
		a = ra
	}
	if rb, err := filepath.EvalSymlinks(b); err == nil {
		b = rb
	}
	return filepath.Clean(a) == filepath.Clean(b)
}

func isHomeOrRoot(dir string) bool {
	if dir == "/" {
		return true
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false
	}
	return dir == home
}

func gitRepoRoot() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func isInsideGitRepo() bool {
	_, err := exec.Command("git", "rev-parse", "--git-dir").Output()
	return err == nil
}

func detectGitRepo() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))
	url = strings.TrimSuffix(url, ".git")
	if idx := strings.Index(url, "github.com"); idx >= 0 {
		path := url[idx+len("github.com"):]
		path = strings.TrimPrefix(path, ":")
		path = strings.TrimPrefix(path, "/")
		return path
	}
	return ""
}

func mustGetwd() string {
	dir, _ := os.Getwd()
	if dir == "" {
		return "."
	}
	return dir
}

// readChoiceFrom reads a 1..maxChoice selection. It fails when input runs out
// rather than defaulting, so a non-interactive run can't silently pick option 1.
func readChoiceFrom(in *bufio.Reader, maxChoice int) (int, error) {
	for {
		line, err := in.ReadString('\n')
		if err != nil && line == "" {
			break
		}
		text := strings.TrimSpace(line)
		var choice int
		if _, err := fmt.Sscanf(text, "%d", &choice); err != nil || choice < 1 || choice > maxChoice {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d: ", maxChoice)
			continue
		}
		return choice, nil
	}
	fmt.Println()
	return 0, fmt.Errorf("no selection made (input ended). To link without prompts: devdash link <project-id-or-name> [--here]")
}

func readLineFrom(in *bufio.Reader, defaultVal string) string {
	line, _ := in.ReadString('\n')
	if text := strings.TrimSpace(line); text != "" {
		return text
	}
	return defaultVal
}
