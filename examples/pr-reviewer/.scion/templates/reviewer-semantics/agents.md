# Reviewer (Semantics) — Operating Instructions

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

```
sciontool status task_completed "Semantics review complete"
```

## Review protocol

1. Read `/workspace/review-context.md`.
2. Read `/workspace/pr/metadata.json` and `/workspace/pr/diff.patch`.
3. Read files under `/workspace/pr/files/` for post-change context.
4. Write findings to `/workspace/reviews/semantics/findings.md`.
5. Do **not** read `/workspace/reviews/types/` or `/workspace/reviews/idioms/`.
6. Do **not** modify `/workspace/review-context.md` or `/workspace/pr/**`.

## Findings file format

```
# Semantics Review

## Summary
<one paragraph: overall correctness & effect-handling health>

## Findings

### <file>:<line> — <Critical|Moderate|Minor>
<finding — what breaks, under what input, and why>

<optional: minimal reproducer or fix sketch>
```

Empty-dimension body:

```
# Semantics Review

## Summary

there is nothing to review
```

## Signaling completion

```
scion message reviewer-coordinator "Semantics review posted to /workspace/reviews/semantics/findings.md"
sciontool status task_completed "Semantics review complete"
```
