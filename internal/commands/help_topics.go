package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

// RegisterHelpTopics overrides cobra's default help command to support topic subcommands.
func RegisterHelpTopics() {
	// Replace the default help command
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [topic]",
		Short: "Help about devdash or a specific topic",
		Long:  "Available topics: cli, workflow, close, pr, projects, report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return rootCmd.Help()
			}

			// Check if it's a known topic
			topic := args[0]
			if text, ok := helpTopics[topic]; ok {
				fmt.Println(text)
				return nil
			}

			// Fall back to cobra's command help
			target, _, err := rootCmd.Find(args)
			if err == nil && target != nil {
				return target.Help()
			}

			fmt.Printf("Unknown help topic: %s\n\nAvailable topics: cli, workflow, close, pr, projects, report\n", topic)
			return nil
		},
	})
}

var helpTopics = map[string]string{
	"cli": `# DevDash CLI Reference

## Commands
  ready [--since=X]                    Pending + unblocked issues sorted by priority
  list [--status=X] [--since=X]       All issues, optionally filtered
  blocked                              Pending issues with unsatisfied dependencies
  show <id>                            Full issue detail
  find <uuid>                          Cross-project bead lookup (full UUID required)
  stats                                Project health: open/closed/blocked counts
  stale [--since=X]                    In-progress issues with no recent activity

  create --title="..." [flags]         Create a new issue
  update <id> [flags]                  Update an issue
  close <id> [<id>...] [flags]        Close one or more issues
  delete <id> [--force] [--cascade]   Delete issues
  report <id> --status=X [flags]      Report agent progress

  dep add <issue> <depends-on>         Add a dependency
  dep remove <issue> <depends-on>      Remove a dependency
  comment <id> --body="..."            Add a comment
  comments <id>                        List comments
  activity [<id>]                      View activity log

  login [--no-browser]                 Authenticate
  init                                 Initialize project in current repo
  doctor                               Check configuration
  team                                 List team members
  prime                                Output agent workflow context
  version                              Print version

## ID Formats
  Full UUID:    27bf66bd-945f-4714-93fd-0c3322b720f4
  Short prefix: 27bf (any unique prefix)
  Local ID:     dev-dash-42 (project-scoped)

## --since Format
  Nh          N hours (e.g., 24h)
  Nd          N days (e.g., 7d)
  Nw          N weeks (e.g., 2w)
  YYYY-MM-DD  Exact date

## Priority
  0=critical, 1=high, 2=medium (default), 3=low, 4=backlog`,

	"workflow": `# DevDash Workflow Guide

## Starting Work
  devdash ready                             See what's available
  devdash show <id>                         Read the full issue
  devdash update <id> --status=in_progress  Mark as started

## Completing Work
  git add <files>
  git commit -m "message"
  git push
  devdash close <id> --summary="..." --commit=$(git rev-parse HEAD)

  Git operations MUST succeed before closing. Never close before push.

## When to Create Issues
  - Before writing ANY code (issue-first rule)
  - One issue per commit
  - When scope expands mid-task, create a new issue
  - For multi-step plans: one parent + child issues per step
  - Even spikes/investigations deserve an issue

## Decomposition Patterns
  - Parent/child: "parts of a whole" (--parent=<id>)
  - Dependencies: "X must happen before Y" (dep add)
  - Prefer parent/child for breakdown, dependencies for ordering

## Before Starting Work
  Always run devdash show <id> and check:
  - parentBeadId: understand the larger goal
  - blockedBy/blocks: understand ordering constraints
  - preInstructions: agent-specific context`,

	"close": `# Close Summary Guide

Close summaries are institutional memory. Future agents and humans will
read them to understand what happened.

## What to Include
  - What changed (files, functions, behavior)
  - Why (the motivation, not just "fixed the bug")
  - Decisions made and alternatives considered
  - Surprises or gotchas discovered
  - Follow-up work needed

## Examples

  Good: "Added cursor-based pagination to FetchAll. Chose generic approach
  with type parameter to avoid duplication. API returns plain arrays for
  some endpoints — added fallback unmarshaling."

  Bad: "Done"
  Bad: "Fixed the issue"
  Bad: "Implemented as described"

## Metadata
  --summary="..."        What changed and why
  --commit=SHA           Git commit SHA
  --pr=URL               Pull request URL (if applicable)`,

	"pr": `# Pull Request Format

## DevDash Footer
Every PR should include a DevDash footer section:

  ## DevDash
  Project: ` + "`95ca3de0-7e4f-4f9e-9b17-36f5609cfa11`" + `
  Issues:
  - [<issue-id>](https://dev-dash-blue.vercel.app/issue/<issue-id>)

Replace <issue-id> with the full UUID of each devdash issue.
If the PR addresses multiple issues, list each on its own line.`,

	"projects": `# Cross-Project Work

## Targeting a Different Project
  DD_PROJECT_ID=<full-uuid> devdash <command>

## Finding Beads Across Projects
  devdash find <full-uuid>

  Returns bead with project context (projectId, projectName).
  Requires full UUID — prefix lookup only works within a project.

## Multi-Repo Dependencies
  Dependencies can only be created within a single project.
  For cross-project coordination, use comments or external tracking.`,

	"report": `# Progress Reporting

## Why Report
  Reporting prevents issues from appearing stale (30+ min without activity).
  It's fire-and-forget — if it fails, nothing breaks.

## Status Values
  code_complete   Code written, ready for commit
  committed       Git commit created
  pushed          Pushed to remote
  error           Something went wrong

## When to Report
  After completing code: devdash report <id> --status=code_complete --summary="..."
  After commit:          devdash report <id> --status=committed --commit=$(git rev-parse HEAD)
  After push:            devdash report <id> --status=pushed --branch=$(git branch --show-current)
  On error:              devdash report <id> --status=error --error="description"

## Always Report Before Exiting
  Even if the work is incomplete, report the current status so the issue
  doesn't appear abandoned.`,
}
