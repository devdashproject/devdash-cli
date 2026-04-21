# DevDash — AI Agent Task Tracking

This project uses **devdash** for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself in the terminal; do not ask the user to run them for you.
Do NOT use markdown files or other tools for task tracking.

Issues are called "beads" internally. You'll see this in fields like `parentBeadId`.

Project ID: 47eb046a-b02a-41b4-926f-8bc7138ab470

## Core Principles

**Be a capture reflex.**
When the user mentions a bug, idea, TODO, or "we should probably..." — offer to create
an issue. Don't wait to be asked.

**Issue-first.**
Create an issue before doing work. Your first action when asked to implement something
should be `devdash create`.

**One issue per logical unit of work.**
If a task has multiple steps, create a parent issue and child issues. Scope creep during
a task = new issue, not an expanded current one. Every git commit must map to a devdash
issue.

## Rules

1. Create an issue before starting work. No exceptions.
2. `devdash update <id> --status=in_progress` before starting work on an issue.
3. Close with a substantive summary — write it for a future reader with zero context.
4. Don't batch unrelated work into a single issue.
5. **Close after push**: Only close issues after `git push` succeeds — never before.
6. No orphaned work: at session end, every commit must map to a closed issue.
7. Git operations MUST succeed before closing. Never run git and devdash close in parallel.
8. Preserve stderr: avoid `2>/dev/null` on devdash commands.

## Completing Work

`git add` → `git commit` → `git push` → `devdash close <id>`
On successful completions: `devdash close <id> --summary="..." --commit=$(git rev-parse HEAD)`.
If a PR exists, include `--pr=URL` too.
Close summaries are institutional memory — include what, why, decisions, surprises, follow-ups.
One issue per commit. Scope creep = new issue. Multi-step = parent + children.

## On-Demand Reference

Run these when you need detailed guidance:
- `devdash help cli` — Full command reference (flags, ID formats, --since syntax)
- `devdash help workflow` — When to create issues, decomposition patterns, bead relationships
- `devdash help close` — Close summary expectations with examples
- `devdash help pr` — PR footer format and multi-issue PRs
- `devdash help projects` — Cross-project dependencies and multi-repo work

## Session Startup

Run `devdash prime` at the start of every new session and after any context loss
(compaction, clear, handoff). It provides dynamic project context — team, health
stats, and output format guidance — that these static instructions cannot.

## Agent-Specific Instructions

- Minimize redundant startup work after `devdash prime`. Don't run broad repo scans or repeated discovery commands unless the current request needs them.
- Read only the context needed for the current request. Prefer targeted repo reads over whole-repo exploration when the task is narrow.
- Preserve existing user changes. Do not revert unrelated modifications or overwrite work you did not make.
- Run the narrowest verification that meaningfully covers the change, then summarize the result for the user.
