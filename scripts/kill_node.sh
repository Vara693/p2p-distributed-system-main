#!/usr/bin/env bash
set -e

PORT=$1

if [ -z "$PORT" ]; then
  echo "Usage: ./scripts/kill_node.sh <http_port>"
  echo "Example: ./scripts/kill_node.sh 8081"
  exit 1
fi

echo "💥 Simulating node crash/failure for instance bound to HTTP port $PORT..."
pkill -f "http 127.0.0.1:$PORT" || pkill -f "http.*$PORT" || true

# Precise Windows targeted process termination by network port
if command -v netstat >/dev/null 2>&1; then
  PID=$(netstat -ano | grep ":$PORT" | awk '{print $5}' | tr -d '\r' | head -n 1)
  if [ -n "$PID" ] && [ "$PID" != "0" ]; then
    taskkill //F //PID "$PID" //T >/dev/null 2>&1 || true
  fi
fi

echo "✅ Node terminated. Overlay heartbeat timeouts will mark this peer inactive shortly."
