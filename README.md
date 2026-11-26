# devdash-cli

Dev-Dash CLI — AI-powered task tracking for coding agents.

A fast, single-binary CLI that wraps the [Dev-Dash](https://dev-dash-blue.vercel.app/) REST API for managing tasks, dependencies, and agent jobs from the terminal. Built in Go with zero runtime dependencies.

This CLI can be run in your terminal but is optimized for agents. Don't feel like you need to memorize the commands; your agent should be familiar enough!

**New here?** Check out the **[Getting Started guide](GETTING-STARTED.md)** for a friendly walkthrough, or read **[Understanding the Workflow](UNDERSTANDING-THE-WORKFLOW.md)** for the philosophy behind Dev-Dash.

## Install

```bash
curl -fsSL https://raw.githubusercontent.com/devdashproject/devdash-cli/main/install.sh | sh
```

Installs the latest release to `~/.local/bin` (set `DEVDASH_INSTALL_DIR` to change it). If that directory isn't on your `PATH`, the installer prints the line to add. Check it worked:

```bash
devdash version
```

Or download a binary directly from [GitHub Releases](https://github.com/devdashproject/devdash-cli/releases) (macOS, Linux, Windows; amd64 and arm64; `checksums.txt` is attached to each release).

### Build from source

```bash
go build -o devdash ./cmd/devdash
```

## Setup

First run, in order:

```bash
# 1. Log in (opens your browser to sign in to Dev-Dash)
devdash login

# 2. In your repo, link it to a project
cd your-repo
devdash link                    # auto-detects the project from your git remote
devdash link MyProject          # or match by name (case-insensitive)
devdash link 896b3dbc           # or by project ID prefix

# 3. Give your AI agent the workflow instructions (optional, recommended)
devdash agent-setup

# 4. Verify
devdash doctor
```

- **`devdash init` is an alias for `devdash link`.** It writes a `.devdash` file (project ID and `close_gate`) at the repo root; commit it so teammates get the same project. It does not set up agents or the `dd` alias for you; it prints the next step.
- **`agent-setup`** detects the agents in your repo and writes the matching instructions: `CLAUDE.md` (Claude, the default when none are found), `AGENTS.md` (Codex), `.github/copilot-instructions.md` (Copilot). For Claude it also adds `SessionStart` hooks that run `devdash prime`. Use `--agent=claude,codex` to pick agents and `--force` to overwrite.
- **Optional `dd` shortcut:** `devdash alias-setup` adds `dd` as an alias. It shadows `/usr/bin/dd` (Unix disk copy), so only enable it if you don't use that tool.

### Logging in without a browser (SSH, CI, coding agents)

`devdash login` needs a browser on the same machine. Otherwise use an API token:

```bash
devdash login --token=dd_...                  # verify and save the token
echo "$TOKEN" | devdash login --with-token    # same, keeps it out of shell history
export DEVDASH_TOKEN=dd_...                   # use it without saving anything
```

Create a token with `devdash token create "my laptop"` on a machine that is already logged in. See `devdash help auth`.

## Quick Start

```bash
devdash create --title="Fix login bug" --type=bug --priority=1   # 1. track it first
devdash ready                                                    # what is ready to work on
devdash update <id> --status=in_progress                         # 2. claim it
# ... do the work, commit, git push ...
devdash close <id> --summary="What changed and why" --commit=$(git rev-parse HEAD)
```

`<id>` can be the first 8 characters of the UUID. `devdash show <id> --pretty` shows an issue's details. Always give `close` a real `--summary`: it is the permanent record of the work.

### Creating tasks

```bash
devdash create --title="Fix login bug" --type=bug --priority=1
devdash create --title="Add dark mode" --type=feature --description="Support system theme"
```

### Managing dependencies

```bash
devdash create --title="Write tests" --type=task
devdash dep add <test-id> <feature-id>   # Tests depend on feature
devdash blocked                           # See what's waiting
```

## All Commands

| Command | Description |
|---------|-------------|
| `login` | Authenticate via browser (tries ports 18787-18792), or `--token=dd_...` / `--with-token` / `DEVDASH_TOKEN` when headless |
| `link [name-or-id]` | Link repo to a project (auto-detect, name, ID prefix, or interactive picker). `init` is an alias |
| `project create` | Create a new project (`--name`, `--repo`, `--description`) |
| `project list` | List all your projects |
| `project delete <id>` | Delete a project (`--force` to skip confirmation) |
| `list [--status=X] [--since=X] [--mine] [--tree]` | List tasks (`--status=open` = pending, in progress, blocked) |
| `ready [--since=X]` | Tasks with no blockers, sorted by automability then priority |
| `blocked` | Tasks waiting on dependencies |
| `stale [--since=X]` | In-progress tasks with no recent activity |
| `show <id> [--pretty]` | Task details with dependencies (JSON; `--pretty` for a readable view) |
| `create --title="..."` | Create task (`--type`, `--priority`, `--description`, `--parent`) |
| `update <id> --key=val` | Update fields (`--status`, `--priority`, `--owner`, `--title`, `--description`, `--pre-instructions`) |
| `close <id> [...] [--pr=URL] [--commit=SHA] [--summary="..."]` | Close one or more tasks (with optional completion metadata) |
| `delete <id> [--force] [--cascade]` | Delete a task (`--cascade` deletes children) |
| `dep add <id> <dep-id>` | Add dependency (id depends on dep-id) |
| `dep remove <id> <dep-id>` | Remove dependency |
| `comment <id> --body="..."` | Add a comment to a task |
| `comments <id> [--pretty]` | List comments on a task |
| `session save --summary="..."` | Save a session summary/handoff to project memory |
| `session list [--limit=N]` | List recent session summaries |
| `activity [--since=X]` | View activity log |
| `report <id> --status=X` | Log agent progress (`code_complete\|committed\|pushed\|error`) |
| `find <uuid>` | Cross-project task lookup |
| `diagnose <id>` | Investigate task: status, job history, failure details |
| `jobs [--bead=<id>]` | List recent jobs (optionally filtered by task) |
| `reconcile-tasks` | Audit and fix backlog inconsistencies |
| `score <id>` | Score task automability |
| `analyze <id>` | Analyze a task |
| `dispatch <id>` | Dispatch work to an agent |
| `sync` | Trigger full reconcile with GitHub |
| `import <num> \| --all` | Import GitHub issue(s) into the project |
| `stats` | Project statistics |
| `prime` | Output workflow context for AI agents (includes project health) |
| `doctor` | Check prerequisites and configuration |
| `agent-setup` | Configure AI agent instructions (`--agent=X`, `--all`, `--force`, `--close-on=commit\|push`) |
| `alias-setup` | Add `dd` shortcut alias |
| `move <id> --to=<project> [--from=<project>]` | Move a task to a different project |
| `version` | Print the CLI version |
| `team` | List project team members |
| `token create\|list\|revoke` | Manage API tokens |
| `self-update` | Update to latest version |
| `uninstall` | Remove the CLI |

### Bead IDs

Commands accept UUIDs, short prefixes (e.g., `27bf`), or local IDs. Ambiguous prefixes are rejected with suggestions.

A **full UUID needs no project**: `show`, `comment`, `comments`, `update`, `close`, `delete`, `report` and `dep add|remove` work from any directory with no `.devdash`, `DD_PROJECT_ID` or `--project` (the server uses the issue's own project, and checks your access to it). Prefixes and local IDs are only unique within one project, so they still use the current project. Against an older server that still requires a project, the CLI looks up the issue's project with one extra request and retries.

### Agent integration

`devdash agent-setup` (see [Setup](#setup)) writes the instructions your agent reads. You can also run it with options:

```bash
devdash agent-setup                      # detect agents in this repo
devdash agent-setup --agent=claude,codex
devdash agent-setup --all --force
```

The `prime` command outputs structured context for AI coding agents:

```bash
devdash prime
```

It includes project name, health stats, command reference, and workflow patterns. Use it as a session hook to inject workflow context automatically.

### Multi-project setup

Each repo gets its own `.devdash` file linking it to a project. The CLI reads this file to determine which project to use:

```bash
cd my-project && devdash link    # Links this repo
cd other-repo && devdash link    # Links that repo separately
```

You can also set `DD_PROJECT_ID` to choose a project or use the `--project` flag on any command. The CLI reads `project_id` from `.devdash`, but deliberately ignores `api_url` in that repository file: a repository can be controlled by someone else and must not choose where your saved bearer token goes.

For a self-hosted API, set `DD_API_URL` explicitly or put `{"api_url":"https://your-api.example"}` in `~/.config/dev-dash/settings.json` (or `$DD_CONFIG_DIR/settings.json`). The environment variable takes precedence. Credentialed endpoints must use HTTPS, except `http://localhost` or a loopback IP for local development. Redirects to a different origin are rejected. Run `devdash doctor` to see which endpoint source is active.

## Troubleshooting

| Issue | Fix |
|-------|-----|
| `devdash: command not found` | The install directory isn't on your `PATH`. Add the line the installer printed (default `~/.local/bin`) |
| `not logged in` | Run `devdash login`, or set `DEVDASH_TOKEN` (see [Logging in without a browser](#logging-in-without-a-browser-ssh-ci-coding-agents)) |
| `no project configured` | Run `devdash link` in your repo, or set `DD_PROJECT_ID`, or pass `--project` (not needed when you pass a full issue UUID) |
| `Port 18787 in use` during login | CLI auto-tries ports 18787-18792 |
| `API error (401)` | Token expired or revoked: re-run `devdash login` |
| Commands seem slow | API latency: ensure a good network connection |

Run `devdash doctor` to check all prerequisites and configuration; it shows which API endpoint and token source are in use.

## Exit Codes

| Code | Meaning |
|------|---------|
| 0 | Success |
| 1 | Any error (bad arguments, not logged in, no project, API or network failure). The message on stderr says which |

## Platforms

Pre-built binaries are available for:

- macOS (Apple Silicon and Intel)
- Linux (amd64 and arm64)
- Windows (amd64 and arm64)

## License

MIT
