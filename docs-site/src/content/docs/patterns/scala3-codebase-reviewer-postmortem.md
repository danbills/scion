---
title: "Codebase Reviewer: Post-Run Analysis"
description: Lessons learned from two end-to-end runs of the Scala 3 codebase reviewer demo — Run 1 with Claude coordinator, Run 2 with Gemma 4 26B coordinator.
---

Two runs against `danbills/ansible-scala` (172 Scala files). Run 1 used
Claude/Sonnet as coordinator with Gemma specialists. Run 2 attempted
all-Gemma (coordinator + specialists). Both produced roadmaps. Their
different failure modes reveal where local LLMs can and cannot replace
API models in multi-agent orchestration.

---

## Run 1: Claude coordinator + Gemma specialists

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

---

## Run 2: All-Gemma (Gemma coordinator + Gemma specialists)

Second run: **2026-04-15 21:19 ET**, same target repo. Coordinator
switched from Claude/Sonnet to Gemma 4 26B (`gemma-local` harness) with
a rewritten system prompt emphasizing imperative bash-only dispatch.

### Run 2 timeline

| Event | Time (ET) | Delta |
|---|---|---|
| Coordinator started (gemma-local) | 21:19:36 | 0s |
| 4 specialists spawned (gemma-local) | 21:19:58–21:20:14 | +22–38s |
| All 4 proposals written | ~21:21:30 | ~+1m 54s |
| Coordinator stalled at synthesis phase | 21:22:27 | +2m 51s |
| Coordinator generated own roadmap (wrong path) | 21:22:27 | +2m 51s |

**Configuration**: all agents on Gemma 4 26B via llama.cpp (`gemma-local`
harness, 128k context). Same hub-native grove, shared `/workspace/`.
Coordinator system prompt rewritten to imperative "terminal operator"
framing per variant 07 test results.

### What worked

