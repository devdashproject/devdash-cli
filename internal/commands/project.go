package commands

import (
	"encoding/json"
	"fmt"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/spf13/cobra"
)

func newProjectCmd(d *Deps) *cobra.Command {
	projectCmd := &cobra.Command{Use: "project", Short: "Manage projects"}

	createCmd := &cobra.Command{
		Use: "create", Short: "Create a new project",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			name, _ := cmd.Flags().GetString("name")
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			body := map[string]string{"name": name}
			if repo, _ := cmd.Flags().GetString("repo"); repo != "" {
				body["githubRepo"] = repo
			}
			if desc, _ := cmd.Flags().GetString("description"); desc != "" {
				body["description"] = desc
			}
			data, err := d.Client.Post("/projects", body)
			if err != nil {
				return err
			}
			var project api.Project
			_ = json.Unmarshal(data, &project)
			fmt.Printf("Created project %s: %s\n", project.ID, project.Name)
			return nil
		},
	}
	createCmd.Flags().String("name", "", "Project name (required)")
	createCmd.Flags().String("repo", "", "GitHub repo (owner/repo format)")
	createCmd.Flags().String("description", "", "Project description")
	projectCmd.AddCommand(createCmd)

	projectCmd.AddCommand(&cobra.Command{
		Use: "list", Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			data, err := d.Client.Get("/projects")
			if err != nil {
				return err
			}
			var projects []api.Project
			_ = json.Unmarshal(data, &projects)
			for _, p := range projects {
				repo := ""
				if p.GithubRepo != "" {
					repo = fmt.Sprintf(" (%s)", p.GithubRepo)
				}
				fmt.Printf("%s  %s%s\n", shortID(p.ID), p.Name, repo)
			}
			return nil
		},
	})

	deleteCmd := &cobra.Command{
		Use: "delete <project-id>", Short: "Delete a project", Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := d.requireAuth(); err != nil {
				return err
			}
			_, err := d.Client.Delete("/projects/" + args[0])
			if err != nil {
				return err
			}
			fmt.Printf("Deleted project: %s\n", args[0])
			return nil
		},
	}
	deleteCmd.Flags().BoolP("force", "f", false, "Skip confirmation")
	projectCmd.AddCommand(deleteCmd)

	return projectCmd
}
