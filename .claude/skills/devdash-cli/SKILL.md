---
name: devdash-cli
description: Use the DevDash (dd) CLI for task tracking. Covers the issue-first workflow, parent/child issue decomposition for multi-step work, dependency graphs, and close summaries.
---

# devdash-cli

You are working in a repo that uses **DevDash** for task tracking. DevDash is a CLI (`devdash`, aliased to `dd`) plus a backing service that lets you and the user share a persistent record of every meaningful unit of work — features, bugs, ideas, spikes. Each unit is called a **bead** internally and an **issue** externally.

This skill makes you fluent in the DevDash workflow. Read it once at the start of a session where DevDash is in use; consult the linked references on demand.

## When to activate

Activate this skill when any of these hold:

- The current repo contains `CLAUDE.md`, `AGENTS.md`, or `.github/copilot-instructions.md` mentioning `devdash`
- `.claude/settings.local.json` registers a `devdash prime` hook
- The user references `dd`, `devdash`, "tasks", "issues", or "tickets"
- You are planning multi-step work that another agent might pick up later
- The user says "let's track this" or "open an issue for that"

If none of these apply, the repo probably does not use DevDash; do not invoke `devdash` commands speculatively.

## The five-step workflow

Every unit of work follows this exact loop. No exceptions.

```bash
# 1. CREATE — before any file reads or code edits.
devdash create --subject="Fix login redirect bug" --type=bug --priority=2 \
  --description="When users hit /login while authenticated, they should..."
# → Created: 642e62d4-... - Fix login redirect bug

# 2. UPDATE to in_progress — when you actually start.
devdash update 642e62d4 --status=in_progress

# 3. WORK — read files, edit code, run tests.

# 4. COMMIT + PUSH — git push must succeed before step 5.
git add <files> && git commit -m "..." && git push

# 5. CLOSE — with a substantive summary and the commit SHA.
devdash close 642e62d4 \
  --summary="Redirect now respects ?next= query param. Found a related XSS \
  hole in the redirect target validation; opened follow-up 8b1c..." \
  --commit=$(git rev-parse HEAD) \
  --pr=https://github.com/org/repo/pull/123
```

Short IDs (first 8 chars of the UUID) are accepted by every command that takes an ID.

## Non-negotiable rules

- **Issue-first.** `devdash create` is your very first action on any task — before reading files, before writing code. If the user gives you a task and you start exploring the repo, you've already broken the rule.
- **One issue per commit.** Scope creep mid-task → new issue, not an expanded current one.
- **Close after push.** `git push` must complete successfully before `devdash close`. Never run them in parallel. If push fails, fix the push; do not close.
- **Run commands yourself.** Invoke `devdash` via your shell. Never instruct the user to run `devdash` commands for you.
- **Capture reflex.** When the user mentions a bug, idea, or "we should probably…", offer to open an issue. Don't let context evaporate.
- **No orphaned work.** Every commit you author must map to a closed issue by session end.
- **Don't run inventory commands on startup.** Skip `devdash ready`, `stats`, `list --status=pending`. The `devdash prime` hook (configured in `.claude/settings.local.json`) handles session orientation automatically. Only run `prime` manually after context loss (compaction, `/clear`, handoff).

## When to use a single issue vs a parent + children

**One issue is right when:** the work is 1–2 commits, touches one tight concern, or completes within a single session.

**Parent + children is right when:** the work is 3+ commits, spans multiple distinct concerns (e.g., a Go generator + a Jekyll site + CI plumbing), has steps that can be parallelized, or sub-tasks should be reviewable independently.

The pattern:

```bash
# Parent first — capture the big-picture goal, approach, acceptance criteria.
devdash create --subject="Build CLI docs site" --type=feature \
  --description="Hybrid: auto-generated reference + hand-written guides..."
# → PARENT_ID = 9b5dae5b

# Children with --parent — each gets enough detail that another agent could execute it cold.
devdash create --parent=9b5dae5b --subject="Build doc generator" \
  --description="cmd/gen-docs/main.go that walks Cobra tree..."
devdash create --parent=9b5dae5b --subject="Set up Jekyll" \
  --description="docs/_config.yml with just-the-docs theme..."

# Dependencies — when one child blocks another.
devdash dep add <child-b-id> <child-a-id>   # child-b depends on child-a
```

Sequence of completion: work through children (each: in_progress → commit → close), then close the parent **last** with a roll-up summary citing the child commits.

For the full pattern with a worked example, see `references/decomposition.md`.

## Top-10 command cheatsheet

