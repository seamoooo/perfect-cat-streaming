#!/usr/bin/env bash
# Drive a real Chromium against the dev app in a forever loop so NR Browser
# Agent + Video Agent keep emitting telemetry continuously. Designed to run
# on the host (not in docker) — uses the host's localhost networking so the
# AJAX URLs and HLS manifest URLs Just Work.
#
# Stop with Ctrl+C.

set -euo pipefail

cd "$(dirname "$0")/.."

ITERS_PER_BATCH="${ITERS_PER_BATCH:-10}"
PLAYBACK_MS="${PLAYBACK_MS:-8000}"
PAUSE_MS="${PAUSE_MS:-1500}"
PAUSE_BETWEEN_BATCHES="${PAUSE_BETWEEN_BATCHES:-5}"  # seconds
API_BASE="${API_BASE:-http://localhost:8080}"
E2E_BASE_URL="${E2E_BASE_URL:-http://localhost:5173}"

cat <<EOF
[forever] starting NR browser-driven loop
  iters per batch:   $ITERS_PER_BATCH
  playback per clip: ${PLAYBACK_MS}ms
  pause between batches: ${PAUSE_BETWEEN_BATCHES}s
  app:               $E2E_BASE_URL  (API: $API_BASE)
  Press Ctrl+C to stop.
EOF

# Quick sanity check — fail loudly if backend / frontend isn't up.
for u in "$E2E_BASE_URL" "$API_BASE/healthz"; do
  if ! curl -fsS -m 5 "$u" >/dev/null; then
    echo "[forever] $u is not reachable — start the dev stack with 'make up'" >&2
    exit 1
  fi
done

trap 'echo "[forever] stopping"; exit 0' INT TERM

batch=0
cd frontend
while true; do
  batch=$((batch + 1))
  start_ts=$(date '+%H:%M:%S')
  echo "[forever] batch #$batch start=$start_ts"
  if ! ITERS="$ITERS_PER_BATCH" \
       PLAYBACK_MS="$PLAYBACK_MS" \
       PAUSE_MS="$PAUSE_MS" \
       API_BASE="$API_BASE" \
       E2E_BASE_URL="$E2E_BASE_URL" \
       npx playwright test e2e/telemetry-loop.spec.ts --reporter=line; then
    echo "[forever] batch #$batch failed (continuing in ${PAUSE_BETWEEN_BATCHES}s)"
  fi
  sleep "$PAUSE_BETWEEN_BATCHES"
done
