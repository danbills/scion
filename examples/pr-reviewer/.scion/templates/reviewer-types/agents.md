# Reviewer (Types) — Operating Instructions

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

When you have written your findings file, run:

```
sciontool status task_completed "Types review complete"
```

## Review protocol

1. Read `/workspace/review-context.md` (PR summary, severity taxonomy, output contract).
2. Read `/workspace/pr/metadata.json` for PR title, number, author, base/head sha.
3. Read `/workspace/pr/diff.patch` as the authoritative view of what changed.
4. Read files under `/workspace/pr/files/` for post-change context. Files may be truncated to ±40 lines around touched hunks; `diff.patch` is canonical.
5. Write your findings to `/workspace/reviews/types/findings.md` using the exact template below.
6. Do **not** read `/workspace/reviews/idioms/` or `/workspace/reviews/semantics/`. You work independently. The synthesizer merges later.
7. Do **not** modify `/workspace/review-context.md` or `/workspace/pr/**`.

## Findings file format

```
# Types Review

## Summary
<one paragraph: overall type-strength health of the change>

## Findings

### <file>:<line> — <Critical|Moderate|Minor>
<finding>

<optional: suggested change as a short Scala snippet>

### <file>:<line> — <Critical|Moderate|Minor>
...
```

If there is nothing to flag in this dimension, the file body is exactly:

```
# Types Review

## Summary

there is nothing to review
```

## Signaling completion

After writing the file, direct-message the coordinator:

```
scion message reviewer-coordinator "Types review posted to /workspace/reviews/types/findings.md"
```

Then `sciontool status task_completed`.
