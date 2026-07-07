# Chunkster — System Architecture

> Complete technical documentation covering every folder, file, algorithm, data structure, and protocol used in the Chunkster P2P distributed file storage system.

---

## Table of Contents

1. [What is Chunkster?](#what-is-chunkster)
2. [Project Folder Structure](#project-folder-structure)
3. [Detailed Folder & File Breakdown](#detailed-folder--file-breakdown)
4. [How the System Works — End to End](#how-the-system-works--end-to-end)
5. [Algorithms & Data Structures](#algorithms--data-structures)
6. [The gRPC Protocol Contract](#the-grpc-protocol-contract)
7. [Tools & Technologies](#tools--technologies)
8. [Scripts Reference](#scripts-reference)

---

## What is Chunkster?

**Chunkster** is a fully decentralized, peer-to-peer distributed file storage system — similar in concept to **IPFS (InterPlanetary File System)** or **BitTorrent**, but built from scratch in Go.

The core idea is simple: instead of uploading your files to a central server (like Google Drive or Dropbox), your files are **split into small chunks** and **distributed across multiple computers** on the network. Any computer on the network can retrieve those chunks and reassemble the original file — even if the original uploader is offline, as long as at least one other computer stored a copy.

### Key Properties

| Property | Description |
|---|---|
| **Decentralized** | No single server owns your data |
| **Content-Addressed** | Files are identified by their SHA-256 hash (CID), not by a name or URL |
| **Fault-Tolerant** | Each chunk is replicated to 3 different nodes, so the network survives node failures |
| **Self-Healing** | Dead nodes are automatically detected and replaced |
| **Bounded** | The network enforces an upper bound of 20 active peers |

---

## Project Folder Structure

```
chunkster/
├── cmd/                    ← Entry points (the programs you compile and run)
│   ├── bootstrap/          ← The directory/discovery server
│   └── node/               ← The main storage node program
├── internal/               ← All core logic packages
│   ├── api/                ← HTTP REST API exposed to the frontend
│   ├── coordinator/        ← The brain — orchestrates all node operations
│   ├── dht/                ← Kademlia Distributed Hash Table logic
│   ├── gen/                ← Auto-generated gRPC code (do not edit manually)
│   ├── heartbeat/          ← Peer health monitoring
│   ├── merkle/             ← Merkle DAG file tree structures
│   ├── network/            ← gRPC connection pool and transport
│   ├── peer/               ← Peer registry and state management
│   ├── replication/        ← Block replication across peers
│   ├── scheduler/          ← Background cleanup and rebalancing tasks
│   ├── storage/            ← Block store, chunker, CID logic
│   └── utils/              ← Shared constants and logger
├── proto/                  ← The gRPC service contract definition
│   └── node.proto          ← Defines all messages and RPC methods
├── scripts/                ← Shell scripts for managing the cluster
├── docs/                   ← Documentation
├── Dockerfile              ← Docker container definition (optional)
├── docker-compose.yml      ← Docker multi-container setup (optional)
├── buf.yaml                ← Buf tool config for compiling .proto files
└── go.mod                  ← Go module definition (lists all dependencies)
```

---

## Detailed Folder & File Breakdown

### `cmd/bootstrap/` — The Directory Server

**What it is:** The bootstrap server is the first thing you start. It acts like a **phone book** for the network. When a new node wants to join, it calls the bootstrap server and says "I exist at this address." The bootstrap server records this and shares the full peer list with anyone who asks.

**Why it exists:** In a truly decentralized network, how does a brand new node know where anyone else is? It doesn't. The bootstrap server solves the "cold start" problem by being a well-known, stable address that everyone connects to first.

**Key file: `main.go`**
- Listens on port **9099** by default (configurable via `BOOTSTRAP_ADDR` environment variable)
- Exposes three HTTP endpoints:
  - `POST /v1/register` — A node calls this when it starts up to register itself
  - `GET /v1/peers` — Returns the full list of all known peers
  - `DELETE /v1/peers/{id}` — Removes a peer when it gracefully shuts down
- Optionally connects to Redis (via `REDIS_URL` env var) to persist the peer list across restarts
- Enforces a maximum of 20 peers (`utils.MaxNetworkPeers`)
- Deduplicates peers by PeerID, gRPC address, and HTTP address to prevent ghost entries on restart

---

### `cmd/node/` — The Storage Node Program

**What it is:** This is the main program. Every computer on the network runs this. It is responsible for storing file chunks, serving them to other nodes, and participating in the DHT routing network.

**Files:**
- **`main.go`** — Starts the gRPC server, HTTP API server, and the background loop goroutines. Also reads the `-advertise-grpc` and `-advertise-http` flags to support Ngrok/Tailscale NAT traversal.
- **`config.go`** — Defines all command-line flags the node accepts:

| Flag | Default | Purpose |
|---|---|---|
| `-data` | `storage/node1` | Where to store data on disk |
| `-grpc` | `127.0.0.1:50051` | Local gRPC bind address |
| `-http` | `127.0.0.1:8080` | Local HTTP API bind address |
| `-advertise-grpc` | *(empty)* | Public gRPC address to tell other peers (for NAT traversal) |
| `-advertise-http` | *(empty)* | Public HTTP address to advertise (for Ngrok) |
| `-bootstrap` | `http://127.0.0.1:9099` | Bootstrap server URL to register with |
| `-replication` | `3` | How many copies of each chunk to store |

---

### `internal/coordinator/` — The Brain

**What it is:** The coordinator is the central orchestrator of every node. It ties together all the sub-packages (storage, DHT, peers, replication, heartbeat) into a single cohesive system.

**Files:**
- **`node.go`** — The most important file in the project. Defines the `PeerHost` struct which contains references to every subsystem. Key functions:
  - `NewPeerHost()` — Initializes a new node, creates/loads the peer ID from disk
  - `AddFile()` — The upload pipeline: chunks file → stores locally → announces to DHT → replicates to peers
  - `GetFile()` — The download pipeline: looks up providers in DHT → fetches chunks from peers → reassembles file
  - `RunLoops()` — Starts all background goroutines (heartbeat, bootstrap sync, catalog updates)
  - `UpdateGlobalCatalog()` — Syncs the file catalog across all peers in the network

- **`config.go`** — The `Config` struct that `NewPeerHost` needs to initialize

- **`bootstrap_sync.go`** — Background loop that periodically re-registers with the bootstrap server and fetches the latest peer list to keep the routing table fresh

- **`grpc_service.go`** — Implements the gRPC server methods (Ping, StoreBlock, GetBlock, FindProvider, JoinNetwork, LeaveNetwork, Heartbeat, AddProvider). This is the code that responds when another node calls your node over the network.

---

### `internal/dht/` — Kademlia Distributed Hash Table

**What it is:** The DHT is the routing brain of the network. Instead of asking a central server "who has file X?", a node uses DHT to intelligently route its query to the nodes most likely to have the answer.

**Files:**
- **`xor_distance.go`** — The mathematical heart of Kademlia. Implements XOR distance calculation between two 32-byte keys. Distance is interpreted as a `big.Int` for numeric comparison.
- **`routing_table.go`** — Maintains a sorted view of all known peers by their XOR distance to a target key. `ClosestPeerIDs()` returns the N nearest peers to a given CID.
- **`provider_records.go`** — An in-memory map of `CID → []PeerEndpoint`. Records which nodes have announced they hold a particular chunk.
- **`lookup.go`** — `LookupProviders()` first checks local provider records, then fans out `FindProvider` gRPC calls to neighboring nodes to find who holds a chunk.

---

### `internal/storage/` — Block Store & Chunker

**What it is:** Everything related to actually reading and writing raw bytes to your hard drive.

**Files:**
- **`cid.go`** — Defines the `CID` type (a 64-character hex string of a SHA-256 hash). `NewCID(data []byte)` computes the hash of a block and returns its unique identifier.
- **`chunker.go`** — `PutFile()` opens a file, reads it in `256 KiB` chunks using a buffered reader, stores each chunk in the block store, and returns an ordered list of chunk CIDs.
- **`blockstore.go`** — The `Store` struct manages reading/writing raw binary blocks to disk. Each block is saved as a file named after its CID inside the node's data directory. `Put()` writes a block; `Get()` reads it back.
- **`file_reconstruct.go`** — Given an ordered list of chunk CIDs, fetches each one from the block store and writes them sequentially to reconstruct the original file.

---

### `internal/peer/` — Peer Registry

**What it is:** Every node keeps its own local address book of all known peers on the network.

**Files:**
- **`peer_state.go`** — Defines the `State` enum: `Active`, `Inactive`, `Left`
- **`peer_registry.go`** — The `Registry` struct is a thread-safe map of `PeerID → PeerRecord`. Methods: `Upsert()`, `Get()`, `All()`, `Remove()`. This is what the frontend reads to display the "Online Systems Registry" table. Includes address-based deduplication to prune stale entries from node restarts.
- **`peer_manager.go`** — The `Manager` wraps the registry and adds higher-level operations like marking a peer as inactive when its heartbeat times out.

---

### `internal/replication/` — Block Replication

**What it is:** After a chunk is stored locally, this package handles pushing copies of it to nearby nodes for fault tolerance.

**Files:**
- **`replication_manager.go`** — `ReplicateBlock()` takes a CID + raw bytes + list of target peers, and concurrently sends `StoreBlock` gRPC calls to each target peer.
- **`replica_selection.go`** — Picks which peers should receive a replica. Uses the DHT routing table to find the K closest peers to the chunk's CID that are not the local node.
- **`recovery.go`** — Handles the case where a target node is unreachable during replication. Tries the next-closest peer as a fallback.

---

### `internal/network/` — gRPC Connection Management

**What it is:** Managing network connections efficiently. Opening a new TCP connection for every gRPC call would be slow. This package maintains a pool of reusable connections.

**Files:**
- **`connection_pool.go`** — A thread-safe `Pool` that caches open `grpc.ClientConn` connections by address. `Get()` returns an existing connection or dials a new one.
- **`grpc_client.go`** — Helper function that wraps a connection in a typed `NodeClient` for making RPC calls.
- **`grpc_server.go`** — Creates a standard `grpc.Server` instance.
- **`transport.go`** — Configures gRPC dial options (e.g., insecure transport for local connections).

---

### `internal/heartbeat/` — Peer Health Monitoring

**What it is:** Every node continuously pings its known peers to check if they are still alive.

**Files:**
- **`health_check.go`** — Sends `Ping` gRPC calls to all known peers on a regular timer. If a peer responds, its `LastSeen` timestamp in the registry is updated.
- **`peer_timeout.go`** — Checks the registry for peers whose `LastSeen` is older than 60 seconds and marks them as `Inactive`.

---

### `internal/scheduler/` — Background Maintenance

**What it is:** Background jobs that run automatically to keep the network clean.

**Files:**
- **`cleanup.go`** — Periodically scans the peer registry and removes peers that have been `Inactive` for too long. This frees up slots for new peers to join.
- **`rebalance.go`** — Stub for future rebalancing logic (re-distributing chunks when the network topology changes significantly).

---

### `internal/merkle/` — Merkle DAG File Tree

**What it is:** A data structure borrowed from Git and IPFS that represents a file as a tree of content-addressed nodes.

**Files:**
- **`file_node.go`** — Defines the `FileNode` struct: a JSON object containing the filename, total size, and an ordered list of chunk CIDs. This is the "root" object stored for each uploaded file.
- **`directory_node.go`** — Defines the `DirectoryNode` struct for future directory support.
- **`dag_traversal.go`** — Functions for walking the DAG tree to collect all chunk CIDs needed to reconstruct a file.

---

### `internal/api/` — HTTP REST API

**What it is:** The HTTP server that the React frontend communicates with. Every node runs its own HTTP API so users can connect to it via a browser.

**Files:**
- **`server.go`** — Creates and configures the `http.ServeMux` router, registering all route handlers.
- **`cors.go`** — CORS middleware that allows the React app (running on a different origin like Vercel) to make cross-origin requests. Also allows the `ngrok-skip-browser-warning` header.
- **`host.go`** — Defines the `Host` interface that API handlers use to access node internals.
- **`health_peers.go`** — Handles `GET /api/health` (returns node status, peer counts) and `GET /api/peers` (returns the full peer list for the registry table).
- **`upload_handler.go`** — Handles `POST /api/upload`. Receives a file via multipart form, saves it to a temp location, calls `AddFile()` on the coordinator, returns the root CID.
- **`download_handler.go`** — Handles `GET /api/download?cid=...`. Calls `GetFile()` on the coordinator, streams the reconstructed file bytes back to the browser.
- **`search_handler.go`** — Handles `GET /api/search?cid=...`. Looks up providers for a CID and returns which nodes hold it.
- **`catalog_handler.go`** — Handles `GET /api/catalog`. Returns the global shared file catalog (list of all files uploaded to the network).

---

### `internal/gen/` — Auto-Generated gRPC Code

**What it is:** These files are machine-generated from `proto/node.proto` by the `buf` tool. They contain the Go struct definitions and client/server interfaces for all gRPC messages. **You should never edit these files manually.**

- **`node.pb.go`** — Go structs for all protobuf messages (PingRequest, StoreBlockRequest, etc.)
- **`node_grpc.pb.go`** — Go interfaces and implementations for the gRPC `Node` service client and server.

---

### `internal/utils/` — Shared Constants

- **`constants.go`** — Defines `MaxNetworkPeers = 20` and the default chunk size.
- **`logger.go`** — Configures the structured logger used across all packages.

---

### `proto/node.proto` — The Network Contract

**What it is:** The single source of truth for how nodes communicate with each other. Written in Protocol Buffers (protobuf) language.

**RPC Methods defined:**

| Method | Purpose |
|---|---|
| `Ping` | Check if a peer is alive |
| `StoreBlock` | Push a raw chunk to another node |
| `GetBlock` | Fetch a raw chunk from another node |
| `FindProvider` | Ask "who has chunk X?" |
| `JoinNetwork` | Announce yourself to an existing peer |
| `LeaveNetwork` | Gracefully remove yourself from the network |
| `Heartbeat` | Keep-alive signal sent periodically |
| `AddProvider` | Tell a peer "I now have chunk X" |

---

## How the System Works — End to End

### Phase 1: Node Startup

1. `node.exe` starts and reads its command-line flags.
2. It checks the `-data` directory for a file called `peer.id`. If found, it loads the existing 64-character hex peer ID. If not found, it generates a new random 32-byte ID and saves it. **This ID is permanent and survives restarts.**
3. It initializes its block store (the on-disk chunk database), peer registry, routing table, and DHT provider records.
4. It starts the gRPC server (listens for calls from other nodes) and HTTP API server (listens for calls from the browser frontend).
5. It registers itself with the bootstrap server: `POST /v1/register {peer_id, grpc_addr, http_addr}`.
6. It fetches the current peer list from bootstrap and sends `JoinNetwork` gRPC calls to existing peers to announce its presence and receive their known peer lists in return.
7. Background goroutines begin: heartbeat loop, bootstrap re-sync loop, catalog sync loop.

### Phase 2: File Upload

1. User drags a file onto the React dashboard.
2. The frontend sends a `POST /api/upload` HTTP request to the node.
3. The node calls `coordinator.AddFile()`:
   - The chunker reads the file in 256 KiB blocks.
   - Each block is hashed with SHA-256 to produce its CID.
   - Each block is written to the block store on the local hard drive.
   - A `FileNode` JSON is constructed (filename + ordered list of chunk CIDs) and also stored.
   - For each chunk, the DHT routing table finds the 3 nearest peers (by XOR distance).
   - The replication manager concurrently pushes each chunk to those 3 peers via `StoreBlock` gRPC.
   - Provider records are announced: `AddProvider` gRPC is called so peers know who holds which chunk.
4. The root CID of the `FileNode` is returned to the frontend and displayed.

### Phase 3: File Download

1. User enters a CID in the search box and clicks Download.
2. The frontend calls `GET /api/download?cid=...`.
3. The node calls `coordinator.GetFile()`:
   - It fetches the `FileNode` by its root CID (either locally or from a peer via `GetBlock`).
   - For each chunk CID in the FileNode, it calls `dht.LookupProviders()` to find which nodes hold that chunk.
   - It sends `GetBlock` gRPC calls to those peers concurrently.
   - If a peer is unreachable, it automatically falls back to the next-closest provider.
   - Chunks are reassembled in order and streamed back to the browser as a file download.

### Phase 4: Peer Failure & Recovery

1. Every 30 seconds, the heartbeat loop pings all known peers.
2. If a peer does not respond, its `LastSeen` timestamp stops updating.
3. After 60 seconds without a heartbeat response, the scheduler marks the peer as `Inactive`.
4. The peer's record remains in the registry but is flagged. The frontend shows it as "Inactive" (yellow dot).
5. If a 21st peer tries to join, the coordinator checks for any `Left` or long-`Inactive` peers, removes them, and accepts the new peer.
6. When the offline peer comes back online, it re-registers with the bootstrap server. The heartbeat loop detects the peer as active again and updates the registry.

---

## Algorithms & Data Structures

### 1. SHA-256 Content Addressing
- **What:** Every chunk of data is identified by the SHA-256 hash of its raw bytes.
- **Why:** This makes storage content-addressed rather than location-addressed. Two identical chunks produce the same CID, enabling automatic deduplication. It also allows any node to verify chunk integrity.
- **Where:** `internal/storage/cid.go`

### 2. Fixed-Size Chunking (256 KiB)
- **What:** Files are split into fixed 256 KiB (262,144 byte) chunks.
- **Data structure:** An ordered `[]CID` slice (the chunk list inside `FileNode`)
- **Why:** A uniform chunk size ensures predictable storage behavior and simplifies replication logic. The last chunk may be smaller.
- **Where:** `internal/storage/chunker.go`

### 3. Merkle DAG (Directed Acyclic Graph)
- **What:** A tree structure where every parent node contains only the hashes of its children, not the data itself.
- **Data structure:** `FileNode` struct containing a `[]string` of chunk CIDs.
- **Why:** The root CID of a FileNode is a compact, unforgeable fingerprint of the entire file. Any tampering with any chunk changes its CID, which changes the root CID, making corruption detectable.
- **Where:** `internal/merkle/`

### 4. Kademlia XOR Distance Metric
- **What:** The topological "distance" between two nodes or between a node and a key is calculated as the bitwise XOR of their 32-byte IDs, interpreted as a 256-bit unsigned integer.
- **Data structure:** `*big.Int` (arbitrary-precision integer)
- **Why:** XOR satisfies the mathematical properties of a metric (identity, symmetry, triangle inequality). It creates a perfectly balanced address space where each node is naturally "closest" to a unique region of the keyspace.
- **Formula:** `Distance(A, B) = A ⊕ B`
- **Where:** `internal/dht/xor_distance.go`

### 5. Routing Table (K-Nearest Neighbor Sort)
- **What:** To find the K nodes closest to a target CID, the routing table takes a snapshot of all active peers, computes their XOR distance to the target, sorts them, and returns the top K.
- **Data structure:** A slice of `peer.Record` sorted by `*big.Int` XOR distance.
- **Time complexity:** O(N log N) for the sort, where N is the number of active peers.
- **Where:** `internal/dht/routing_table.go`

### 6. Provider Records (DHT Announcement)
- **What:** An in-memory hash map from `CID string → []PeerEndpoint`.
- **Data structure:** `map[string][]PeerEndpoint` protected by a `sync.RWMutex`
- **Why:** Allows O(1) lookup of who holds a particular chunk without querying the entire network.
- **Where:** `internal/dht/provider_records.go`

### 7. gRPC Connection Pool
- **What:** A cache of open TCP connections to peer nodes.
- **Data structure:** `map[string]*grpc.ClientConn` protected by a `sync.Mutex`
- **Why:** Opening a new TCP + TLS handshake for every RPC call would add 100–500ms latency. Reusing cached connections makes block transfers ~10x faster.
- **Where:** `internal/network/connection_pool.go`

### 8. Peer Registry (Thread-Safe Map)
- **What:** A thread-safe map of all known peers and their current state.
- **Data structure:** `map[string]*Record` (keyed by PeerID) with `sync.RWMutex`
- **Why:** Multiple goroutines (heartbeat, bootstrap sync, gRPC handlers) read and write the peer list concurrently. The mutex prevents data races.
- **Where:** `internal/peer/peer_registry.go`

---

## The gRPC Protocol Contract

The file `proto/node.proto` defines exactly how every node communicates. Key message flows:

```
Node A starts up:
  A → Bootstrap: POST /v1/register {peer_id: "abc...", grpc_addr: "...", http_addr: "..."}
  A → Bootstrap: GET /v1/peers → [{Node B}, {Node C}, ...]
  A → Node B: gRPC JoinNetwork({self: A_endpoint}) → {known_peers: [C, D, ...]}

User uploads file on Node A:
  A → A (local): StoreBlock(chunk1_cid, chunk1_bytes)
  A → Node B: gRPC StoreBlock(chunk1_cid, chunk1_bytes) → {ok: true}
  A → Node C: gRPC StoreBlock(chunk1_cid, chunk1_bytes) → {ok: true}
  A → Node B: gRPC AddProvider(chunk1_cid, A_endpoint)

User downloads file on Node D:
  D → D (local DHT): LookupProviders(chunk1_cid) → not found locally
  D → Node E: gRPC FindProvider(chunk1_cid) → {providers: [A_endpoint, B_endpoint]}
  D → Node A: gRPC GetBlock(chunk1_cid) → {data: <raw bytes>}
  D assembles chunks → streams file to browser
```

---

## Tools & Technologies

| Tool/Technology | Version | Purpose |
|---|---|---|
| **Go** | 1.24 | Backend language for all node and bootstrap logic |
| **gRPC** | v1.71 | High-performance RPC framework for node-to-node communication |
| **Protocol Buffers** | proto3 | Interface definition language for gRPC messages |
| **Buf** | latest | Tool for compiling `.proto` files into Go code |
| **Redis** | optional | Persistent storage for bootstrap peer list (cloud deployments) |
| **Ngrok** | latest | HTTP tunnel to expose local node HTTP API to deployed frontend |
| **Tailscale** | latest | Secure P2P mesh VPN for cross-internet node-to-node gRPC |

---

## Scripts Reference

| Script | Purpose |
|---|---|
| `start_cluster.sh` | Starts a local 3-node cluster for development |
| `start_20_nodes.sh` | Starts a full 20-node simulation for stress testing |
| `start_deployed_cluster.sh` | Starts 1 bootstrap server for real-world deployment |
| `stop_cluster.sh` | Forcefully kills all running node and bootstrap processes |
| `stop_cluster.bat` | Windows batch equivalent of `stop_cluster.sh` |
| `kill_node.sh` | Kills a single specific node by port number |
| `simulate_failure.sh` | Kills a random node to test the network's self-healing behavior |
