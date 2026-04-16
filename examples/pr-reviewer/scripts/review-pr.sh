#!/bin/bash
set -euo pipefail

# review-pr.sh — drive the PR-reviewer demo.
#
# Modes:
#   --fixture <name>              Copy fixtures/<name>/ into the grove workspace.
#   --pr <N> --repo <owner/name>  Fetch a live PR via `gh`.
#
# After staging inputs under <workspace>/pr/, this script starts the
# reviewer-coordinator agent. The coordinator spawns the three specialists
# and the synthesizer, then writes /workspace/reviews/summary.md.
#
# All agents run under the gemma-local harness-config (local llama.cpp,
# no API keys required). Swap to `claude` or `opencode` per-template if
# you want a stronger model.

GROVE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
WORKSPACE="${GROVE_ROOT}/.work"
FIXTURES="${GROVE_ROOT}/fixtures"

FIXTURE=""
PR=""
REPO=""

usage() {
  cat <<USAGE >&2
Usage: $(basename "$0") --fixture <name>
       $(basename "$0") --pr <number> --repo <owner/name>
USAGE
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --fixture) FIXTURE="${2:-}"; shift 2 ;;
    --pr)      PR="${2:-}"; shift 2 ;;
    --repo)    REPO="${2:-}"; shift 2 ;;
    -h|--help) usage ;;
    *)         echo "unknown arg: $1" >&2; usage ;;
  esac
done

if [[ -n "$FIXTURE" && -n "$PR" ]]; then
  echo "--fixture and --pr are mutually exclusive" >&2
  exit 2
fi
if [[ -z "$FIXTURE" && -z "$PR" ]]; then
  usage
fi

rm -rf "$WORKSPACE"
mkdir -p "$WORKSPACE/pr/files" "$WORKSPACE/reviews"

if [[ -n "$FIXTURE" ]]; then
  SRC="${FIXTURES}/${FIXTURE}"
  if [[ ! -d "$SRC" ]]; then
    echo "fixture not found: $SRC" >&2
    exit 1
  fi
  cp "$SRC/metadata.json" "$WORKSPACE/pr/metadata.json"
  cp "$SRC/diff.patch"    "$WORKSPACE/pr/diff.patch"
  if [[ -d "$SRC/files" ]]; then
    cp -r "$SRC/files/." "$WORKSPACE/pr/files/"
  fi
  echo "staged fixture '$FIXTURE' → $WORKSPACE/pr/"
else
  if [[ -z "$REPO" ]]; then
    echo "--pr requires --repo" >&2; exit 2
  fi
  command -v gh >/dev/null || { echo "gh CLI required for live mode" >&2; exit 1; }

  gh pr view "$PR" --repo "$REPO" \
    --json number,title,body,author,baseRefName,headRefName,baseRefOid,headRefOid,labels,files \
    > "$WORKSPACE/pr/metadata.json"
  gh pr diff "$PR" --repo "$REPO" > "$WORKSPACE/pr/diff.patch"

  # Fetch post-change content of each changed file (capped at 500 lines).
  HEAD_OID="$(jq -r .headRefOid "$WORKSPACE/pr/metadata.json")"
  jq -r '.files[].path' "$WORKSPACE/pr/metadata.json" | while IFS= read -r path; do
    dest="$WORKSPACE/pr/files/$path"
    mkdir -p "$(dirname "$dest")"
    if gh api "repos/${REPO}/contents/${path}?ref=${HEAD_OID}" --jq .content 2>/dev/null \
         | base64 -d \
         | head -n 500 > "$dest"; then
      :
    else
      rm -f "$dest"
    fi
  done
  echo "staged live PR #$PR from $REPO → $WORKSPACE/pr/"
fi

cd "$GROVE_ROOT"

echo ""
echo "Starting reviewer-coordinator..."
echo "Workspace: $WORKSPACE"
echo ""

scion start reviewer-coordinator \
  -t reviewer-coordinator \
  --workspace "$WORKSPACE" \
  --yes \
  "Coordinate a PR review. PR inputs are already staged at /workspace/pr/. Follow your agents.md: write review-context.md, spawn the three specialists, wait for findings, spawn the synthesizer, report completion."
