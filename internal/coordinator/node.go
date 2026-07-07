package coordinator

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	nodev1 "edi_sem2/internal/gen"
	"edi_sem2/internal/dht"
	"edi_sem2/internal/merkle"
	"edi_sem2/internal/network"
	"edi_sem2/internal/peer"
	"edi_sem2/internal/replication"
	"edi_sem2/internal/scheduler"
	"edi_sem2/internal/storage"
	"edi_sem2/internal/utils"
)

// PeerHost is a single storage node: local blockstore, overlay registry, DHT tables, and gRPC.
type PeerHost struct {
	Log *slog.Logger

	SelfID    string
	GrpcAddr  string // host:port as dialed
	HTTPAddr  string // host:port for dashboard API
	DataDir   string
	Bootstrap string // optional e.g. http://127.0.0.1:9099

	ReplFactor int

	mu          sync.RWMutex
	store       *storage.Store
	chunker     storage.Chunker
	reg         *peer.Registry
	rtm         *peer.Manager
	rt          *dht.RoutingTable
	providers   *dht.ProviderRecords
	pool        *network.Pool
	repl        *replication.Manager
	localCatalogCid string
}

func NewPeerHost(cfg Config) (*PeerHost, error) {
	if cfg.DataDir == "" {
		return nil, errors.New("data dir required")
	}
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return nil, err
	}
	idPath := filepath.Join(cfg.DataDir, "peer.id")
	pid, err := loadOrCreatePeerID(idPath)
	if err != nil {
		return nil, err
	}
	bs, err := storage.NewStore(cfg.DataDir)
	if err != nil {
		return nil, err
	}
	log := cfg.Logger
	if log == nil {
		log = utils.NewLogger()
	}
	reg := peer.NewRegistry(pid)
	reg.Upsert(&peer.Record{
		ID:       pid,
		GrpcAddr: cfg.GrpcListenAddr,
		HttpAddr: cfg.HTTPListenAddr,
		State:    peer.StateActive,
		LastSeen: time.Now(),
	})
	rf := cfg.ReplicationFactor
	if rf < 1 {
		rf = 3
	}
	h := &PeerHost{
		Log:        log,
		SelfID:     pid,
		GrpcAddr:   cfg.GrpcListenAddr,
		HTTPAddr:   cfg.HTTPListenAddr,
		DataDir:    cfg.DataDir,
		Bootstrap:  cfg.BootstrapURL,
		ReplFactor: rf,
		store:      bs,
		chunker:    storage.Chunker{ChunkSize: storage.DefaultChunkSize},
		reg:        reg,
		rtm:        peer.NewManager(reg),
		rt:         dht.NewRoutingTable(reg),
		providers:  dht.NewProviderRecords(),
		pool:       network.NewPool(),
		repl:       &replication.Manager{Pool: network.NewPool()},
	}
	// replication uses same pool as host for connection reuse
	h.repl.Pool = h.pool
	return h, nil
}

func loadOrCreatePeerID(path string) (string, error) {
	if b, err := os.ReadFile(path); err == nil {
		s := string(bytesTrim(b))
		if len(s) == 64 {
			if _, err := dht.KeyFromHex(s); err == nil {
				return s, nil
			}
		}
	}
	var buf [32]byte
	if _, err := io.ReadFull(rand.Reader, buf[:]); err != nil {
		return "", err
	}
	s := hex.EncodeToString(buf[:])
	if err := os.WriteFile(path, []byte(s+"\n"), 0o600); err != nil {
		return "", err
	}
	return s, nil
}

func bytesTrim(b []byte) string {
	for len(b) > 0 && (b[0] == ' ' || b[0] == '\n' || b[0] == '\r') {
		b = b[1:]
	}
	for len(b) > 0 && (b[len(b)-1] == ' ' || b[len(b)-1] == '\n' || b[len(b)-1] == '\r') {
		b = b[:len(b)-1]
	}
	return string(b)
}

func (h *PeerHost) SelfEndpoint() *nodev1.PeerEndpoint {
	return &nodev1.PeerEndpoint{
		PeerId:   h.SelfID,
		GrpcAddr: h.GrpcAddr,
		HttpAddr: h.HTTPAddr,
	}
}

func (h *PeerHost) Registry() *peer.Registry { return h.reg }

func (h *PeerHost) Store() *storage.Store { return h.store }

func (h *PeerHost) Providers() *dht.ProviderRecords { return h.providers }

func (h *PeerHost) Pool() *network.Pool { return h.pool }

func (h *PeerHost) PeerID() string { return h.SelfID }

func (h *PeerHost) DialGRPC() string { return h.GrpcAddr }

func (h *PeerHost) ListenHTTP() string { return h.HTTPAddr }

