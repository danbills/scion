---
title: "Codebase Reviewer: Post-Run Analysis"
description: Lessons learned, synthesizer output analysis, and prompt improvement recommendations from the first successful end-to-end run of the Scala 3 codebase reviewer demo.
---

First successful end-to-end run: **2026-04-15**, target repo
`danbills/ansible-scala` (172 Scala files). Six agents produced a ranked
modernization roadmap in **4 minutes 35 seconds**. This page captures
what worked, what broke, and how the specialist prompts should evolve.

## Run summary

| Event | Time (ET) | Delta |
|---|---|---|
| Coordinator started (claude/sonnet) | 20:10:36 | 0s |
| 4 specialists spawned (gemma-local) | 20:11:01–03 | +25–27s |
| Broadcast `=== CODEBASE REVIEW ===` | 20:11:13 | +37s |
| All 4 proposals written | ~20:12:46 | ~+2m 10s |
| Synthesizer spawned | 20:13:16 | +2m 40s |
| Broadcast `*** ROADMAP READY ***` | 20:15:11 | **+4m 35s** |

**Configuration**: coordinator on Claude Sonnet 4.6 via Anthropic API;
four specialists + synthesizer on Gemma 4 26B via llama.cpp
(`gemma-local` harness, 128k context window). Hub-native grove
(`398f1038-93b0-4c1a-ad12-5444dfa25ba6`), shared `/workspace/` mount.
Five containers running concurrently at peak. Three attempts were needed
before the successful run (failures documented below).

## Failure modes and lessons learned

### 1. Gemma 4 26B cannot reliably dispatch

