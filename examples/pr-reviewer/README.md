# PR Reviewer Demo

A concrete application of the [Athenaeum coordination pattern](../../docs-site/src/content/docs/patterns/athenaeum-coordination.md)
to something useful: a multi-dimensional code review where each review
dimension is its own specialist agent, and a synthesizer produces the
unified verdict.

## What it does

For a given pull request (fixture or live), four agents run:

1. **reviewer-coordinator** — stages the PR, spawns the specialists, waits
   for findings, spawns the synthesizer, reports completion.
2. **reviewer-types** — Scala 3 type strength (opaque types, Option over
   null, narrow return types, value classes around primitive IDs).
3. **reviewer-idioms** — Scala 3 idiom conformance (`using`/`given`,
   `enum`, `derives`, `scala.jdk.CollectionConverters`, extension methods).
4. **reviewer-semantics** — correctness & effect handling (swallowed
   failures, SQL injection, missing resource closes, racy mutation,
   misused `ExecutionContext`).

Then:

5. **synthesizer** — merges all three findings into `reviews/summary.md`,
   grouped by severity (**Critical / Moderate / Minor**), each finding
   tagged by dimension.

The shared `/workspace/` is the coordination substrate: the coordinator
writes inputs under `pr/`, each specialist writes to its own
`reviews/<dimension>/findings.md`, the synthesizer reads all three.

## Harness

All agents use **`gemma-local`** — a local llama.cpp server, no API keys,
no cost. To swap in a stronger model per template, edit
`default_harness_config:` in each `scion-agent.yaml`
(e.g. to `claude` or `opencode`) — nothing else changes.

## Run the fixture

```bash
# From the repo root.
make pr-reviewer-demo

# Or directly:
bash examples/pr-reviewer/scripts/review-pr.sh --fixture pr-sample-1
```

Expect four files to appear under `examples/pr-reviewer/.work/reviews/`:

```
reviews/types/findings.md
reviews/idioms/findings.md
reviews/semantics/findings.md
reviews/summary.md        ← the deliverable
```

## Run against a live PR

```bash
bash examples/pr-reviewer/scripts/review-pr.sh --pr 42 --repo owner/repo
```

Requires `gh` authenticated against the repo. The driver uses
`gh pr view` + `gh pr diff` + `gh api …/contents` to stage metadata, the
unified diff, and post-change file content (capped at 500 lines per file)
into `/workspace/pr/`.

## Empty PRs

If a PR is docs-only or has nothing in a given dimension to flag, the
specialist still writes its findings file — body exactly
`there is nothing to review` under the dimension heading. The file's
existence is the signal to the coordinator; the synthesizer collapses
clean dimensions into a "Clean dimensions" list rather than showing
empty severity buckets.

## Scope

This is Scala 3-first. The system prompts encode Scala 3 idioms directly.
Generalizing to other languages means swapping each specialist's
`system-prompt.md` — a cheap edit, but out of scope for v1.

## Directory layout

```
examples/pr-reviewer/
├── README.md                  # this file
├── .scion/
│   └── templates/
│       ├── reviewer-coordinator/
│       ├── reviewer-types/
│       ├── reviewer-idioms/
│       ├── reviewer-semantics/
│       └── synthesizer/
├── fixtures/
│   └── pr-sample-1/
│       ├── README.md
│       ├── metadata.json
│       ├── diff.patch
│       └── files/…
└── scripts/
    └── review-pr.sh
```
