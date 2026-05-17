# DevDash Command Reference

Full flag and argument detail for every non-hidden `devdash` command. Read this when SKILL.md's cheatsheet isn't enough — for instance, when you need a flag you haven't seen, or when the user invokes a command outside the core workflow.

**Source of truth:** flag definitions live in `internal/commands/*.go`. When you must verify a flag, read the relevant `.go` file rather than trusting paraphrase.

**Conventions used below:**
- `<id>` → an issue/bead UUID. The CLI accepts the first 8 characters as a unique prefix.
- `<flag>` → a required value.
- `[<flag>]` → optional.
- All commands honor the persistent `--project <uuid-or-prefix>` flag to override the project.

---

## Core workflow

### `devdash create`

Create a new issue. Returns the created UUID on stdout.

```bash
devdash create --subject="<title>" [flags]
```

| Flag | Type | Default | Description |
|------|------|---------|-------------|
| `--subject` | string | — | Issue title. Required (or use `--title`). |
| `--title` | string | — | Deprecated alias for `--subject`. |
| `--description` | string | `""` | Markdown body. Use bash heredoc for multi-line. |
| `--type` | string | `task` | One of: `task`, `bug`, `feature`, `enhancement`, `thought`. |
| `--priority` | int | `2` | `0`=critical, `1`=high, `2`=medium, `3`=low, `4`=backlog. |
| `--parent` | string | — | Parent issue UUID/prefix. Use for sub-issues. |
| `--due` | string | — | `YYYY-MM-DD`. |
| `--estimate` | int | — | Minutes to complete. |
| `--sort-order` | int | — | Display order among siblings (0-based). |

Subjects starting with `-` are rejected to avoid flag-collision bugs; use `--subject="-foo"` explicitly if needed.

```bash
devdash create --subject="Fix login redirect" --type=bug --priority=1 \
  --description="$(cat <<'EOF'
When ?next=/admin is supplied to /login, the redirect ignores it.
Acceptance: ?next= is honored when target is same-origin.
EOF
)"
# → Created: 642e62d4-3165-4102-ad31-53abe7c512ff - Fix login redirect
```

### `devdash update <id>`

Mutate one or more fields on an existing issue in a single call. At least one flag must be passed.

```bash
devdash update <id> [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--status` | string | `pending`, `in_progress`, or `completed`. |
| `--title` | string | New subject/title. |
| `--description` | string | New markdown body. |
| `--priority` | int | `0`–`4`. |
| `--owner` | string | Assignee email or name. |
| `--parent` | string | New parent issue. |
| `--pre-instructions` | string | Agent-specific context attached to the issue. |
| `--due` | string | `YYYY-MM-DD`. |
| `--estimate` | int | Minutes. |
| `--sort-order` | string | Integer, or `none` to clear explicit ordering. |

```bash
devdash update 642e62d4 --status=in_progress
# → Updated: 642e62d4-...

devdash update 642e62d4 --priority=0 --owner=jason.w.massey@gmail.com
```

### `devdash close <id> [<id>...]`

Close one or more issues. Single-ID closes via PATCH; multiple IDs use a bulk endpoint.

```bash
devdash close <id> [<id2> ...] [flags]
```

| Flag | Type | Description |
|------|------|-------------|
| `--summary` | string | Completion summary. Always include this. |
| `--commit` | string | Git SHA — pair with `$(git rev-parse HEAD)`. |
| `--pr` | string | Pull request URL. |

```bash
devdash close 642e62d4 \
  --summary="Fixed by honoring ?next= when same-origin. Edge case: javascript: scheme rejected." \
  --commit=$(git rev-parse HEAD) \
  --pr=https://github.com/org/repo/pull/123
# → Closed: 642e62d4-...
```

When closing multiple sibling issues at once (e.g. on a roll-up commit), pass several IDs:

```bash
devdash close abc123ab def456cd --summary="Both fixed by..." --commit=$(git rev-parse HEAD)
```

### `devdash ready`

Pending, unblocked issues sorted by priority. The default "what to work on next" query.

```bash
devdash ready [--since=<duration>]
```

| Flag | Description |
|------|-------------|
| `--since` | Created within `Nh` / `Nd` / `Nw` or since `YYYY-MM-DD`. |

---

## Inspection

### `devdash show <id>`

Full issue detail as JSON: subject, description, status, priority, parent, dependencies, comments-count, etc. Use this to confirm an issue's state before acting on it.

```bash
devdash show 642e62d4
```

### `devdash list`

List issues with filters. Composes flags freely.

| Flag | Description |
|------|-------------|
| `--status` | `pending`, `in_progress`, or `completed`. |
| `--since` | `Nh` / `Nd` / `Nw` / `YYYY-MM-DD` (filters on `updatedAt`). |
| `--parent` | Show only children of a given parent. |
| `--mine` | Issues assigned to the current user (calls `/auth/me`). |

```bash
devdash list --status=in_progress --mine
devdash list --parent=9b5dae5b --status=pending
```

### `devdash find <uuid>`

Cross-project lookup by full UUID. Use when you have an issue ID but don't know which project it lives in.

### `devdash blocked`

Pending issues whose dependencies aren't yet complete. The inverse of `ready`.

### `devdash stale`

In-progress issues with no recent activity — agents that forgot to close, or work that stalled.

