---
title: Sequential Worklist
description: How an orchestrator walks a list of ordered tasks one agent at a time, handing state forward through the workspace.
---

This page documents the sequential worklist pattern — the ordered,
single-lane cousin of [Fan-out Parallel](/scion/patterns/fan-out-parallel/).
One orchestrator reads a worklist, creates a single worker, waits for it to
finish, deletes it, and moves on to the next item. Each worker inherits the
workspace state left by its predecessor.

For a working example that implements this pattern, see
[`examples/orchestration-basics/sequence/`](https://github.com/GoogleCloudPlatform/scion/tree/main/examples/orchestration-basics/sequence).

## When to use

Reach for sequential worklist when:

- Tasks **must execute in order** — step 2 reads files that step 1
  created, step 3 runs scripts that step 2 wrote, and so on.
- You want to **limit concurrency** to one agent at a time (resource
  constraints, API rate limits, or simply easier to debug).
- The handoff between steps is **file-based** — each worker leaves
  artifacts in the workspace for the next worker to consume.

If your tasks are independent and can run simultaneously, use
[Fan-out Parallel](/scion/patterns/fan-out-parallel/) instead.

## The shape

```
orchestrator ──start──► worker-1 ──notify──► orchestrator
                                                  │
                                             delete worker-1
                                                  │
             ──start──► worker-2 ──notify──► orchestrator
                                                  │
                                             delete worker-2
                                                  │
                                                 ...
                                                  │
             ──start──► worker-N ──notify──► orchestrator ──► done
```

- **One orchestrator** owns the worklist and drives the sequence.
- **One worker at a time** — created, runs to completion, deleted.
- The workspace accumulates state across workers: worker 1's output
  files are on disk when worker 2 starts.

## Communication channels

| Channel | Used? | Role |
|---|---|---|
| `--notify` callback | Yes | Worker → orchestrator: "I'm done." |
| `scion message` | No | Not needed — workers don't overlap. |
| Shared filesystem | **Yes, critically** | The workspace is the state handoff mechanism between sequential workers. |

The workspace is the pipeline. Each worker reads what previous workers
left and adds its own output. No explicit message passing is needed
because workers never run concurrently.

## State handoff via the workspace

This is the key difference from fan-out parallel. In fan-out, workers are
stateless and independent. In sequential worklist, the workspace is a
**cumulative artifact** that grows with each step:

```
After worker-1:  fetch_data.py, raw_data.json
After worker-2:  fetch_data.py, raw_data.json, process_data.py
After worker-3:  fetch_data.py, raw_data.json, process_data.py, processed_log.txt
After worker-4:  ... + generate_report.py, report.html
After worker-5:  ... + project_delivery.tar.gz
```

Each worker's task prompt tells it what files already exist and what to
produce. The orchestrator does not move files or transform data — it only
sequences the workers.

## Template structure

Sequential worklist typically uses a single generic template for all
workers, varying only the task prompt:

```
.scion/templates/
└── default/
    └── system-prompt.md    # general-purpose worker instructions
```

The task-specific behavior comes entirely from the prompt passed to
`scion start`. This keeps the template simple and reusable.

If different steps need fundamentally different capabilities (e.g., step 1
needs web search, step 4 needs an HTML renderer), you can use different
templates per step via the `--template` flag.

## Orchestrator prompt

The orchestrator's system prompt defines the sequencing logic. A minimal
version:

```
Your job is to orchestrate a set of agents across a sequence of work.

For each task in the worklist, use the scion CLI to start an agent and
assign it a task. Include the --notify argument so it will alert you
when it has completed.

After assigning each task, wait idle for the agent's notification.

Stop and delete that agent, then proceed to assign the next task.

When all agents have completed their work, you are done.
```

The orchestrator reads the worklist from a file in the workspace (e.g.,
`work-sequence.md`) and walks it item by item.

## Worked example: data pipeline

The
[`examples/orchestration-basics/sequence/`](https://github.com/GoogleCloudPlatform/scion/tree/main/examples/orchestration-basics/sequence)
directory contains a five-step data pipeline.

### 0. Prerequisites

Like fan-out parallel, this pattern requires **Hub mode** for `--notify`.

### 1. Initialize and connect

```bash
cd my-pipeline-project
scion init
scion server start
scion config set hub.endpoint http://localhost:8080
scion hub enable
scion hub link
```

### 2. Prepare the worklist

Place `work-sequence.md` in the workspace. Each item describes one step:

1. Find a public API that returns random trivia, write `fetch_data.py`,
   save output to `raw_data.json`.
2. Run `fetch_data.py`, write `process_data.py` to parse the JSON and
   append to `processed_log.txt`.
3. Run both scripts three times to accumulate entries in
   `processed_log.txt`, verify with `cat`.
4. Write `generate_report.py` to convert `processed_log.txt` into a
   styled `report.html`.
5. Run `generate_report.py`, bundle everything into
   `project_delivery.tar.gz`.

### 3. Launch the orchestrator

```bash
scion start -a orchestrator
```

Provide the sequencing prompt (or embed it in the orchestrator's template).

### 4. What happens

The orchestrator:

1. Reads `work-sequence.md` — 5 tasks.
2. Runs `scion start --notify agent-1 "Task 1: find an API, write fetch_data.py..."`.
3. Idles until `agent-1` notifies completion.
4. Runs `scion delete agent-1`.
5. Runs `scion start --notify agent-2 "Task 2: run fetch_data.py, write process_data.py..."`.
   Agent-2 finds `fetch_data.py` and `raw_data.json` already in the workspace.
6. Repeats through task 5.

### 5. Results

The workspace contains the full pipeline output: scripts, intermediate
data, the final HTML report, and the compressed archive. Each step built
on artifacts left by the previous step.

## When to choose sequential vs. parallel

| Factor | Sequential | Fan-out Parallel |
|---|---|---|
| Task dependencies | Each step reads prior output | Tasks are independent |
| Concurrency | 1 worker at a time | N workers simultaneously |
| Speed | Slower (serial by design) | Faster (wall-clock) |
| Resource usage | Minimal (one container) | Higher (N containers) |
| Debugging | Easy (replay one step) | Harder (N agents running) |
| Workspace state | Cumulative, ordered | Independent per worker |

You can also **combine** the two: an orchestrator that runs steps 1-3
sequentially, then fans out steps 4a/4b/4c in parallel, then runs step 5
sequentially after all parallel workers finish.

## Limitations

- **Slow by design.** Only one worker runs at a time. If your steps are
  truly independent, fan-out parallel will finish faster.
- **Single point of failure.** If the orchestrator crashes mid-sequence,
  the pipeline stalls. The workspace retains all artifacts, so a human
  can restart from where it left off, but there is no automatic resume.
- **No partial results.** You only get output after the full sequence
  completes (or after each step individually, if you inspect the
  workspace between steps).
- **`--notify` requires Hub mode.** Same as fan-out parallel — without a
  Hub, the orchestrator must poll `scion list`.
- **Workspace accumulation.** The workspace grows monotonically. If
  workers produce large intermediate files, disk usage can climb. Clean
  up explicitly in later steps if needed.
