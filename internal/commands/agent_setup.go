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
	if _, err := os.Stat(".github/copilot-instructions.md"); err == nil {
		agents = append(agents, "copilot")
	}
	return agents
}

func setupAgent(agent, pid, closeOn string, force bool) error {
	switch agent {
	case "claude":
		return setupClaude(pid, closeOn, force)
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
