# Worked Example: A Multi-Step Feature

End-to-end walkthrough of decomposing a real feature request into a parent issue plus 6 children with a dependency graph. The user's request and the resulting issue tree are taken verbatim from a session in this repo.

The task: "Create friendly help docs for this CLI, auto-updating when the CLI is updated, hosted as a static site on GitHub Pages."

---

## 1. The user's request, and why a single issue would be wrong

The work has at least 5 distinct concerns:
1. A Go program that walks the Cobra command tree and emits Markdown.
2. A Jekyll-based site scaffold so GitHub Pages can serve it.
3. Hand-written guides (landing page, getting started, workflow tutorials).
4. Turning on GitHub Pages in the repo settings.
5. A CI workflow that re-runs the generator when CLI code changes.
6. The initial publishing + README link-up.

Each concern has its own review surface (Go code vs Jekyll YAML vs prose vs repo settings vs GitHub Actions YAML). Bundling them into one issue would create a too-large, hard-to-review PR. The decomposition signal is clear: **3+ distinct concerns ⇒ parent + children.**

## 2. Create the parent issue first

The parent captures the big-picture goal, the approach, the rough sequencing, and the acceptance criteria. It does not duplicate the children's detail.

```bash
devdash create \
  --title="Build auto-generated CLI help docs hosted on GitHub Pages" \
  --type=feature \
  --priority=2 \
  --description="$(cat <<'EOF'
# Goal
Ship a friendly public help site for the DevDash CLI...

# Approach: Hybrid
1. Auto-generated command reference — Go program walks Cobra tree
2. Hand-written guides — landing, getting started, workflow walkthroughs
3. Static site via Jekyll + GitHub Pages — serve from main:/docs
4. CI keeps things fresh — Action regenerates reference on every push

# Sub-issue order
1. Doc generator — independent
2. docs/ scaffolding + Jekyll config — independent
3. Hand-written guides — depends on (2)
4. GitHub Pages configuration — depends on (2)
5. CI auto-regeneration — depends on (1)
6. Initial publishing + verification — depends on (1, 2, 3, 4, 5)

# Acceptance
- Public URL loads with landing page, getting started, command reference
- Editing a command's Short/Long in internal/commands/ updates the docs site automatically
- README links to the docs site
EOF
)"
# → Created: 642e62d4-3165-4102-ad31-53abe7c512ff - Build auto-generated CLI help docs...
```

`642e62d4` is the parent. Every child below will set `--parent=642e62d4`.

## 3. Create each child with enough detail to execute cold

The principle: **a different agent picking up child `acf16262` should be able to execute it without reading anything else.** That means file paths, flag values, acceptance criteria, all inline.

```bash
# Child 1: doc generator
devdash create \
  --title="Build Go program to auto-generate Markdown docs from Cobra command tree" \
  --parent=642e62d4 --type=task --priority=2 --estimate=120 \
  --description="$(cat <<'EOF'
# Goal
Create cmd/gen-docs/main.go: walk the Cobra command tree (internal/commands.NewRootCmd)
and write one Markdown page per command into docs/reference/.

# Files
- cmd/gen-docs/main.go (new)
- docs/reference/*.md (generated; commit them)

# Acceptance criteria
- `go run ./cmd/gen-docs` regenerates docs/reference/ deterministically (second run = no diff)
- Each page has the command's Short, Long, usage, and flag table
EOF
)"
# → Created: acf16262-2efa-430c-ba6a-972d366f5b10 - Build Go program to auto-generate...

# Child 2: docs/ scaffolding
devdash create \
  --title="Create docs/ directory structure with Jekyll config for GitHub Pages" \
  --parent=642e62d4 --type=task --priority=2 --estimate=45 \
  --description="$(cat <<'EOF'
# Goal
Scaffold docs/ so GitHub Pages can serve it with the just-the-docs theme.

# Files
- docs/_config.yml (title, theme: just-the-docs, baseurl)
- docs/index.md (placeholder landing page)
- docs/Gemfile

# Acceptance criteria
- `bundle exec jekyll build` succeeds locally from docs/
EOF
)"
# → Created: aa754ab0-f71a-488b-a1f8-74d8235fa946 - Create docs/ directory structure...

# Child 3: hand-written guides
devdash create \
  --title="Write hand-curated guide content (landing, getting started, workflows)" \
  --parent=642e62d4 --type=task --priority=2 --estimate=180 \
  --description="$(cat <<'EOF'
# Goal
Write the human-facing pages: landing, getting started, and a workflow walkthrough.

# Files
- docs/index.md, docs/getting-started.md, docs/workflow.md
- Each starts with just-the-docs front matter (title, nav_order)

# Acceptance criteria
- Getting started goes from install to first closed issue with copy-pasteable commands
EOF
)"
# → Created: 58ae4f35-3823-43bc-96e8-941bd5c706d9 - Write hand-curated guide content...

# Child 4: enable GitHub Pages
devdash create \
  --title="Enable GitHub Pages serving from /docs folder on main" \
  --parent=642e62d4 --type=task --priority=2 --estimate=20 \
  --description="$(cat <<'EOF'
# Goal
Turn on Pages: Settings > Pages > Source = Deploy from branch, main, /docs.

# Acceptance criteria
- First deploy succeeds and the site URL returns 200
- The URL is recorded in the parent issue
EOF
)"
# → Created: 28e102b5-ea44-4f30-988c-403fb265cfe8 - Enable GitHub Pages serving...

# Child 5: CI workflow
devdash create \
  --title="Add GitHub Action to regenerate command reference docs on CLI changes" \
  --parent=642e62d4 --type=task --priority=2 --estimate=60 \
  --description="$(cat <<'EOF'
# Goal
Keep docs/reference/ in sync: on push to main touching internal/commands/**, run
`go run ./cmd/gen-docs` and open a PR (peter-evans/create-pull-request) if there is a diff.

# Files
- .github/workflows/docs.yml (new)

# Acceptance criteria
- Editing a command's Short text produces a bot PR updating docs/reference/
EOF
)"
# → Created: d4868102-91ab-4aea-aa93-62fa131eaf8e - Add GitHub Action to regenerate...

# Child 6: publish + README
devdash create \
  --title="Generate initial reference, publish first version, link from README" \
  --parent=642e62d4 --type=task --priority=2 --estimate=45 \
  --description="$(cat <<'EOF'
# Goal
Run the generator, commit, push, verify the public site, and link it from README.md.

# Acceptance criteria
- Public site shows landing page, getting started, and the command reference
- README.md links to the site
EOF
)"
# → Created: 073f834a-4577-4423-a2d6-985a6b29726f - Generate initial reference...
```