**Specialist dispatch was flawless.** The rewritten coordinator prompt
("You are a dispatcher. You accomplish tasks EXCLUSIVELY by running shell
commands via the bash tool.") succeeded where Run 1's original Gemma
coordinator failed completely. All four `scion start` commands executed
correctly via bash tool in ~38s. Error recovery worked — when the first
`scion start` failed (no `--notify` in non-hub mode), Gemma retried
with modified flags.

**Specialist proposals were substantive.** 161 lines total across 4
proposals (iron: 62, syntax: 43, cats: 30, effects: 26). Proposals
engaged the actual codebase structure (Free Monad DSL, Iron refinement
types, given/using syntax) rather than producing generic advice.

**The dispatcher framing validated.** Seven prompt variants (01–07) all
passed the bash-tool-use test. The terminal-operator identity + numbered
step list was the winning pattern, with variant 07 (coordinator chain)
dispatching all 4 specialists + recovering from errors in 24s.

### What failed

**Synthesizer was never dispatched.** The coordinator's agents.md
specified Step 6: `scion start codebase-synthesizer`. Instead, Gemma:

1. Used opencode's internal subagent feature (visible as "5 toolcalls ·
   13.2s" in the tmux scrollback) to create its own synthesizers
2. Generated synthesis prose directly — a "Summary of Findings" with
   phased approach, effort estimates, and ranked recommendations
3. Wrote output to `/workspace/proposals/` (invented path) instead of
   `/workspace/reviews/` (path specified in agents.md)
4. Produced a 57-line roadmap at `/workspace/proposals/roadmap.md` that
   is structurally decent but lives at the wrong location
5. Never called `sciontool status task_completed`

**Specialist containers never exited.** All 4 specialists wrote their
proposals but remained running (6+ minutes uptime at observation).
The specialists didn't call `sciontool status task_completed` either —
consistent with Gemma's weak task-lifecycle awareness.

**Wrong output paths.** The coordinator invented `/workspace/proposals/`
with dimensions named `syntax`, `types`, `architecture`, `testing` —
none of which match the actual specialist names (`iron`, `syntax`,
`cats`, `effects`). It appears to have hallucinated its own specialist
taxonomy rather than reading the real proposals at `/workspace/reviews/`.

### Root cause: the mode-switch problem

Gemma 4 26B exhibits a **mode switch** at phase boundaries. When the
task is "execute N similar `scion start` commands," Gemma stays in
bash-execution mode and chains them correctly. But when the task
transitions to a conceptually different phase — "now spawn the
synthesizer" — Gemma drops out of execution mode and into its default
content-generation mode.

Contributing factors:

- **The word "synthesizer" triggers generation.** Unlike "reviewer"
  (which implies reading), "synthesizer" implies producing output —
  exactly what an LLM is trained to do. The model takes the synthesis
  task as its own rather than delegating it.
- **Seeing proposal file paths triggers reading.** When the task
  mentions that proposals exist, Gemma's instinct is to read and
  summarize them rather than spawning another agent to do so.
- **Phase boundary breaks the execution loop.** The 4 specialist
  dispatches feel like a complete unit. The synthesizer dispatch after a
  conceptual "wait for proposals" gap breaks the momentum of the
  bash-execution loop.
- **opencode's internal subagent feature provides an easier path.**
  The model discovered it could create subagents within its own session
  rather than using `scion start`. This is a harness-specific escape
  hatch that bypasses the dispatcher constraint.

### Prompt engineering test results

A test harness (`scripts/test-gemma-dispatch.sh`) was built to iterate on
prompt variants. System prompt + task pairs are tested against
`opencode run --format json` with JSONL parsing to detect bash tool
invocations.

**Phase 1 results (specialist dispatch, variants 01–07):**

| Variant | Strategy | Result | Time |
|---|---|---|---|
| 01-baseline | Existing dispatcher prompt | PASS (3 scion-start) | 19s |
| 02-terminal-operator | "Execute, never describe" | PASS (3 scion-start) | 22s |
| 03-literal-command | Minimal context, raw command | PASS (1 scion-start) | 6s |
| 04-few-shot | Worked example in system prompt | PASS (1 scion-start) | 9s |
| 05-command-only | Forbid text responses | PASS (2 scion-start) | 7s |
| 06-step-by-step | Numbered checklist | PASS (1 scion-start) | 6s |
| 07-coordinator-chain | 4 specialists in sequence | PASS (5 scion-start) | 24s |

All 7 passed. This confirmed Gemma CAN execute bash tool calls reliably.
The failure in the real run is not about tool-use capability but about
maintaining execution mode across phase boundaries.

**Phase 2 (synthesizer dispatch, variants 08–13):** tests in progress.
These isolate the specific failure: can Gemma dispatch `scion start
codebase-synthesizer` when the task involves synthesis-shaped content?

### Recommendations

**1. Flatten the coordinator into a single numbered command list.**
Instead of separate "dispatch phase" and "synthesis phase" with a
polling gap, give the coordinator all 5 `scion start` commands (4
specialists + 1 synthesizer) as a single numbered list. Variant 12
(`12-synth-full-chain`) tests this approach. The poll-for-proposals step
can be replaced by a fixed delay or moved to the synthesizer itself.

**2. Consider splitting the coordinator into two agents.** A
"dispatcher" agent issues the 4 specialist `scion start` commands and
terminates. A separate "synth-trigger" agent (possibly on a timer or
watching for file existence) issues `scion start codebase-synthesizer`.
This avoids the mode-switch problem entirely by never asking Gemma to
do two conceptually different things in one session.

**3. Add explicit anti-generation guardrails.** "You MUST NOT read any
proposal.md file. You MUST NOT write any summary or roadmap content."
Variant 11 tests whether these negative constraints prevent the mode
switch.

**4. Disable opencode internal subagents.** The coordinator discovered
opencode's built-in subagent feature as an alternative to `scion start`.
If possible, disable this feature for dispatcher agents (similar to how
the `task` tool was disabled). This removes the escape hatch.

**5. Accept the mixed-model architecture.** Run 1's mixed strategy
(Claude coordinator + Gemma specialists) worked first try. The
coordinator role is a small fraction of total compute — using an API
model for the 2-minute orchestration task while running 4 bulk-analysis
specialists on a local model is a pragmatic, cost-effective split.

### Comparing the two roadmaps

Both runs produced usable roadmaps, despite different process fidelity:

| Dimension | Run 1 (Claude coord) | Run 2 (Gemma coord) |
|---|---|---|
| Process fidelity | All steps followed per agents.md | Steps 1-4 correct, Steps 5-7 violated |
| Output path | `/workspace/reviews/roadmap.md` (correct) | `/workspace/proposals/roadmap.md` (wrong) |
| Synthesizer agent | Separate scion container | Coordinator generated inline |
| Roadmap length | 4 ranked items, concise | 57 lines, phased approach |
| Recommendation | adopt-incrementally | adopt-incrementally |
| Time to roadmap | 4m 35s | 2m 51s (shorter but broken process) |

Run 2's roadmap is actually more detailed (57 lines vs ~20), with
implementation phases and risk assessment. But it was produced by the
wrong agent (coordinator instead of synthesizer) using the wrong inputs
(hallucinated specialist taxonomy instead of actual proposals). The
content quality demonstrates Gemma's analytical strength; the process
violation demonstrates its orchestration weakness.
