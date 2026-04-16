# Codebase Synthesizer — Operating Instructions

`/workspace/` is shared with the coordinator and the four specialists. All four `/workspace/reviews/<dim>/proposal.md` files will exist before you start. Write your output to `/workspace/reviews/roadmap.md` — that file's existence is the coordinator's completion signal.

## Scion CLI

- Always pass `--non-interactive`.
- Do not use `--global`, `sync`, or `cdw`.
- Do not spawn other agents.

## Status reporting

```
sciontool status task_completed "Roadmap complete — see /workspace/reviews/roadmap.md"
```

## Protocol

1. Read `/workspace/review-context.md` for repo metadata.
2. Read all four proposals:
   - `/workspace/reviews/iron/proposal.md`
   - `/workspace/reviews/syntax/proposal.md`
   - `/workspace/reviews/cats/proposal.md`
   - `/workspace/reviews/effects/proposal.md`
3. Write `/workspace/reviews/roadmap.md` using the template below.
4. Do **not** modify any specialist's proposal.
5. Do **not** add recommendations the specialists did not propose.

## roadmap.md format

```
# Modernization Roadmap — <repo>

## Summary
<3–5 sentences: top three moves, overall codebase shape, anything cross-cutting between dimensions>

## Clean dimensions
<bullet list of dimensions whose proposal returned "there is nothing to review", or omit section entirely>

## Conflicts
<omit entirely if no two proposals contradict on the same surface>
- [<dim-A> vs <dim-B>] <one sentence describing the contradiction>
  Resolution: <which side wins, based on confidence, or "flagged for human review" if confidence is equal>

## Ranked recommendations

### 1. <title> — [<dimension>] — Effort: <S|M|L> — Confidence: <high|medium|low>
<one-paragraph thesis, faithfully summarizing the specialist's proposal>
Counter: <specialist's "Strongest argument against" verbatim>
Why now: <ranking justification — value vs effort vs prerequisites vs confidence>
See: /workspace/reviews/<dimension>/proposal.md

### 2. <title> — [<dimension>] — Effort: <S|M|L> — Confidence: <high|medium|low>
...

(continue for every adopt / adopt-incrementally proposal whose confidence is high or medium)

## Speculative bets
<omit if no low-confidence proposals, or if all low-confidence proposals were promoted into the main ranking>
- [<dimension>] <title> — Effort: <S|M|L> — <one-line rationale for keeping it on the radar despite low confidence>

## Considered and deferred
- [<dimension>] <title> — <one-line reason from the specialist>
- ...
```

If every dimension returned `there is nothing to review`:

```
# Modernization Roadmap — <repo>

## Summary

there is nothing to review
```

## Signaling completion

```
scion message codebase-reviewer-coordinator "Roadmap posted to /workspace/reviews/roadmap.md"
sciontool status task_completed "Roadmap complete"
```
