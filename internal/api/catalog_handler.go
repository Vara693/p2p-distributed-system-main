package api

import (
	"context"
	"net/http"
)

func handleCatalog(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		
		rdb := h.Registry().RDB()
		catalogCid := ""
		if rdb != nil {
			if c, err := rdb.Get(context.Background(), "chunkster:catalog_cid").Result(); err == nil && c != "" {
				catalogCid = c
			}
		} else {
			catalogCid = h.LocalCatalogCid()
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"catalog_cid":"` + catalogCid + `"}` + "\n"))
	}
}
