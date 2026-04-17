---
title: Moderated Multi-Agent
description: How a coordinator, a set of peers, and an independent auditor collaborate around shared state with structured messaging.
---

This page documents the moderated multi-agent pattern — the coordination
shape for scenarios where multiple agents interact in real time, governed
by a coordinator who owns shared state, with an independent auditor
ensuring correctness.

The pattern combines three communication channels: broadcast messages for
public actions, direct messages for private coordination, and a
coordinator-owned state file that every agent can read but only one can
write. An independent auditor watches everything and flags violations.

For a working example that implements this pattern, see the
[Agent Poker demo](https://github.com/GoogleCloudPlatform/scion/tree/main/examples/agent-poker).

## When to use

Reach for moderated multi-agent when:

- **Multiple peers** need to take turns, make decisions, or interact
  with each other in real time.
- A single **coordinator** should own the ground truth (the state file)
  and enforce rules about who can do what and when.
- An independent **auditor** should verify correctness without
  participating in the activity.

If your agents do not interact with each other and just report back to a
dispatcher, use [Fan-out Parallel](/scion/patterns/fan-out-parallel/).
If you need multi-phase pipelines with review gates, see
[Athenaeum Coordination](/scion/patterns/athenaeum-coordination/).

## The shape

Three distinct roles:

```
                  ┌──────────────────────────────────────────┐
                  │            Coordinator (1)               │
                  │  - Sole writer of shared state file      │
                  │  - Sends direct messages (private)       │
                  │  - Sends broadcasts (public)             │
                  │  - Spawns peers + auditor at startup     │
                  └────────┬─────────────┬──────────────────┘
                           │             │
              direct msgs  │             │  direct msgs
              (private)    │             │  (violations)
                           ▼             ▼
        ┌──────────────────┐     ┌──────────────────┐
        │   Peers (N)      │     │   Auditor (1)    │
        │  - Read state    │     │  - Reads state   │
        │  - Broadcast     │     │  - Reads msgs    │
        │    actions       │     │  - Private logs  │
        │  - Wait for turn │     │  - Reports       │
        └──────────────────┘     │    violations    │
                                 └──────────────────┘
```

The coordinator is the sole writer of the shared state file. Peers read
it and broadcast their actions. The auditor observes both channels and
maintains a private shadow ledger.

## The three communication channels

| Channel | Mechanism | Who uses it | Purpose |
|---|---|---|---|
| **Broadcast** | `scion message --broadcast "..."` | Coordinator (announcements), Peers (public actions) | Table-wide visibility — everyone sees it |
| **Direct message** | `scion message <agent> "..."` | Coordinator → Peer (private prompts), Auditor → Coordinator (violations) | Private, point-to-point |
| **Shared state file** | A JSON/YAML file in `/workspace/` | Coordinator (write), Peers + Auditor (read) | Structured ground truth |

### Broadcasts are doorbells

A broadcast tells everyone "something happened" — a phase transition, a
player's action, a rule enforcement. It does not carry the full state;
agents read the state file for that.

### Direct messages are private channels

The coordinator uses direct messages to send information that not everyone
should see: a player's cards, a specific turn prompt, a warning. The
auditor uses direct messages to report violations to the coordinator
without tipping off the peers.

### The state file is the source of truth

One file (e.g., `card-table.json`) holds the canonical state. The
coordinator is the **sole writer** — this is the invariant that prevents
race conditions. Peers read it before making decisions; the auditor reads
it to cross-reference actions.

## Template anatomy

Each role gets its own template with a `scion-agent.yaml` and a
`system-prompt.md`. The poker example uses three:

```
.scion/templates/
├── poker-dealer/              # coordinator
│   ├── scion-agent.yaml
│   ├── system-prompt.md       # game rules, state file schema, turn management
│   └── home/
│       └── deck.py            # private tooling (only dealer sees this)
├── poker-player/              # peer
│   ├── scion-agent.yaml
│   └── system-prompt.md       # how to read state, when to act, how to broadcast
└── poker-auditor/             # auditor
    ├── scion-agent.yaml
    └── system-prompt.md       # what to watch for, where to log, when to report
```

### The private `home/` directory

Each template's `home/` directory is mounted **only** into that agent's
container at `~/`. Other agents cannot see it. This provides information
isolation within a shared workspace:

- **Coordinator** stores private tooling (`deck.py`) and answer keys in
  `~/`. Peers never see the deck state or upcoming cards.
- **Auditor** writes a private audit log to `~/audit-log.md` — never to
  the workspace, which all agents can read.
- **Peers** write private strategy notes to `~/`. Their cards and
  reasoning stay hidden from other peers.

This is how you give agents secrets in a shared-workspace setup: put
sensitive material in the template's `home/` directory.

## Turn management and concurrency

The coordinator serializes access by controlling whose turn it is:

1. Coordinator sets `active_player` in the state file.
2. Coordinator sends a **direct message** to that peer: "It's your turn.
   Current bet is 20. You have 75 chips."
3. Peer reads the state file, decides, and **broadcasts** its action:
   "I raise to 40."
4. Coordinator validates the action, updates the state file, and
   advances `active_player` to the next peer.

Peers **must wait** for their direct turn prompt before acting. Acting
out of turn is a violation that the auditor will flag.

This turn-by-turn protocol avoids the need for locking or
compare-and-swap on the state file. Only one agent acts at a time, and
the coordinator is the only writer.

## The auditor role

The auditor is a read-only observer with three responsibilities:

1. **Shadow tracking.** The coordinator sends the auditor private copies
   of all dealt information (e.g., each player's cards). The auditor
   maintains an independent ledger to compare against claims.
2. **Violation detection.** The auditor watches broadcasts for illegal
   actions: out-of-turn play, impossible claims, bet manipulation. It
   classifies violations by severity:
   - **Warnings** (procedural infractions) — the auditor direct-messages
     the offending peer and informs the coordinator.
   - **Cheating violations** (fraud, impossible claims) — the auditor
     broadcasts evidence publicly. The coordinator enforces punishment.
3. **End-of-round verification.** After each round/hand/phase, the
   auditor cross-checks the outcome against its shadow records and
   confirms or disputes the result.

The auditor never modifies the state file and never writes to the shared
workspace. All records stay in its private `~/` directory.

## Worked example: 4-player poker

### 0. Setup

```bash
scion init poker-night

scion templates import --all \
  https://github.com/GoogleCloudPlatform/scion/tree/main/examples/agent-poker/templates
```

This imports three templates: `poker-dealer`, `poker-player`, and
`poker-auditor`.

### 1. Launch the coordinator

```bash
scion create dealer --template poker-dealer
scion message dealer "Start a 4-player Texas Hold'em game"
```

### 2. Coordinator bootstraps the game

The dealer agent:

1. Spawns `player-1` through `player-4` using `scion start --template poker-player`.
2. Spawns `auditor` using `scion start --template poker-auditor`.
3. Initializes the deck: `python3 ~/deck.py init`.
4. Creates `card-table.json` in the workspace with initial state (4
   players, 100 chips each, blinds at 5/10).
5. Sends each player a direct message confirming their identity and
   position.
6. Broadcasts: "Game starting. 4 players, 100 chips each, blinds 5/10."

### 3. A hand plays out

**Pre-flop:**

1. Dealer runs `python3 ~/deck.py draw 2` for each player.
2. Dealer direct-messages each player their hole cards: "player-2, you
   are Small Blind this hand. Your hole cards are: [Ace of spades, 7 of
   hearts]"
3. Dealer direct-messages the auditor with shadow copies: "DEAL:
   player-2 received [Ace of spades, 7 of hearts]"
4. Dealer sets `active_player: "player-1"` in `card-table.json` and
   direct-messages player-1: "Your turn. Current bet: 10. Your chips: 100."

**Betting round:**

5. Player-1 reads `card-table.json`, evaluates hand strength, and
   broadcasts: "I call."
6. Dealer validates the action, updates `card-table.json` (chips, bet
   history, `active_player`), and direct-messages the next player.
7. Repeat for all active players.

**Flop, turn, river:**

8. Dealer draws community cards: `python3 ~/deck.py draw 3` (flop),
   then `draw 1` (turn), then `draw 1` (river).
9. Updates `card-table.json` with community cards.
10. Broadcasts each reveal: "--- FLOP: 8 of hearts, King of diamonds,
    3 of clubs ---"
11. Runs a betting round after each reveal.

**Showdown:**

12. Dealer asks remaining players to reveal hands via broadcast.
13. Evaluates best 5-card hands, awards pot, updates chip stacks.
14. Broadcasts result: "player-2 wins with pair of Kings. Pot: 45 chips."

**Audit:**

15. Auditor reads revealed hands, compares against its shadow records,
    and broadcasts: "AUDIT: Hand 1 verified clean."

### 4. Violation handling

If the auditor detects player-3 claiming cards that were never dealt:

1. Auditor broadcasts: "AUDIT VIOLATION: player-3 — Card fraud. Claimed
   [Queen of hearts] but was dealt [4 of clubs]. Evidence: shadow record
   from dealer DM at pre-flop."
2. Dealer sets `player-3.status: "banned"` in `card-table.json`.
3. Dealer broadcasts: "player-3 has been banned for cheating. Chips
   forfeited and distributed."
4. Game continues with remaining players.

## Generalizing beyond poker

The coordinator/peer/auditor triad is domain-agnostic. Replace poker with
any scenario that has:

- A single source of truth that must be updated atomically.
- Peers who take turns or interact under rules.
- A need for independent verification.

| Domain | Coordinator | Peers | Auditor |
|---|---|---|---|
| Code review | PM or lead (assigns reviews, merges) | Developers (review, approve, request changes) | Linter or style checker (automated validation) |
| Simulation | Environment engine (updates world state) | Actors (take actions, observe state) | Logger/validator (ensures physics/rules hold) |
| Debate/discussion | Facilitator (manages turns, enforces time) | Participants (make arguments) | Fact-checker (verifies claims) |
| Auction | Auctioneer (manages lots, accepts bids) | Bidders (place bids) | Compliance monitor (validates bid legality) |

In every case, the shape is the same: one writer of shared state,
broadcast for public actions, direct messages for private coordination,
and a read-only observer for integrity.

## Building your own moderated multi-agent grove

### Step 1: Define the state file schema

Decide what goes in the shared state file. It should contain everything
a peer needs to make a decision, plus metadata the coordinator uses for
sequencing (`active_player`, `phase`, etc.). Write the schema into the
coordinator's `system-prompt.md`.

### Step 2: Write three templates

- **Coordinator** — owns the state file, spawns peers and auditor,
  manages turns, enforces rules. Put private tooling and data in
  `home/`.
- **Peers** — read-only access to the state file, broadcast public
  actions, wait for turn prompts. All peer behavior comes from the
  system prompt.
- **Auditor** — receives shadow copies via direct message, maintains a
  private ledger in `~/`, reports violations. Never writes to the
  workspace.

### Step 3: Establish communication rules in each template's prompt

Every template's `system-prompt.md` should explicitly state:

- **When to broadcast** vs. **when to direct-message**.
- **What files the agent may read** vs. **what files it may write**
  (and where — workspace vs. home directory).
- **When to act** (after receiving a turn prompt) vs. **when to wait**.

These rules are the contract. Scion does not enforce them technically —
agents can write anywhere the container filesystem allows. The
enforcement is in the prompts and in the auditor watching for violations.

### Step 4: Decide on violation severity

Classify what counts as a warning (procedural, recoverable) vs. a
serious violation (fraud, corruption). Encode the classification in the
auditor's system prompt and the coordinator's enforcement rules.

## Limitations

- **Coordinator is a single point of failure.** If it crashes, the game
  stalls. Peers cannot self-organize because they don't own the state
  file. A human must restart the coordinator.
- **No automatic recovery.** Scion does not checkpoint the state file or
  replay missed messages. If you need durability, have the coordinator
  commit the state file to git after each round (see the
  [Athenaeum pattern](/scion/patterns/athenaeum-coordination/#git-commits-as-the-official-ledger-optional-strengthening)
  for this technique).
- **Broadcast delivery is at-most-once.** An agent that isn't running
  when a broadcast is sent simply misses it. The state file is the
  backstop — on reconnect, agents should re-read the state file rather
  than relying on broadcast history.
- **No filesystem locking.** The single-writer convention (only the
  coordinator writes the state file) is enforced by the prompts, not by
  the OS. A misbehaving agent could write to the state file. The auditor
  can detect this by watching for unexpected state changes, but cannot
  prevent it.
- **Turn serialization limits throughput.** Only one peer acts at a
  time. If you need parallel work within phases, combine this pattern
  with fan-out parallel for the work steps and moderated multi-agent for
  the coordination steps.
