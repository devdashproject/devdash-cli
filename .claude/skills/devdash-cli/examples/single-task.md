# Worked Example: A Single-Issue Task

End-to-end walkthrough of a small task — one issue, one commit, one PR — using the actual command sequence from a real session in this repo. Read this when you want to see the workflow in full motion before applying it yourself.

The task: clean up a stray 10 MB binary (`devdash-test`) that had been accidentally committed in an earlier session.

---

## 1. User raises the topic, agent offers an issue

> User: "What is the devdash-test binary?"

The agent investigates (file type, git blame, grep for references) and reports back that it's a stale build artifact with no source references. Then offers to open an issue.

> User: "yes"

The capture reflex triggers: a piece of cleanup work has been identified and agreed to, so the very next action is `devdash create`. **Not** a `git rm` first.

## 2. Create the issue (before any file changes)

```bash
devdash create \
  --title="Remove tracked devdash-test binary and gitignore build artifacts" \
  --type=task \
  --priority=3 \
  --estimate=15 \
  --description="$(cat <<'EOF'
# Goal
Remove the ~10 MB devdash-test Mach-O binary from version control and
prevent the bare devdash and devdash-test build outputs from being
re-committed.

# Background
devdash-test is a stale build artifact (the CLI itself, compiled
— devdash 0.4.0, ARM64 macOS Mach-O). First added in commit 5f7c324.
It is referenced by nothing in source, Makefile, or CI.

# Files to change
- git rm devdash-test
- Edit .gitignore: add /devdash-test (the existing /devdash already covers the standard build output)

# Acceptance criteria
- devdash-test no longer exists in the working tree
- .gitignore prevents future re-introduction
- PR opened against main
EOF
)"
# → Created: 5af2b7f4-e700-4a1a-9e56-bd69c708b3af - Remove tracked devdash-test binary...
```

Capture that UUID — every subsequent command needs it.

## 3. Mark in progress and create a branch

```bash
devdash update 5af2b7f4 --status=in_progress
# → Updated: 5af2b7f4-e700-4a1a-9e56-bd69c708b3af

git checkout -b chore/remove-devdash-test-binary
# → Switched to a new branch 'chore/remove-devdash-test-binary'
```

## 4. Do the work — but stay open to scope adjustments

The agent started editing `.gitignore` to add `/devdash-test`. Mid-edit, the user pushed back:

> User: "let's just remove it from the repo. It never should have been there, so we shouldn't need to watch for it in the future"

The agent adjusts: skip the `.gitignore` edit, just remove the file. (And saves a feedback memory entry capturing the rule for future sessions: "When cleaning up a mistake, don't also add defensive guards.")

```bash
git rm devdash-test
# → rm 'devdash-test'

git status
# → deleted: devdash-test
```

## 5. Commit and push (sequential, never parallel with close)

```bash
git commit -m "$(cat <<'EOF'
Remove stale devdash-test build artifact

Removes a 10 MB ARM64 macOS Mach-O binary (compiled devdash 0.4.0)
that was accidentally committed in 5f7c324 and re-added on subsequent
builds. Nothing in source, Makefile, or CI references it. Release
binaries come from goreleaser.

Closes devdash 5af2b7f4.
EOF
)"
# → [chore/remove-devdash-test-binary 483aab3] Remove stale devdash-test build artifact

git push -u origin chore/remove-devdash-test-binary
# → branch 'chore/remove-devdash-test-binary' set up to track 'origin/chore/remove-devdash-test-binary'.
```

## 6. Open the PR

```bash
gh pr create --title "Remove stale devdash-test build artifact" --body "..."
# → https://github.com/devdashproject/devdash-cli/pull/9
```

## 7. Close the issue with a substantive summary

```bash
devdash close 5af2b7f4 \
  --summary="Removed devdash-test binary via git rm (commit 483aab3). Skipped the .gitignore additions originally specified in the issue per user direction — the binary should never have been committed in the first place, and the existing /devdash gitignore entry already covers the natural 'go build' output name. PR #9 opened against main, awaiting CI. Note: the binary still exists in prior commits (5f7c324 forward); deliberately did not rewrite history to avoid disrupting public main." \
  --commit=$(git rev-parse HEAD) \
  --pr=https://github.com/devdashproject/devdash-cli/pull/9
# → Closed: 5af2b7f4-e700-4a1a-9e56-bd69c708b3af
```

Note the summary records the **deviation** from the original issue (`.gitignore` skipped) and the **deliberate omission** (no history rewrite). Future readers will know both what was done and what wasn't, and why.

---

## What this teaches

- **Issue-first works even for trivial tasks.** The whole loop took ~5 minutes including the create + close ceremony. The tracking value (a future reader can answer "why is this binary not in the repo anymore?" by reading one issue) far exceeds the cost.
- **Scope can shift mid-task.** The original issue called for `.gitignore` edits. The user changed direction. The agent didn't silently expand or shrink scope — it adjusted, recorded the change in memory, and reflected the deviation in the close summary.
- **Commit message and close summary serve different purposes.** The commit message explains the *change*; the close summary explains the *resolution* including what didn't happen.
- **Push before close, always.** The `git push` completed and was verified before `devdash close` ran. If push had failed (auth, conflict, hook rejection), `devdash close` would not have run yet.

Total time: ~5 minutes from "what is this binary?" to "closed PR opened against main."

Tracking value: a future contributor can search for the binary name or the commit SHA and land on a complete record — the question that prompted it, the work, the decisions, the deliberate omissions, and the PR.
