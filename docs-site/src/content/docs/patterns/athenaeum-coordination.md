---
title: Athenaeum Coordination Pattern
description: How a shared workspace, structured directories, and a validator agent let multiple Scion agents collaborate on a staged, multi-phase task.
---

This page documents the multi-agent coordination pattern used by the
[Relics of the Athenaeum](https://github.com/danbills/scion-athenaeum) demo.
It is a reference for how to design a grove in which many agents hand off work
to one another across multiple phases, with a dedicated reviewer guarding
quality at each step.

The pattern combines three channels:

1. **Real-time messages** between agents, via `scion message`.
2. **A shared filesystem** (the `/workspace` mount every agent sees) used as a
   structured message queue keyed by directory convention.
3. **Git commits** inside that workspace, as an optional durable ledger of
   "official" updates.

A single orchestrator (the **Game Runner**) owns the script and privately
holds the answer keys. A dedicated validator (**Thorne the Sentinel**)
reviews peer output before the orchestrator accepts it.

## The three coordination channels

| Channel | Good for | Not good for |
|---|---|---|
| `scion message` (direct / broadcast) | Doorbells, short questions, "I'm done", act transitions | Payloads, audit trail, anything you need to re-read later |
| Shared filesystem (`/workspace/...`) | Handing off files, queued work, accumulating per-phase artifacts | Authorship, atomicity across multiple files, cross-broker sync without push/pull |
| Git commits in the workspace | Attribution, history, atomic snapshots, remote mirroring | Fast-churning scratch state (committing every partial would drown real signal) |

Messages are the doorbell. Files are the package. Commits are the signed
receipt.

See [About Workspaces](/scion/advanced-local/workspace/) for how Scion
resolves a grove to a workspace directory, and [Templates & Harnesses](/scion/advanced-local/templates/)
for the template anatomy referenced below.

## The shared `/workspace` substrate

In the stock single-agent flow, Scion gives each agent its own git worktree
(see [About Workspaces](/scion/advanced-local/workspace/)). This pattern
instead arranges for **every agent in the grove to see the same directory**
at `/workspace/`. Two ways to do that:

- Pass `--workspace <dir>` when starting each agent; all of them mount the
  same host directory.
- Declare a `shared_dirs:` entry in the grove's `settings.yaml` (host path is
  bind-mounted under `<workspace>/.scion-volumes/<name>` or
  `/scion-volumes/<name>` inside every container). Athenaeum itself uses a
  single shared workspace mount; `shared_dirs` is the more granular variant
  when you only want *parts* of the state shared.

Whichever route you take, the coordination contract is identical: agents
agree on a **directory convention**, write there, and read from there.

### Directory convention

The Game Runner sets up this layout at startup
(`scion-athenaeum/.scion/templates/game-runner/system-prompt.md:187-205`):

```
/workspace/
├── game-context.md           # living quest state (orchestrator maintains)
├── current-challenge.md      # active phase description (orchestrator writes)
├── challenges/               # inbox from orchestrator → party
│   ├── act-1/
│   ├── act-2a/               # parallel sub-tracks
│   ├── act-2b/
│   ├── act-2c/
│   ├── act-3/
│   ├── act-4/
│   └── act-5/
├── solutions/                # outbox from party → orchestrator
│   ├── act-1/
│   └── ...
├── inventory/                # durable artifacts (recovered fragments)
├── notes/                    # agent-private working notes
├── sprites/                  # task specs + results for ephemeral workers
└── oracle-responses/         # one-off expert answers
```

Two root files act as rendezvous points: `current-challenge.md` (orchestrator
→ everyone: "here's what to work on") and `game-context.md` (running summary
of state).

### The private playbook

The Game Runner's *template* includes a `home/playbook/` directory
(`scion-athenaeum/.scion/templates/game-runner/home/playbook/`). Scion mounts
each template's `home/` into that agent's container home directory, and it is
**not visible to any other agent**. The Game Runner keeps challenge inputs,
answer keys, and escalation rules here:

```
~/playbook/
├── act-1/
│   ├── challenge-1.1/
│   │   ├── challenge.md
│   │   ├── data/             # files to deploy into /workspace
│   │   └── solutions/        # authoritative answer keys (never shared)
│   └── challenge-1.2/
└── ...
```

