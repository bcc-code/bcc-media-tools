package main

import (
	"io"
	"net/http"

	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/bcc-code/mediabank-bridge/log"
)

// vaultThumbnailHandler streams an item's thumbnail through the server (the
// Vidispine thumbnail endpoints require basic auth, so they cannot be loaded
// directly by the browser). GET /vault/thumbnail?vxid=VX-123[&t=<time>].
type vaultThumbnailHandler struct {
	vault *VaultAPI
}

func (h vaultThumbnailHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanVault() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	vxID := r.URL.Query().Get("vxid")
	if vxID == "" {
		http.Error(w, "missing vxid", http.StatusBadRequest)
		return
	}

	// Optional fraction (0..1) along the asset for trick-play; empty = poster.
	data, contentType, err := h.vault.fetchThumbnail(vxID, r.URL.Query().Get("f"))
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault: thumbnail not available")
		http.Error(w, "no thumbnail", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "private, max-age=3600")
	_, _ = w.Write(data)
}

// vaultPreviewHandler proxies an item's preview shape through the server so no
// upstream (Cantemo) URL is exposed to the browser. It forwards the Range
// header so the <video> element can seek. GET /vault/preview?vxid=VX-123.
type vaultPreviewHandler struct {
	cantemo      *cantemo.Client
	cantemoToken string
}

func (h vaultPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanVault() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	vxID := r.URL.Query().Get("vxid")
	if vxID == "" {
		http.Error(w, "missing vxid", http.StatusBadRequest)
		return
	}

	previewURL, err := h.cantemo.GetPreviewUrl(vxID)
	if err != nil || previewURL == "" {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault: preview not available")
		http.Error(w, "no preview", http.StatusNotFound)
		return
	}

	upstream, err := http.NewRequestWithContext(r.Context(), http.MethodGet, previewURL, nil)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusInternalServerError)
		return
	}
	if rng := r.Header.Get("Range"); rng != "" {
		upstream.Header.Set("Range", rng)
	}
	if h.cantemoToken != "" {
		upstream.Header.Set("Auth-Token", h.cantemoToken)
	}

	resp, err := http.DefaultClient.Do(upstream)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault: preview upstream failed")
		http.Error(w, "preview upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	for _, hdr := range []string{"Content-Type", "Content-Length", "Content-Range", "Accept-Ranges", "Last-Modified", "ETag"} {
		if val := resp.Header.Get(hdr); val != "" {
			w.Header().Set(hdr, val)
		}
	}
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}
