---
title: Fan-out Parallel
description: How an orchestrator agent dispatches N independent workers in parallel and collects results via notify callbacks.
---

This page documents the fan-out parallel pattern, the simplest multi-agent
coordination shape in Scion. One orchestrator reads a list of independent
tasks, spawns a worker agent for each, and idles until every worker has
finished.

For a working example that implements this pattern, see
[`examples/orchestration-basics/fan-out-parallel/`](https://github.com/GoogleCloudPlatform/scion/tree/main/examples/orchestration-basics/fan-out-parallel).

## When to use

Reach for fan-out parallel when:

- You have a **list of independent, homogeneous tasks** (research topics,
  file reviews, data transforms, test runs).
- No task needs the output of any other task.
- All workers can use the **same template** — the only thing that varies
  is the task prompt.

If your tasks must run in order because each consumes the prior step's
output, use [Sequential Worklist](/scion/patterns/sequential-worklist/) instead.

## The shape

```
                        ┌── worker-1 ──► notify ──┐
                        │                          │
orchestrator ───start───┼── worker-2 ──► notify ──┼──► orchestrator (idle)
                        │                          │         │
                        ├── worker-3 ──► notify ──┤    collect / delete
                        │      ...                 │
                        └── worker-N ──► notify ──┘
```

- **One orchestrator** owns the task list and the lifecycle of every worker.
- **N workers** are stateless — each receives a task prompt at creation
  time, does its work, and exits.
- The only coordination channel is **`--notify`**: when a worker's harness
  reports `task_completed`, Scion sends a notification back to the
  orchestrator so it knows the work is done.

## Communication channel

| Channel | Used? | Role |
|---|---|---|
| `--notify` callback | Yes | Worker → orchestrator: "I'm done." |
| `scion message` | No | Not needed — tasks are independent. |
| Shared filesystem | Optional | Workers may write output files to the workspace, but they don't read each other's output. |

This is the lightest coordination footprint in Scion. Workers never talk
to each other and never broadcast.

## Template structure

Two templates are involved:

### Orchestrator template

The orchestrator's system prompt contains the fan-out instructions. It
does not need special tooling — it reads a task list, loops over it, and
calls `scion start` for each entry.

A minimal orchestrator prompt:

```
Use the scion CLI to start a researcher agent for each of the topics
in topics.txt.

Be sure to ask to be --notified when they are done.

Once you have started each researcher, wait idle for notifications —
no need to poll or check on them.

When a researcher completes its work, you may delete it.
```

The key elements:

1. **Read the task list** — a file in the workspace (`topics.txt`, a
   JSON manifest, a directory listing — whatever fits your domain).
2. **`scion start --notify <name> "<task>"`** — the `--notify` flag is
   what wires the callback. Without it, the orchestrator would have to
   poll `scion list` to detect completion.
3. **Idle after dispatch** — the orchestrator does nothing until a
   notification arrives. No busy-waiting, no polling loops.
4. **Delete on completion** — `scion delete <name>` frees the container
   and worktree.

### Worker template

Each worker is cloned from the same template. The template's
`system-prompt.md` defines the worker's role and methodology; the task
prompt (passed as the positional argument to `scion start`) provides the
specific assignment.

The example uses a detailed research-specialist prompt
(`examples/orchestration-basics/fan-out-parallel/researcher-prompt.md`)
and a structured output template
(`examples/orchestration-basics/fan-out-parallel/research-template.md`).
Workers fill in the template and write the result to the workspace.

```
.scion/templates/
├── default/           # orchestrator uses the default template
│   └── system-prompt.md
└── researcher/        # worker template, cloned from default
    └── system-prompt.md   # ← researcher-prompt.md content
```

## Worked example: parallel research

The
[`examples/orchestration-basics/fan-out-parallel/`](https://github.com/GoogleCloudPlatform/scion/tree/main/examples/orchestration-basics/fan-out-parallel)
directory contains everything needed to run this pattern end-to-end.

### 0. Prerequisites

Fan-out parallel requires **Hub mode** because `--notify` is a Hub
feature. You need a running Scion server (Hub + broker).

### 1. Initialize the grove

```bash
cd my-research-project
scion init
```

### 2. Set up the worker template

Clone the default template and replace its system prompt with the
researcher role:

```bash
scion templates clone default researcher
mv ./researcher-prompt.md $(scion config dir)/templates/researcher/system-prompt.md
```

### 3. Connect to Hub

```bash
scion server start          # starts the local Hub + broker
scion config set hub.endpoint http://localhost:8080
scion hub enable
scion hub link              # registers this grove; syncs templates
```

### 4. Prepare the task list

The example uses `topics.txt` — one topic per line:

```
Precision burr grinders
Pour-over kettles
High-end espresso machines
Digital coffee scales
Temperature-controlled milk frothers
Cold brew systems
Manual espresso makers
Gooseneck electric kettles
```

This file lives in the workspace so the orchestrator can read it.

### 5. Launch the orchestrator

```bash
scion start -a orchestrator
```

Inside the orchestrator's session, provide the fan-out prompt (or bake it
into the orchestrator's template system-prompt):

```
Use the scion CLI to start a researcher agent for each of the topics
in topics.txt.

Be sure to ask to be --notified when they are done.

Once you have started each researcher, wait idle for notifications.
When a researcher completes its work, delete it.
```

### 6. What happens

The orchestrator:

1. Reads `topics.txt` — 8 topics.
2. Runs `scion start --notify researcher-1 --template researcher "Research: Precision burr grinders"`,
   then `researcher-2`, `researcher-3`, etc.
3. All 8 workers launch in parallel across available brokers.
4. Each worker researches its topic, writes a report to the workspace
   using the output template, and exits.
5. As each worker completes, the orchestrator receives a notification,
   acknowledges it, and runs `scion delete <name>`.
6. After all 8 notifications arrive, the orchestrator reports completion.

### 7. Results

The workspace now contains 8 completed research reports, one per topic.
The orchestrator can summarize them, merge them into a single document,
or simply leave them for human review.

## Scaling knobs

- **Number of workers**: Limited only by available broker capacity. If
  the Hub has multiple brokers registered, Scion dispatches workers
  across them automatically.
- **Template variation**: All workers share a template, but you can use
  different templates for different task types (e.g., `researcher` for
  some topics, `analyst` for others) by varying the `--template` flag
  per `scion start` call.
- **Output format**: Control what workers produce by embedding an output
  template in the worker's `system-prompt.md` or by placing a
  `research-template.md` in the workspace for workers to follow.

## Limitations

- **No inter-worker communication.** Workers cannot coordinate, share
  partial results, or depend on each other. If you need that, step up
  to [Moderated Multi-Agent](/scion/patterns/moderated-multi-agent/) or
  [Athenaeum Coordination](/scion/patterns/athenaeum-coordination/).
- **No built-in retry.** If a worker fails, the orchestrator receives a
  notification but must decide what to do (re-start, skip, alert the
  human). There is no automatic retry loop.
- **`--notify` requires Hub mode.** This pattern does not work in
  purely local mode. If you cannot run a Hub, the orchestrator must poll
  `scion list` to detect worker completion — which works but is less
  efficient.
- **At-most-once delivery.** If the orchestrator is not running when a
  notification arrives, it misses it. In practice this rarely matters
  because the orchestrator is idle-waiting, but be aware of it.