This is how the orchestrator can hold secrets (solutions, future-act content)
without them leaking into the shared workspace. Every other agent sees only
`/workspace/`; no agent should read other agents' `home/` directories.

## The shared filesystem as a staged message queue

The Game Runner plays back the quest act by act. Each act runs the same
cycle, and the directory structure **is** the protocol.

### Per-act cycle

For act *N*:

1. **Deploy**. The Game Runner copies inputs from its private playbook into
   the shared workspace:
   ```bash
   cp -r ~/playbook/act-N/challenge-X/data/* /workspace/challenges/act-N/
   ```
2. **Announce**. It writes `/workspace/current-challenge.md` (narrative +
   acceptance criteria), then broadcasts a message so running agents
   immediately see the new challenge:
   ```bash
   scion message --broadcast "=== QUEST UPDATE === Act N challenge deployed. See challenges/act-N/ and current-challenge.md."
   ```
3. **Work**. Character agents read `current-challenge.md`, consume inputs
   from `challenges/act-N/`, and write artifacts into `solutions/act-N/`.
   They can coordinate among themselves with direct messages and
   broadcasts while they work.
4. **Signal**. When an agent believes its piece is done, it direct-messages
   the orchestrator: `scion message game-runner "Solution posted to
   solutions/act-N/<file>"`.
5. **Validate**. The Game Runner diffs the submission against the private
   answer key in `~/playbook/act-N/challenge-X/solutions/`, or runs a
   validator script from the same directory. It then broadcasts a verdict
   using a structured marker (`*** EVALUATION RESULT ***`) — PASS, PARTIAL,
   or FAIL.
6. **Advance** (on PASS). The Game Runner updates `game-context.md`, then
   loops back to step 1 for act *N+1*.

The **directory path is the queue address**. `challenges/act-N/` is the inbox
from orchestrator to party; `solutions/act-N/` is the outbox from party to
orchestrator. Agents never negotiate filenames ad-hoc — the convention is the
contract.

This is *pull, not push*. Writing a file does not automatically wake anyone
up. The `scion message` broadcast is the doorbell that tells agents to go
look; the filesystem is the package that was delivered.

### Variations across the five acts

- **Act I (serial)** — single challenge directory, full party collaborates.
- **Act II (parallel split)** — three concurrent sub-tracks keyed by
  subdirectory: `challenges/act-2a/`, `act-2b/`, `act-2c/`. Three sub-teams
  queue work independently without any locking because each team only writes
  under its own letter.
- **Act III (convergence)** — artifacts from 2a/b/c flow back into a single
  `challenges/act-3/` input set; agents read multiple prior outboxes at once.
- **Act IV (layered dependencies)** — three sequential layers, where each
  layer's `solutions/` output becomes the next layer's `challenges/` input.
  The orchestrator gates progression by only deploying layer 2 after
  layer 1 passes.
- **Act V (finale)** — a coordinated ordering across all five characters;
  the shape is still the same, just with more handoffs per act.

### Concurrency notes

The filesystem has no locking. The pattern relies on **one writer per
path** by convention: each character owns its filename prefix within a
`solutions/` directory (e.g., `solutions/act-1/lyra-decoded.txt`), and the
orchestrator is the sole writer of `current-challenge.md` and
`game-context.md`. The Scribe agent is deliberately *read-only* from the
game's perspective — it observes and records in its own journal but does not
edit orchestrator-owned files.

If two agents genuinely need to produce the same artifact together, have one
write a draft and message the other to review; don't co-edit.

## Git commits as the official ledger (optional strengthening)

Athenaeum as shipped does **not** commit act-by-act. The shared workspace is
a working directory — agents write files, the orchestrator reads and
evaluates, and the state of the quest at any moment is just "whatever the
directory looks like right now."

That is fine for a demo. For a longer-lived or audited grove, promote the
`solutions/` handoff to a git commit and you get four things for free:

- **Author attribution.** Each agent has its own git identity (set in the
  task prompt or template env), so `git log solutions/act-3/` tells you
  exactly which agent claimed completion and when.
- **Atomic snapshot.** Staging multiple files and committing them as one
  unit avoids races where a partial write looks like a complete submission.
- **Diff-based review.** The orchestrator (or a human) can review the
  delta, not the final state.
