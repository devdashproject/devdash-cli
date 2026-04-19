# Getting Started with Dev-Dash

Welcome! This guide will get you from zero to managing tasks in about 5 minutes.

## What is Dev-Dash?

Dev-Dash is a task tracker built for developers who work with AI coding agents. It has two parts:

- **Web dashboard** at [dev-dash-blue.vercel.app](https://dev-dash-blue.vercel.app) — create projects, manage your team, connect GitHub, dispatch AI agents
- **CLI** (`devdash`) — manage tasks without leaving your terminal

They stay in sync automatically. Use whichever feels right for the moment.

---

## Step 1: Install the CLI

```bash
curl -fsSL https://raw.githubusercontent.com/devdashproject/devdash-cli/main/install.sh | sh
```

Or via npm:

```bash
npm install -g @devdashproject/devdash-cli
```

Verify it worked:

```bash
devdash version
```

If anything looks off, run `devdash doctor` — it'll tell you exactly what's missing.

---

## Step 2: Log in

```bash
devdash login
```

Your browser will open for Google sign-in. After you authenticate, the auth token is saved locally and you're done — no passwords to remember, no keys to paste.

> **First time?** If you don't have a Dev-Dash account yet, one is created automatically when you sign in with Google. No separate signup required.

---

## Step 3: Connect to a project

### If you have a git repo

Navigate to the root of the repo you want to work in and run:

```bash
devdash link
```

`link` detects your GitHub remote and auto-matches it to a Dev-Dash project if one exists. If there's no match, it walks you through picking an existing project or creating a new one.

A `.devdash` file is written to the directory you chose to link — this is how the CLI knows which devdash project to use when you're working in this repo. Commit it so teammates automatically get the same project connection without having to run `devdash link` themselves.

### If you don't have a git repo

Create a project either from the [web dashboard](https://dev-dash-blue.vercel.app) or the CLI:

```bash
devdash project create --name="My Project"
```

Then set your project ID in your shell so the CLI knows which project to use:

```bash
export DD_PROJECT_ID=<your-project-id>   # add to ~/.zshrc or ~/.bashrc to persist
```

You can find the project ID in your dashboard URL or by running `devdash project list`.

---

## Step 4: Set up the shortcut (optional)

Run `devdash alias-setup` to add a `dd` shortcut to your shell — it's offered automatically after login, or you can run it any time.

From here on, the examples use `dd`. If you skipped the alias, substitute `devdash` everywhere.

> The alias shadows `/usr/bin/dd` (a Unix disk-copy utility). If you use that tool, skip this step.

---

## Step 5: Configure your AI agents (optional)

If you use AI coding agents (Claude Code, Codex, Cursor, Copilot, Windsurf, Cline), you can auto-generate config files so they know to use `dd` for task tracking.

```bash
dd agent-setup
```

This auto-detects which agents you use and writes the appropriate config files. You can also specify agents directly:

```bash
dd agent-setup --agent=claude,codex    # Just these two
dd agent-setup --all                   # All supported agents
dd agent-setup --all --force           # Overwrite existing configs
```

Each agent gets instructions telling it to use `dd` commands, run `dd prime` at session start, and follow the git-push-before-close workflow.

---

## Step 6: Explore your project

See what's ready to work on:

```bash
dd ready
```

This shows all unblocked tasks sorted by priority, with the most actionable tasks first. Pick one and dig in:

```bash
dd show <id>
```

You can use the full UUID, a short prefix (like `27bf`), or a local ID. The CLI figures it out.

---

## Your first workflow

Here's the day-to-day loop:

```bash
# What needs doing?
dd ready

# Claim a task
dd update <id> --status=in_progress

# ... write code, fix bugs, ship features ...

# Done!
dd close <id>
```

### Create a task

```bash
dd create --title="Fix login redirect" --type=bug --priority=1
```

Priority runs from 0 (critical) to 4 (backlog). Type can be `task`, `bug`, `feature`, or `enhancement`.

### Add a dependency

Some things need to happen in order:

```bash
dd create --title="Write API endpoint" --type=task
dd create --title="Write tests for endpoint" --type=task
dd dep add <tests-id> <endpoint-id>
```

Now the tests task won't show up in `dd ready` until the endpoint task is closed.

### Check the big picture

```bash
dd stats
```

```
Total:       42
Pending:     28
In Progress: 3
Completed:   11
Blocked:     7
Ready:       21
```

---

## The web dashboard

The CLI handles task management, but the dashboard gives you a few extras:

| Feature | Where to find it |
|---------|-----------------|
| **Kanban board** | Project > Board tab |
| **AI agent dispatch** | Project > Agents tab |
| **GitHub sync settings** | Project > Settings tab |
| **Team invitations** | Project > Settings > Members |
| **API keys** (Anthropic, OpenAI) | Settings page (top-right menu) |
| **GitHub connection** | Settings > GitHub |

To invite a teammate, go to your project settings and add their email. When they sign into Dev-Dash with Google, they'll automatically get access to your project, following the permissions you specify.

---

## Working with AI agents

Dev-Dash is designed to work alongside AI coding agents like Claude Code. *Any agent* that can run a CLI command can work with Dev-Dash to create and retrieve issues in real time. Tell your agent to use `devdash` to create an issue for you, or retrieve details about existing issues in your projects. Dev-Dash is designed to work with agents directly, and the CLI installation (see `agent-setup`) helps your agent understand how to use `devdash` commands effectively.

### Inject context about Dev-Dash and the current project

```bash
dd prime
```

This outputs a structured block of context — your project name, health stats, available commands, and workflow patterns. Set it up as a session hook so your agent always knows how to use `dd`.

If in doubt, tell the agent to run `dd prime` and `dd help`.

### Dispatch agents from the dashboard

In the **Agents** tab, you can assign a task to an AI agent and watch it work. The agent gets your repo, the task description, and project context, then runs in a sandboxed environment.

Check on agent jobs from the CLI:

```bash
dd jobs                  # Recent runs
dd jobs --bead=<id>      # Jobs for a specific task
dd diagnose <id>         # Investigate a task: status, jobs, failures
```

---

## Multi-repo setup

Each repo gets its own `.devdash` file. Just run `link` in each one:

```bash
cd ~/projects/frontend && dd link
cd ~/projects/backend && dd link
cd ~/projects/infra && dd link
```

The CLI reads `.devdash` to know which project you're working in. No global state to juggle.

---

## Quick reference

| Command | What it does |
|---------|-------------|
| `dd login` | Sign in via browser |
| `dd link` | Link repo to project |
| `dd ready` | Show unblocked tasks |
| `dd list` | All open tasks |
| `dd show <id>` | Task details |
| `dd create --title="..."` | New task |
| `dd update <id> --status=in_progress` | Claim work |
| `dd close <id>` | Mark done |
| `dd dep add <a> <b>` | A depends on B |
| `dd blocked` | Show stuck tasks |
| `dd stale` | In-progress tasks with no activity |
| `dd diagnose <id>` | Investigate a task |
| `dd jobs` | Recent agent jobs |
| `dd stats` | Project health |
| `dd prime` | AI agent context |
| `dd agent-setup` | Configure AI agents |
| `dd doctor` | Check setup |

---

## Troubleshooting

**"Not logged in"** — Run `dd login`.

**"No project configured"** — Run `dd link` inside your repo. If you don't have a git repo, either pass `--project=<id>` to each command, or set `DD_PROJECT_ID=<your-project-id>` in your shell. Find your project ID in the web dashboard or with `devdash project list`.

**Login port in use** — The CLI auto-tries ports 18787-18792. If all are busy, free one up and retry.

**Tasks not syncing with GitHub** — Check your GitHub connection in the web dashboard under Settings. You may need to re-authorize.

When in doubt: `dd doctor` checks everything.

---

That's it! You're set up and ready to ship. Happy building.
