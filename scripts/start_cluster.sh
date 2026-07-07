#!/usr/bin/env bash
set -e

echo "=========================================================="
echo "🚀 Bootstrapping Chunkster Distributed Cluster (3+1 Nodes)"
echo "=========================================================="

# Ensure output directories exist
mkdir -p bin
mkdir -p storage/node1 storage/node2 storage/node3

# Build binaries
echo "🔨 Compiling static Go binaries..."
go build -o bin/bootstrap ./cmd/bootstrap
go build -o bin/node ./cmd/node

# Cleanup any previous instances running locally
echo "🧹 Cleaning up background node instances..."
pkill -f "bin/bootstrap" || taskkill //F //IM bootstrap.exe //T >/dev/null 2>&1 || true
pkill -f "bin/node serve" || taskkill //F //IM node.exe //T >/dev/null 2>&1 || true
sleep 1

# Launch Bootstrap Node
echo "🌐 Starting overlay bootstrap directory service (:9099)..."
./bin/bootstrap > storage/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!

sleep 1

# Launch Peer Nodes
echo "📦 Starting Storage Peer Node 1 (gRPC:50051 / HTTP:8080)..."
./bin/node serve -data storage/node1 -grpc 127.0.0.1:50051 -http 127.0.0.1:8080 -bootstrap http://127.0.0.1:9099 -replication 3 > storage/node1.log 2>&1 &
NODE1_PID=$!

echo "📦 Starting Storage Peer Node 2 (gRPC:50052 / HTTP:8081)..."
./bin/node serve -data storage/node2 -grpc 127.0.0.1:50052 -http 127.0.0.1:8081 -bootstrap http://127.0.0.1:9099 -replication 3 > storage/node2.log 2>&1 &
NODE2_PID=$!

echo "📦 Starting Storage Peer Node 3 (gRPC:50053 / HTTP:8082)..."
./bin/node serve -data storage/node3 -grpc 127.0.0.1:50053 -http 127.0.0.1:8082 -bootstrap http://127.0.0.1:9099 -replication 3 > storage/node3.log 2>&1 &
NODE3_PID=$!

echo ""
echo "✨ Cluster operational! Network layer routing initialized."
echo "----------------------------------------------------------"
echo "  Bootstrap Logs : cat storage/bootstrap.log"
echo "  Node 1 Logs    : cat storage/node1.log"
echo "  Node 2 Logs    : cat storage/node2.log"
echo "  Node 3 Logs    : cat storage/node3.log"
echo "----------------------------------------------------------"
echo "👉 Run frontend dashboard: cd frontend && npm run dev"
echo "To shutdown cluster: pkill -f bin/node && pkill -f bin/bootstrap"
