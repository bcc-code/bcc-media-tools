package main

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// subtitleStylePreviewHandler serves the preview image paired with a subtitle
// style: the style file's basename with a .png extension, from the same
// SUBTITLE_STYLES_DIR the styles are listed from.
// GET /subtitle-style-preview?name=<style-filename>
type subtitleStylePreviewHandler struct{}

func (subtitleStylePreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanVBExport() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	dir := os.Getenv("SUBTITLE_STYLES_DIR")
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

	// Pair by basename: "Foo.ass" -> "Foo.png".
	base := strings.TrimSuffix(name, filepath.Ext(name))
	path := filepath.Join(dir, base+".png")

	w.Header().Set("Cache-Control", "public, max-age=300")
	http.ServeFile(w, r, path)
}
