package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"io"
	"net/http"

	"github.com/bcc-code/mediabank-bridge/log"
)

// editorialPreviewHandler proxies a session's source video through the server,
// the way the transcription and Vault previews already do. Resolving the
// Cantemo URL server-side is not enough on its own: the upstream wants an
// Auth-Token header the browser will not send, and its host only resolves on
// the internal network, so a <video> pointed straight at it fails with "no
// video with supported format and MIME type found". It forwards the Range
// header so the element can seek.
//
// Keyed on the session rather than the VXID, so that being allowed to open the
// review stays what authorizes its video — the client never names an asset.
// GET /editorial/preview?session=<id>.
type editorialPreviewHandler struct {
	editorial    *EditorialAPI
	cantemoToken string
}

func (h editorialPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	email := getEmailFromHttp(r)
	if email == "" || PermissionsForEmail(email).EditorialRole() < apiv1.EditorialReviewer {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	id := r.URL.Query().Get("session")
	if id == "" {
		http.Error(w, "missing session", http.StatusBadRequest)
		return
	}

	// A live session has no asset to preview yet, so it has no VXID either.
	sess, err := h.editorial.loadSession(r.Context(), id)
	if err != nil || sess.GetVXID() == "" {
		log.L.Debug().Err(err).Str("session", id).Msg("editorial: preview session not available")
		http.Error(w, "no preview", http.StatusNotFound)
		return
	}

	previewURL, err := h.editorial.cantemo.GetPreviewUrl(sess.GetVXID())
	if err != nil || previewURL == "" {
		log.L.Debug().Err(err).Str("vxid", sess.GetVXID()).Msg("editorial: preview not available")
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
		log.L.Debug().Err(err).Str("vxid", sess.GetVXID()).Msg("editorial: preview upstream failed")
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
