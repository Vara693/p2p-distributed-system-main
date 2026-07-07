# Chunkster — Deployment Guide

> Step-by-step commands for deploying the Chunkster P2P network globally (Tailscale + Ngrok) or on a local LAN.

---

> **Legend for this document:**
> - 🔴 `CHANGE THIS` — You must replace this value before running the command
> - ✅ Run as-is — No changes needed
> - 🖥️ Run on YOUR laptop (the host)
> - 👥 Run on your FRIEND'S PC

---

## Scenario A: Global Internet Deployment (Tailscale + Ngrok)

### For 5 users anywhere in the world

This is the recommended setup when you and your friends are in **different locations/networks**. Tailscale creates a secure P2P mesh so nodes can transfer file chunks directly. Ngrok creates a public HTTPS URL so each person's browser (Vercel frontend) can connect to their local node.

---

### STEP 1 — Install and Set Up Tailscale (EVERY person does this once)

1. Every participant downloads Tailscale from: **https://tailscale.com/download**
2. Install it (Windows installer, takes 30 seconds).
3. The **host (you)** logs in at **https://login.tailscale.com** using Google or GitHub.
4. **Each friend** goes to the same login page and signs in with the **same account** (simplest) or the host shares an invite link from the Admin Console.
5. Once logged in, all devices automatically appear on the same private mesh network.

**After this step, everyone clicks the Tailscale icon in their taskbar tray and notes their `100.x.x.x` IP address.**

---

### STEP 2 — Know Your IPs (Fill in these values for yourself)

Before running any commands, note down everyone's Tailscale IP and fill them in wherever you see 🔴:

```
YOUR Tailscale IP:     100.xx.xx.xx    ← Get this from your Tailscale app
Friend 1 Tailscale IP: 100.xx.xx.xx
Friend 2 Tailscale IP: 100.xx.xx.xx
Friend 3 Tailscale IP: 100.xx.xx.xx
Friend 4 Tailscale IP: 100.xx.xx.xx
```

---

### STEP 3 — Start the Bootstrap Server (Host only 🖥️)

Open your Git Bash terminal in the project root and run:

```bash
# ✅ Run as-is — compiles fresh binaries and starts the bootstrap server
sh ./scripts/start_deployed_cluster.sh
```

This will:
- Compile `bin/bootstrap.exe` and `bin/node.exe`
- Start the Bootstrap discovery server on port `9099`

Wait for the terminal to print `✅ Bootstrap server is running!` before continuing.

---

### STEP 4 — Start Your Ngrok HTTP Tunnel (Host 🖥️)

Open a **new terminal window** and run:

```bash
# ✅ Run as-is
ngrok http 8080
```

The Ngrok terminal will show something like:
```
Forwarding    https://abc123.ngrok-free.app → http://localhost:8080
```

**Copy your HTTPS Ngrok URL.** 🔴 You will need it in Step 5.

---

### STEP 5 — Start Your Node With Advertise Flags (Host 🖥️)

Open a **new terminal window** and run (replacing the 🔴 values):

**PowerShell version:**
```powershell
# 🔴 Replace 100.xx.xx.xx with YOUR Tailscale IP
# 🔴 Replace https://abc123.ngrok-free.app with YOUR Ngrok URL from Step 4
.\bin\node.exe serve `
  -data .\storage\node1 `
  -grpc 0.0.0.0:50051 `
  -http 0.0.0.0:8080 `
  -advertise-grpc 100.xx.xx.xx:50051 `
  -advertise-http https://abc123.ngrok-free.app `
  -bootstrap http://127.0.0.1:9099 `
  -replication 2
```

**Git Bash version:**
```bash
./bin/node.exe serve \
  -data ./storage/node1 \
  -grpc 0.0.0.0:50051 \
  -http 0.0.0.0:8080 \
  -advertise-grpc 100.xx.xx.xx:50051 \
  -advertise-http https://abc123.ngrok-free.app \
  -bootstrap http://127.0.0.1:9099 \
  -replication 2
