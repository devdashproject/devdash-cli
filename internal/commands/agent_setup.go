package commands

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	managedBlockStart = "<!-- devdash:agent-instructions -->"
	managedBlockEnd   = "<!-- /devdash:agent-instructions -->"
)

// writeManagedInstructions wraps instructions in sentinel markers and writes
// them to target. Behavior:
//   - file missing: write the managed block.
//   - file has markers: skip if !force; on force, replace the markers-to-markers
//     block in place, preserving surrounding user content.
//   - file lacks markers: skip if !force and the file mentions devdash
//     (legacy substring check); otherwise append the managed block.
func writeManagedInstructions(target, instructions string, force bool) error {
	block := managedBlockStart + "\n\n" + instructions + "\n" + managedBlockEnd

	existing, err := os.ReadFile(target)
	if err != nil {
		return os.WriteFile(target, []byte(block+"\n"), 0644)
	}

	s := string(existing)
	startIdx := strings.Index(s, managedBlockStart)
	endIdx := strings.Index(s, managedBlockEnd)

	if startIdx >= 0 && endIdx > startIdx {
		if !force {
			fmt.Printf("  %s already contains devdash instructions (use --force to overwrite)\n", target)
			return nil
		}
		end := endIdx + len(managedBlockEnd)
		newContent := s[:startIdx] + block + s[end:]
		return os.WriteFile(target, []byte(newContent), 0644)
	}

	if !force && strings.Contains(s, "devdash") {
		fmt.Printf("  %s already contains devdash instructions (use --force to overwrite)\n", target)
		return nil
	}

	sep := "\n\n"
	if strings.HasSuffix(s, "\n\n") {
		sep = ""
	} else if strings.HasSuffix(s, "\n") {
		sep = "\n"
	}
	return os.WriteFile(target, []byte(s+sep+block+"\n"), 0644)
}

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

// agentConfig holds per-agent customization for the shared instruction template.
type agentConfig struct {
	// Preamble is agent-specific text inserted before the shared core
	// (e.g., tool disambiguation, naming conventions).
	Preamble string
	// Postamble is agent-specific text appended after the shared core
	// (e.g., agent-specific behavioral tips).
	Postamble string
}

// buildInstructions generates the shared devdash agent instructions,
// sandwiching agent-specific content around the universal core.
func buildInstructions(pid, closeOn string, cfg agentConfig) string {
	var b strings.Builder

	// Agent-specific preamble
	if cfg.Preamble != "" {
		b.WriteString(cfg.Preamble)
		b.WriteString("\n")
	}

	// --- Shared core (identical across all agents) ---
	fmt.Fprintf(&b, `## Core Principles

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

## Session Startup

Run `+"`devdash prime`"+` at the start of every new session and after any context loss
(compaction, clear, handoff). It provides dynamic project context — team, health
stats, and output format guidance — that these static instructions cannot.
`, closeOn, closeOn, closeOn)

	// Agent-specific postamble
	if cfg.Postamble != "" {
		b.WriteString("\n")
		b.WriteString(cfg.Postamble)
		b.WriteString("\n")
	}

	return b.String()
}

func setupClaude(pid, closeOn string, force bool) error {
	target := "CLAUDE.md"

	preamble := fmt.Sprintf(`# DevDash — AI Agent Task Tracking

This project uses **devdash** (`+"`dd`"+`) for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself via the terminal — do not just tell the user to run them.
Do NOT use TodoWrite, TaskCreate, `+"`bd`"+`, or markdown files for tracking. When the user
says "dd", they mean the devdash CLI, not the Unix `+"`dd`"+` data-copy utility.

Issues are called "beads" internally. You'll see this in fields like `+"`parentBeadId`"+`.

Project ID: %s
`, pid)

	postamble := `## Agent-Specific Instructions

- You may use your built-in task tools (TaskCreate, TodoWrite, etc.) for your own tracking, but you **must also** create and update devdash issues. Devdash is the system of record.
- When the user asks you to implement a plan, feature, or fix: your **very first action** is ` + "`devdash create`" + `. Do not read files, do not write code — create the issue first.
- For multi-step plans, create one devdash issue per step before starting any implementation. Group them under a parent issue. Then work through them sequentially: mark in-progress, implement, commit, close, move to next.
- After creating issues, follow the normal workflow: mark in-progress, do the work, commit, then close.`

	instructions := buildInstructions(pid, closeOn, agentConfig{
		Preamble:  preamble,
		Postamble: postamble,
	})

	if err := writeManagedInstructions(target, instructions, force); err != nil {
		return err
	}
	fmt.Printf("  ✓ %s configured for devdash\n", target)

	if err := setupClaudeHooks(force); err != nil {
		return err
	}

	return nil
}

