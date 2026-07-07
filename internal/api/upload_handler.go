package api

import (
	"io"
	"net/http"
	"os"
)

func handleUpload(h Host) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		if err := r.ParseMultipartForm(256 << 20); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "missing file field", http.StatusBadRequest)
			return
		}
		defer file.Close()
		tmp, err := os.CreateTemp("", "upload-*")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer os.Remove(tmp.Name())
		defer tmp.Close()
		if _, err := io.Copy(tmp, file); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := tmp.Close(); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		root, err := h.AddFile(r.Context(), tmp.Name())
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		
		ext := ""
		if len(header.Filename) > 0 {
			// Extract extension manually or import path, we can just use strings.LastIndex
			ext = "unknown"
			for i := len(header.Filename) - 1; i >= 0; i-- {
				if header.Filename[i] == '.' {
					ext = header.Filename[i:]
					break
				}
			}
		}
		catalogCid, _ := h.UpdateGlobalCatalog(r.Context(), header.Filename, string(root), header.Size, ext)

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"root_cid":"` + string(root) + `","catalog_cid":"` + string(catalogCid) + `"}` + "\n"))
	}
}
