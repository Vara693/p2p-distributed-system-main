package api

import (
	"encoding/json"
	"net/http"

	"edi_sem2/internal/peer"
)

func handleHealth(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"peer_id":      h.PeerID(),
			"grpc":       h.DialGRPC(),
			"http":       h.ListenHTTP(),
			"active_peers": h.Registry().ActiveCount(),
			"total_peers":  len(h.Registry().Snapshot()),
		})
	}
}

func handlePeers(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var out []map[string]any
		for _, p := range h.Registry().Snapshot() {
			out = append(out, map[string]any{
				"id":         p.ID,
				"grpc_addr":  p.GrpcAddr,
				"http_addr":  p.HttpAddr,
				"state":      string(p.State),
				"last_seen":  p.LastSeen.UTC().Format("2006-01-02T15:04:05Z"),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	}
}

func handleGraph(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		snap := h.Registry().Snapshot()
		var nodes []map[string]any
		var edges [][2]string
		for _, p := range snap {
			nodes = append(nodes, map[string]any{
				"id":    p.ID,
				"label": p.ID[:8],
				"state": string(p.State),
			})
			if p.State == peer.StateActive && p.ID != h.PeerID() {
				edges = append(edges, [2]string{h.PeerID(), p.ID})
			}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"nodes": nodes, "edges": edges})
	}
}
