package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"edi_sem2/internal/utils"
)

type regPeer struct {
	PeerID   string `json:"peer_id"`
	GrpcAddr string `json:"grpc_addr"`
	HttpAddr string `json:"http_addr"`
}

func main() {
	addr := ":9099"
	if v := os.Getenv("BOOTSTRAP_ADDR"); v != "" {
		addr = v
	}
	var mu sync.Mutex
	peers := make([]regPeer, 0, utils.MaxNetworkPeers)

	var rdb *redis.Client
	if rdbUrl := os.Getenv("REDIS_URL"); rdbUrl != "" {
		if opt, err := redis.ParseURL(rdbUrl); err == nil {
			rdb = redis.NewClient(opt)
			log.Println("bootstrap server connected to persistent cloud Redis cluster")
		}
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/v1/register", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		var p regPeer
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if p.PeerID == "" || p.GrpcAddr == "" {
			http.Error(w, "peer_id and grpc_addr required", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for i := range peers {
			if peers[i].PeerID == p.PeerID || peers[i].GrpcAddr == p.GrpcAddr || peers[i].HttpAddr == p.HttpAddr {
				if rdb != nil && peers[i].PeerID != p.PeerID {
					rdb.Del(context.Background(), "bootstrap:peer:"+peers[i].PeerID)
				}
				peers[i] = p
				if rdb != nil {
					data, _ := json.Marshal(p)
					rdb.Set(context.Background(), "bootstrap:peer:"+p.PeerID, data, 5*time.Minute)
				}
				w.WriteHeader(http.StatusOK)
				return
			}
		}
		if len(peers) >= utils.MaxNetworkPeers {
			http.Error(w, "network full", http.StatusForbidden)
			return
		}
		peers = append(peers, p)
		if rdb != nil {
			data, _ := json.Marshal(p)
			rdb.Set(context.Background(), "bootstrap:peer:"+p.PeerID, data, 5*time.Minute)
		}
		w.WriteHeader(http.StatusOK)
	})
	mux.HandleFunc("/v1/peers", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(peers)
	})
	mux.HandleFunc("/v1/peers/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		id := r.URL.Path[len("/v1/peers/"):]
		if id == "" {
			http.Error(w, "missing id", http.StatusBadRequest)
			return
		}
		mu.Lock()
		defer mu.Unlock()
		for i, p := range peers {
			if p.PeerID == id {
				peers = append(peers[:i], peers[i+1:]...)
				if rdb != nil {
					rdb.Del(context.Background(), "bootstrap:peer:"+id)
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}
		http.NotFound(w, r)
	})

	log.Printf("bootstrap listening on %s (max %d peers)", addr, utils.MaxNetworkPeers)
	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