// AddFile chunks locally, stores DAG root, registers providers, replicates blocks to closest peers.
func (h *PeerHost) AddFile(ctx context.Context, localPath string) (storage.CID, error) {
	chunks, err := h.chunker.PutFile(h.store, localPath)
	if err != nil {
		return "", err
	}
	chunkStrs := make([]string, len(chunks))
	for i, c := range chunks {
		chunkStrs[i] = string(c)
	}
	fn := merkle.NewFileNode(chunkStrs)
	nodeBytes, err := merkle.EncodeCanonicalJSON(fn)
	if err != nil {
		return "", err
	}
	root := storage.NewCID(nodeBytes)
	if _, err := h.store.Put(nodeBytes); err != nil {
		return "", err
	}
	selfEp := h.SelfEndpoint()
	h.providers.Add(string(root), selfEp)
	for _, c := range chunks {
		h.providers.Add(string(c), selfEp)
	}
	// Per-file human-readable mirror (optional)
	_ = h.writeFileBundle(root, chunks, nodeBytes)

	allCIDs := append([]storage.CID{root}, chunks...)
	for _, c := range allCIDs {
		data, err := h.store.Get(c)
		if err != nil {
			return "", err
		}
		ids, err := replication.SelectReplicaPeerIDs(h.rt, string(c), h.ReplFactor, h.SelfID)
		if err != nil {
			h.Log.Warn("replica select", "cid", c, "err", err)
			continue
		}
		var addrs []string
		for _, id := range ids {
			if rec, ok := h.reg.Get(id); ok && rec.GrpcAddr != "" {
				addrs = append(addrs, rec.GrpcAddr)
			}
		}
		if len(addrs) == 0 {
			continue
		}
		if err := h.repl.ReplicateToAddrs(ctx, addrs, c, data); err != nil {
			h.Log.Warn("replication incomplete", "cid", c, "err", err)
		}
	}
	return root, nil
}

func (h *PeerHost) writeFileBundle(root storage.CID, chunks []storage.CID, nodeBytes []byte) error {
	dir := filepath.Join(h.store.RootDir(), "files", string(root))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, string(root)+".block"), nodeBytes, 0o644); err != nil {
		return err
	}
	for _, ch := range chunks {
		b, err := h.store.Get(ch)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, string(ch)+".block"), b, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// GetFile resolves root CID using local store first, then DHT + remote GetBlock.
func (h *PeerHost) GetFile(ctx context.Context, root storage.CID, w io.Writer) error {
	rootBytes, err := h.store.Get(root)
	if err != nil {
		provs := dht.LookupProviders(ctx, h.pool, h.providers, h.GrpcAddr, h.reg.Snapshot(), string(root))
		b, err2 := replication.FetchBlock(ctx, h.pool, root, provs)
		if err2 != nil {
			return fmt.Errorf("root block: %v: %w", err, err2)
		}
		if err := storage.VerifyCID(root, b); err != nil {
			return err
		}
		rootBytes = b
		if _, err := h.store.Put(rootBytes); err != nil {
			h.Log.Debug("cache root", "err", err)
		}
	}
	var fn merkle.FileNode
	if err := json.Unmarshal(rootBytes, &fn); err != nil {
		return fmt.Errorf("invalid file node: %w", err)
	}
	for _, ch := range fn.Chunks {
		cid := storage.CID(ch)
		data, err := h.store.Get(cid)
		if err != nil {
			provs := dht.LookupProviders(ctx, h.pool, h.providers, h.GrpcAddr, h.reg.Snapshot(), ch)
			if len(provs) > 1 {
				h.Log.Info("resilient chunk retrieval trace initiated", "cid", ch, "available_providers", len(provs))
			}
			data, err = replication.FetchBlock(ctx, h.pool, cid, provs)
			if err != nil {
				h.Log.Error("chunk reconstruction dropoff: exhausted all DHT providers", "cid", ch, "err", err)
				return fmt.Errorf("chunk %s: %w", ch, err)
			}
			h.Log.Info("chunk payload successfully fetched & resumed", "cid", ch, "bytes", len(data))
			if err := storage.VerifyCID(cid, data); err != nil {
				return err
			}
			if _, err := h.store.Put(data); err != nil {
				h.Log.Debug("cache chunk", "cid", ch, "err", err)
			}
		}
		if _, err := w.Write(data); err != nil {
			return err
		}
	}
	return nil
}

func (h *PeerHost) InspectCID(c storage.CID) (string, error) {
	b, err := h.store.Get(c)
	if err != nil {
		provs := dht.LookupProviders(context.Background(), h.pool, h.providers, h.GrpcAddr, h.reg.Snapshot(), string(c))
		b, err = replication.FetchBlock(context.Background(), h.pool, c, provs)
		if err != nil {
			return "", err
		}
	}
	if t, err := merkle.DetectType(b); err == nil {
		switch t {
		case merkle.NodeTypeFile:
			var fn merkle.FileNode
			_ = json.Unmarshal(b, &fn)
			return fmt.Sprintf("CID: %s\nType: file\nChunks: %d\n", c, len(fn.Chunks)), nil
		case merkle.NodeTypeDirectory:
			var dn merkle.DirectoryNode
			_ = json.Unmarshal(b, &dn)
			return fmt.Sprintf("CID: %s\nType: directory\nEntries: %d\n", c, len(dn.Entries)), nil
		}
	}
	return fmt.Sprintf("CID: %s\nType: raw block\nSize: %d bytes\n", c, len(b)), nil
}

