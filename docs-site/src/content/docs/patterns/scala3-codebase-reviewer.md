---
title: Scala 3 Codebase Reviewer Demo
description: A whole-repo modernization reviewer — four specialist agents (Iron, Scala 3 syntax, Cats, effects) each argue a thesis about the codebase, and a synthesizer ranks them into a single roadmap.
---

The [PR Reviewer Demo](/scion/patterns/pr-reviewer/) shows the
[Athenaeum coordination pattern](/scion/patterns/athenaeum-coordination/)
applied to a single PR diff. **This** demo is the same pattern applied to
a whole codebase: clone the repo, run four specialists in parallel, get
back a single ranked roadmap of opinionated improvement directions.

It lives at `examples/scala3-codebase-reviewer/` in the main Scion repository.

## The shape

One repo → six agents running under the same `/workspace/`:

| Agent | Template | Role |
|---|---|---|
| `codebase-reviewer-coordinator` | `codebase-reviewer-coordinator` | Confirms clone, writes context, dispatches, polls, finalizes |
| `reviewer-iron` | `reviewer-iron` | Argues *where Iron refinement types replace primitive obsession* |
| `reviewer-syntax` | `reviewer-syntax` | Argues *how far to go on significant indentation, `enum`, `using`/`given`, `extension`, `derives`* |
| `reviewer-cats` | `reviewer-cats` | Argues *where Cats core typeclasses & `cats.syntax` simplify boilerplate* |
| `reviewer-effects` | `reviewer-effects` | Argues *what effect type fits — `Future` vs Cats Effect `IO` vs ZIO — and the migration shape* |
| `codebase-synthesizer` | `codebase-synthesizer` | Re-orders all proposals into a single `roadmap.md`, ranked by value/effort |

All six run under the **`gemma-local`** harness-config — no API keys, no
cost. Swap to `claude` per template by editing `default_harness_config:`
in `scion-agent.yaml`.

## Workspace layout

```
/workspace/
├── code/                       # full git clone of the target repo
├── reviews/
│   ├── iron/proposal.md
│   ├── syntax/proposal.md
│   ├── cats/proposal.md
│   ├── effects/proposal.md
│   └── roadmap.md              ← the deliverable
└── review-context.md
```

## Per-review cycle

1. **Stage.** The host driver `scripts/review-codebase.sh` runs
   `gh auth status`, `gh repo view`, then `gh repo clone <repo>` into
   `/workspace/code/`. Private repos work because `gh` carries the user's auth.
2. **Coordinate.** The coordinator confirms the clone, gathers a quick
   summary (branch, head commit, Scala-file count), and writes
   `/workspace/review-context.md` pinning the proposal-output contract
   and the empty-dimension rule.
3. **Dispatch.** Coordinator spawns the four specialists in parallel
   (each with `--notify`).
4. **Argue.** Each specialist reads `/workspace/code/`, picks **one
   most-impactful proposal** for its dimension (not a checklist walk),
   and writes a proposal containing: thesis, 2–6 evidence citations,
   before/after sketch, **S/M/L** effort estimate, risk note, and a
   recommendation (**adopt / adopt-incrementally / defer / reject**).
5. **Synthesize.** Coordinator polls for all four `proposal.md` files,
   then spawns the synthesizer. The synthesizer **only re-orders** the
   proposals — it never invents new recommendations. Items marked
   `defer` / `reject` go into a "Considered and deferred" tail section.
6. **Done.** Coordinator broadcasts `*** ROADMAP READY ***` and exits.

## Effort & recommendation taxonomy

- **Effort**: `S` (hours) / `M` (days) / `L` (weeks).
- **Recommendation**: `adopt` / `adopt-incrementally` / `defer` / `reject`.

The synthesizer **preserves** the specialist's effort estimate and
recommendation — no re-grading across the seam. Ranking is by
**value / effort, with quick wins first**: an `S`-effort `adopt` that
catches real bugs ranks above an `L`-effort `adopt-incrementally` even
if the latter has higher long-term value.

## Empty dimensions

If a specialist's dimension genuinely doesn't apply (e.g. an effects
review on a pure-data library), it still writes a proposal — body
exactly:

```
# <Dimension> Proposal

## Thesis

there is nothing to review
```

The file's existence is the signal to the coordinator. The synthesizer
collapses clean dimensions into a "Clean dimensions" list. If all four
return clean, the roadmap is one line: `there is nothing to review`.

## Running it

From the repo root:

```bash
# Default target (private — danbills/ansible-scala).
make codebase-reviewer-demo

# Any other repo:
bash examples/scala3-codebase-reviewer/scripts/review-codebase.sh \
  --repo owner/repo

# Specific branch:
bash examples/scala3-codebase-reviewer/scripts/review-codebase.sh \
  --repo owner/repo --branch feature/x
```

Requires `gh` authenticated against the target repo. The driver fails
fast if `gh auth status` or `gh repo view` reports an issue.

## How this differs from the PR reviewer

| Aspect | PR Reviewer | Codebase Reviewer |
|---|---|---|
| Input | Single PR diff + post-change file content | Full git clone of the repo |
| Specialist's job | Walk a checklist, find nits | Argue one thesis about the codebase as a whole |
| Output shape | Findings grouped by **Critical/Moderate/Minor** | Single ranked roadmap, **S/M/L** effort |
| Synthesizer | Merges findings, dedupes, groups by severity | Re-orders proposals by value/effort, never invents |

## Non-goals for v1

- No incremental / re-review mode (re-running produces a fresh roadmap, not a diff).
- No GitHub write-back — the roadmap stays in the workspace.
- No build-tool, test-strategy, or documentation specialists yet.
- No reviewer-of-reviewers (Thorne-style gate). Specialist proposals flow straight to the synthesizer.
- No automatic application of recommendations. The roadmap is advisory.
- Scala 3 + Iron + Cats prompts only — generalizing means swapping each specialist's `system-prompt.md`.
