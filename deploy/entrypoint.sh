#!/bin/sh
# All-in-one Herald process supervisor:
#   - N× herald-worker (asynq consumers; DELIVERY_WORKER_COUNT)
#   - herald-api :8080 — only published port
# Remote Delivery Workers (e.g. befship-notification) run as separate processes/containers.
set -eu

DELIVERY_WORKER_COUNT="${DELIVERY_WORKER_COUNT:-1}"
HERALD_API_BIN="${HERALD_API_BIN:-/app/herald-api}"
HERALD_WORKER_BIN="${HERALD_WORKER_BIN:-/app/herald-worker}"

pids=""

cleanup() {
  for pid in $pids; do
    kill "$pid" 2>/dev/null || true
  done
  wait 2>/dev/null || true
}
trap cleanup INT TERM

i=1
while [ "$i" -le "$DELIVERY_WORKER_COUNT" ]; do
  echo "starting herald-worker #$i / $DELIVERY_WORKER_COUNT (queue consumer)"
  "$HERALD_WORKER_BIN" &
  pids="$pids $!"
  i=$((i + 1))
done

echo "starting herald-api (DELIVERY_WORKER_COUNT=${DELIVERY_WORKER_COUNT})"
"$HERALD_API_BIN" &
api_pid=$!
pids="$pids $api_pid"

wait "$api_pid"
status=$?
cleanup
trap - INT TERM
exit "$status"
