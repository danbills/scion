#!/bin/bash
set -euo pipefail

# Gemma multi-agent fan-out demo.
#
# Spawns two opencode agents backed by the local gemma-4-26B endpoint, each on
# its own git worktree, each doing a small independent task. Waits for both to
# finish, then prints their branches so a human can merge.
#
# Usage: scripts/gemma-multiagent-demo.sh <grove-path>
#
# The grove must already be initialized (scion init) and contain FILE1.md and
# FILE2.md at the root.

GROVE="${1:-}"
if [[ -z "$GROVE" || ! -d "$GROVE/.scion" ]]; then
  echo "usage: $0 <grove-path>" >&2
  echo "  grove must contain .scion/ (run 'scion init' there first)" >&2
  exit 2
fi

TASK1="Append the line 'AGENT-1 was here' to FILE1.md at the repo root. Then run: git add FILE1.md && git -c user.email=a1@demo -c user.name=agent1 commit -m 'agent1: mark FILE1'."
TASK2="Append the line 'AGENT-2 was here' to FILE2.md at the repo root. Then run: git add FILE2.md && git -c user.email=a2@demo -c user.name=agent2 commit -m 'agent2: mark FILE2'."

for a in impl-1 impl-2; do
  if scion -g "$GROVE" list 2>/dev/null | awk '{print $1}' | grep -qx "$a"; then
    echo "stopping/removing existing agent $a"
    scion -g "$GROVE" stop "$a" --rm 2>/dev/null || true
    scion -g "$GROVE" delete "$a" --yes 2>/dev/null || true
  fi
done

echo "launching impl-1..."
scion -g "$GROVE" start impl-1 --harness-config gemma-local --yes "$TASK1" >/dev/null
echo "launching impl-2..."
scion -g "$GROVE" start impl-2 --harness-config gemma-local --yes "$TASK2" >/dev/null

echo "polling for completion (ctrl-c to abort)..."
deadline=$(( $(date +%s) + 900 ))  # 15 min budget
while :; do
  now=$(date +%s)
  if (( now > deadline )); then
    echo "TIMEOUT after 15min" >&2
    break
  fi
  s1=$(podman ps -a --format "{{.Names}}|{{.Status}}" | awk -F'|' -v n="$(basename "$GROVE")--impl-1" '$1==n{print $2}')
  s2=$(podman ps -a --format "{{.Names}}|{{.Status}}" | awk -F'|' -v n="$(basename "$GROVE")--impl-2" '$1==n{print $2}')
  printf "  impl-1: %-30s  impl-2: %-30s\n" "${s1:-?}" "${s2:-?}"
  [[ "$s1" == Exited* && "$s2" == Exited* ]] && break
  sleep 20
done

echo
echo "=== RESULT ==="
cd "$GROVE"
for b in impl-1 impl-2; do
  echo "--- branch: $b ---"
  git log --oneline "$b" ^master 2>/dev/null | head -5 || echo "(no branch)"
  git diff --stat master.."$b" 2>/dev/null || true
done
