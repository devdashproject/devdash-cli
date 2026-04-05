package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/jasonmassey/devdash-cli-go/internal/config"
	"github.com/spf13/cobra"
)

func newInitCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Initialize devdash in the current repository",
		Long: `Initialize devdash in the current Git repository.

Detects the GitHub remote from git config and attempts to match it against
your existing devdash projects. If a match is found, it links automatically;
otherwise you are prompted to select an existing project or create a new one.

Writes a .devdash configuration file in the repository root. If .devdash
already exists, the command exits early without overwriting it.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}

			if _, err := os.Stat(config.ProjectFileName); err == nil {
				fmt.Println(".devdash already exists in this directory.")
				return nil
			}

			repoName := detectGitRepo()

			data, err := d.Client.Get("/projects")
			if err != nil {
				return fmt.Errorf("failed to fetch projects: %w", err)
			}

			var projects []api.Project
			_ = json.Unmarshal(data, &projects)

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
				fmt.Printf("Matched project: %s (%s)\n", matched.Name, matched.ID)
				projectID = matched.ID
			} else {
				fmt.Println("Available projects:")
				for i, p := range projects {
					repo := ""
					if p.GithubRepo != "" {
						repo = fmt.Sprintf(" (%s)", p.GithubRepo)
					}
					fmt.Printf("  %d. %s%s\n", i+1, p.Name, repo)
				}
				fmt.Printf("  %d. Create new project\n", len(projects)+1)

				fmt.Print("\nSelect project number: ")
				var choice int
				if _, err := fmt.Scan(&choice); err != nil || choice < 1 || choice > len(projects)+1 {
					return fmt.Errorf("invalid selection")
				}

				if choice <= len(projects) {
					projectID = projects[choice-1].ID
				} else {
					name := filepath.Base(mustGetwd())
					fmt.Printf("Project name [%s]: ", name)
					var input string
					_, _ = fmt.Scanln(&input)
					if input != "" {
						name = input
					}

					reqBody := map[string]string{"name": name}
					if repoName != "" {
						reqBody["githubRepo"] = repoName
					}

					data, err := d.Client.Post("/projects", reqBody)
					if err != nil {
						return fmt.Errorf("failed to create project: %w", err)
					}

					var newProject api.Project
					_ = json.Unmarshal(data, &newProject)
					projectID = newProject.ID
					fmt.Printf("Created project: %s (%s)\n", newProject.Name, newProject.ID)
				}
			}

			pf := config.ProjectFile{
				ProjectID:   projectID,
				APIURL:      d.Cfg.APIURL,
				FrontendURL: d.Cfg.FrontendURL,
				CloseGate:   config.DefaultCloseGate,
			}

			data, _ = json.MarshalIndent(pf, "", "  ")
			if err := os.WriteFile(config.ProjectFileName, append(data, '\n'), 0644); err != nil {
				return fmt.Errorf("failed to write %s: %w", config.ProjectFileName, err)
			}

			fmt.Printf("Initialized: %s\n", config.ProjectFileName)
			return nil
		},
	}
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
