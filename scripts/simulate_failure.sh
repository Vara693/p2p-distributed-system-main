#!/usr/bin/env bash
set -e

echo "========================================================================="
echo "🛡️  Simulating Distributed Storage Fault Tolerance & File Recovery Workflow"
echo "========================================================================="

# Ensure cluster is fully clean and started
./scripts/start_cluster.sh
sleep 4

echo "📝 Creating 1MB random payload file (test_payload.bin)..."
dd if=/dev/urandom of=storage/test_payload.bin bs=1024 count=1000 status=none

echo "📤 Uploading test_payload.bin to Node 1 (Ingestion Gateway on :8080)..."
UPLOAD_RESP=$(curl -s -X POST -F "file=@storage/test_payload.bin" http://127.0.0.1:8080/api/upload)
echo "   Response: $UPLOAD_RESP"

ROOT_CID=$(echo $UPLOAD_RESP | grep -o '"root_cid":"[^"]*' | cut -d'"' -f4)
if [ -z "$ROOT_CID" ]; then
  echo "❌ Upload failed to return root CID"
  exit 1
fi
echo "🔑 Extracted Root CID: $ROOT_CID"
sleep 2

echo "💥 Injecting catastrophic failure: forcefully crashing Node 2 (:8081)..."
./scripts/kill_node.sh 8081
sleep 5

echo "⏳ Waiting 12 seconds for overlay heartbeat timeout checks to mark Node 2 Inactive..."
sleep 12

echo "📥 Querying remaining active nodes to reconstruct file via Node 3 resolver (:8082)..."
curl -s -o storage/reconstructed.bin "http://127.0.0.1:8082/api/download?cid=$ROOT_CID"

echo "⚖️ Verifying cryptographic SHA-256 signatures of original vs recovered payloads..."
if command -v sha256sum >/dev/null 2>&1; then
  ORIG_HASH=$(sha256sum storage/test_payload.bin | awk '{print $1}')
  RECOV_HASH=$(sha256sum storage/reconstructed.bin | awk '{print $1}')
  echo "   Original SHA-256 : $ORIG_HASH"
  echo "   Recovered SHA-256: $RECOV_HASH"
  if [ "$ORIG_HASH" = "$RECOV_HASH" ]; then
    echo "🎉 SUCCESS: Recovered file is a bit-perfect binary match! Fault tolerance verified."
  else
    echo "❌ FAILURE: Checksums do not match."
    exit 1
  fi
else
  echo "⚠️ sha256sum utility not available, skipping strict byte comparison."
  ls -lh storage/test_payload.bin storage/reconstructed.bin
fi
