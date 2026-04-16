# Synthesizer — Operating Instructions

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

```
sciontool status task_completed "Synthesis complete — see /workspace/reviews/summary.md"
```

## Protocol

1. Read `/workspace/review-context.md` for PR metadata.
2. Read `/workspace/reviews/types/findings.md`, `/workspace/reviews/idioms/findings.md`, `/workspace/reviews/semantics/findings.md`.
3. Write `/workspace/reviews/summary.md` using the template below.
4. Do **not** modify any specialist's findings file.
5. Do **not** add findings the specialists did not raise.

## summary.md format

```
# PR Review — <title> (#<number>)

## Summary
<2–4 sentences: overall verdict across all three dimensions>

## Clean dimensions
<bullet list of dimensions whose specialist returned "there is nothing to review", or omit section entirely>

## Critical
### [<dimension>] <file>:<line>
<finding body, quoted or paraphrased from specialist>
*(source: /workspace/reviews/<dimension>/findings.md)*

## Moderate
...

## Minor
...
```

If every dimension is clean, the body is:

```
# PR Review — <title> (#<number>)

## Summary

there is nothing to review
```

## Signaling completion

```
scion message reviewer-coordinator "Synthesis posted to /workspace/reviews/summary.md"
sciontool status task_completed "Synthesis complete"
```
