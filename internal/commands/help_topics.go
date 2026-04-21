package commands

import (
	"fmt"

	"github.com/spf13/cobra"
)

func registerHelpTopics(rootCmd *cobra.Command) {
	rootCmd.SetHelpCommand(&cobra.Command{
		Use:   "help [topic]",
		Short: "Help about devdash or a specific topic",
		Long:  "Available topics: cli, workflow, close, pr, projects, report",
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return rootCmd.Help()
			}

			if text, ok := helpTopics[args[0]]; ok {
				fmt.Println(text)
				return nil
			}

			target, _, err := rootCmd.Find(args)
			if err == nil && target != nil {
				return target.Help()
			}

			fmt.Printf("Unknown help topic: %s\n\nAvailable topics: cli, workflow, close, pr, projects, report\n", args[0])
			return nil
		},
	})
}

var helpTopics = map[string]string{
	"cli": `# DevDash CLI Reference

## Issue Tracking (Core)
  ready [--since=X]                    Pending + unblocked issues sorted by priority. Excludes thoughts. Use this when you need to choose what to work on next.
  list [--status=X] [--since=X]       All issues, optionally filtered
  blocked                              Pending issues with unsatisfied dependencies
  show <id>                            Full issue detail: description, dependencies, pre-instructions, parent. Start here when the user already named the issue.
  find <uuid>                          Cross-project bead lookup (full UUID required)
  stats                                Project health: open/closed/blocked counts
  stale [--since=X]                    In-progress issues with no recent activity

  create --title="..." [flags]         Create a new issue
  update <id> [flags]                  Update an issue
  close <id> [<id>...] [flags]        Close one or more issues
  delete <id> [--force] [--cascade]   Delete issues

  dep add <issue> <depends-on>         Add a dependency
  dep remove <issue> <depends-on>      Remove a dependency
  comment <id> --body="..."            Add a comment
  comments <id>                        List comments
  activity [<id>]                      View activity log

  team                                 List team members
  prime                                Output agent workflow context
  report <id> --status=X [flags]      Report agent progress (optional, see: help report)

## Setup & Utility
  login [--no-browser]                 Authenticate
  link                                 Link current repo to a devdash project
  doctor                               Check configuration
  version                              Print version
  self-update                          Update devdash to the latest version
  uninstall [--dry-run]               Remove devdash and its configuration
  agent-setup [--agent=X] [--force]   Configure agent instructions for this repo
  alias-setup                          Add 'dd' alias to your shell RC file

## Project Management
  project create --name="..." [flags] Create a new project (--repo, --description)
  project list                         List all projects
  project delete <project-id>         Delete a project

## Token Management
  token create <name>                  Create a new API token
  token list                           List API tokens
  token revoke <id>                    Revoke an API token

## Execution & Automation (paid plan)
  dispatch <id> [--priority] [--worker]   Queue an issue for execution as a job
  jobs [--bead=X]                         List recent jobs
  jobs show <id>                          Job details (JSON)
  jobs log <id> [--tail=N]               Job output log
  jobs failures [--bead=X]               Recent failed jobs
  analyze <id>                            Trigger sandbox analysis for an issue
  diagnose <id>                           Investigate: status, job history, failures
  score [<id>]                            Score beads for automability
  reconcile-tasks [--auto-fix] [--json]  Audit and fix backlog inconsistencies

## Integration (WIP — being reworked)
  sync                                 Trigger full GitHub reconciliation
  import <issue-number> | --all       Import GitHub issues (--state=open|all)

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
  0=critical, 1=high, 2=medium (default), 3=low, 4=backlog

## Examples

  # Mark an issue as in-progress
  devdash update abc123 --status=in_progress

  # Close with summary and commit
  devdash close abc123 --summary="Added projectId to request bodies" --commit=$(git rev-parse HEAD)

  # Batch close multiple issues
  devdash close abc123 def456 --summary="Shipped both fixes"

  # Create a child issue under a parent
  devdash create --title="Fix API validation" --parent=abc123

  # Report progress (optional — for monitored workflows)
  devdash report abc123 --status=code_complete --summary="Tests passing"`,

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

	"pr": "# Pull Request Format\n\n## DevDash Footer\nEvery PR should include a DevDash footer section:\n\n  ## DevDash\n  Project: `95ca3de0-7e4f-4f9e-9b17-36f5609cfa11`\n  Issues:\n  - [<issue-id>](https://dev-dash-blue.vercel.app/issue/<issue-id>)\n\nReplace <issue-id> with the full UUID of each devdash issue.\nIf the PR addresses multiple issues, list each on its own line.",

	"projects": `# Cross-Project Work

## Targeting a Different Project
  DD_PROJECT_ID=<full-uuid> devdash <command>

## Finding Beads Across Projects
  devdash find <full-uuid>

  Returns bead with project context (projectId, projectName).
  Requires full UUID — prefix lookup only works within a project.

## Cross-Project Dependencies
  Dependencies work across projects. The blocker bead is resolved globally
  (not project-scoped), so you can link issues from different projects.

  Use full UUIDs for cross-project bead references:
    devdash dep add <issue-uuid> <blocker-uuid-from-other-project>

  The server verifies you have access to both projects.`,

	"report": `# Progress Reporting (Optional)

The report command is NOT part of the default workflow. The core workflow is:
  create → update --status=in_progress → [work] → close

Report exists for clients that want granular, real-time progress tracking —
dashboards, CI bots, multi-agent orchestrators, or any system polling for
intermediate status between "in progress" and "closed".

## When to Use Report
  - A monitoring dashboard is watching issue progress
  - Multiple agents coordinate and need to signal milestones to each other
  - Long-running tasks where a human or system needs visibility before completion
  - Error reporting when work is abandoned mid-task

  If none of these apply, you don't need report. Just close the issue when done.

## Status Values
  code_complete   Code written, ready for commit
  committed       Git commit created
  pushed          Pushed to remote
  error           Something went wrong

## Usage
  devdash report <id> --status=code_complete --summary="..."
  devdash report <id> --status=committed --commit=$(git rev-parse HEAD)
  devdash report <id> --status=pushed --branch=$(git branch --show-current)
  devdash report <id> --status=error --error="description"

  Report is fire-and-forget — if it fails, nothing breaks.

## For Client Developers
  Report statuses are available in the API for any client to consume.
  Clients that want agents to report should include reporting instructions
  in their agent context (e.g., via prime output or custom agent prompts).
  The default prime output does not include reporting instructions.`,
}
