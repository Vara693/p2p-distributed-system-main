package peer

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"edi_sem2/internal/utils"
)

// Record is a known peer in the overlay (registry + routing view).
type Record struct {
	ID       string    `json:"id"`
	GrpcAddr string    `json:"grpc_addr"`
	HttpAddr string    `json:"http_addr"`
	State    State     `json:"state"`
	LastSeen time.Time `json:"last_seen"`
	JoinedAt time.Time `json:"joined_at"`
}

type Registry struct {
	mu    sync.RWMutex
	self  string
	peers map[string]*Record
	max   int
	rdb   *redis.Client
}

func NewRegistry(selfID string) *Registry {
	reg := &Registry{
		self:  selfID,
		peers: make(map[string]*Record),
		max:   utils.MaxNetworkPeers,
	}

	if rdbUrl := os.Getenv("REDIS_URL"); rdbUrl != "" {
		if opt, err := redis.ParseURL(rdbUrl); err == nil {
			rdb := redis.NewClient(opt)
			reg.rdb = rdb
			go reg.startRedisSync()
		}
	}

	return reg
}

func (r *Registry) SelfID() string { return r.self }

// RDB returns the underlying Redis client, if configured.
func (r *Registry) RDB() *redis.Client { return r.rdb }

// Upsert inserts or updates a peer. Returns false if the network is at capacity and this is a new peer.
func (r *Registry) Upsert(rec *Record) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if rec == nil || rec.ID == "" {
		return false
	}
	// Prune any stale entries with the same address but a different ID (due to node restart)
	for id, existing := range r.peers {
		if id != rec.ID && (existing.GrpcAddr == rec.GrpcAddr || (existing.HttpAddr != "" && existing.HttpAddr == rec.HttpAddr)) {
			delete(r.peers, id)
			if r.rdb != nil {
				r.rdb.Del(context.Background(), "peers:"+id)
			}
			break
		}
	}
	if existing, ok := r.peers[rec.ID]; ok {
		existing.GrpcAddr = rec.GrpcAddr
		existing.HttpAddr = rec.HttpAddr
		if rec.State != "" {
			existing.State = rec.State
		}
		existing.LastSeen = rec.LastSeen
		r.publishRedis(existing)
		return true
	}
	if len(r.peers) >= r.max {
		// Attempt peer replacement: evict an existing Left peer to make room
		var evictID string
		for id, p := range r.peers {
			if p.State == StateLeft {
				evictID = id
				break
			}
		}
		if evictID == "" {
			return false
		}
		delete(r.peers, evictID)
		if r.rdb != nil {
			r.rdb.Del(context.Background(), "peers:"+evictID)
		}
	}
	rec.JoinedAt = time.Now()
	r.peers[rec.ID] = rec
	r.publishRedis(rec)
	return true
}

func (r *Registry) Get(id string) (*Record, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p, ok := r.peers[id]
	return p, ok
}

func (r *Registry) MarkState(id string, s State) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.peers[id]; ok {
		p.State = s
		r.publishRedis(p)
		if s == StateLeft && r.rdb != nil {
			r.rdb.Del(context.Background(), "peers:"+id)
		}
	}
}

func (r *Registry) Touch(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if p, ok := r.peers[id]; ok {
		p.LastSeen = time.Now()
		if p.State == StateInactive {
			p.State = StateActive
		}
		r.publishRedis(p)
	}
}

func (r *Registry) Remove(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.peers, id)
	if r.rdb != nil {
		r.rdb.Del(context.Background(), "peers:"+id)
	}
}

func (r *Registry) Snapshot() []*Record {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]*Record, 0, len(r.peers))
	for _, p := range r.peers {
		cp := *p
		out = append(out, &cp)
	}
	return out
}

func (r *Registry) ActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n := 0
	for _, p := range r.peers {
		if p.State == StateActive {
			n++
		}
	}
	return n
}

func (r *Registry) publishRedis(rec *Record) {
	if r.rdb == nil || rec == nil {
		return
	}
	data, err := json.Marshal(rec)
	if err == nil {
		// SETEX caches peer info with a 60-second automatic cloud expiration
		r.rdb.Set(context.Background(), "peers:"+rec.ID, data, 60*time.Second)
	}
}

func (r *Registry) startRedisSync() {
	t := time.NewTicker(5 * time.Second)
	ctx := context.Background()
	for range t.C {
		if r.rdb == nil {
			return
		}
		keys, err := r.rdb.Keys(ctx, "peers:*").Result()
		if err != nil {
			continue
		}
		for _, key := range keys {
			data, err := r.rdb.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}
			var rec Record
			if err := json.Unmarshal(data, &rec); err == nil {
				r.mu.Lock()
				if existing, exists := r.peers[rec.ID]; exists {
					// Merge states carefully favoring newer seen timestamps
					if rec.LastSeen.After(existing.LastSeen) {
						existing.GrpcAddr = rec.GrpcAddr
						existing.HttpAddr = rec.HttpAddr
						existing.State = rec.State
						existing.LastSeen = rec.LastSeen
					}
				} else if len(r.peers) < r.max {
					r.peers[rec.ID] = &rec
				}
				r.mu.Unlock()
			}
		}
	}
}
