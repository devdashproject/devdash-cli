# Project Management: IDs, Scopes, and Cross-Project Moves

Most `devdash` invocations operate in the scope of a single project. This file covers how the CLI figures out which project that is, how to override it, and what to do when work needs to move from one to another.

## How "current project" is resolved

When a command needs a project ID, the CLI looks in this order, taking the first non-empty source:

1. The `--project` persistent flag — set on the command line: `devdash --project=47eb046a list`.
2. The `DD_PROJECT_ID` environment variable.
3. The `.devdash` file in the current directory (written by `devdash link`).

If none of these is set, the CLI returns an error like `no project configured — run 'devdash link' or set DD_PROJECT_ID`.

## Short prefix resolution

Every flag, argument, or env var that expects a project ID also accepts the first 8+ characters of the UUID, as long as the prefix is unambiguous across projects you can access:

```bash
DD_PROJECT_ID=47eb046a devdash ready              # works
DD_PROJECT_ID=47eb devdash ready                  # works if no other project starts with "47eb"
DD_PROJECT_ID=47 devdash ready                    # fails: ambiguous prefix
```

The same applies to issue IDs. Use prefixes — they read better and the cost of an occasional collision is low (the CLI tells you when it happens, with the matching candidates).

## Listing projects

```bash
devdash project list
```

Output is one row per project: short UUID, name, GitHub repo if any. Use this when you need to find a target project for a `move`, or when the user mentions a project by name.

## Moving issues between projects

`devdash move <id>` relocates an issue and all its history (comments, status changes, dependencies that don't cross project boundaries) into a target project.

```bash
devdash move <issue-id> --to=<target-project>
```

Both `--from` (optional) and `--to` (required) accept UUIDs or short prefixes.

### When `--from` matters

By default the source project is the current project (resolved via the `--project` / env / `.devdash` chain above). Use `--from` when the issue lives in a project other than the current one — for instance, when you're in repo A but moving an issue that was accidentally filed in repo B's project:

```bash
# You're in repo A. The issue belongs to repo B's project. Move it to repo C's project.
devdash move 642e62d4 --from=<repo-b-project> --to=<repo-c-project>
```

Source precedence inside `move` is:

```
--from   >   --project   >   DD_PROJECT_ID   >   .devdash file
```

`--from` overrides everything. Without `--from`, the move uses the same project the rest of the CLI is using.

### Common move errors and what they mean

| Error | What's going on | Fix |
|-------|-----------------|-----|
| `target project not found: <prefix>` | Your `--to` prefix doesn't match any accessible project. | Run `devdash project list` to find the right ID. |
| `source bead not found: <id>` | The issue isn't in the source project the CLI is currently scoped to. | Add `--from=<correct-project>`. |
| `ambiguous prefix: <prefix>` | More than one project matches the prefix. | Lengthen the prefix or use the full UUID; the error lists matches. |

## Worked example: rescuing a misfiled issue

Suppose you ran `devdash create --subject="Fix typo in landing page"` from inside the wrong repo, and the issue landed in the `devdash-cli-go` project (`47eb046a-...`) instead of `dd-landing-page` (`aebcb113-...`) where it belongs.

The issue's short ID, from the `create` output, is `7c8d9e0f`.

```bash
# Verify where it lives now and confirm you have the right issue
devdash show 7c8d9e0f
# → status: pending, projectId: 47eb046a-..., subject: Fix typo in landing page

# Move it. --from is optional here because we're still in the devdash-cli-go repo,
# but being explicit makes the command self-documenting.
devdash move 7c8d9e0f --from=47eb046a --to=aebcb113
# → Moved 7c8d9e0f to project aebcb113-...

# Confirm
DD_PROJECT_ID=aebcb113 devdash show 7c8d9e0f
# → projectId is now aebcb113-...
```

If you cd into the `dd-landing-page` repo first, the `--from` becomes unnecessary because that repo's `.devdash` will already supply the source.

## Cross-project dependencies

Within a single project, `devdash dep add` works as documented in `commands.md`. Cross-project dependencies (issue in project A blocked by issue in project B) are not supported by `dep add` — the dependency model assumes a single project. If the user wants something close to it, the common pattern is:

- Create the issue in the project that owns the work.
- Add a `devdash comment` on the depending issue with the blocker's full UUID and a note.
- Optionally, the depending issue stays `pending` until you manually close the comment thread; it doesn't appear in `ready` until the blocker is resolved by convention rather than enforcement.

For richer cross-project tracking, see `devdash help projects` for any project-specific conventions in this repo.

## Targeting a single command at another project

You don't need to change your shell's environment to operate on another project temporarily. Two ergonomic options:

```bash
# Persistent flag for one command:
devdash --project=896b3dbc ready
devdash --project=896b3dbc list --status=in_progress

# Inline env var:
DD_PROJECT_ID=896b3dbc devdash show 642e62d4
```

Either works the same. Choose whichever reads better in context.

## When the user mentions a project by name

Don't guess the UUID — run `devdash project list`, find the matching row, and use that project's short UUID. Hallucinating a project ID is the same class of error as hallucinating a function name; it fails noisily once you actually run the command, but you've wasted a turn.
