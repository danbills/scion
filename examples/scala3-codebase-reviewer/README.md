# Scala 3 Codebase Reviewer Demo

A whole-codebase modernization reviewer. Same multi-agent shape as the
[PR reviewer](../pr-reviewer/), but the input is a **full repo clone**
(no PR, no diff) and the output is **a single ranked roadmap** of opinionated
improvement directions, not file:line findings.

## What it does

For a given Scala 3 repo (default: `danbills/ansible-scala`), six agents run:

1. **codebase-reviewer-coordinator** — confirms the clone, writes
   `review-context.md`, spawns specialists, waits, spawns the synthesizer.
2. **reviewer-iron** — argues *where Iron refinement types
   (`io.github.iltotore.iron`) would replace stringly/intly-typed values*.
3. **reviewer-syntax** — argues *how far this codebase should go on
   significant indentation, `enum`, `using`/`given`, `extension`, `derives`*.
4. **reviewer-cats** — argues *where Cats core typeclasses and
   `cats.syntax` would simplify hand-rolled boilerplate*.
5. **reviewer-effects** — argues *what effect type fits this codebase*
   (`Future` vs Cats Effect `IO` vs ZIO) and what a phased migration looks like.
6. **codebase-synthesizer** — merges all four proposals into a single
   ranked roadmap (`reviews/roadmap.md`), ordered by **value / effort with
   quick wins first**.

Each specialist argues **one thesis** about the codebase as a whole,
with 2–6 evidence citations from `/workspace/code/`, a sample
before/after sketch, an effort estimate (**S / M / L**), and a
recommendation (**adopt / adopt-incrementally / defer / reject**).

## Workspace layout

```
/workspace/
├── code/                       # full clone of the target repo
├── reviews/
│   ├── iron/proposal.md
│   ├── syntax/proposal.md
│   ├── cats/proposal.md
│   ├── effects/proposal.md
│   └── roadmap.md              ← the deliverable
└── review-context.md
```

## Harness

All six agents use **`gemma-local`** — a local llama.cpp server, no API
keys. The synthesizer in particular benefits from a stronger model;
swap to `claude` by editing `default_harness_config:` in
`codebase-synthesizer/scion-agent.yaml`.

## Run it

The default target is `danbills/ansible-scala` (private):

```bash
# From the repo root.
make codebase-reviewer-demo

# Or against any other repo:
bash examples/scala3-codebase-reviewer/scripts/review-codebase.sh \
  --repo owner/some-scala3-repo

# Specific branch:
bash examples/scala3-codebase-reviewer/scripts/review-codebase.sh \
  --repo owner/some-scala3-repo --branch feature/x
```

Requires `gh` authenticated against the target repo (the driver runs
`gh auth status` and `gh repo view` as a precheck).

Output lands under `~/.scion/groves/scala3-codebase-reviewer/reviews/`
(that directory is the shared `/workspace/` mount — see **How it works**).

## How this differs from the PR reviewer

| Aspect | PR Reviewer | Codebase Reviewer |
|---|---|---|
| Input | Single PR diff + post-change file content | Full git clone of the repo |
| Specialists' job | Walk a checklist, find nits | Argue one thesis about the codebase as a whole |
| Output shape | Findings grouped by **Critical/Moderate/Minor** | Single ranked roadmap, **S/M/L** effort |
| Synthesizer | Merges findings, dedupes, groups by severity | Re-orders proposals by value/effort |
| Re-runnable? | Different PR → different review | Re-running on the same repo gives a fresh roadmap (no diff mode v1) |

## Empty dimensions

If a specialist's dimension genuinely doesn't apply (e.g. effects review
on a pure-data library), it writes a proposal whose body is exactly
`there is nothing to review` under the dimension heading. The
synthesizer collapses these into a "Clean dimensions" list. If all
four are clean, the roadmap itself is one line.

## Scope

Scala 3 + Iron + Cats prompts only. Generalizing means swapping each
specialist's `system-prompt.md`. Out of scope for v1: build-tooling,
test-strategy, and documentation specialists; GitHub write-back; an
incremental "what changed since last review" mode.

## How it works

This grove is **hub-native** (no git remote) — the same pattern
`scion-athenaeum` uses. All agents in the grove mount the same
`~/.scion/groves/scala3-codebase-reviewer/` directory on the broker as
`/workspace/`, so they share state directly:

1. The driver (`scripts/review-codebase.sh`) `gh repo clone`s the target
   into `~/.scion/groves/scala3-codebase-reviewer/code/` **before**
   launching the coordinator. Pre-cloning on the host keeps the
   coordinator's job purely orchestrational.
2. Coordinator starts, sees `/workspace/code/` already populated,
   writes `/workspace/review-context.md`, and spawns the four
   specialists via `scion start`.
3. Each specialist reviews `/workspace/code/` and writes its proposal
   to `/workspace/reviews/<dim>/proposal.md`. `scion message --broadcast`
   carries notifications; the actual artifacts are files.
4. Coordinator polls for the four proposals, spawns the synthesizer,
   which writes `/workspace/reviews/roadmap.md`.

All agents are visible in the web UI at http://localhost:8080.

## Setup

First time using this grove on a fresh clone:

```bash
# 1. scion server must be running with hub enabled.
scion server start
scion config set hub.endpoint http://localhost:8080
scion hub enable

# 2. Link the grove and sync templates to the hub.
cd examples/scala3-codebase-reviewer
scion hub link
scion templates sync

# 3. The gemma-local harness config must exist at
#    ~/.scion/harness-configs/gemma-local/config.yaml, pointing at
#    your llama.cpp endpoint. See scion-athenaeum for an example.
#    To use claude instead, edit default_harness_config in each
#    template's scion-agent.yaml.

# 4. gh must be authenticated for the target repo.
gh auth status
```

## Coordinator as Gemma: mode-switch caveat

Gemma 4 26B can drive the coordinator, but it has a mode-switch failure
at the synthesis phase — it will start *writing* the roadmap itself
instead of dispatching the synthesizer. The coordinator template's
system prompt is tuned against this (anti-narration + few-shot bash
examples), but it's not bulletproof. See the
[postmortem](../../docs-site/src/content/docs/patterns/scala3-codebase-reviewer-postmortem.md)
for what went wrong and the prompt engineering that followed. The
dispatch test harness lives at `scripts/test-gemma-dispatch.sh` with
variants under `scripts/prompt-variants/`; run it with
`bash scripts/test-gemma-dispatch.sh` to iterate on coordinator prompts.