// RunLoops starts background tasks (bootstrap sync, heartbeats, cleanup). Non-blocking.
func (h *PeerHost) RunLoops(ctx context.Context) {
	go h.bootstrapLoop(ctx)
	go h.heartbeatLoop(ctx)
	go h.cleanupLoop(ctx)
}

func (h *PeerHost) bootstrapLoop(ctx context.Context) {
	if h.Bootstrap == "" {
		return
	}
	do := func() {
		if err := h.registerBootstrap(); err != nil {
			h.Log.Debug("bootstrap register", "err", err)
		}
		if err := h.mergeBootstrapPeers(ctx); err != nil {
			h.Log.Debug("bootstrap merge", "err", err)
		}
	}
	do()
	t := time.NewTicker(time.Duration(utils.BootstrapPollSec) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			do()
		}
	}
}

func (h *PeerHost) heartbeatLoop(ctx context.Context) {
	t := time.NewTicker(time.Duration(utils.DefaultHeartbeatSec) * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			h.reg.Touch(h.SelfID)
			h.rtm.Sweep(time.Now())
			for _, rec := range h.reg.Snapshot() {
				if rec.ID == h.SelfID || rec.State != peer.StateActive {
					continue
				}
				if rec.GrpcAddr == "" {
					continue
				}
				_ = network.WithTimeout(ctx, 3*time.Second, func(cctx context.Context) error {
					cli, err := network.NodeClient(cctx, h.pool, rec.GrpcAddr)
					if err != nil {
						return err
					}
					_, err = cli.Ping(cctx, &nodev1.PingRequest{PeerId: h.SelfID})
					if err != nil {
						return err
					}
					h.reg.Touch(rec.ID)
					return nil
				})
			}
		}
	}
}

func (h *PeerHost) cleanupLoop(ctx context.Context) {
	scheduler.RunCleanup(ctx, h.reg, h.providers)
}

// Close releases outbound gRPC connections.
func (h *PeerHost) Close() error {
	if h == nil || h.pool == nil {
		return nil
	}
	return h.pool.Close()
}

func (h *PeerHost) LocalCatalogCid() string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.localCatalogCid
}

// UpdateGlobalCatalog safely appends to a global distributed text file using Redis as a distributed lock.
func (h *PeerHost) UpdateGlobalCatalog(ctx context.Context, name, cid string, size int64, ext string) (storage.CID, error) {
	rdb := h.reg.RDB()
	var currentCatalogBytes []byte

	// 1. Acquire Distributed Lock (if Redis available)
	if rdb != nil {
		lockKey := "chunkster:catalog_lock"
		for {
			locked, err := rdb.SetNX(ctx, lockKey, h.SelfID, 10*time.Second).Result()
			if err == nil && locked {
				break
			}
			select {
			case <-ctx.Done():
				return "", ctx.Err()
			case <-time.After(500 * time.Millisecond):
				// retry lock
			}
		}
		defer rdb.Del(context.Background(), lockKey)
	} else {
		// Fallback local lock if no Redis
		h.mu.Lock()
		defer h.mu.Unlock()
	}

	// 2. Fetch the current catalog CID
	var currentCatalogCid string
	if rdb != nil {
		if c, err := rdb.Get(ctx, "chunkster:catalog_cid").Result(); err == nil && c != "" {
			currentCatalogCid = c
		}
	} else {
		currentCatalogCid = h.localCatalogCid
	}

	// 3. Download the current catalog if it exists
	if currentCatalogCid != "" {
		var buf bytes.Buffer
		err := h.GetFile(ctx, storage.CID(currentCatalogCid), &buf)
		if err == nil {
			currentCatalogBytes = buf.Bytes()
		} else {
			h.Log.Warn("failed to fetch previous catalog, creating new one", "cid", currentCatalogCid, "err", err)
		}
	}

	// 4. Append the new entry
	entry := fmt.Sprintf("Name: %s | CID: %s | Size: %d bytes | Type: %s\n", name, cid, size, ext)
	newCatalogBytes := append(currentCatalogBytes, []byte(entry)...)

	// 5. Write to temp file
	tmp, err := os.CreateTemp("", "catalog-*")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	
	if _, err := tmp.Write(newCatalogBytes); err != nil {
		tmp.Close()
		return "", err
	}
	tmp.Close()

	// 6. Upload new catalog to the overlay
	newCatalogCid, err := h.AddFile(ctx, tmp.Name())
	if err != nil {
		return "", err
	}

	// 7. Update the global pointer in Redis
	if rdb != nil {
		err = rdb.Set(ctx, "chunkster:catalog_cid", string(newCatalogCid), 0).Err()
		if err != nil {
			h.Log.Error("failed to update catalog_cid in redis", "err", err)
		}
	} else {
		h.localCatalogCid = string(newCatalogCid)
	}

	h.Log.Info("global catalog updated successfully", "new_catalog_cid", newCatalogCid)
	return newCatalogCid, nil
}
