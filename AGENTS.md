<!-- devdash:agent-instructions -->

# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself in the terminal; do not ask the user to run them for you.
Do NOT use markdown files or other tools for task tracking.

Issues are called "beads" internally. You'll see this in fields like `parentBeadId`.

Project ID: 47eb046a-b02a-41b4-926f-8bc7138ab470

## The Workflow

Every task follows this exact sequence:

1. **`devdash create --title="..."`** — before any file reads or code. No exceptions.
2. **`devdash update <id> --status=in_progress`** — mark it started.
3. Do the work.
4. **`git add` → `git commit` → `git push`**
5. **`devdash close <id> --summary="..." --commit=$(git rev-parse HEAD)`**

For multi-step work: create a parent issue + one child per step. Work through children sequentially (create → in_progress → commit → close), then close the parent.

## Rules

- **Issue-first**: No exceptions. `devdash create` is your first action — before reading files or writing code.
- **One issue per commit**: Scope creep mid-task = new issue, not an expanded current one.
- **Close after push**: `git push` must succeed before closing. Never run git and devdash close in parallel.
- **Capture reflex**: When the user mentions a bug, idea, or "we should probably..." — offer to create an issue.
- **No orphaned work**: Every commit must map to a closed issue by session end.

## Close Summaries

Write for a future reader with zero context: what changed, why, decisions made, surprises, follow-ups. Not "Fixed the bug" or "Implemented as described."

`devdash close <id> \`
  `--summary="Added cursor-based pagination to FetchAll. Chose generic type param approach to avoid duplication. API returns plain arrays on some endpoints — added fallback unmarshaling." \`
  `--commit=$(git rev-parse HEAD)`

Add `--pr=URL` if a PR exists.

## Quick Reference

```
devdash ready                                          What to work on (pending, unblocked)
devdash show <id>                                      Full issue detail (description, deps, parent)
devdash create --title="Fix login redirect bug"        Create an issue
devdash update abc123 --status=in_progress             Mark started
devdash close abc123 --summary="Fixed X by doing Y" --commit=$(git rev-parse HEAD)  Close
devdash comment abc123 --body="Blocked on API response format"  Add a comment
devdash project list                                   List all projects
```

Run `devdash help cli` for the full reference — deps, activity, report, dispatch, and more.

## Session Startup

Run `devdash prime` at the start of every new session and after any context loss
(compaction, clear, handoff). It provides dynamic project context — project health,
command quirks, and output format guidance — that these static instructions cannot.

Treat `devdash prime` as sufficient session-start DevDash orientation. Do not also run
`devdash stats`, `devdash ready`, `devdash list --status=pending`, or similar broad
inventory commands just to get oriented. Run those commands only when the user asks
about project status, choosing the next issue, triage, backlog health, or when the
current task explicitly depends on that information.

## On-Demand Reference

- `devdash help workflow` — Decomposition patterns, bead relationships
- `devdash help close` — Close summary examples
- `devdash help pr` — PR footer format
- `devdash help projects` — Cross-project dependencies

## Agent-Specific Instructions

- Minimize redundant startup work after `devdash prime`; continue directly into the user's request using the issue-first workflow.
- Don't run broad repo scans or repeated discovery commands unless the current request needs them.
- Read only the context needed for the current request. Prefer targeted repo reads over whole-repo exploration when the task is narrow.
- Preserve existing user changes. Do not revert unrelated modifications or overwrite work you did not make.
- Run the narrowest verification that meaningfully covers the change, then summarize the result for the user.

<!-- /devdash:agent-instructions -->
