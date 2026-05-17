# `devdash help` and `devdash prime`: When to Call Which

The CLI ships two on-demand orientation surfaces. They serve different purposes and you should not confuse them.

## `devdash help <topic>` — narrow reference

`devdash help` is for looking up specific, durable information about the CLI itself: command flags, ID format rules, workflow conventions. It is text bundled into the binary, identical for every user.

The available topics:

| Topic | What it contains | Invoke when |
|-------|------------------|-------------|
| `cli` | Full command reference: every flag, every command, every ID-format rule, every `--since` accepted form. | You hit a flag or syntax you don't recognize and `references/commands.md` doesn't cover it. |
| `workflow` | Decomposition patterns, parent/child rules, dependency semantics, when an issue should be split. | You're about to decompose a feature and want the canonical pattern. |
| `close` | Close-summary expectations with several worked examples. | Before writing a close summary on a non-trivial issue. |
| `pr` | The footer block agents are expected to append to PRs that close issues (e.g. `Closes devdash 642e...`), and how to scope multi-issue PRs. | Before opening a PR that closes one or more devdash issues. |
| `projects` | Cross-project dependencies and multi-repo work. | When the user's task spans more than one project, or when you need to reason about `DD_PROJECT_ID`. |

Invocation pattern is uniform:

```bash
devdash help cli
devdash help workflow
devdash help close
devdash help pr
devdash help projects
```

These output to stdout in human-readable form (not JSON). Pipe to a pager or read inline.

## `devdash prime` — dynamic project context

`devdash prime` is different in kind. It calls the API and prints **dynamic** project context: the current project's open/in-progress/blocked counts, the list of all projects you can access, command quirks specific to your installed CLI version, and output-format guidance.

It is the canonical "orient an agent at session start" surface.

### How `prime` is normally triggered

In any repo where `devdash agent-setup` has run, `.claude/settings.local.json` contains `SessionStart` hooks that call `devdash prime` automatically:

```json
"hooks": {
  "SessionStart": [
    { "matcher": "startup", "hooks": [{ "type": "command", "command": "devdash prime" }] },
    { "matcher": "clear",   "hooks": [{ "type": "command", "command": "devdash prime" }] }
  ]
}
```

The harness runs the command and pipes its output into the conversation as a system-reminder. You see it before you see the user's first message.

### When to invoke `prime` manually

Almost never. The hook covers it. Run it manually only when:

- The conversation was compacted and the prime output may have been summarized away.
- The user ran `/clear` mid-session and you want fresh project context.
- An agent handoff happened and you have no startup context at all.
- The user asks "what's the state of the project?" or "what should I work on next?" and you want a fresh snapshot.

### When NOT to invoke `prime`

- On startup when the hook already ran. Running it twice wastes tokens.
- Before doing a concrete task the user just asked for. Just start the task — the prime context from startup is enough.
- Repeatedly during a session for "freshness." The state doesn't change fast enough to justify it.

## Quick decision tree

```
User asks about CLI flags / commands you don't know
  → `devdash help cli`

User is asking you to decompose work
  → `devdash help workflow` (or `references/decomposition.md`)

You're about to write a close summary
  → `devdash help close` (or `references/close-summaries.md`)

You're opening a PR that closes issues
  → `devdash help pr`

User mentions another project, or you need cross-project context
  → `devdash help projects` (or `references/project-management.md`)

User asks "what's open" / "what to work on" / context loss happened
  → `devdash prime`
```

## Why `help` and `prime` are both worth knowing

`help` is cheap and durable: it's just text. Call it freely the moment you're uncertain about a flag — better than guessing.

`prime` is more expensive (an API call, dynamic output) but it's also the only way to get fresh project state. Don't call it casually, but don't be afraid of it when the context genuinely calls for it.
