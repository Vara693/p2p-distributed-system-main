package api

import (
	"encoding/json"
	"net/http"

	"edi_sem2/internal/dht"
	"edi_sem2/internal/storage"
)

func handleSearch(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		cid := r.URL.Query().Get("cid")
		if cid == "" {
			http.Error(w, "missing cid", http.StatusBadRequest)
			return
		}
		provs := dht.LookupProviders(r.Context(), h.Pool(), h.Providers(), h.DialGRPC(), h.Registry().Snapshot(), cid)
		info, err := h.InspectCID(storage.CID(cid))
		if err != nil {
			info = err.Error()
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"cid":       cid,
			"providers": provs,
			"inspect":   info,
		})
	}
}
