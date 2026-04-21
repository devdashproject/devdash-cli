<!-- devdash:agent-instructions -->

# DevDash — AI Agent Task Tracking

This project uses **devdash** (`dd`) for task tracking. DevDash is a shared memory
between you and the user — a place where ideas, decisions, and progress are captured so
nothing gets lost.

Run devdash commands yourself via the terminal — do not just tell the user to run them.
Do NOT use TodoWrite, TaskCreate, `bd`, or markdown files for tracking. When the user
says "dd", they mean the devdash CLI, not the Unix `dd` data-copy utility.

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

- You may use your built-in task tools (TaskCreate, TodoWrite, etc.) for your own tracking, but you **must also** create and update devdash issues. Devdash is the system of record.
- When the user asks you to implement a plan, feature, or fix: your **very first action** is `devdash create`. Do not read files, do not write code — create the issue first.
- For multi-step plans, create one devdash issue per step before starting any implementation. Group them under a parent issue. Then work through them sequentially: mark in-progress, implement, commit, close, move to next.
- After creating issues, follow the normal workflow: mark in-progress, do the work, commit, then close.
<!-- /devdash:agent-instructions -->
