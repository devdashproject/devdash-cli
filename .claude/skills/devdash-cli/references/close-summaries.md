# Writing Close Summaries That Earn Their Keep

A close summary is the only thing future readers — humans or agents three weeks from now — will see when they ask "what was issue `642e62d4` about and how did we resolve it?" The commit, the PR, the conversation context have all gone cold. The summary is what survives. Write it for them.

## The shape

Every good summary touches some subset of these five elements, in roughly this order:

1. **What changed** — concrete: file, function, behavior. One sentence.
2. **Why** — the motivation, especially if non-obvious from the diff.
3. **Decisions made** — paths considered and rejected, trade-offs taken.
4. **Surprises** — anything you discovered mid-task that future readers should know.
5. **Follow-ups** — issues you opened, deferred work, related cleanup.

Length: 2–5 sentences for a simple change, up to a paragraph for something with real decisions. Avoid the extremes — single-fragment summaries and multi-paragraph essays both age badly.

## The required mechanics

Every close should include the commit:

```bash
devdash close <id> \
  --summary="..." \
  --commit=$(git rev-parse HEAD)
```

Add `--pr=<url>` whenever a pull request exists:

```bash
devdash close <id> \
  --summary="..." \
  --commit=$(git rev-parse HEAD) \
  --pr=https://github.com/org/repo/pull/123
```

The `--commit` flag is what lets a reader jump from issue to code; never skip it.

## Good examples (verbatim from real closes)

**Cursor pagination, the canonical example from project CLAUDE.md:**

> Added cursor-based pagination to FetchAll. Chose generic type param approach to avoid duplication. API returns plain arrays on some endpoints — added fallback unmarshaling.

Three sentences. What (cursor pagination to `FetchAll`), decision (generic type param), surprise (some endpoints don't follow the standard shape; fallback needed). A reader six months later can answer "why is there a fallback path in `FetchAll`?" without grepping for the bug.

**Removing the devdash-test binary, from this session:**

> Removed devdash-test binary via git rm (commit 483aab3). Skipped the .gitignore additions originally specified in the issue per user direction — the binary should never have been committed in the first place, and the existing /devdash gitignore entry already covers the natural 'go build' output name. PR #9 opened against main, awaiting CI. Note: the binary still exists in prior commits (5f7c324 forward); deliberately did not rewrite history to avoid disrupting public main.

Four sentences. What, why-the-original-plan-changed, current state, and a deliberate omission with reasoning. A future reader knows both what was done and what wasn't.

**Building SKILL.md, also from this session:**

> Added .claude/skills/devdash-cli/SKILL.md (178 lines, 170-char description, under 200-char cap). 8 sections: activation triggers, 5-step workflow, rules, decomposition heuristic, top-10 command cheatsheet, create-flag table, anti-patterns, pointers to references. Followed Anthropic Agent Skills format with progressive disclosure — deep detail intentionally deferred to references/ and examples/ files in follow-on issues 25c08147 and 7a6858dc. Description front-loads triggers (DevDash, dd, task tracking, issue-first, parent/child) to survive any truncation. PR #10 opened.

Length sits at the high end of the range, justified because there are multiple decisions worth recording (line budget, format choice, deferred scope).

## Bad examples (what to avoid)

**"Fixed it."**
Useless. Tells the next reader nothing they couldn't get from the commit subject. If your summary fits in two words, you didn't write one.

**"Done."**
Same problem. The status field already says it's done.

**"Implemented as described."**
Implies the description is the summary. It isn't — the description is what you were going to do, the summary is what actually happened, including the deviations.

**"See PR for details."**
The whole point of the summary is to spare the reader from re-reading the PR. Putting the work back on them defeats the format. Cite the PR with `--pr=`, but say something in the summary.

**"Fixed the bug by updating the function."**
Vague to the point of uselessness. Which bug? Which function? "Updating" means what? This is the auto-pilot summary; the next reader will have to read the diff anyway.

## Side-by-side: same change, two summaries

A bug fix where the issue was an off-by-one error in pagination, but the actual fix involved switching to cursor-based pagination and adding a fallback for non-standard responses:

**Bad:** `Fixed pagination bug.`

**Good:** `Off-by-one in offset pagination was the symptom; root cause was that offset-based pagination doesn't compose with the server's eventual-consistency model. Switched FetchAll to cursor-based. Surprise: /comments returns a bare array, not a wrapped paged response — added fallback unmarshaling. Follow-up: open issue to migrate /comments endpoint to standard shape (filed 8b1c...).`

The bad one says nothing. The good one warns the next reader that there's a pending follow-up, explains why the code looks the way it does, and surfaces the surprise so nobody else has to discover it.

## When the change is genuinely trivial

Some closes really are trivial — a typo fix, a dependency bump, a CI trigger commit. Two short sentences are fine:

> Bumped `golang.org/x/net` from 0.17 to 0.20 to pull in the GO-2024-XXXX patch. No code changes; vendor only.

Don't pad. But do still include the why ("to pull in the patch"), not just the what.

## Quick checklist before closing

- Does the summary mention what changed in concrete terms (file, function, behavior)?
- Does it say why, especially if the diff alone wouldn't make that clear?
- If you made a non-obvious decision, is it in there?
- If you discovered something mid-task, is it in there?
- If you opened follow-up issues, are they cited (or at least mentioned)?
- Did you pass `--commit=$(git rev-parse HEAD)`?
- If a PR exists, did you pass `--pr=<url>`?

When the answer to all six is yes, close.
