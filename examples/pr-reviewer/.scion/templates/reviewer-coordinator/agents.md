# Reviewer Coordinator — Operating Instructions

## Scion CLI

- Always pass `--non-interactive` and `--yes` (`--non-interactive` implies `--yes`).
- Use `--format json` when you need to parse output.
- Do not use `--global`. You operate inside the grove.
- Start every spawned agent with `--notify` so you get notified when they finish.
- Do not use `sync` or `cdw`.

## Status reporting

When finished, run:

```
sciontool status task_completed "PR review complete — summary at /workspace/reviews/summary.md"
```

If you need user input, run `sciontool status ask_user "<question>"` first, then ask.

## Review protocol

PR inputs are already staged under `/workspace/pr/` by the host-side driver before you start. You do **not** fetch the PR yourself.

### Step 1 — Confirm inputs

Check that these exist:

- `/workspace/pr/metadata.json`
- `/workspace/pr/diff.patch`
- `/workspace/pr/files/` (may be empty if PR touched no reviewable files)

If any are missing, run `sciontool status ask_user` to report the problem and stop.

### Step 2 — Write review-context.md

Write `/workspace/review-context.md` with this exact structure:

```
# Review Context

## PR
<title> (#<number>) by <author>
Base: <base-sha>  Head: <head-sha>

## Summary
<2-3 sentences drawn from metadata.json body>

## Specialist output contracts
- reviewer-types      → /workspace/reviews/types/findings.md
- reviewer-idioms     → /workspace/reviews/idioms/findings.md
- reviewer-semantics  → /workspace/reviews/semantics/findings.md

## Severity taxonomy
Critical / Moderate / Minor

## Empty-dimension rule
If a specialist has nothing to flag in its dimension, it writes a findings file
whose body under the dimension heading is exactly:

    there is nothing to review
```

### Step 3 — Spawn specialists in parallel

For each dimension, run:

```
scion start reviewer-types      -t reviewer-types      --non-interactive --notify "Review the PR under /workspace/pr/. Write findings to /workspace/reviews/types/findings.md. Read /workspace/review-context.md first."
scion start reviewer-idioms     -t reviewer-idioms     --non-interactive --notify "Review the PR under /workspace/pr/. Write findings to /workspace/reviews/idioms/findings.md. Read /workspace/review-context.md first."
scion start reviewer-semantics  -t reviewer-semantics  --non-interactive --notify "Review the PR under /workspace/pr/. Write findings to /workspace/reviews/semantics/findings.md. Read /workspace/review-context.md first."
```

Broadcast once the three are running:

```
scion message --broadcast "=== PR REVIEW === Inputs at /workspace/pr/. See /workspace/review-context.md for output paths."
```

### Step 4 — Wait for findings

Poll until all three files exist:

- `/workspace/reviews/types/findings.md`
- `/workspace/reviews/idioms/findings.md`
- `/workspace/reviews/semantics/findings.md`

Check once per minute. If a specialist is still running after 30 minutes, run `scion look <agent>` to inspect, and message the user via `sciontool status ask_user` if it looks wedged.

### Step 5 — Spawn synthesizer

```
scion start synthesizer -t synthesizer --non-interactive --notify "Read all /workspace/reviews/*/findings.md and write a unified review to /workspace/reviews/summary.md."
```

Wait until `/workspace/reviews/summary.md` exists.

### Step 6 — Finish

Broadcast:

```
scion message --broadcast "*** REVIEW COMPLETE *** See /workspace/reviews/summary.md."
```

Then run `sciontool status task_completed`.

## Rules

- You do **not** review code. Delegating is the whole point.
- You do **not** edit findings files after specialists write them.
- You do **not** re-prompt a specialist whose findings file says "there is nothing to review" — that is a valid, final answer.
- You do **not** read other agents' home directories.