```

**Single-line version (works in any terminal):**
```
node.exe serve -data ./storage/node1 -grpc 0.0.0.0:50051 -http 0.0.0.0:8080 -advertise-grpc 100.xx.xx.xx:50051 -advertise-http https://abc123.ngrok-free.app -bootstrap http://127.0.0.1:9099 -replication 2
```

---

### STEP 6 — Each Friend Runs Their Own Ngrok Tunnel (Friends 👥)

Each friend opens Ngrok in a terminal on their PC:

```bash
# ✅ Every friend runs this on their own PC
ngrok http 8080
```

Each friend copies their own unique Ngrok HTTPS URL.

---

### STEP 7 — Each Friend Starts Their Node (Friends 👥)

Each friend places `node.exe` in a folder on their PC and opens a terminal. They run (replacing 🔴 values):

```
node.exe serve -data ./storage/mynode -grpc 0.0.0.0:50051 -http 0.0.0.0:8080 -advertise-grpc [THEIR_TAILSCALE_IP]:50051 -advertise-http [THEIR_NGROK_URL] -bootstrap http://[HOST_TAILSCALE_IP]:9099 -replication 2
```

> **Pattern:** Every friend's `-bootstrap` always points to YOUR Tailscale IP on port `9099`. Their own `-advertise-grpc` uses their own Tailscale IP. Their own `-advertise-http` uses their own Ngrok URL.

---

### STEP 8 — Connect to the Dashboard (Everyone)

1. **First**, open your Ngrok URL directly in a browser tab (e.g., `https://abc123.ngrok-free.app/api/health`) and click **"Visit Site"** on the Ngrok warning page. This sets a browser cookie that bypasses the warning for all future API calls.
2. Open the deployed Vercel frontend: **[Chunkster Dashboard](https://p2p-distributed-system-vite-fronten.vercel.app)**
3. In the entry box, paste **YOUR OWN** Ngrok HTTPS URL and click **Join**.
4. After a few seconds, all 5 nodes appear in the **ONLINE SYSTEMS REGISTRY** table!

---

### STEP 9 — Share Files!

- **Upload:** Drag any file onto the upload panel on your dashboard. The system splits it into chunks and replicates them across all nodes automatically.
- **Download:** Copy the CID shown after upload. Any friend can paste this CID into their search box and download the file from the network.

---

### STEP 10 — Shut Down the Network (Host 🖥️)

When you are done:
```bash
# ✅ Run as-is in the project root
sh ./scripts/stop_cluster.sh
```

---

---

## Scenario B: Same Local Wi-Fi Network (LAN)

### No Ngrok, No Tailscale, No Deployed Site Required

This is the simplest setup. If everyone is connected to **the same Wi-Fi router** (same room, same home network, college lab, etc.), you connect directly using local IP addresses. No internet services are needed at all.

> **Key advantage:** Your friend does NOT need the source code. They only need `node.exe`. The frontend can run locally on any machine that has the `frontend/` folder.

---

### STEP 1 — Find Everyone's Local Wi-Fi IP

**Every person** opens their terminal/command prompt and runs:

```cmd
# ✅ Run on every PC (Windows Command Prompt or PowerShell)
ipconfig
```

Look for **"IPv4 Address"** under **"Wireless LAN adapter Wi-Fi"**.
Example output:
```
Wireless LAN adapter Wi-Fi:
   IPv4 Address. . . . . . . : 192.168.1.15   ← This is your LAN IP
```

Note down everyone's LAN IP. Replace 🔴 values in the commands below with your actual IPs.

---

### STEP 2 — Start the Bootstrap Server and Your Node (Host 🖥️)

Open your Git Bash terminal in the project root and run:

```bash
# ✅ Run as-is
sh ./scripts/start_deployed_cluster.sh
```

Then open a **new terminal window** and start your node advertising your LAN IP:

```bash
# 🔴 Replace 192.168.1.15 with YOUR actual Wi-Fi IP from Step 1
./bin/node.exe serve \
  -data ./storage/node1 \
  -grpc 0.0.0.0:50051 \
  -http 0.0.0.0:8080 \
  -advertise-grpc 192.168.1.15:50051 \
  -bootstrap http://127.0.0.1:9099 \
  -replication 2
```

---

### STEP 3 — Each Friend Starts Their Node (Friends 👥)

Each friend places `node.exe` in a folder on their PC and opens a terminal. They run:

```
node.exe serve -data ./storage/mynode -grpc 0.0.0.0:50051 -http 0.0.0.0:8080 -advertise-grpc [THEIR_WIFI_IP]:50051 -bootstrap http://[HOST_WIFI_IP]:9099 -replication 2
```

> **Pattern:** `-advertise-grpc` = their own LAN IP. `-bootstrap` = always the host's LAN IP on port `9099`.

---

### STEP 4 — Allow Through Windows Firewall (Friends 👥 — Critical!)

The first time `node.exe` runs, Windows Defender Firewall will show a popup asking:
> *"Do you want to allow node.exe to communicate on private and public networks?"*

**Both checkboxes (Private AND Public) must be checked, then click "Allow Access".**

If this popup was cancelled or missed, open Windows Defender Firewall → "Allow an app through firewall" → find `node.exe` → check both Private and Public boxes.

---

### STEP 5 — Access the Frontend

Since you are on a local network, there is **no need for the deployed Vercel site**.

**Option A: Everyone who has the `frontend/` folder runs it locally**

```bash
cd path/to/frontend
npm install    # first time only
npm run dev
```

Open **`http://localhost:5173`** in your browser. In the entry box, type **`http://127.0.0.1:8080`** and click **Join**.

**Option B: Your friend does NOT have the frontend code**

Your friend can access your frontend directly from their browser using your laptop's LAN IP:

1. You run `npm run dev` on your laptop.
2. Your friend navigates to `http://192.168.1.15:5173` (🔴 your LAN IP).
3. In the entry box, your friend types **`http://127.0.0.1:8080`** and clicks **Join**.

> **Why does this work?** Vite's dev server accepts connections from any device on the LAN. Your friend's browser downloads the React app from your laptop, but it runs entirely in their browser and talks to their own local `node.exe` on `127.0.0.1:8080`.

---

### STEP 6 — Verify the Network is Live

Once everyone is connected, the **ONLINE SYSTEMS REGISTRY** table should show all active nodes. To verify from the command line:
```
http://192.168.1.15:8080/api/peers
```
This should return a JSON array with all registered peers.

---

### STEP 7 — Shut Down (Host 🖥️)

```bash
# ✅ Run as-is in the project root
sh ./scripts/stop_cluster.sh
```

---

---

## Quick Reference: What Each Flag Means

| Flag | Example Value | What it Does |
|---|---|---|
| `-data` | `./storage/node1` | Where this node stores its chunks on disk |
| `-grpc` | `0.0.0.0:50051` | Which local port to listen for gRPC calls on (`0.0.0.0` means "all network cards") |
| `-http` | `0.0.0.0:8080` | Which local port to serve the HTTP API on |
| `-advertise-grpc` | `100.64.1.5:50051` | The **public** gRPC address other nodes should use to reach you |
| `-advertise-http` | `https://abc.ngrok-free.app` | The **public** HTTP address the browser frontend uses to reach you |
| `-bootstrap` | `http://100.64.1.1:9099` | The URL of the Bootstrap server to register with |
| `-replication` | `2` | How many extra copies of each chunk to push to other nodes |

---

## Common Errors and Fixes

| Error Message | Cause | Fix |
|---|---|---|
| `bind: Only one usage of each socket address` | Another node.exe is already running on that port | Run `sh ./scripts/stop_cluster.sh` or kill via `Stop-Process -Id (Get-NetTCPConnection -LocalPort 50051).OwningProcess -Force` |
| `flag provided but not defined: -advertise-grpc` | Running an old version of `node.exe` | Recompile: `go build -o bin/node.exe ./cmd/node` and send the new file |
| `CORS policy: No 'Access-Control-Allow-Origin'` | Browser getting Ngrok interstitial HTML instead of JSON | Open the Ngrok URL directly in a browser tab and click "Visit Site" |
| `connection refused` when friend tries to connect | Windows Firewall is blocking `node.exe` | Check both Private+Public boxes in the Windows Firewall popup |
| Bootstrap returns empty peer list | Bootstrap started too recently, nodes haven't registered yet | Wait 5 seconds after starting the cluster, then refresh |
