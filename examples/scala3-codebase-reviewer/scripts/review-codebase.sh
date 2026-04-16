#!/bin/bash
set -euo pipefail

# review-codebase.sh — drive the Scala 3 codebase-reviewer demo.
#
# Starts the coordinator in a hub-native scion grove. The coordinator clones
# the target repo into the shared /workspace/code/ at startup, then spawns
# four specialists (iron, syntax, cats, effects) and a synthesizer that
# produces a single ranked roadmap. All agents in the grove share
# /workspace/ (this is the hub-native scion pattern, same as scion-athenaeum).
#
# Usage:
#   review-codebase.sh --repo <owner/name> [--branch <branch>]
#
# Requires: scion server running with hub enabled, grove hub-linked, gh auth.

GROVE_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

REPO=""
BRANCH=""

usage() {
  cat <<USAGE >&2
Usage: $(basename "$0") --repo <owner/name> [--branch <branch>]
USAGE
  exit 2
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --repo)   REPO="${2:-}"; shift 2 ;;
    --branch) BRANCH="${2:-}"; shift 2 ;;
    -h|--help) usage ;;
    *)        echo "unknown arg: $1" >&2; usage ;;
  esac
done

[[ -z "$REPO" ]] && usage

command -v gh >/dev/null || { echo "gh CLI required" >&2; exit 1; }

if ! gh auth status >/dev/null 2>&1; then
  echo "gh is not authenticated. Run: gh auth login" >&2
  exit 1
fi
if ! gh repo view "$REPO" >/dev/null 2>&1; then
  echo "gh cannot access $REPO. Check the name and your auth scopes." >&2
  exit 1
fi

cd "$GROVE_ROOT"

export SCION_HUB_ENDPOINT="${SCION_HUB_ENDPOINT:-http://localhost:8080}"

WORKSPACE="$HOME/.scion/groves/scala3-codebase-reviewer"
mkdir -p "$WORKSPACE"
if [ ! -d "$WORKSPACE/code" ] || [ -z "$(ls -A "$WORKSPACE/code" 2>/dev/null)" ]; then
  echo "Cloning $REPO into $WORKSPACE/code..."
  rm -rf "$WORKSPACE/code"
  gh repo clone "$REPO" "$WORKSPACE/code" -- --depth 1 ${BRANCH:+--branch "$BRANCH"}
else
  echo "Reusing existing clone at $WORKSPACE/code"
fi
rm -rf "$WORKSPACE/reviews"

echo "Starting codebase-reviewer-coordinator for $REPO..."
echo ""

scion start codebase-reviewer-coordinator \
  -t codebase-reviewer-coordinator \
  --harness-config gemma-local \
  --yes \
  "Coordinate a whole-codebase Scala 3 modernization review of ${REPO}${BRANCH:+ (branch ${BRANCH})}. Follow your agents.md: clone the target into /workspace/code/, write review-context.md, spawn the four specialists, wait for proposals, spawn the synthesizer, report completion. TARGET_REPO=${REPO} TARGET_BRANCH=${BRANCH:-main}"
