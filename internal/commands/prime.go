package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/devdashproject/devdash-cli/internal/api"
	"github.com/spf13/cobra"
)

func newPrimeCmd(d *Deps) *cobra.Command {
	return &cobra.Command{
		Use:   "prime",
		Short: "Output AI-optimized workflow context for agent injection",
		Long: `Output a full AI-optimized workflow context block designed for agent injection.

The output includes the current project info, team roster, all known projects,
health stats (open/in-progress/blocked counts), mandatory rules, a quick
reference for common workflows, and on-demand help topics.

Run this at the start of every new session, after context compaction, or
whenever the agent has lost track of project state. The output is formatted
as Markdown so it can be injected directly into an agent's context window.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			projData, _ := d.Client.Get("/projects/" + pid)
			var project api.Project
			_ = json.Unmarshal(projData, &project)

			allProjData, _ := d.Client.Get("/projects")
			var allProjects []api.Project
			_ = json.Unmarshal(allProjData, &allProjects)

			beads, _ := api.FetchAll[api.Bead](d.Client, "/beads?projectId="+pid)

			teamData, _ := d.Client.Get("/projects/" + pid + "/members?format=compact")
			var members []api.TeamMember
			_ = json.Unmarshal(teamData, &members)

			var open, inProgress, blocked int
			completedIDs := make(map[string]bool)
			for _, b := range beads {
				switch b.Status {
				case "pending":
					open++
				case "in_progress":
					inProgress++
				case "completed":
					completedIDs[b.ID] = true
				}
			}
			for _, b := range beads {
				if b.Status == "pending" && len(b.BlockedBy) > 0 && isBlocked(b, completedIDs) {
					blocked++
				}
			}

			fmt.Println("# Dev-Dash Workflow Context")
			fmt.Println()
			fmt.Println("> **Context Recovery**: Run `devdash prime` after compaction, clear, or new session.")
			fmt.Println()

			repo := ""
			if project.GithubRepo != "" {
				repo = fmt.Sprintf(" (%s)", project.GithubRepo)
			}
			fmt.Printf("**Project**: %s%s  |  **ID**: `%s`\n", project.Name, repo, shortID(project.ID))
			fmt.Printf("**Health**: %d open, %d in progress, %d blocked\n", open, inProgress, blocked)
			fmt.Println()

			if len(members) > 0 {
				fmt.Println("## Team")
				fmt.Println()
				fmt.Println("| Name | Username | Email | Role |")
				fmt.Println("|------|----------|-------|------|")
				cap := 20
				if len(members) < cap {
					cap = len(members)
				}
				for _, m := range members[:cap] {
					name := m.Name
					if name == "" {
						name = m.Email
					}
					username := m.Username
					if username != "" {
						username = "@" + username
					}
					fmt.Printf("| %s | %s | %s | %s |\n", name, username, m.Email, m.Role)
				}
				fmt.Println()
				fmt.Println("Assign with: `devdash update <id> --owner=<email-or-name>`")
				fmt.Println()
			}

			fmt.Println("## All Projects")
			fmt.Println()
			fmt.Println("| Short | Full UUID | Name | Repo |")
			fmt.Println("|-------|-----------|------|------|")
			for _, p := range allProjects {
				r := "(no repo)"
				if p.GithubRepo != "" {
					r = fmt.Sprintf("(%s)", p.GithubRepo)
				}
				fmt.Printf("| `%s` | `%s` | %s | %s |\n", shortID(p.ID), p.ID, p.Name, r)
			}
			fmt.Println()
			fmt.Println("Use `DD_PROJECT_ID=<full-uuid> devdash <command>` to target a specific project.")
			fmt.Println("Short prefixes work for project IDs too: `DD_PROJECT_ID=47eb046a devdash ready`")
			fmt.Println()

			exe, _ := os.Executable()
			fmt.Printf("**CLI Path**: `%s`\n", exe)
			fmt.Println()

			fmt.Println("## Output Formats")
			fmt.Println("- There is no `--json` flag. Do not try `--json`, `--format`, or `--output`.")
			fmt.Println("- These commands already return JSON: `show`, `find`, `activity`, `comments`, `token list`")
			fmt.Println("- All other commands return human-readable text. Parse UUIDs from `create`/`close`/`update` output directly.")
			fmt.Println()

			fmt.Println("## On-Demand Reference")
			fmt.Println("Run these when you need detailed guidance:")
			fmt.Println("- `devdash help cli` — Full command reference (flags, ID formats, --since syntax)")
			fmt.Println("- `devdash help workflow` — When to create issues, decomposition patterns, bead relationships")
			fmt.Println("- `devdash help close` — Close summary expectations with examples")
			fmt.Println("- `devdash help pr` — PR footer format and multi-issue PRs")
			fmt.Println("- `devdash help projects` — Cross-project dependencies and multi-repo work")

			return nil
		},
	}
}