| Command | What it does | Example |
|---------|--------------|---------|
| `devdash ready` | List pending, unblocked issues you can pick up | `devdash ready` |
| `devdash show <id>` | Full issue detail (description, deps, parent) | `devdash show 642e62d4` |
| `devdash create` | Make a new issue | `devdash create --subject="…" --type=task` |
| `devdash update <id>` | Change status, priority, etc. | `devdash update 642e62d4 --status=in_progress` |
| `devdash close <id>` | Close with summary + commit | `devdash close 642e... --summary="…" --commit=$(git rev-parse HEAD)` |
| `devdash list` | Filter issues | `devdash list --status=pending --mine` |
| `devdash dep add` | Wire a blocking dependency | `devdash dep add <issue> <blocked-by>` |
| `devdash comment <id>` | Add a note to an issue | `devdash comment 642e... --body="Blocked on auth API"` |
| `devdash help <topic>` | On-demand reference | `devdash help workflow` |
| `devdash project list` | All projects you can access | `devdash project list` |

## Common flags for `devdash create`

| Flag | Purpose | Values |
|------|---------|--------|
| `--subject` | Issue title (primary form) | string |
| `--title` | Deprecated alias for `--subject` | string |
| `--description` | Markdown body | string — use bash heredoc for multi-line |
| `--type` | Category | `task` (default), `bug`, `feature`, `enhancement`, `thought` |
| `--priority` | Urgency | `0`=critical, `1`=high, `2`=medium (default), `3`=low, `4`=backlog |
| `--parent` | Link to a parent issue | UUID or 8-char short prefix |
| `--estimate` | Time in minutes | int |
| `--due` | Deadline | `YYYY-MM-DD` |
| `--sort-order` | Display order among siblings | int |

Use bash heredocs for descriptions with markdown — they survive shell escaping cleanly:

```bash
devdash create --subject="..." --description="$(cat <<'EOF'
# Goal
...
# Acceptance criteria
- ...
EOF
)"
```

## Anti-patterns

| Wrong | Right |
|-------|-------|
| Read files, then `devdash create` | `devdash create` FIRST, then read files |
| `--summary="Fixed it"` or `--summary="Done"` | Multi-sentence: what changed, why, decisions, surprises, follow-ups |
| `git push & devdash close ...` (parallel) | Sequential: push, verify, then close |
| "Please run `devdash update abc...`" to the user | Run it yourself via your shell |
| One issue for a 5-commit feature | Parent + 5 children |
| `devdash ready` / `stats` on session start | Trust the `devdash prime` hook; skip manual inventory |
| Sibling issues for one linear task | Single issue is fine — don't over-split |
| Creating an issue, then immediately closing it without commits | Close requires a real commit and a real summary |
| Hardcoding full UUIDs everywhere | Short 8-char prefixes work and read better |

## Pointers to references

Load these on demand — they are not in your context until you read them.

| When you need… | Read |
|----------------|------|
| Full flag details for any command, or to look up a command you don't recognize | `references/commands.md` |
| To decide how to split a feature into parent + children, and how to wire deps | `references/decomposition.md` |
| To write a substantive close summary (with side-by-side good vs bad examples) | `references/close-summaries.md` |
| To understand which `devdash help <topic>` to call and when | `references/help-topics.md` |
| To move issues between projects, use short prefixes, or set `DD_PROJECT_ID` | `references/project-management.md` |
| End-to-end command sequence for a single-commit task | `examples/single-task.md` |
| End-to-end command sequence for a multi-step feature (parent + children + deps) | `examples/multi-step-feature.md` |

The CLI itself also exposes on-demand help:

- `devdash help cli` — flag and ID-format reference for every command
- `devdash help workflow` — decomposition patterns and bead relationships
- `devdash help close` — close summary expectations with examples
- `devdash help pr` — PR footer format and multi-issue PRs
- `devdash help projects` — cross-project dependencies and multi-repo work

## A clean turn looks like this

User: *"Can you fix the login redirect bug we hit yesterday?"*

You:
1. `devdash create --subject="Fix login redirect bug" --type=bug --priority=1 --description="..."` → capture ID
2. `devdash update <id> --status=in_progress`
3. (read code, edit files, run tests)
4. `git add ... && git commit -m "..." && git push`
5. `devdash close <id> --summary="..." --commit=$(git rev-parse HEAD) --pr=<url>`

That sequence — every time, no shortcuts — is what makes DevDash valuable. The repo's commits and DevDash's issue log line up perfectly, and the next agent (or the user three weeks later) can read either side of the trail and reconstruct what happened.
