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

func init() {
	rootCmd.AddCommand(initCmd)
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize devdash in the current repository",
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := requireAuth(); err != nil {
			return err
		}

		// Check if .devdash already exists
		if _, err := os.Stat(config.ProjectFileName); err == nil {
			fmt.Println(".devdash already exists in this directory.")
			return nil
		}

		// Detect repo from git remote
		repoName := detectGitRepo()

		// Fetch projects
		data, err := client.Get("/projects")
		if err != nil {
			return fmt.Errorf("failed to fetch projects: %w", err)
		}

		var projects []api.Project
		if err := json.Unmarshal(data, &projects); err != nil {
			return fmt.Errorf("failed to parse projects: %w", err)
		}

		// Try to match by repo name
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
			// List projects for selection
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
				// Create new project
				name := filepath.Base(mustGetwd())
				fmt.Printf("Project name [%s]: ", name)
				var input string
				fmt.Scanln(&input)
				if input != "" {
					name = input
				}

				reqBody := map[string]string{"name": name}
				if repoName != "" {
					reqBody["githubRepo"] = repoName
				}

				data, err := client.Post("/projects", reqBody)
				if err != nil {
					return fmt.Errorf("failed to create project: %w", err)
				}

				var newProject api.Project
				if err := json.Unmarshal(data, &newProject); err != nil {
					return err
				}
				projectID = newProject.ID
				fmt.Printf("Created project: %s (%s)\n", newProject.Name, newProject.ID)
			}
		}

		// Write .devdash file
		pf := config.ProjectFile{
			ProjectID:   projectID,
			APIURL:      cfg.APIURL,
			FrontendURL: cfg.FrontendURL,
			CloseGate:   config.DefaultCloseGate,
		}

		data, err = json.MarshalIndent(pf, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(config.ProjectFileName, append(data, '\n'), 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", config.ProjectFileName, err)
		}

		fmt.Printf("Initialized: %s\n", config.ProjectFileName)
		return nil
	},
}

func detectGitRepo() string {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return ""
	}
	url := strings.TrimSpace(string(out))

	// Parse owner/repo from git URL
	// Handles: git@github.com:owner/repo.git, https://github.com/owner/repo.git
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
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