```bash
devdash stale --since=7d
```

### `devdash activity [<id>]`

Activity log for the project, or for a single issue if `<id>` is supplied. Includes creation, status changes, comments, closures.

| Flag | Description |
|------|-------------|
| `--limit` | Cap on results. |

### `devdash comments <id>`

List all comments on an issue. (Note: `comments` is plural; `comment` (singular) adds one.)

### `devdash stats`

Project health: open / in-progress / blocked / completed counts.

---

## Relationships

### `devdash dep add <issue> <depends-on>`

Declare `<issue>` is blocked until `<depends-on>` is completed. Both args resolve as UUID or short prefix.

```bash
devdash dep add 25c08147 9cb5aa55   # references blocked until SKILL.md is done
```

### `devdash dep remove <issue> <depends-on>`

Clear a dependency. Inverse of `dep add`.

### `devdash comment <id>`

Add a comment to an issue.

```bash
devdash comment 642e62d4 --body="Spec changed: ?next= must accept relative paths only."
```

---

## Reporting

### `devdash report <id>`

Post a progress update to an issue — used by dispatched/automated runners to report status without closing.

| Flag | Description |
|------|-------------|
| `--status` | Required: `code_complete`, `committed`, `pushed`, or `error`. |
| `--summary` | Progress note. |
| `--files-changed` | Count. |
| `--branch` | Git branch name. |
| `--commit` | Git commit SHA. |
| `--error` | Error message (when `--status=error`). |

### `devdash analyze`, `devdash score`

Higher-level project analysis (issue metrics, health scoring). Rarely needed in agent flows — use when the user asks for project health insights.

---

## Projects

### `devdash project list`

List all projects you can access. Output is one row per project: short UUID, name, GitHub repo (if any).

### `devdash project create --name=<name>`

| Flag | Description |
|------|-------------|
| `--name` | Project name (required). |
| `--repo` | `owner/repo` GitHub identifier. |
| `--description` | Long-form description. |

### `devdash project delete <id>`

Permanently delete a project. `--force` / `-f` skips confirmation.

### `devdash move <id>`

Move an issue between projects.

| Flag | Description |
|------|-------------|
| `--to` | Target project (UUID or short prefix). Required. |
| `--from` | Source project. Optional; overrides `--project` / `DD_PROJECT_ID` / `.devdash`. |

```bash
devdash move 642e62d4 --to=896b3dbc
devdash move 642e62d4 --from=47eb046a --to=896b3dbc
```

See `references/project-management.md` for the full source-project precedence rules.

---

## Jobs & dispatch

### `devdash jobs`

List recent jobs. Optionally filter `--bead=<id>`.

Subcommands:
- `devdash jobs show <id>` — full JSON for one job.
- `devdash jobs log <id>` — output log. Supports `--tail`.
- `devdash jobs failures` — last 10 failures.

### `devdash dispatch`

Run an automated job (build/test/deploy) against an issue. Project-specific; most agents won't invoke this directly.

---

## Auth & configuration

### `devdash login`

Authenticate the CLI. Opens an OAuth flow or accepts an API token.

### `devdash token`

Manage API tokens. Treat tokens like passwords — same scope as your session.

- `devdash token create <name>` — issue a new token.
- `devdash token list` — show active tokens.
- `devdash token revoke <id>` — invalidate one.

### `devdash link`

Link the current repo to a project. (Alias: hidden `init` for backwards compatibility.)

---

## Setup & maintenance

### `devdash prime`

Print dynamic project context: open-issue counts, current project, command quirks, all available projects. Wired to run automatically via `SessionStart` hooks in `.claude/settings.local.json` (`startup` and `clear` matchers).

**Rule:** do not run manually unless context has been lost (compaction, `/clear`, agent handoff). The hook handles it.

### `devdash agent-setup`

Configure agent instruction files for the current repository.

| Flag | Description |
|------|-------------|
| `--agent` | Comma-separated agent names: `claude`, `codex`, `copilot`. |
| `--all` | Setup all detected agents. |
| `--force` | Overwrite existing instructions inside the sentinel block. |
| `--close-on` | `commit` or `push` (default `push`) — workflow gate. |

Auto-detects agents based on existing files: `CLAUDE.md` → claude, `AGENTS.md` → codex, `.github/copilot-instructions.md` → copilot.

### `devdash alias-setup`, `devdash doctor`, `devdash diagnose`, `devdash reconcile`, `devdash sync`, `devdash team`

Operational tooling — invoke only when the user asks. `doctor` checks CLI health, `diagnose` collects debug info, `reconcile` repairs project state drift, `sync` pulls remote state.

### `devdash self-update`, `devdash uninstall`

Update or remove the CLI binary.

---

## Administration

### `devdash admin reset-user <user-id>`

Reset a user's data. Requires `ADMIN_SECRET` env var or `~/.config/dev-dash/admin-secret` file. Almost never used by agents.

---

## Utilities

### `devdash version`

Print the CLI version.

### `devdash help <topic>`

On-demand reference. See `references/help-topics.md` for which topic to invoke when.

- `devdash help cli` — flag / ID-format reference
- `devdash help workflow` — decomposition patterns
- `devdash help close` — close summary expectations
- `devdash help pr` — PR footer format
- `devdash help projects` — cross-project work