Two attempts with Gemma as coordinator failed. Despite explicit
instructions ("The ONLY delegation mechanism is `scion start`", the
opencode `task` tool disabled), Gemma narrated "deploying specialists" as
prose instead of shelling out. It invented specialist personas ("Type
Architect", "DSL Designer") that didn't match any template name, then
declared the review complete.

**Fix**: the coordinator must run on a model that reliably uses tools.
Claude Sonnet worked first try — confirmed inputs, wrote
`review-context.md`, issued four `scion start` commands, polled for
proposals, spawned the synthesizer, and broadcast completion, all
exactly per `agents.md`.

**Implication for prompt design**: dispatcher-role agents need strong
tool-use capability. Narrative-heavy models will "play act" delegation
rather than executing it. If cost is a concern, use a capable model for
the coordinator and a cheaper model for the leaf specialists.

### 2. In-container `gh repo clone` fails on token passthrough

The original design had the coordinator clone the target repo into
`/workspace/code/` at startup using `gh repo clone`. The `GITHUB_TOKEN`
grove secret was correctly injected as an env var, but:

- `gh` prefers `GH_TOKEN` over `GITHUB_TOKEN` in some versions
- Gemma didn't follow the `GH_TOKEN="$GITHUB_TOKEN" gh repo clone ...`
  instruction from `agents.md`
- The error: `fatal: could not read Username for 'https://github.com': No such device or address`

Manual test confirmed `GH_TOKEN="$GITHUB_TOKEN" gh repo clone` works
inside the container. The model simply didn't pass the env var.

**Fix**: the driver pre-clones on the host (where `gh auth` is already
configured) into `~/.scion/groves/<slug>/code/` before starting the
coordinator. Simpler, more robust, removes auth complexity from the
agent entirely.

### 3. `scion templates delete` removes local files

Running `scion templates delete codebase-reviewer-coordinator -y`
deleted both the hub copy AND the local
`.scion/templates/codebase-reviewer-coordinator/` directory. This
destroyed the coordinator template mid-session.

Recovery was possible from
`~/.scion/storage/local/templates/groves/<grove-id>/` cache on the
broker host.

**Recommendation**: commit templates to git before iterating. The
`examples/scala3-codebase-reviewer/` directory was untracked during
development, so there was no safety net.

### 4. Harness config is frozen at agent creation

Changing `default_harness_config: claude` in `scion-agent.yaml` and
re-syncing the template did NOT affect already-existing agents. `scion
list` continued to show `gemma-local-dispatcher` and the container image
remained `scion-opencode:latest`. The `--harness-config claude` CLI flag
on `scion start` also did not override.

**Fix**: `scion delete` the agent, then fresh `scion start`. The harness
config is resolved once at creation time and baked into the agent record.

### 5. Hub endpoint must be explicit in the driver

Despite `hub.endpoint: http://localhost:8080` being set in
`.scion/settings.yaml`, the CLI frequently failed with "Hub is enabled
but no endpoint configured." Root cause unclear — possibly the CLI reads
from the parent scion repo's `.scion/settings.yaml` rather than the
demo grove's.

**Fix**: `export SCION_HUB_ENDPOINT=http://localhost:8080` in the driver
script.

### 6. Tmux capture-pane 502s on re-created specialists

When the coordinator re-created specialists mid-run (after detecting
they'd stopped without writing proposals), the broker failed
`tmux capture-pane` on the new containers (exit 125). This is a race
between container startup and tmux session creation. Self-healed on the
third clean run (all agents started fresh).

---

## Synthesizer output analysis

### The roadmap produced

```
1. Modernize DSL & Improve Ergonomics (High Impact, M effort) — from effects
2. Refinement Type Expansion & Consistency (High Impact, M effort) — from iron
3. Maintain Syntax Standards (High Impact, S effort) — from syntax
4. Enhance Functional Abstractions (Medium Impact, S effort) — from cats
```

### What the synthesizer did well

- **Faithfully summarized** each proposal's core thesis without inventing
  new recommendations — the prompt's "You do NOT invent new
  recommendations" constraint held.
- **Preserved effort estimates** from specialists (no re-grading across
  the seam).
- **Concise**: the roadmap is scannable in 30 seconds. Four ranked items,
  each with originating dimension and a 2–3 line summary.

### What the synthesizer got wrong

**Ranking violated its own stated principle.** The synthesizer prompt
says "value / effort, with quick wins first: an S-effort adopt proposal
ranks above an L-effort adopt-incrementally." But the roadmap ranks
two M-effort items (#1 effects, #2 iron) above an S-effort High Impact
item (#3 syntax). Syntax should rank #2 or higher by the stated rule.

**Missing required structural elements:**

| Required by prompt | Present in output? |
|---|---|
| "Why now" line per ranked item | No |
| Pointer to source proposal file | No |
| "Considered and deferred" section | No (no defer/reject items existed, but section should appear as empty) |
| "Clean dimensions" list | No (correct omission — all 4 had findings — but no explicit note) |
| Summary paragraph (3–5 sentences) | No (jumps straight to ranked list) |

The synthesizer followed the spirit (rank and summarize) but not the
letter (structural template). This is consistent with the specialists'
template compliance issues — Gemma 4 26B on 128k context tends to
capture intent while dropping structural constraints.

---

## Specialist prompt improvement recommendations

### reviewer-iron (refinement types)

**What worked**: correctly identified the existing `RefinedTypes.scala` as
strong prior art, enumerated the full constraint catalog (Port,
Percentage, NonEmptyString, etc.), noted the circe integration via
`io.github.iltotore.iron.circe.given`. Recommendation of
`adopt-incrementally` is appropriate for a codebase that already uses
Iron heavily.

**What to improve**:

- **No file:line citations.** The system prompt asks for "2–6 evidence
  citations from `/workspace/code/`" but the proposal only references
  packages ("in `RefinedTypes.scala`"). Add to `agents.md`: *"Every
  evidence claim MUST include at least one
  `src/main/scala/path/File.scala:NN` citation. If you cannot cite a
  line number, the claim is too vague."*
- **No before/after sketch.** Required by the proposal template but
  absent. Make non-optional: *"Include a concrete before/after code
  snippet of at least 5 lines."*
- **No risk section.** Missing entirely — add as a required heading.
- **Didn't acknowledge the codebase is already a model Iron user.** The
  proposal frames recommendations as if Iron is being newly adopted.
  Add to the system prompt: *"If the codebase already uses your
  dimension heavily, lead with that finding. Your thesis should be about
  the NEXT level of adoption, not the initial one."*

### reviewer-syntax (Scala 3 idioms)

**What worked**: correctly identified the codebase is already deeply
Scala 3 idiomatic — enums as ADTs, given/using, extension methods,
refinement types. The `adopt` recommendation correctly means "keep doing
what you're doing."

**What to improve**:

- **Describes current state, doesn't argue for change.** The system
  prompt says "argue a thesis" but the thesis is "the codebase is
  already good." Add: *"If the codebase already follows your dimension's
  best practices, your thesis is: what is the NEXT frontier? What
  Scala 3 feature is the codebase NOT yet using that would have the
  highest leverage?"*
- **Effort format wrong.** States `S (Hours)` instead of the exact
  `S | M | L` taxonomy. Enforce in the template.
- **No file:line citations.** Same fix as iron.
- **Handwavy enhancement.** The "Minor Enhancement" about export surface
  area cites no specific file or symbol. Require concrete evidence.

### reviewer-cats (typeclasses and syntax)

**What worked**: deep, accurate analysis of the Free Monad architecture,
EitherK coproduct, Writer-based task accumulation. Correctly identified
cats is already deeply integrated and the codebase is a "model example."

**What to improve**:

- **Describes current state more than arguing for change.** The
  "Potential Improvements" section is only 2 vague bullet points
  ("ensure boundary is explicit", "could benefit from more explicit
  usage"). Require: *"Your proposal must contain at least one concrete
  refactoring with a before/after sketch."*
- **Scope violation.** The system prompt says "Cats Effect (IO) is OUT of
  scope — that's reviewer-effects" but the proposal discusses `IO` and
  `IOApp` at length. Add a harder gate: *"If you mention
  `cats.effect.IO`, `Resource`, `IOApp`, or effect-system concerns, you
  are out of scope. Delete that paragraph."*
- **No effort estimate.** The synthesizer had to infer "Low Effort."
  Require `## Effort` as a heading.
- **No risk section.**

### reviewer-effects (effect system choice)

**What worked**: best proposal of the four. Thorough analysis of the Free
Monad DSL architecture, identified the `noCloudInterpreter`
runtime-error antipattern, concrete suggestions (extension methods for
smart constructors, coproduct simplification, compile-time platform
safety).

**What to improve**:

- **Didn't answer the primary question.** The system prompt asks the
  specialist to argue for a specific effect type (`Future` vs `IO` vs
  `ZIO`) but the proposal argues about DSL ergonomics instead. The
  codebase uses `cats.effect.IO` already — the specialist should have
  led with "IO is the right choice, here's the evidence." Reframe:
  *"First verdict: is the current effect type the right one? Then:
  what's the highest-leverage improvement to effect discipline?"*
- **No file:line citations.** Good inline evidence but no line numbers.
- **Overlap with cats reviewer** on EitherK/coproduct analysis. Mirror
  the scope boundary: *"Free Monad structure and coproduct design are
  shared concerns; focus on the EFFECT side (IO/Future/ZIO choice,
  resource management, concurrency, error model), not the algebra
  design."*

### Cross-cutting recommendations

1. **Enforce the proposal template structurally.** All 4 proposals
   deviated from the required template (missing Risk, missing
   before/after, missing file:line citations). Change `agents.md` to
   include the proposal template as a checklist: *"Your proposal will be
   rejected if any of these sections are empty: Thesis, Evidence (with
   file:line), Before/After sketch, Effort, Risk, Recommendation."*

2. **Add a self-check step.** Before `sciontool status task_completed`,
   require: *"Re-read your proposal. Verify: (a) at least 2 file:line
   citations, (b) before/after sketch present, (c) effort S/M/L stated,
   (d) recommendation stated, (e) risk section present. If any missing,
   fix before completing."*

3. **Sharpen scope boundaries.** Cats and effects overlapped heavily on
   Free Monad / EitherK analysis. Add mutual exclusion clauses to each
   specialist's system prompt.

4. **Synthesizer needs structural enforcement too.** The roadmap omitted
   "Why now" lines, source-file pointers, and the "Considered and
   deferred" section. Add the same self-check pattern.

---

## Architecture decisions validated

**Hub-native grove pattern works.** Shared `/workspace/` via
`~/.scion/groves/<slug>/` mounted into all containers — no git clones,
no sync, instant visibility. This is the right primitive for multi-agent
file handoff. Matches the
[Athenaeum coordination pattern](/scion/patterns/athenaeum-coordination/).

**Mixed harness strategy works.** Coordinator on claude/sonnet (needs
reliable tool use), specialists on gemma-local (bulk analysis, no API
cost). The `default_harness_config` per-template model supports this
cleanly — each `scion-agent.yaml` names its own harness.

**Driver pre-clone is simpler than in-container clone.** Host-side
`gh repo clone` avoids auth passthrough complexity entirely. The
coordinator's job is coordination, not devops.

**128k context window was sufficient.** Specialists analyzed 172 Scala
files within the window. The prior 256k setting was unnecessary for this
codebase size. For larger repos (500+ files), may need to revisit.

**Gemma 4 26B is viable for leaf specialists but not dispatchers.**
All four specialists produced substantive, accurate proposals (even if
structurally incomplete). The model correctly explored build files,
identified library usage patterns, and wrote coherent analyses. It fails
only on multi-step procedural tasks requiring reliable tool invocation.
