package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/jasonmassey/devdash-cli-go/internal/api"
	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(primeCmd)
}

var primeCmd = &cobra.Command{
	Use:   "prime",
	Short: "Output AI-optimized workflow context for agent injection",
	RunE: func(cmd *cobra.Command, args []string) error {
		pid, err := requireProject()
		if err != nil {
			return err
		}

		// Fetch project details
		projData, err := client.Get("/projects/" + pid)
		if err != nil {
			return fmt.Errorf("failed to fetch project: %w", err)
		}

		var project api.Project
		json.Unmarshal(projData, &project)

		// Fetch all projects for the listing
		allProjData, err := client.Get("/projects")
		if err != nil {
			return fmt.Errorf("failed to fetch projects: %w", err)
		}

		var allProjects []api.Project
		json.Unmarshal(allProjData, &allProjects)

		// Fetch beads
		beads, err := api.FetchAll[api.Bead](client, "/beads?projectId="+pid)
		if err != nil {
			return fmt.Errorf("failed to fetch beads: %w", err)
		}

		// Fetch team
		teamData, _ := client.Get("/projects/" + pid + "/members?format=compact")
		var members []api.TeamMember
		json.Unmarshal(teamData, &members)

		// Count stats
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

		// Build output
		fmt.Println("# Dev-Dash Workflow Context")
		fmt.Println()
		fmt.Println("> **Context Recovery**: Run `devdash prime` after compaction, clear, or new session.")
		fmt.Println("> Use `devdash` (dev-dash CLI) for ALL task tracking — never `bd`.")
		fmt.Println()

		// Project info
		repo := ""
		if project.GithubRepo != "" {
			repo = fmt.Sprintf(" (%s)", project.GithubRepo)
		}
		fmt.Printf("**Project**: %s%s  |  **ID**: `%s`\n", project.Name, repo, shortID(project.ID))
		fmt.Printf("**Health**: %d open, %d in progress, %d blocked\n", open, inProgress, blocked)
		fmt.Println()

		// Team
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

		// All projects
		fmt.Println("## All Projects")
		fmt.Println()
		fmt.Println("| Short | Full UUID | Name | Repo |")
		fmt.Println("|-------|-----------|------|------|")
		for _, p := range allProjects {
			repo := ""
			if p.GithubRepo != "" {
				repo = fmt.Sprintf("(%s)", p.GithubRepo)
			} else {
				repo = "(no repo)"
			}
			fmt.Printf("| `%s` | `%s` | %s | %s |\n", shortID(p.ID), p.ID, p.Name, repo)
		}
		fmt.Println()
		fmt.Println("Use `DD_PROJECT_ID=<full-uuid> devdash <command>` to target a specific project.")
		fmt.Println()

		// Rules
		fmt.Println("## Rules (MANDATORY)")
		fmt.Println("- **Issue-first**: Create a devdash issue BEFORE writing code. No exceptions.")
		fmt.Println("- **Issue-per-commit**: Every git commit must map to a devdash issue. If scope expands, create new issues.")
		fmt.Println("- **Mark in-progress**: `devdash update <id> --status=in_progress` before starting work.")
		fmt.Println("- **Pre-commit checkpoint**: Before each `git commit`, verify you have an issue. If not, create one.")
		fmt.Println("- **Close after push**: Only close issues after `git push` succeeds — never before.")
		fmt.Println("- **No orphaned work**: At session end, every commit must map to a closed issue.")
		fmt.Printf("- **Path**: `%s`\n", whichDevdash())
		fmt.Println("- **Preserve stderr**: Avoid `2>/dev/null` on devdash commands — stderr contains error details you'll need for debugging failures.")
		fmt.Println("- **Prohibited**: Do NOT use `bd`, TodoWrite, TaskCreate, or markdown files for task tracking")
		fmt.Println()

		// Quick reference
		fmt.Println("## Quick Reference")
		fmt.Println("- **Start**: `devdash ready` → `devdash show <id>` → `devdash update <id> --status=in_progress`")
		fmt.Println("- **Complete**: git add → commit → push → `devdash close <id> --summary=\"...\" --commit=$(git rev-parse HEAD)`")
		fmt.Println("- **Report**: `devdash report <id> --status=code_complete|committed|pushed|error --summary=\"...\"`")
		fmt.Println("- Close summaries are institutional memory — include what, why, decisions, surprises, follow-ups.")
		fmt.Println("- One issue per commit. Scope creep = new issue. Multi-step = parent + children.")
		fmt.Println()

		// Help topics
		fmt.Println("## On-Demand Reference")
		fmt.Println("Run these when you need detailed guidance:")
		fmt.Println("- `devdash help cli` — Full command reference (flags, ID formats, --since syntax)")
		fmt.Println("- `devdash help workflow` — When to create issues, decomposition patterns, bead relationships")
		fmt.Println("- `devdash help close` — Close summary expectations with examples")
		fmt.Println("- `devdash help pr` — PR footer format and multi-issue PRs")
		fmt.Println("- `devdash help projects` — Cross-project dependencies and multi-repo work")
		fmt.Println("- `devdash help report` — Progress reporting cadence and status values")

		return nil
	},
}

func whichDevdash() string {
	exe, err := os.Executable()
	if err != nil {
		return "devdash"
	}
	return exe
}
