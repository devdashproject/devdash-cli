package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func newAgentSetupCmd(d *Deps) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "agent-setup",
		Short: "Configure agent instructions for this repository",
		RunE: func(cmd *cobra.Command, args []string) error {
			pid, err := d.requireProject(cmd)
			if err != nil {
				return err
			}

			force, _ := cmd.Flags().GetBool("force")
			closeOn, _ := cmd.Flags().GetString("close-on")
			agentFlag, _ := cmd.Flags().GetString("agent")
			allFlag, _ := cmd.Flags().GetBool("all")

			var agents []string
			if agentFlag != "" {
				agents = strings.Split(agentFlag, ",")
			} else if allFlag {
				agents = detectAgents()
			} else {
				agents = detectAgents()
				if len(agents) == 0 {
					agents = []string{"claude"}
				}
			}

			for _, agent := range agents {
				agent = strings.TrimSpace(agent)
				if err := setupAgent(agent, pid, closeOn, force); err != nil {
					fmt.Fprintf(os.Stderr, "Warning: %s setup failed: %v\n", agent, err)
				}
			}
			return nil
		},
	}
	cmd.Flags().String("agent", "", "Comma-separated agent names")
	cmd.Flags().Bool("all", false, "Setup all detected agents")
	cmd.Flags().Bool("force", false, "Overwrite existing configs")
	cmd.Flags().String("close-on", "push", "Workflow gate: commit or push")
	return cmd
}

func detectAgents() []string {
	var agents []string
	if _, err := os.Stat("CLAUDE.md"); err == nil {
		agents = append(agents, "claude")
	} else if _, err := os.Stat(".claude"); err == nil {
		agents = append(agents, "claude")
	}
	if _, err := os.Stat("AGENTS.md"); err == nil {
		agents = append(agents, "codex")
	}
	if _, err := os.Stat(".github/copilot-instructions.md"); err == nil {
		agents = append(agents, "copilot")
	}
	return agents
}

func setupAgent(agent, pid, closeOn string, force bool) error {
	switch agent {
	case "claude":
		return setupClaude(pid, closeOn, force)
	case "codex":
		return setupCodex(pid, closeOn, force)
	default:
		return fmt.Errorf("unsupported agent: %s", agent)
	}
}

func setupClaude(pid, closeOn string, force bool) error {
	target := "CLAUDE.md"
	if !force {
		if _, err := os.Stat(target); err == nil {
			data, _ := os.ReadFile(target)
			if strings.Contains(string(data), "devdash") {
				fmt.Printf("  %s already contains devdash instructions (use --force to overwrite)\n", target)
				return nil
			}
		}
	}

	instructions := fmt.Sprintf(`# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking.

## Rules
- Create a devdash issue BEFORE writing code
- Every git commit must map to a devdash issue
- Mark issues in_progress before starting work
- Close issues only after git %s succeeds
- Project ID: %s

## Workflow
devdash ready → devdash show <id> → devdash update <id> --status=in_progress
git add → git commit → git %s → devdash close <id> --summary="..." --commit=$(git rev-parse HEAD)
`, closeOn, pid, closeOn)

	var content []byte
	if existing, err := os.ReadFile(target); err == nil && !force {
		content = append(existing, []byte("\n\n"+instructions)...)
	} else {
		content = []byte(instructions)
	}

	if err := os.WriteFile(target, content, 0644); err != nil {
		return err
	}
	fmt.Printf("  ✓ %s configured for devdash\n", target)
	return nil
}

func setupCodex(pid, closeOn string, force bool) error {
	target := "AGENTS.md"
	if !force {
		if _, err := os.Stat(target); err == nil {
			data, _ := os.ReadFile(target)
			if strings.Contains(string(data), "devdash") {
				fmt.Printf("  %s already contains devdash instructions (use --force to overwrite)\n", target)
				return nil
			}
		}
	}

	instructions := fmt.Sprintf(`# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking. Project ID: %s

## Agent-Specific Instructions

- Run `+"`devdash prime`"+` at the start of every new session. Run it again after any handoff, compaction, or context-loss event.
- If the user already named a specific devdash issue, follow `+"`devdash prime`"+` with `+"`devdash show <id>`"+` and `+"`devdash update <id> --status=in_progress`"+` instead of `+"`devdash ready`"+`.
- Use `+"`devdash ready`"+` only when the user has not already chosen the task.
- For command discovery, prefer `+"`devdash help`"+` and topic help commands such as `+"`devdash help cli`"+` before probing subcommands with `+"`--help`"+`.
- Use `+"`devdash --help`"+` when you need command syntax, capabilities, or supported workflow and a topic help entry is not enough.
- Minimize redundant startup work after `+"`devdash prime`"+`. Don't run broad repo scans or repeated discovery commands unless the current request needs them.
- Read only the context needed for the current request. Prefer targeted repo reads over whole-repo exploration when the task is narrow.
- Avoid commands whose only purpose is to reconfirm information already present in the prompt, local instructions, or recent command output.
- Run devdash commands yourself in the terminal; do not ask the user to run them for you.
- Before making code changes, make sure a devdash issue exists and is marked `+"`in_progress`"+`.
- Before each commit, confirm that the commit maps to exactly one devdash issue.
- Report progress with `+"`devdash report`"+` at `+"`code_complete`"+`, `+"`committed`"+`, and `+"`pushed`"+`.
- Never close a devdash issue until `+"`git %s`"+` succeeds.
- Preserve existing user changes. Do not revert unrelated modifications or overwrite work you did not make.
- Run the narrowest verification that meaningfully covers the change, then summarize the result for the user.
`, pid, closeOn)

	var content []byte
	if existing, err := os.ReadFile(target); err == nil && !force {
		content = append(existing, []byte("\n\n"+instructions)...)
	} else {
		content = []byte(instructions)
	}

	if err := os.WriteFile(target, content, 0644); err != nil {
		return err
	}
	fmt.Printf("  ✓ %s configured for devdash\n", target)
	return nil
}
