#!/usr/bin/env bash
set -e

echo "================================================================="
echo "🌌 Scaling up Chunkster Cluster to Maximum Capacity (20 Nodes)"
echo "================================================================="

mkdir -p bin
echo "🔨 Compiling node and bootstrap binaries..."
go build -o bin/bootstrap.exe ./cmd/bootstrap
go build -o bin/node.exe ./cmd/node

echo "🧹 Stopping any running cluster nodes..."
pkill -f "bin/bootstrap" || taskkill //F //IM bootstrap.exe //T >/dev/null 2>&1 || true
pkill -f "bin/node serve" || taskkill //F //IM node.exe //T >/dev/null 2>&1 || true
sleep 1

# 1. Start Bootstrap Server
echo "🌐 Launching Overlay Directory Discovery Service (:9099)..."
./bin/bootstrap.exe > storage/bootstrap.log 2>&1 &
sleep 1

# 2. Loop to spawn exactly 20 dynamic storage nodes
echo "🚀 Spawning 20 isolated storage peers across incremental ports..."
for i in {1..20}; do
  GRPC_PORT=$((50050 + i))
  HTTP_PORT=$((8079 + i))
  DATA_DIR="storage/node$i"
  
  mkdir -p "$DATA_DIR"
  
  # Launch peer in background
  ./bin/node.exe serve -data "$DATA_DIR" -grpc "127.0.0.1:$GRPC_PORT" -http "127.0.0.1:$HTTP_PORT" -bootstrap "http://127.0.0.1:9099" -replication 3 > "$DATA_DIR.log" 2>&1 &
  
  # Small stagger to prevent bootstrap UDP/TCP flood
  sleep 0.2
done

echo ""
echo "✨ Kademlia Topology Ring fully populated with 20 Active Peers!"
echo "-----------------------------------------------------------------"
echo "🔌 Peer Port Range Mapping:"
echo "   Node 1  -> gRPC: 50051 | HTTP API: 8080"
echo "   Node 2  -> gRPC: 50052 | HTTP API: 8081"
echo "   ...     -> ..."
echo "   Node 20 -> gRPC: 50070 | HTTP API: 8099"
echo "-----------------------------------------------------------------"
echo "🎨 Inspect UI: Open the React Dashboard and enter any HTTP API port (e.g., http://127.0.0.1:8080 up to http://127.0.0.1:8099) to view the localized node ring mapping!"
echo "🛑 Shutdown Command: pkill -f bin/node && pkill -f bin/bootstrap"