Note that priorities are identical here (all `2`) because no single child is on a critical path. If a release deadline forced "ship the public URL before any guides," child 4 (GitHub Pages enable) might bump to priority 1.

## 4. Wire dependencies

Run independently — these can fan out in parallel since they only manipulate the dependency graph:

```bash
devdash dep add 58ae4f35 aa754ab0   # guides need scaffolding
devdash dep add 28e102b5 aa754ab0   # GH Pages needs scaffolding
devdash dep add d4868102 acf16262   # CI needs generator
devdash dep add 073f834a acf16262   # publish needs generator
devdash dep add 073f834a aa754ab0   # publish needs scaffolding
devdash dep add 073f834a 58ae4f35   # publish needs guides
devdash dep add 073f834a 28e102b5   # publish needs Pages enabled
devdash dep add 073f834a d4868102   # publish needs CI
```

Reading: `devdash dep add <issue> <depends-on>` means "the first issue is blocked until the second completes."

## 5. The resulting dependency graph

```
[1] acf16262  Doc generator         ← independent (no deps)
[2] aa754ab0  docs/ + Jekyll        ← independent (no deps)
        ↓
[3] 58ae4f35  Hand-written guides   ← blocked by [2]
[4] 28e102b5  Enable GH Pages       ← blocked by [2]
[5] d4868102  CI auto-regen         ← blocked by [1]
        ↓
[6] 073f834a  Publish + README link ← blocked by [1, 2, 3, 4, 5]
```

`[1]` and `[2]` are the roots — two agents could pick these up in parallel immediately. Once `[2]` closes, `[3]` and `[4]` unblock. Once `[1]` closes, `[5]` unblocks. `[6]` is the sink and closes last.

## 6. How another agent picks up the work

```bash
devdash ready
# → Lists pending, unblocked issues. With the graph above, on day one this returns
#   one line per issue, in the form `<icon> <id> [P<priority>] [<type>] - <subject>`:
#   ○ acf16262 [P2] [task] - Build Go program to auto-generate Markdown docs from Cobra command tree
#   ○ aa754ab0 [P2] [task] - Create docs/ directory structure with Jekyll config for GitHub Pages
```

After `[2]` closes:

```bash
devdash ready
# → Now also shows 58ae4f35 and 28e102b5 (since their dep on [2] is satisfied).
```

This is the payoff: an agent never needs to read the dependency graph or remember which child blocks which. `devdash ready` answers "what can I pick up right now?" and the dep graph handles the rest.

## 7. Closing order

Children close as their work lands. When the **last** child closes, the server closes the parent automatically (with no summary) and `devdash close` tells you so:

```
Parent 642e62d4 "Build auto-generated CLI help docs..." closed automatically: all its children are done.
  Add an overall summary: devdash close 642e62d4 --summary="..."
```

Follow that hint: closing an already-closed issue **replaces** its summary, commit and PR, so this is how the parent gets its roll-up summary citing the children:

```bash
devdash close 642e62d4 \
  --summary="Docs site shipped at https://devdashproject.github.io/devdash-cli/. \
  All 6 sub-issues closed: generator (acf16262), scaffolding (aa754ab0), guides (58ae4f35), \
  Pages (28e102b5), CI (d4868102), publish (073f834a). Auto-regen verified by editing a \
  command Short description and watching the bot PR appear." \
  --commit=$(git rev-parse HEAD)
```

---

## What this teaches

- **Decomposition is up-front work.** It takes ~10 minutes to write all 7 issues with substantive descriptions. That investment pays back every time another agent (or future-you) picks up a child without needing to re-derive the context.
- **Descriptions are the contract.** When you write a child issue, write it as if a different agent will execute it — file paths, flag values, acceptance criteria, all inline. Otherwise the issue becomes useless and you've created tracking overhead with no payoff.
- **The dep graph is the scheduler.** `devdash ready` reads the graph and tells the next agent what to pick up. No coordination required.
- **The parent isn't a child.** It captures *approach* and *acceptance for the whole feature*, not *implementation*. Resist the temptation to copy implementation detail into the parent.
- **Decomposition can be wrong; correct it.** If you start the work and realize the seams are mis-drawn, open a new child, close one that's no longer needed, or add a dep. The structure is malleable.
