#!/bin/bash
set -euo pipefail

# test-gemma-dispatch.sh — iterate on prompt variants to get Gemma 4 26B
# to reliably invoke bash tool calls (specifically `scion start`).
#
# Usage:
#   ./test-gemma-dispatch.sh                  # run all variants
#   ./test-gemma-dispatch.sh 03-literal       # run one variant by prefix

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VARIANTS_DIR="$SCRIPT_DIR/prompt-variants"
OUTPUT_DIR="$SCRIPT_DIR/.test-output"
MODEL="llama-big/gemma-4-26B-it-UD-Q4_K_M.gguf"
TIMEOUT_SECS=120

mkdir -p "$OUTPUT_DIR"

FILTER="${1:-}"

pass=0
fail=0
total=0

run_variant() {
  local name="$1"
  local system_file="$VARIANTS_DIR/${name}.system.md"
  local task_file="$VARIANTS_DIR/${name}.task.txt"
  local output_file="$OUTPUT_DIR/run-${name}.jsonl"

  if [ ! -f "$task_file" ]; then
    echo "SKIP: $name (no task file)"
    return
  fi

  local task
  task=$(<"$task_file")

  # opencode run has no --system-prompt flag, so we prepend system prompt to the task
  if [ -f "$system_file" ]; then
    local system
    system=$(<"$system_file")
    task="$system

---

$task"
  fi

  total=$((total + 1))
  printf "%-30s " "$name"

  local start_time
  start_time=$(date +%s)

  # Run with timeout
  timeout "$TIMEOUT_SECS" opencode run \
    --model "$MODEL" \
    --format json \
    "$task" > "$output_file" 2>&1 || true

  local end_time
  end_time=$(date +%s)
  local elapsed=$((end_time - start_time))

  # Detect bash tool calls containing "scion start"
  local bash_calls
  bash_calls=$(jq -r 'select(.type=="tool_use") | .part.tool // empty' "$output_file" 2>/dev/null | grep -c "bash" || true)

  local scion_start_calls
  scion_start_calls=$(jq -r 'select(.type=="tool_use") | select(.part.tool=="bash") | .part.state.input.command // empty' "$output_file" 2>/dev/null | grep -c "scion start" || true)

  # Count any bash calls at all (even non-scion)
  local any_bash
  any_bash=$(jq -r 'select(.type=="tool_use") | select(.part.tool=="bash") | .part.state.input.command // empty' "$output_file" 2>/dev/null | head -5)

  # Get text output length
  local text_len
  text_len=$(jq -r 'select(.type=="text") | .part.text // empty' "$output_file" 2>/dev/null | wc -c || echo 0)

  if [ "$scion_start_calls" -ge 1 ]; then
    echo "PASS  (${elapsed}s, ${scion_start_calls} scion-start calls, ${bash_calls} total bash calls, ${text_len}c text)"
    pass=$((pass + 1))
  elif [ "$bash_calls" -ge 1 ]; then
    echo "PARTIAL  (${elapsed}s, bash called ${bash_calls}x but no scion-start, ${text_len}c text)"
    echo "  bash commands: $(echo "$any_bash" | head -3 | tr '\n' ' ; ')"
    fail=$((fail + 1))
  else
    echo "FAIL  (${elapsed}s, no bash calls, ${text_len}c text)"
    # Show what model actually said
    local text_preview
    text_preview=$(jq -r 'select(.type=="text") | .part.text // empty' "$output_file" 2>/dev/null | head -3 | tr '\n' ' ')
    [ -n "$text_preview" ] && echo "  model said: ${text_preview:0:120}..."
    fail=$((fail + 1))
  fi
}

echo "=== Gemma 4 26B Dispatch Test ==="
echo "Model: $MODEL"
echo "Timeout: ${TIMEOUT_SECS}s per variant"
echo ""

# Collect variant names (strip .task.txt suffix, deduplicate)
variants=()
for f in "$VARIANTS_DIR"/*.task.txt; do
  [ -f "$f" ] || continue
  name=$(basename "$f" .task.txt)
  if [ -z "$FILTER" ] || [[ "$name" == *"$FILTER"* ]]; then
    variants+=("$name")
  fi
done

if [ ${#variants[@]} -eq 0 ]; then
  echo "No variants found matching '${FILTER}' in $VARIANTS_DIR"
  exit 1
fi

for name in "${variants[@]}"; do
  run_variant "$name"
done

echo ""
echo "=== Summary: $pass PASS / $fail FAIL / $total total ==="
