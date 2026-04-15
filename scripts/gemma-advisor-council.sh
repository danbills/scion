#!/bin/bash
set -euo pipefail

# Point every scion invocation at the local hub so agents appear in the
# dashboard at http://127.0.0.1:8080. Override by exporting before invocation.
: "${SCION_HUB_ENDPOINT:=http://127.0.0.1:8080}"
export SCION_HUB_ENDPOINT

# Civ 5 Advisor Council — sequential turn-based multi-agent demo, hub mode.
#
# For each of three scenarios, spawn four opencode+gemma agents one at a time.
# Each advisor's container is provisioned via the hub: the hub tells the broker
# to clone the grove's git remote, and the advisor's commits are pushed back to
# origin so the coordinator and the next advisor can see them.
#
# Usage: scripts/gemma-advisor-council.sh <grove-path>
#
# Requires:
#   - Hub running at SCION_HUB_ENDPOINT with the grove linked and these secrets:
#       * GITHUB_TOKEN (Contents: read+write on the grove's repo)
#       * OPENAI_API_KEY (any non-empty value; llama.cpp ignores it)
#   - ~/.scion/harness-configs/gemma-local/  AND  <grove>/.scion/harness-configs/gemma-local/
#   - Advisor templates: .scion/templates/advisor-{economic,foreign,military,scientific}/
#   - <grove-path>/{POTATO_FAMINE,TORNADO,ALLY_INVASION}.md scenario files
#   - `git remote get-url origin` resolves to a GitHub repo the user can push to
#     via ssh (host-side) and that sciontool can clone via https+GITHUB_TOKEN.

GROVE="${1:-}"
if [[ -z "$GROVE" || ! -d "$GROVE/.scion" ]]; then
  echo "usage: $0 <grove-path>" >&2
  exit 2
fi

ADVISORS=(economic foreign military scientific)
SCENARIOS=(POTATO_FAMINE TORNADO ALLY_INVASION)

PER_TURN_TIMEOUT=900   # 15 min per advisor per scenario (gemma is slow)

cd "$GROVE"

# Default branch on the hub-linked remote.
BASE=main
if ! git rev-parse --verify "$BASE" >/dev/null 2>&1; then
  if git rev-parse --verify master >/dev/null 2>&1; then BASE=master; fi
fi
echo "base branch: $BASE"

# Push anything the coordinator has staged locally but not yet pushed. Ensures
# the advisor containers see scenarios + harness configs + templates.
git fetch origin --prune
git -c user.email=council@demo -c user.name=council checkout "$BASE"
git push origin "$BASE"

remote_sha() {
  local branch=$1
  gh api "repos/{owner}/{repo}/branches/$branch" --jq .commit.sha 2>/dev/null || true
}

wait_for_remote_commit() {
  local branch=$1 baseline_sha=$2 deadline
  deadline=$(( $(date +%s) + PER_TURN_TIMEOUT ))
  while :; do
    local cur
    cur=$(remote_sha "$branch")
    if [[ -n "$cur" && "$cur" != "$baseline_sha" ]]; then
      echo "  new commit on origin/$branch: $cur"
      return 0
    fi
    if (( $(date +%s) > deadline )); then
      echo "TIMEOUT waiting for commit on origin/$branch" >&2
      return 1
    fi
    sleep 15
    printf "."
  done
}

cleanup_agent() {
  local name=$1
  scion -g "$GROVE" stop "$name" --rm 2>/dev/null || true
  scion -g "$GROVE" delete "$name" --yes 2>/dev/null || true
}

for scen in "${SCENARIOS[@]}"; do
  if [[ ! -f "$GROVE/$scen.md" ]]; then
    echo "missing scenario file $scen.md; skipping" >&2
    continue
  fi

  council="council-${scen,,}"
  transcript="TRANSCRIPT-${scen}.md"

  echo
  echo "============================================================"
  echo " Scenario: $scen  (branch: $council, transcript: $transcript)"
  echo "============================================================"

  # Create scenario branch with empty transcript header; push to origin so the
  # first advisor's clone checks out the right state.
  git branch -D "$council" 2>/dev/null || true
  git checkout -B "$council" "$BASE"
  printf "# Council Transcript: %s\n\nAdvisors speak in order: Economic, Foreign, Military, Scientific.\n\n" \
    "$scen" > "$transcript"
  git -c user.email=council@demo -c user.name=council add "$transcript"
  git -c user.email=council@demo -c user.name=council commit -m "council: open ${scen} transcript" >/dev/null
  git push -f origin "$council"
  git checkout "$BASE"

  for role in "${ADVISORS[@]}"; do
    agent="advisor-${role}"
    cleanup_agent "$agent"

    ROLE_CAP="$(echo "$role" | sed 's/./\U&/')"
    TASK="You are the ${ROLE_CAP} Advisor. Read the scenario briefing in ${scen}.md and the running discussion in ${transcript}. Append a new section to ${transcript} with the exact heading \"## ${ROLE_CAP} Advisor\" giving your concise advice (under 200 words). If prior advisors have already written sections in ${transcript}, you MUST explicitly reference at least one of them by role name, stating whether you agree or disagree and why. Then run: git -c user.email=${role}@council -c user.name=${role}-advisor add ${transcript} && git -c user.email=${role}@council -c user.name=${role}-advisor commit -m \"${role}: advice on ${scen}\" && git push origin HEAD:${council}. Do not ask any questions; do not wait for input."

    # Baseline SHA for the council branch before the advisor runs — we detect
    # completion when origin/$council advances past this point.
    baseline=$(remote_sha "$council")
    echo
    echo "--- Turn: ${ROLE_CAP} Advisor on ${scen} (baseline ${baseline:0:8}) ---"

    scion -g "$GROVE" start "$agent" \
      --branch "$council" \
      -t "$agent" \
      --harness-config gemma-local \
      --yes "$TASK" >/dev/null

    if wait_for_remote_commit "$council" "$baseline"; then
      echo " ✓ ${ROLE_CAP} pushed to origin/${council}"
      # Pull the advisor's commit into the local council branch so the next
      # printout + any manual inspection reflects the full transcript.
      git fetch origin "$council"
      git checkout "$council"
      git reset --hard "origin/${council}"
      git checkout "$BASE"
    else
      echo " ✗ ${ROLE_CAP} did not push in time; continuing"
    fi

    scion -g "$GROVE" stop "$agent" --rm 2>/dev/null || true
  done

  echo
  echo "=== Final transcript: $transcript (branch $council) ==="
  git show "origin/$council:$transcript" 2>/dev/null | sed 's/^/  /' || \
    git show "$council:$transcript" | sed 's/^/  /'
done

echo
echo "DONE. Council branches on origin:"
for scen in "${SCENARIOS[@]}"; do
  echo "  council-${scen,,}"
done
