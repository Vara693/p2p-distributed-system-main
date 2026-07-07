#!/usr/bin/env bash
# start_deployed_cluster.sh
# Starts ONLY the Bootstrap discovery server.
# Start your storage node separately with your advertise flags (see NETWORK_COMMANDS.md).

echo "================================================================="
echo "🌐 Starting Bootstrap Discovery Server"
echo "================================================================="

mkdir -p bin storage
echo "🔨 Compiling latest binaries..."
go build -o bin/bootstrap.exe ./cmd/bootstrap
go build -o bin/node.exe ./cmd/node

echo "🧹 Stopping any previously running processes..."
pkill -f "bin/bootstrap" 2>/dev/null || taskkill //F //IM bootstrap.exe //T >/dev/null 2>&1 || true
pkill -f "bin/node serve" 2>/dev/null || taskkill //F //IM node.exe //T >/dev/null 2>&1 || true
sleep 1

echo "🌐 Launching Overlay Directory Discovery Service (:9099)..."
mkdir -p storage
./bin/bootstrap.exe > storage/bootstrap.log 2>&1 &
BOOTSTRAP_PID=$!
sleep 1

echo ""
echo "✅ Bootstrap server is running! (PID: $BOOTSTRAP_PID)"
echo "-----------------------------------------------------------------"
echo "  Bootstrap Logs : tail -f storage/bootstrap.log"
echo "-----------------------------------------------------------------"
echo ""
echo "👉 NOW start YOUR node in a new terminal with your advertise flags:"
echo ""
echo "   ./bin/node.exe serve \\"
echo "     -data ./storage/node1 \\"
echo "     -grpc 0.0.0.0:50051 \\"
echo "     -http 0.0.0.0:8080 \\"
echo "     -advertise-grpc <YOUR_TAILSCALE_IP>:50051 \\"
echo "     -advertise-http <YOUR_NGROK_URL> \\"
echo "     -bootstrap http://127.0.0.1:9099 \\"
echo "     -replication 2"
echo ""
echo "-----------------------------------------------------------------"
echo "👉 To stop everything: sh ./scripts/stop_cluster.sh"
