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
	"github.com/spf13/cobra"
)

func newLinkCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "link",
		Short: "Link a git repo to a devdash project",
		Long: `Link a git repository to a devdash project.

Detects the GitHub remote from git config and attempts to match it against
your existing devdash projects. If a match is found, it links automatically;
otherwise you are prompted to select an existing project or create a new one.

Writes a .devdash configuration file at the repository root (or the directory
you specify). If .devdash already exists, the command exits without overwriting it.`,
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

			writeDir := repoRoot

			// Scope selection: if not at repo root, ask
			if cwd != repoRoot {
				repoName := detectGitRepo()
				if repoName == "" {
					repoName = filepath.Base(repoRoot)
				}

				fmt.Printf("Detected git repo: github.com/%s  (root: %s)\n", repoName, repoRoot)
				fmt.Printf("Current directory:  %s\n\n", cwd)
				fmt.Println("Link the whole repo or just this directory?")
				fmt.Printf("  1. Whole repo  %s\n", repoRoot)
				fmt.Printf("  2. This directory  %s\n\n", cwd)
				fmt.Print("Select [1-2]: ")

				choice := readChoice(2)
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

			// Try auto-match
			repoName := detectGitRepo()
			var matched *api.Project
			if repoName != "" {
				for i, p := range projects {
					if strings.EqualFold(p.GithubRepo, repoName) {
						matched = &projects[i]
						break
					}
				}
			}

			var projectID string
			if matched != nil {
				fmt.Printf("Found a matching project: %s\n\n", matched.Name)
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

				choice := readChoice(len(projects) + 1)
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
					name := readLine(defaultName)

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

			// Write .devdash
			pf := config.ProjectFile{
				ProjectID:   projectID,
				APIURL:      d.Cfg.APIURL,
				FrontendURL: d.Cfg.FrontendURL,
				CloseGate:   config.DefaultCloseGate,
			}

			data, _ = json.MarshalIndent(pf, "", "  ")
			if err := os.WriteFile(devdashPath, append(data, '\n'), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", devdashPath, err)
			}

			fmt.Println("Wrote .devdash")
			fmt.Println()
			fmt.Println("Next: run `devdash agent-setup` to configure your AI agent, or `devdash create` to add your first issue.")
			return nil
		},
	}
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

func readChoice(maxChoice int) int {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		var choice int
		if _, err := fmt.Sscanf(text, "%d", &choice); err != nil || choice < 1 || choice > maxChoice {
			fmt.Printf("Invalid selection. Please enter a number between 1 and %d: ", maxChoice)
			continue
		}
		return choice
	}
	return 1
}

func readLine(defaultVal string) string {
	scanner := bufio.NewScanner(os.Stdin)
	if scanner.Scan() {
		text := strings.TrimSpace(scanner.Text())
		if text == "" {
			return defaultVal
		}
		return text
	}
	return defaultVal
}
