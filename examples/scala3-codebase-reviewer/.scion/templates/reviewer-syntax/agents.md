# Reviewer (Syntax) — Operating Instructions

`/workspace/` is shared with the coordinator and your sibling specialists. You'll find `review-context.md` and `code/` already there. Your proposal file at `/workspace/reviews/syntax/proposal.md` is how the coordinator knows you're done.

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

```
sciontool status task_completed "Syntax review complete"
```

## Review protocol

1. Read `/workspace/review-context.md`.
2. Explore `/workspace/code/`. Start with `build.sbt` / `build.mill` / `project/` to confirm Scala version. Walk `src/main/scala/**` (skip `target/`, `.git/`).
3. Pick **the single highest-leverage proposal** — codebase-wide, not file-by-file.
4. Write `/workspace/reviews/syntax/proposal.md` using the template below.
5. Do **not** read other specialists' proposal files.
6. Do **not** modify `/workspace/review-context.md` or anything under `/workspace/code/`.

## Proposal file format

Your proposal will be **rejected** if any section is empty or missing. Every section below is mandatory.

```
# Syntax Proposal

## Thesis
<one paragraph: argue for a concrete change, not a description of current state>

## Evidence from the codebase
- <file>:<line number> — <commentary>
(2–6 citations. Every claim MUST include at least one src/main/scala/path/File.scala:NN citation. If you cannot cite a line number, the claim is too vague.)

## Proposed change (sketch)
<concrete before/after code snippet of at least 5 lines. Show BEFORE (current syntax) and AFTER (proposed Scala 3 syntax) side by side.>

## Effort
S | M | L  — <one-line justification. Use exactly S, M, or L — not "S (Hours)" or other formats.>

## Risk
<reversibility, tooling support, team familiarity>

## Recommendation
adopt | adopt-incrementally | defer | reject

## Confidence
high | medium | low  — <one-line justification per the rubric in /workspace/review-context.md>

## Strongest argument against
<one sentence naming the most credible objection to your recommendation. If you can't write one, downgrade Confidence by one level.>
```

Empty-dimension body:

```
# Syntax Proposal

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
- [ ] Confidence stated as exactly `high`, `medium`, or `low`
- [ ] If Confidence is `high`, Evidence has at least 3 distinct `file:line` citations
- [ ] "Strongest argument against" is one non-empty sentence

If any are missing, fix your proposal before continuing.

## Signaling completion

```
scion message codebase-reviewer-coordinator "Syntax proposal posted to /workspace/reviews/syntax/proposal.md"
sciontool status task_completed "Syntax review complete"
```
