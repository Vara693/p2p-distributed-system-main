@echo off
echo ==========================================
echo 🛑 Terminating all Chunkster instances
echo ==========================================

taskkill /F /IM node.exe /T 2>nul
taskkill /F /IM bootstrap.exe /T 2>nul

echo ✅ All background cluster nodes and bootstrap servers have been stopped successfully.
