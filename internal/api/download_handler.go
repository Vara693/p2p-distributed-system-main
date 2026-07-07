package api

import (
	"net/http"

	"edi_sem2/internal/storage"
)

func handleDownload(h Host) http.HandlerFunc {
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
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=\"download.bin\"")
		if err := h.GetFile(r.Context(), storage.CID(cid), w); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
	}
}
