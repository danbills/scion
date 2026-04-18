---
title: Patterns
description: Reusable shapes for organizing Scion agents to get real work done.
---

Scion patterns are reusable shapes for organizing agents around a task.
Each pattern names a coordination topology, describes the communication
channels it relies on, and explains when to reach for it.

The patterns below are ordered from simplest to most complex. Start with
the one that matches your problem shape; graduate to a heavier pattern only
when the simpler one can't express the coordination you need.

## Choosing a pattern

| I need to... | Pattern | Agents |
|---|---|---|
| Run N independent tasks with no coordination | [Fan-out Parallel](/scion/patterns/fan-out-parallel/) | 1 orchestrator + N workers |
| Execute ordered steps where each consumes the prior step's output | [Sequential Worklist](/scion/patterns/sequential-worklist/) | 1 orchestrator + 1 worker at a time |
| Have peers interact in real time with shared state and an arbiter | [Moderated Multi-Agent](/scion/patterns/moderated-multi-agent/) | 1 coordinator + N peers + 1 auditor |
| Run a multi-phase quest with review gates and a private playbook | [Athenaeum Coordination](/scion/patterns/athenaeum-coordination/) | 1 orchestrator + N characters + 1 validator |

## Communication primitives

Every pattern is built from the same three primitives. The difference is
which ones a pattern uses and how:

| Primitive | Mechanism | Good for |
|---|---|---|
| **Notify** | `scion start --notify` | "Tell me when you're done." One-shot, worker → orchestrator. |
| **Messages** | `scion message` (direct) / `scion message --broadcast` | Real-time coordination, doorbells, short payloads. |
| **Shared filesystem** | `/workspace/` mount (all agents) or `home/` mount (per-agent private) | Handing off files, structured state, audit logs. |

See [About Workspaces](/scion/advanced-local/workspace/) for how Scion
resolves workspace directories, and [Templates & Roles](/scion/advanced-local/templates/)
for how templates wire these primitives into agent behavior.