- **Remote mirroring.** When the workspace has an `origin` remote, every
  push shows up in the Scion hub dashboard and on GitHub — useful when the
  grove runs on a different machine than the one you're watching from.

Recommendation for what to commit vs. leave uncommitted:

| Path | Commit? | Why |
|---|---|---|
| `challenges/**` | Yes | Authoritative input snapshot; what the party was asked. |
| `solutions/**` | Yes | The deliverable. Attribution and diff matter. |
| `current-challenge.md`, `game-context.md` | Yes | Quest state transitions. |
| `inventory/**` | Yes | Recovered fragments carry across acts. |
| `notes/**` | No | Working scratch; commit clutter. |
| `sprites/**` | No | Ephemeral task specs + results. |
| `oracle-responses/**` | No | One-shot expert output. |

A minimal `.gitignore` for this pattern:

```gitignore
notes/
sprites/
oracle-responses/
```

When you enable commits, bake the git identity into each agent's template so
it commits as itself, and have the task prompt finish with an explicit
`git add … && git commit … && git push` so the orchestrator can detect the
update by polling `git log` or the remote branch head rather than parsing
broadcasts.

See the [Hub User Guide](/scion/hub-user/git-groves/) for how pushes flow
back through a hub-linked grove.

## Thorne the Sentinel: the validator agent

The Game Runner holds the final answer key, but it is expensive to run
(long reads, careful narrative, LLM time). You do not want the orchestrator
grading every half-finished attempt. **Thorne the Sentinel** is athenaeum's
answer: a dedicated validator that does cheap local review first.

### Role

From `scion-athenaeum/.scion/templates/thorne/agents.md`:

- Reads peer solutions out of `/workspace/solutions/act-N/`.
- Writes validation reports back into the same directory (e.g.
  `solutions/act-N/validation-decode.md`) so any agent can see them.
- Can spawn **Ward Echo** sprites (up to two at a time) to run test suites
  in parallel.
- When satisfied, direct-messages the Game Runner: "Lyra's decode passes my
  checks. Ready for your evaluation."

### What Thorne is *not*

Thorne is **not** the final arbiter. The orchestrator still diffs against
the private answer key before declaring PASS. Thorne is peer review: it
catches obvious bugs, missing fields, and edge-case failures before the
Game Runner spends a turn on them. Think of Thorne as the first-pass
reviewer who vets whether a submission is worth the DM's attention.

### Collaboration patterns

Thorne's `agents.md` establishes explicit relationships with the other
characters:

- **Lyra (build/verify cycle).** Lyra writes algorithms, Thorne writes
  tests, Lyra fixes what fails.
- **Mira (format validation).** Mira transforms data, Thorne validates
  schema conformance and record integrity.
- **Zara (pre-integration).** Thorne validates components before Zara
  integrates them, so integration bugs trace to glue code, not inputs.

Thorne also cannot *create* solutions — the role is explicitly advisory.
This separation matters: if your validator also writes code, you lose the
independence that makes validation meaningful.

### Building a Thorne-style agent in your own grove

The essentials:

1. A dedicated template (`.scion/templates/<validator>/`) with a
   `system-prompt.md` that emphasizes skepticism, coverage, and no
   solution-writing.
2. An `agents.md` that tells the agent:
   - Read from `/workspace/solutions/<phase>/`.
   - Write reports named `validation-<topic>-<author>.md` alongside the
     submissions they review.
   - Message the orchestrator only when a submission passes local review.
3. A small budget of worker sprites (athenaeum allows two) so Thorne can
   run test suites in parallel without monopolizing runtime.

The validator template can use a stronger model than the
solution-producing agents if you want conservative gatekeeping; athenaeum
runs Thorne on Claude Opus while several of the implementers run on faster
Gemini variants.

## Worked example: Act I end-to-end

A single scenario from start to finish, exercising all three channels.

### 0. Startup

The operator launches the Game Runner:

```bash
scion start game-runner --type game-runner \
  "Begin Relics of the Athenaeum quest. Start all character agents, set the scene, and deploy Act I Challenge 1.1."
```

Game Runner spawns the five characters + the Scribe, creates the
`/workspace/` directory skeleton, writes an initial `game-context.md`,
and deploys the first challenge.

### 1. Deploy

```bash
cp -r ~/playbook/act-1/challenge-1.1/data/* /workspace/challenges/act-1/
```

