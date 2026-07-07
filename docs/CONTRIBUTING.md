# Contributing to Chunkster

Thank you for your interest in contributing to Chunkster! This guide covers everything you need to get set up and start contributing.

---

## Prerequisites

| Tool | Version | Installation |
|---|---|---|
| **Go** | 1.21+ | [go.dev/dl](https://go.dev/dl/) |
| **Git** | any | [git-scm.com](https://git-scm.com/) |
| **Buf** (optional) | latest | [buf.build/docs/installation](https://buf.build/docs/installation) — only needed if you modify `proto/node.proto` |
| **Node.js** (optional) | 18+ | [nodejs.org](https://nodejs.org) — only needed if working on the [frontend](https://github.com/Vara693/p2p-distributed-system-frontend.git) |

---

## Getting Started

### 1. Clone the Repository

```bash
git clone https://github.com/Vara693/p2p-distributed-system-main.git
cd p2p-distributed-system-main
```

### 2. Build the Binaries

```bash
go build -o bin/bootstrap.exe ./cmd/bootstrap
go build -o bin/node.exe ./cmd/node
```

### 3. Run a Local Test Cluster

```bash
# Start a 3-node local cluster
sh ./scripts/start_cluster.sh

# Stop the cluster when done
sh ./scripts/stop_cluster.sh
```

### 4. Verify Everything Works

```bash
# Check the bootstrap server is responding
curl http://127.0.0.1:9099/v1/peers

# Check the node API is responding
curl http://127.0.0.1:8080/api/health
```

---

## Project Layout

```
cmd/           → Entry-point binaries (bootstrap server, storage node)
internal/      → All core logic packages (not importable externally)
proto/         → gRPC Protocol Buffer definitions
scripts/       → Cluster management shell scripts
docs/          → Documentation
```

For detailed explanations of every package and file, see [ARCHITECTURE.md](ARCHITECTURE.md).

---

## Code Style

- **Go conventions**: Follow standard `gofmt` formatting. Run `go fmt ./...` before committing.
- **Error handling**: Always handle errors explicitly. Do not ignore errors with `_`.
- **Comments**: Use `//` comments for non-obvious logic. All exported functions should have a doc comment.
- **Naming**: Use descriptive names. Avoid single-letter variables except in tight loops.
- **Concurrency**: Always protect shared state with `sync.Mutex` or `sync.RWMutex`. Document which lock protects which fields.

---

## Making Changes

### Modifying Go Code

1. Make your changes in the relevant `internal/` package.
2. Run `go build ./...` to verify compilation.
3. Test with a local cluster using `sh ./scripts/start_cluster.sh`.

### Modifying the gRPC Protocol

If you need to add or change RPC methods:

1. Edit `proto/node.proto`.
2. Regenerate the Go code:
   ```bash
   buf generate
   ```
3. Update the gRPC service implementation in `internal/coordinator/grpc_service.go`.
4. Rebuild the binaries.

### Modifying the Frontend

The frontend lives in a [separate repository](https://github.com/Vara693/p2p-distributed-system-frontend.git). See that repo's README for frontend contribution guidelines.

---

## Pull Request Guidelines

1. **One PR per feature/fix** — Keep PRs focused and reviewable.
2. **Describe your changes** — Explain what changed and why in the PR description.
3. **Test locally** — Verify your changes work with at least a 3-node local cluster.
4. **No broken builds** — Ensure `go build ./...` passes before submitting.

---

## Reporting Issues

When reporting a bug, please include:
- Go version (`go version`)
- Operating system
- Steps to reproduce
- Expected vs actual behavior
- Any relevant terminal output or error messages

---

## License

By contributing to Chunkster, you agree that your contributions will be licensed under the [MIT License](../LICENSE).
