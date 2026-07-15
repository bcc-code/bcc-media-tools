package main

import (
	"net/http"
	"os"
	"path/filepath"
)

// overlayPreviewHandler serves an overlay/watermark image directly from
// OVERLAYS_DIR (the overlay file itself is the preview).
// GET /overlay-preview?name=<overlay-filename>
type overlayPreviewHandler struct{}

func (overlayPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanExport() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	dir := os.Getenv("OVERLAYS_DIR")
	if dir == "" {
		http.NotFound(w, r)
		return
	}

	name := r.URL.Query().Get("name")
	// Only a bare filename is allowed, guarding against path traversal.
	if name == "" || name != filepath.Base(name) {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeFile(w, r, filepath.Join(dir, name))
}