func setupCodex(pid, closeOn string, force bool) error {
	target := "AGENTS.md"

	preamble := fmt.Sprintf(`# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself in the terminal; do not ask the user to run them for you.
Do NOT use markdown files or other tools for task tracking.

Issues are called "beads" internally. You'll see this in fields like `+"`parentBeadId`"+`.

Project ID: %s
`, pid)

	postamble := `## Agent-Specific Instructions

- Minimize redundant startup work after ` + "`devdash prime`" + `. Don't run broad repo scans or repeated discovery commands unless the current request needs them.
- Read only the context needed for the current request. Prefer targeted repo reads over whole-repo exploration when the task is narrow.
- Preserve existing user changes. Do not revert unrelated modifications or overwrite work you did not make.
- Run the narrowest verification that meaningfully covers the change, then summarize the result for the user.`

	instructions := buildInstructions(pid, closeOn, agentConfig{
		Preamble:  preamble,
		Postamble: postamble,
	})

	if err := writeManagedInstructions(target, instructions, force); err != nil {
		return err
	}
	fmt.Printf("  ✓ %s configured for devdash\n", target)
	return nil
}

func setupCopilot(pid, closeOn string, force bool) error {
	target := ".github/copilot-instructions.md"

	preamble := fmt.Sprintf(`# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself via the terminal — do not just tell the user to run them.
Do NOT use markdown files or other tools for task tracking. When the user
says "dd", they mean the devdash CLI, not the Unix `+"`dd`"+` data-copy utility.

Issues are called "beads" internally. You'll see this in fields like `+"`parentBeadId`"+`.

Project ID: %s
`, pid)

	instructions := buildInstructions(pid, closeOn, agentConfig{
		Preamble: preamble,
	})

	if err := os.MkdirAll(".github", 0755); err != nil {
		return err
	}

	if err := writeManagedInstructions(target, instructions, force); err != nil {
		return err
	}
	fmt.Printf("  ✓ %s configured for devdash\n", target)
	return nil
}

type settingsConfig struct {
	Hooks       map[string]interface{} `json:"hooks,omitempty"`
	Permissions map[string]interface{} `json:"permissions,omitempty"`
}

type sessionStartHook struct {
	Matcher string        `json:"matcher"`
	Hooks   []hookCommand `json:"hooks"`
}

type hookCommand struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

func setupClaudeHooks(force bool) error {
	configDir := ".claude"
	configFile := configDir + "/settings.local.json"

	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	config := settingsConfig{
		Hooks: map[string]interface{}{
			"SessionStart": []sessionStartHook{
				{
					Matcher: "startup",
					Hooks: []hookCommand{
						{
							Type:    "command",
							Command: "dd prime",
						},
					},
				},
				{
					Matcher: "clear",
					Hooks: []hookCommand{
						{
							Type:    "command",
							Command: "dd prime",
						},
					},
				},
			},
		},
	}

	// Merge with existing config if file exists
	if data, err := os.ReadFile(configFile); err == nil {
		var existing settingsConfig
		if err := json.Unmarshal(data, &existing); err != nil {
			return err
		}
		// Always preserve existing permissions and other settings
		if existing.Permissions != nil {
			config.Permissions = existing.Permissions
		}
		// Merge other hook types if they exist (unless force is true)
		if !force && len(existing.Hooks) > 0 {
			for k, v := range existing.Hooks {
				if k != "SessionStart" {
					config.Hooks[k] = v
				}
			}
		}
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(configFile, append(data, '\n'), 0644); err != nil {
		return err
	}

	fmt.Printf("  ✓ %s configured with SessionStart hooks\n", configFile)
	return nil
}
