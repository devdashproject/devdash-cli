# Issue Decomposition: When to Split, When to Keep One

How to break a unit of work into the right shape of DevDash issues — a single ticket, or a parent with children plus a dependency graph. Read this when the user asks for something that feels bigger than one commit.

## The split decision

| Signal | Right shape |
|--------|-------------|
| 1–2 commits, one file, single concern | **Single issue** |
| 3+ commits expected | **Parent + children** |
| Multiple distinct concerns (code + docs + CI) | **Parent + children** |
| Steps that can be parallelized by separate agents | **Parent + children** |
| Sub-tasks that should be reviewed in their own PRs | **Parent + children** |
| You can describe the work in one sentence without "and" | **Single issue** |
| You catch yourself writing "Step 1… Step 2…" in the description | **Parent + children** |
| Pure refactor or rename across many files | **Single issue** (one commit, even if many files) |

When in doubt, prefer single — over-decomposition creates ceremony with no payoff. Re-split later if the scope grows.

## Anti-patterns

- **Sibling issues for a linear task.** If child B can only start after child A finishes, and child C only after B, you have one issue with three steps, not three issues.
- **Parent with one child.** If you only need one sub-task, it's a single issue.
- **Children that aren't independently reviewable.** If closing each child individually would leave `main` in a broken state, you've split wrong; rethink the seams.
- **Decomposing during execution.** Decompose up front, before you start coding. Mid-stream splits muddle the per-commit mapping rule.

## Setting priority and estimate per level

- **Parent priority** = max of the children's priorities (the work is as urgent as its most urgent piece).
- **Parent estimate** = sum of children's estimates plus 10–20% integration overhead, or just omit it; estimates on the children matter more for `devdash ready`.
- **Child priority** typically matches parent unless one child is on the critical path; bump that one up by a level.

## The construction pattern

```bash
# 1. Create the parent first. Capture goal, approach, acceptance criteria.
devdash create --subject="<big-picture goal>" --type=feature --priority=2 \
  --description="$(cat <<'EOF'
# Goal
...
# Approach
...
# Acceptance criteria
...
EOF
)"
# → Note the UUID it prints; you'll need it.

# 2. Create each child with --parent=<parent-uuid>. Give each child enough
#    detail that another agent could execute it cold — file paths, flag
#    values, acceptance criteria.
devdash create --parent=<parent-uuid> --subject="<sub-task>" \
  --description="$(cat <<'EOF'
# Goal
...
# Files to change
...
# Acceptance criteria
...
EOF
)"

# 3. Wire dependencies for children that block others.
devdash dep add <child-b-uuid> <child-a-uuid>   # B is blocked by A
```

Closing order: work through children sequentially (or in parallel if independent). Close each as its commit lands. **Close the parent LAST**, with a roll-up summary citing the child commits.

## Worked example: building a docs site

In this session we decomposed "build auto-generated CLI help docs hosted on GitHub Pages" into 1 parent + 6 children with a real dep graph. The full structure:

**Parent:** `642e62d4` — Build auto-generated CLI help docs hosted on GitHub Pages
- Description named the hybrid approach (auto-generated reference + hand-written guides), the static-site choice (Jekyll on GitHub Pages from `/docs`), and the CI auto-regeneration plan
- Acceptance criteria covered: site loads, command edits update docs automatically, README links to site

**Children:**

| ID | Subject | Estimate | Depends on |
|----|---------|----------|------------|
| `acf16262` | Build Go doc generator (walks Cobra tree → markdown) | 120 min | — |
| `aa754ab0` | docs/ scaffolding + Jekyll config | 45 min | — |
| `58ae4f35` | Hand-written guides (landing, getting started, workflows) | 180 min | `aa754ab0` |
| `28e102b5` | Enable GitHub Pages | 20 min | `aa754ab0` |
| `d4868102` | CI auto-regeneration workflow | 60 min | `acf16262` |
| `073f834a` | Publish + README link | 45 min | `acf16262`, `aa754ab0`, `58ae4f35`, `28e102b5`, `d4868102` |

**Dep graph:**

```
[1] acf16262  Doc generator         ← independent
[2] aa754ab0  docs/ + Jekyll        ← independent
        ↓
[3] 58ae4f35  Hand-written guides   (needs 2)
[4] 28e102b5  Enable GH Pages       (needs 2)
[5] d4868102  CI auto-regen         (needs 1)
        ↓
[6] 073f834a  Publish + README      (needs 1, 2, 3, 4, 5)
```

**Commands that produced this:**

```bash
# Parent
devdash create --title="Build auto-generated CLI help docs..." --type=feature \
  --priority=2 --description="$(cat <<'EOF' ... EOF)"
# → Created: 642e62d4-...

# Children — note --parent= on each
devdash create --title="Build Go program..." --parent=642e62d4-... --type=task \
  --priority=2 --estimate=120 --description="..."
# ... five more children ...

# Dependencies — note "X depends on Y" reads as: X is blocked by Y
devdash dep add 58ae4f35 aa754ab0   # guides need scaffolding
devdash dep add 28e102b5 aa754ab0   # GH Pages needs scaffolding
devdash dep add d4868102 acf16262   # CI needs generator
devdash dep add 073f834a acf16262
devdash dep add 073f834a aa754ab0
devdash dep add 073f834a 58ae4f35
devdash dep add 073f834a 28e102b5
devdash dep add 073f834a d4868102
```

**Why this shape:**

- **Two roots (1, 2).** Generator and scaffolding are independent — different agents could pick them up in parallel.
- **Pages and guides fan out from scaffolding.** Both need the `docs/` directory to exist; once it does, both can proceed.
- **CI fans out from the generator.** No point wiring CI before the thing it runs exists.
- **Publish is the sink.** It validates everything together. Closing it triggers closing the parent.

**Why not different shapes:**

- **Why not 3 children instead of 6?** Each step here has a distinct review surface — Go code vs Jekyll config vs prose vs Pages settings vs YAML workflow. Bundling them would create a too-large PR.
- **Why not 12 children?** Splitting "hand-written guides" into one issue per page would be over-decomposition; the prose is one cohesive write-up.
- **Why are some children independent?** Independence lets the user (or another agent) parallelize. Forcing a sequential chain when one isn't needed costs time.

## When to add a child to an in-progress parent

It's fine. The capture reflex applies — if mid-execution you realize there's a step you missed, create a new child with `--parent=<parent-uuid>`. Wire any new deps. Don't expand an existing child's scope to absorb it.

## When to NOT decompose

- Bug fixes — usually one commit, one issue, even if you touch several files.
- Renames / refactors that span many files but are one logical change.
- Documentation typo runs.
- Reverts.
- Dependency bumps.
