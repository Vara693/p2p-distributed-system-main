#!/usr/bin/env bash

echo "=========================================="
echo "🛑 Terminating all Chunkster instances"
echo "=========================================="

# Forcefully kill processes on Windows
taskkill //F //IM node.exe //T 2>/dev/null || true
taskkill //F //IM bootstrap.exe //T 2>/dev/null || true

# Also try pkill for Linux/WSL compatibility
pkill -9 -f "bin/node" 2>/dev/null || true
pkill -9 -f "bin/bootstrap" 2>/dev/null || true

echo "✅ All background cluster nodes and bootstrap servers have been stopped successfully."