`challenges/act-1/summons.txt` now holds the encoded input.

### 2. Announce

The Game Runner writes `/workspace/current-challenge.md` with the narrative
framing and acceptance criteria, then broadcasts:

```bash
scion message --broadcast \
  "=== QUEST UPDATE === The ancient summons has appeared at challenges/act-1/summons.txt. Decode it to learn your mission."
```

### 3. Work

Lyra reads `summons.txt`, pipelines through base64 → ROT13 → substitution
cipher, and writes the plaintext to
`/workspace/solutions/act-1/decoded-summons.txt`. She broadcasts:

```bash
scion message --broadcast \
  "I've written the decoded summons to solutions/act-1/decoded-summons.txt. Thorne, can you verify?"
```

### 4. Validate (peer)

Thorne reads Lyra's file, runs local sanity checks (does it contain the
expected sentinel phrase? is the length plausible?), and writes
`/workspace/solutions/act-1/validation-decode.md` with the result. She DMs
the Game Runner:

```bash
scion message game-runner \
  "Lyra's decode passes my checks. Validation report at solutions/act-1/validation-decode.md."
```

### 5. Evaluate (authoritative)

The Game Runner reads `solutions/act-1/decoded-summons.txt`, compares
against `~/playbook/act-1/challenge-1.1/solutions/summons-decoded.txt`, and
on match broadcasts:

```bash
scion message --broadcast \
  "*** EVALUATION RESULT *** The Gateway recognizes your solution. PASS. The path is open. See current-challenge.md for Challenge 1.2."
```

It then updates `game-context.md` and loops back to step 1 with Challenge 1.2
data.

### 6. (Optional strengthening) Commit

If the grove enables the optional git-ledger discipline, each of steps 1, 3,
4, and 5 ends with `git add … && git commit … && git push`. Later, anyone
can run:

```bash
git log --oneline solutions/act-1/
```

to reconstruct who did what and in what order, without replaying broadcasts.

### Channel flow diagram

```d2
direction: right
operator: {
  label: "Human operator"
  shape: person
}
game_runner: {
  label: "Game Runner\n(orchestrator)"
  shape: rectangle
}
party: {
  label: "Party\n(Lyra, Kael, Mira, Zara)"
  shape: rectangle
}
thorne: {
  label: "Thorne\n(validator)"
  shape: rectangle
}
workspace: {
  label: "/workspace/\n(shared FS + git)"
  shape: cylinder
}
playbook: {
  label: "~/playbook/\n(private to Game Runner)"
  shape: cylinder
}

operator -> game_runner: "start"
playbook -> game_runner: "read inputs &\nanswer keys"
game_runner -> workspace: "deploy challenges/act-N/\nwrite current-challenge.md"
game_runner -> party: "broadcast: doorbell"
party -> workspace: "write solutions/act-N/"
party -> thorne: "DM: please review"
thorne -> workspace: "read solutions\nwrite validation-*.md"
thorne -> game_runner: "DM: passes review"
game_runner -> workspace: "diff vs playbook key"
game_runner -> party: "broadcast: PASS/FAIL"
```

## Limitations and non-goals

- **No atomic locking.** Two agents writing the same file race. The
  mitigation is convention (one writer per path); if you need real atomicity,
  commit groups of files as one git commit and treat an advance of the
  branch head as the atomic event.
- **No schema enforcement.** Directory contents are whatever filenames the
  agents agree on in their templates. If you want structured payloads,
  bolt JSON Schema onto each `solutions/<phase>/` directory and have the
  validator enforce it.
- **At-most-once broadcast delivery.** An agent that isn't running when a
  broadcast is sent simply misses it. The filesystem is the backstop: on
  restart, an agent should re-read `current-challenge.md` and `solutions/`
  before assuming it knows what's happening.
- **Not a workflow engine.** There is no retry-on-failure, no fan-out
  primitive, no dead-letter queue. The Game Runner's loop is the workflow
  engine; if the orchestrator agent crashes, the quest stalls until a human
  restarts it.

For a complete working grove that implements this pattern end-to-end, see
the [scion-athenaeum repository](https://github.com/danbills/scion-athenaeum).

For a second worked example — the same coordination shape applied to
multi-dimensional PR review — see the [PR Reviewer Demo](/scion/patterns/pr-reviewer/).
