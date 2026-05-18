#!/usr/bin/env bash

set -euo pipefail

# Optional: dev config mounted at /workspace/dev.auto.yaml
CONFIG_FILE=""
if [ -f "/workspace/dev.auto.yaml" ]; then
  CONFIG_FILE="dev.auto.yaml"
fi

# Start wsproxy (websocket -> UDP ENet proxy) if present
if [ -x "/workspace/proxy/wsproxy" ]; then
  # Run wsproxy to handle /service/proxy/u/<host:port>
  # We will front this via the Go server later; for now, bind on 1338
  /workspace/proxy/wsproxy 1338 &
  PROXY_PID=$!
fi

if [ -n "$CONFIG_FILE" ]; then
  /workspace/bin/sour serve "$CONFIG_FILE" &
else
  /workspace/bin/sour serve &
fi
SERVER_PID=$!
# Ensure the script stops and cleans up on SIGINT and SIGTERM

cleanup() {
  [[ -n "${PROXY_PID:-}" ]] && kill "${PROXY_PID}" >/dev/null 2>&1 || true
  [[ -n "${SERVER_PID:-}" ]] && kill "${SERVER_PID}" >/dev/null 2>&1 || true
  exit 0
}

trap cleanup INT TERM

# Wait specifically for the server process - if it dies, the container should exit
wait $SERVER_PID

# If we reach here, the server has exited - clean up the proxy
[[ -n "${PROXY_PID:-}" ]] && kill "${PROXY_PID}" >/dev/null 2>&1 || true
