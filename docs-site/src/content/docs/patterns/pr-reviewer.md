---
title: PR Reviewer Demo
description: A concrete application of the Athenaeum coordination pattern — specialist reviewer agents per review dimension plus a synthesizer, against a real pull request.
---

The [Athenaeum Coordination Pattern](/scion/patterns/athenaeum-coordination/)
documents the generic shape: one orchestrator, several worker agents, a
shared `/workspace/`, messages as doorbells, filesystem as the queue. The
**PR reviewer** demo is that pattern wearing a suit.

It lives at `examples/pr-reviewer/` in the main Scion repository.

## The shape

One PR → four agents running under the same `/workspace/`:

| Agent | Template | Role |
|---|---|---|
| `reviewer-coordinator` | `reviewer-coordinator` | Stages inputs, spawns specialists, waits on findings, spawns synthesizer, reports done |
| `reviewer-types` | `reviewer-types` | Scala 3 type-strength review |
| `reviewer-idioms` | `reviewer-idioms` | Scala 3 idiom conformance |
| `reviewer-semantics` | `reviewer-semantics` | Correctness & effect-handling review |
| `synthesizer` | `synthesizer` | Merges findings into `reviews/summary.md` |

All five run under the **`gemma-local`** harness-config — a local
llama.cpp server, no API keys, no cost. Swap in `claude` or `opencode` by
editing `default_harness_config:` in any template's `scion-agent.yaml`;
nothing else changes.

## Workspace layout

```
/workspace/
├── pr/
│   ├── metadata.json       # host driver writes this (from fixture or `gh`)
│   ├── diff.patch
│   └── files/              # post-change file content, capped at 500 lines
├── reviews/
│   ├── types/findings.md       # reviewer-types output
│   ├── idioms/findings.md      # reviewer-idioms output
│   ├── semantics/findings.md   # reviewer-semantics output
│   └── summary.md              # synthesizer output — the deliverable
└── review-context.md       # coordinator writes this; specialists read it
```

The directory path is the queue address, identical to athenaeum. No
specialist reads another specialist's file — they work independently,
like athenaeum's Act II sub-teams.

## Per-review cycle

1. **Stage.** The host-side driver `scripts/review-pr.sh` writes
   `pr/metadata.json`, `pr/diff.patch`, and `pr/files/**` into the shared
   workspace. This is the only host-side step.
2. **Coordinate.** `reviewer-coordinator` starts, confirms the inputs
   exist, and writes `/workspace/review-context.md` (PR summary, output
   contracts, severity taxonomy, empty-dimension rule).
3. **Dispatch.** Coordinator spawns the three specialists in parallel
   (each with `--notify`), then broadcasts the doorbell.
4. **Review.** Each specialist reads `pr/` and `review-context.md`,
   writes `reviews/<dimension>/findings.md`, direct-messages the
   coordinator, and exits via `sciontool status task_completed`.
5. **Synthesize.** Coordinator polls for all three findings files, then
   spawns the synthesizer. The synthesizer reads all three and writes
   `reviews/summary.md`.
6. **Done.** Coordinator broadcasts `*** REVIEW COMPLETE ***` and exits.

## Severity

**Critical / Moderate / Minor**, matching `.design/progeny-review.md`.
The synthesizer preserves the specialist's assigned severity — no
re-grading across the seam.

## Empty dimensions

If a specialist has nothing to flag (e.g. a docs-only PR for
`reviewer-semantics`), it still writes a findings file — body exactly:

```
# <Dimension> Review

## Summary

there is nothing to review
```

The file's **existence** is the signal to the coordinator. The fixed
phrase lets the synthesizer detect and collapse clean dimensions into a
"Clean dimensions" list rather than showing empty severity buckets. If
every dimension is clean, the summary itself is one line: `there is
nothing to review`.

## Running it

From the repo root:

```bash
# Fixture mode — reproducible demo input.
make pr-reviewer-demo

# or
bash examples/pr-reviewer/scripts/review-pr.sh --fixture pr-sample-1

# Live mode — requires `gh` authenticated against the target repo.
bash examples/pr-reviewer/scripts/review-pr.sh --pr 42 --repo owner/repo
```

Output lands under `examples/pr-reviewer/.work/reviews/`.

## Where this diverges from athenaeum

Athenaeum is staged across five acts; the PR reviewer runs one cycle
only. Athenaeum's Game Runner grades submissions against a private
answer key in `~/playbook/`; the PR reviewer has no authoritative
answer key — the synthesizer is strictly a merge step, not a grader.
Athenaeum's Thorne is a peer-review validator that gates the DM's
attention; the PR reviewer has no analogue yet (all three specialists'
output flows straight to the synthesizer).

If you want a Thorne-style gate in this demo — a critic that reviews
the specialists' findings before the synthesizer sees them — add a
template between the specialists and the synthesizer that reads all
three findings files, writes `reviews/review-of-reviews.md`, and only
then unblocks the synthesizer. Athenaeum's Thorne template is the
pattern to copy.

## Related

- [Scala 3 Codebase Reviewer](/scion/patterns/scala3-codebase-reviewer/) — same
  coordination shape, but the input is a whole repo clone (not a PR diff)
  and the output is a single ranked modernization roadmap (not severity-grouped findings).

## Non-goals for v1

- No test-coverage, docs, performance, or security specialists — three
  dimensions is enough to prove the shape.
- No GitHub API write-back. Once `summary.md` exists,
  `gh pr comment $N --body-file reviews/summary.md` is a one-line add.
- No multi-round reviews (reviewers reading each other, then revising).
  Athenaeum's Act III convergence is the model for that.
- No language-agnostic refactor. Generalizing to TypeScript, Go, etc.
  means swapping each specialist's `system-prompt.md` — cheap, but out
  of scope for v1.
