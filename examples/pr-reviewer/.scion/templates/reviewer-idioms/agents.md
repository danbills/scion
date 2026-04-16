# Reviewer (Idioms) — Operating Instructions

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

When you have written your findings file, run:

```
sciontool status task_completed "Idioms review complete"
```

## Review protocol

1. Read `/workspace/review-context.md`.
2. Read `/workspace/pr/metadata.json` and `/workspace/pr/diff.patch`.
3. Read files under `/workspace/pr/files/` for post-change context.
4. Write findings to `/workspace/reviews/idioms/findings.md` using the template below.
5. Do **not** read `/workspace/reviews/types/` or `/workspace/reviews/semantics/`.
6. Do **not** modify `/workspace/review-context.md` or `/workspace/pr/**`.

## Findings file format

```
# Idioms Review

## Summary
<one paragraph: overall Scala 3 idiom conformance>

## Findings

### <file>:<line> — <Critical|Moderate|Minor>
<finding>

<optional: suggested Scala 3 rewrite>
```

Empty-dimension body:

```
# Idioms Review

## Summary

there is nothing to review
```

## Signaling completion

```
scion message reviewer-coordinator "Idioms review posted to /workspace/reviews/idioms/findings.md"
sciontool status task_completed "Idioms review complete"
```
