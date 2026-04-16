---
title: Recursive Metacognition (DISCIPL-style Self-Steering)
description: A speculative design note on letting a Planner agent write the inference program that cheap Follower agents execute — inspired by Grand et al., "Self-Steering Language Models."
---

A design-space note, not a roadmap. Anchored in Grand et al., *Self-Steering
Language Models* (MIT / Yale, 2025) — the DISCIPL framework.
See [github.com/gabegrand/self-steering](https://github.com/gabegrand/self-steering)
for the paper's reference implementation.

## The DISCIPL idea in one paragraph

A capable **Planner** language model generates an *inference program* — Python
in the `llamppl` probabilistic-programming framework — that tells a population
of cheap **Follower** language models how to search the solution space for a
given task. The Planner writes `sample`, `observe`, and scoring operations;
the Followers execute them in parallel under a sequential-Monte-Carlo engine
that culls low-scoring partial generations and reallocates compute to
promising ones. On constrained-generation tasks, a small Follower
(Llama-3.2-1B, Qwen3-1.7B) running under a DISCIPL program written by GPT-4o
matches or beats o1 running alone. The Planner runs once; the Followers run
many times in parallel. The inference procedure is *generated per-task* rather
than hand-engineered.

## Why this shape matters for scion

Scion's existing coordination pattern — see
[Scala 3 Codebase Reviewer](/scion/patterns/scala3-codebase-reviewer/) and
[Athenaeum Coordination](/scion/patterns/athenaeum-coordination/) — already
has a Planner/Follower shape, but only implicitly:

| DISCIPL                    | Scion today                                     |
|----------------------------|-------------------------------------------------|
| Planner LM                 | Coordinator agent, driven by a hand-written template |
| Inference program in Python | Static `agents.md` + `system-prompt.md` + specialist spawn list |
| Follower LM population     | Specialist agents spawned via `scion start`     |
| SMC engine, score-based culling | No analog — specialists run once, outputs all survive |
| Error → re-prompt Planner  | Coordinator retries its own bash but does not rewrite its decomposition |

The interesting question is what scion would look like if the coordinator's
job were to *generate* an inference program — a concrete specialist-spawn
plan, scoring rubric, and merge rule — on the fly for a given repo and task,
rather than execute a hand-written template.

## Four directions worth sketching

### 1. Planner-written coordinator program

Today the coordinator template is static: four specialists, fixed roles,
one synthesizer. A Planner-class coordinator (Claude/GPT-4o) could read a
repo's `build.sbt`, `README.md`, and git log once, then emit the specialist
topology — which dimensions to review, how many specialists per dimension,
what each is allowed to cite — as structured output that scion turns into
`scion start` calls. Specialists stay small and cheap.

### 2. Score-based culling of parallel specialist drafts

Run **N** reviewer-iron instances in parallel instead of one. Score each
draft with a cheap function (citation count × declared confidence × overlap
with peer specialists' evidence) and feed only the top-**k** to the
synthesizer. This imports DISCIPL's culling mechanism without needing a
probabilistic-programming runtime — the score function is just a shell
script or a scoring agent. The confidence-score addition to the Scala 3
reviewer's output contract (`Confidence: high | medium | low` plus a
"Strongest argument against" sentence, defined in the coordinator's shared
`review-context.md`) is a prerequisite: specialists must self-report
confidence and counter-arguments for culling to have signal.

### 3. Retry-with-trace at the plan level

DISCIPL's outer loop re-prompts the Planner with the error + traceback when
an inference program crashes. Scion's coordinator today retries individual
`scion start` calls but never revises the plan. When a specialist produces
an empty or malformed proposal, the coordinator could hand the failure trace
back to a Planner-class model and ask it to rewrite the specialist's prompt
or split the dimension into finer sub-dimensions. This is the closest
available mapping of DISCIPL's recursive-correction loop onto multi-agent
coordination.

### 4. Honest Planner/Follower model partitioning

The
[codebase reviewer post-run writeup](/scion/patterns/scala3-codebase-reviewer-postmortem/)
found that Gemma 4 26B can dispatch via bash but drifts into content-generation
mode at phase boundaries — the mode-switch failure. DISCIPL gives that
finding a name: a Follower is not a Planner. The coordinator is structurally
Planner work and should run on a Planner-class model; specialists and the
synthesizer are closer to Followers and can be smaller. The current
"everything on `gemma-local`" configuration inverts the partition DISCIPL
assumes.

## Smallest experiment worth running

Pick direction **2** (score-based culling) as the first experiment — it
requires no Planner-written code, composes with the existing coordinator
template, and reuses the confidence field we're already adding to the
specialist output contract.

Procedure:

1. Spawn **3** reviewer-iron instances with different seeds / model
   temperatures (or different harness configs).
2. Define a score function: `score = n_citations * confidence_weight` where
   `confidence_weight ∈ {high: 1.0, medium: 0.5, low: 0.2}`, with a penalty
   if the "Strongest argument against" field is empty or generic.
3. The synthesizer reads all three, references the highest-scoring draft as
   the authoritative proposal for that dimension, and cross-checks its claims
   against the runners-up — if a claim only appears in the top draft, flag it
   as low-consensus.

This is the minimum shape that tests the DISCIPL hypothesis in scion's
idiom: *does orchestrating multiple Follower drafts through a scoring gate
yield a better roadmap than a single draft, for the same wall-clock time?*

## Open questions

- **Who writes the score function?** DISCIPL lets the Planner write scoring
  inside the inference program. A first scion pass will hand-write the
  scorer; the natural next step is to have the coordinator (Planner) emit a
  score function per-task, which starts to look recursive.
- **Token budget vs. wall-clock budget.** DISCIPL's budget is token-count
  because Followers are fast. Scion's bottleneck is container startup and
  filesystem coordination, not tokens. The culling math changes accordingly.
- **Is LLAMPPL the right substrate?** The MIT group's framework targets
  token-level SMC inside one model. Scion operates at the agent level —
  seconds-to-minutes per sample, not milliseconds. A useful question is
  whether agent-level SMC is the same algorithm at different time scales,
  or whether it needs a different coordination primitive entirely.
