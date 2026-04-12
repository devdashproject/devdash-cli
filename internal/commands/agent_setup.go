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
		Long: `Configure AI agent instruction files for the current repository so that
agents automatically follow the devdash workflow.

By default the command auto-detects which agents are present (looking for
CLAUDE.md, AGENTS.md, .github/copilot-instructions.md) and writes devdash
instructions into each. Use --agent to target specific agents by name, and
--force to overwrite configs that already contain devdash instructions.

The --close-on flag controls the workflow gate: set it to "commit" or "push"
to determine when agents are allowed to close issues.`,
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
	case "copilot":
		return setupCopilot(pid, closeOn, force)
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

	instructions := fmt.Sprintf(`<!-- devdash:agent-instructions -->

# DevDash — AI Agent Task Tracking

This project uses **devdash** (`+"`dd`"+`) for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself via the terminal — do not just tell the user to run them.
Do NOT use TodoWrite, TaskCreate, `+"`bd`"+`, or markdown files for tracking. When the user
says "dd", they mean the devdash CLI, not the Unix `+"`dd`"+` data-copy utility.

Issues are called "beads" internally. You'll see this in fields like `+"`parentBeadId`"+`.

Project ID: %s

## Core Principles

**Be a capture reflex.**
When the user mentions a bug, idea, TODO, or "we should probably..." — offer to create
an issue. Don't wait to be asked.

**Issue-first.**
Create an issue before doing work. Your first action when asked to implement something
should be `+"`devdash create`"+`.

**One issue per logical unit of work.**
If a task has multiple steps, create a parent issue and child issues. Scope creep during
a task = new issue, not an expanded current one. Every git commit must map to a devdash
issue.

## Rules

1. Create an issue before starting work. No exceptions.
2. `+"`devdash update <id> --status=in_progress`"+` before starting work on an issue.
3. Close with a substantive summary — write it for a future reader with zero context.
4. Don't batch unrelated work into a single issue.
5. **Close after %s**: Only close issues after `+"`git %s`"+` succeeds — never before.
6. No orphaned work: at session end, every commit must map to a closed issue.
7. Git operations MUST succeed before closing. Never run git and devdash close in parallel.
8. Preserve stderr: avoid `+"`2>/dev/null`"+` on devdash commands.

## Completing Work

`+"`git add`"+` → `+"`git commit`"+` → `+"`git %s`"+` → `+"`devdash close <id>`"+`
On successful completions: `+"`devdash close <id> --summary=\"...\" --commit=$(git rev-parse HEAD)`"+`.
If a PR exists, include `+"`--pr=URL`"+` too.
Close summaries are institutional memory — include what, why, decisions, surprises, follow-ups.
One issue per commit. Scope creep = new issue. Multi-step = parent + children.

## On-Demand Reference

Run these when you need detailed guidance:
- `+"`devdash help cli`"+` — Full command reference (flags, ID formats, --since syntax)
- `+"`devdash help workflow`"+` — When to create issues, decomposition patterns, bead relationships
- `+"`devdash help close`"+` — Close summary expectations with examples
- `+"`devdash help pr`"+` — PR footer format and multi-issue PRs
- `+"`devdash help projects`"+` — Cross-project dependencies and multi-repo work

## Agent-Specific Instructions

- You may use your built-in task tools (TaskCreate, TodoWrite, etc.) for your own tracking, but you **must also** create and update devdash issues. Devdash is the system of record.
- When the user asks you to implement a plan, feature, or fix: your **very first action** is `+"`devdash create`"+`. Do not read files, do not write code — create the issue first.
- For multi-step plans, create one devdash issue per step before starting any implementation. Group them under a parent issue. Then work through them sequentially: mark in-progress, implement, commit, close, move to next.
- After creating issues, follow the normal workflow: mark in-progress, do the work, commit, then close.
<!-- /devdash:agent-instructions -->
`, pid, closeOn, closeOn, closeOn)

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

func setupCopilot(pid, closeOn string, force bool) error {
	target := ".github/copilot-instructions.md"
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

This project uses **devdash** for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself via the terminal — do not just tell the user to run them.
Do NOT use markdown files or other tools for task tracking. When the user
says "dd", they mean the devdash CLI, not the Unix `+"`dd`"+` data-copy utility.

Issues are called "beads" internally. You'll see this in fields like `+"`parentBeadId`"+`.

Project ID: %s

## Core Principles

**Be a capture reflex.**
When the user mentions a bug, idea, TODO, or "we should probably..." — offer to create
an issue. Don't wait to be asked.

**Issue-first.**
Create an issue before doing work. Your first action when asked to implement something
should be `+"`devdash create`"+`.

**One issue per logical unit of work.**
If a task has multiple steps, create a parent issue and child issues. Scope creep during
a task = new issue, not an expanded current one. Every git commit must map to a devdash
issue.

## Rules

1. Create an issue before starting work. No exceptions.
2. `+"`devdash update <id> --status=in_progress`"+` before starting work on an issue.
3. Close with a substantive summary — write it for a future reader with zero context.
4. Don't batch unrelated work into a single issue.
5. **Close after %s**: Only close issues after `+"`git %s`"+` succeeds — never before.
6. No orphaned work: at session end, every commit must map to a closed issue.
7. Git operations MUST succeed before closing. Never run git and devdash close in parallel.
8. Preserve stderr: avoid `+"`2>/dev/null`"+` on devdash commands.

## Completing Work

`+"`git add`"+` → `+"`git commit`"+` → `+"`git %s`"+` → `+"`devdash close <id>`"+`
On successful completions: `+"`devdash close <id> --summary=\"...\" --commit=$(git rev-parse HEAD)`"+`.
If a PR exists, include `+"`--pr=URL`"+` too.
Close summaries are institutional memory — include what, why, decisions, surprises, follow-ups.
One issue per commit. Scope creep = new issue. Multi-step = parent + children.

## On-Demand Reference

Run these when you need detailed guidance:
- `+"`devdash help cli`"+` — Full command reference (flags, ID formats, --since syntax)
- `+"`devdash help workflow`"+` — When to create issues, decomposition patterns, bead relationships
- `+"`devdash help close`"+` — Close summary expectations with examples
- `+"`devdash help pr`"+` — PR footer format and multi-issue PRs
- `+"`devdash help projects`"+` — Cross-project dependencies and multi-repo work
`, pid, closeOn, closeOn, closeOn)

	if err := os.MkdirAll(".github", 0755); err != nil {
		return err
	}

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
