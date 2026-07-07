package api

import (
	"net/http"
)

// RegisterRoutes wires dashboard HTTP endpoints on mux.
func RegisterRoutes(mux *http.ServeMux, h Host) {
	mux.HandleFunc("/api/health", handleHealth(h))
	mux.HandleFunc("/api/peers", handlePeers(h))
	mux.HandleFunc("/api/graph", handleGraph(h))
	mux.HandleFunc("/api/upload", handleUpload(h))
	mux.HandleFunc("/api/download", handleDownload(h))
	mux.HandleFunc("/api/search", handleSearch(h))
	mux.HandleFunc("/api/catalog", handleCatalog(h))
}
