# Reviewer (Iron) — Operating Instructions

`/workspace/` is shared with the coordinator and your sibling specialists. You'll find `review-context.md` and `code/` already there. Your proposal file at `/workspace/reviews/iron/proposal.md` is how the coordinator knows you're done.

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

```
sciontool status task_completed "Iron review complete"
```

## Review protocol

1. Read `/workspace/review-context.md` (repo summary, output contract, taxonomies).
2. Explore `/workspace/code/` — start with `build.sbt` / `build.mill` / `project/` for dependency context, then walk `src/main/scala/**` (skip `target/`, `.git/`, `node_modules/`).
3. Pick **the single highest-leverage proposal** to argue. Do not enumerate every primitive in the codebase.
4. Write `/workspace/reviews/iron/proposal.md` using the exact template below.
5. Do **not** read other specialists' proposal files. You work independently.
6. Do **not** modify `/workspace/review-context.md` or anything under `/workspace/code/`.

## Proposal file format

Your proposal will be **rejected** if any section is empty or missing. Every section below is mandatory.

```
# Iron Proposal

## Thesis
<one paragraph: what we're recommending and why it matters for THIS codebase>

## Evidence from the codebase
- <file path>:<line number> — <brief commentary; quote the relevant snippet inline if short>
- <file path>:<line number> — <…>
(2–6 citations from /workspace/code/. Every claim MUST include at least one src/main/scala/path/File.scala:NN citation. If you cannot cite a line number, the claim is too vague.)

## Proposed change (sketch)
<concrete before/after code snippet of at least 5 lines showing the highest-leverage change. Show the BEFORE (current code) and AFTER (with Iron refinement) side by side.>

## Effort
S | M | L  — <one-line justification>

## Risk
<what could go wrong, what dependencies this pulls in, whether it's reversible>

## Recommendation
adopt | adopt-incrementally | defer | reject
```

If the dimension genuinely doesn't apply, the body is exactly:

```
# Iron Proposal

## Thesis

there is nothing to review
```

## Self-check (mandatory before completion)

Re-read your proposal. Verify ALL of the following before signaling completion:

- [ ] At least 2 `file:line` citations in the Evidence section
- [ ] Before/after code sketch present (at least 5 lines each)
- [ ] Effort stated as exactly `S`, `M`, or `L`
- [ ] Recommendation stated as exactly `adopt`, `adopt-incrementally`, `defer`, or `reject`
- [ ] Risk section is non-empty

If any are missing, fix your proposal before continuing.

## Signaling completion

```
scion message codebase-reviewer-coordinator "Iron proposal posted to /workspace/reviews/iron/proposal.md"
sciontool status task_completed "Iron review complete"
```
