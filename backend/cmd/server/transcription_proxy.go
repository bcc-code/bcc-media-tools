package main

import (
	"io"
	"net/http"

	"github.com/bcc-code/mediabank-bridge/log"
)

// transcriptionPreviewHandler proxies the source video for the transcription
// editor through the server. The upstream (Cantemo) host only resolves on the
// internal network, so handing its URL to the browser breaks for anyone whose
// DNS does not answer internally. It forwards the Range header so the <video>
// element can seek. GET /transcription/preview?vxid=VX-123.
type transcriptionPreviewHandler struct {
	transcription *TranscriptionAPI
	cantemoToken  string
}

func (h transcriptionPreviewHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	vxID := r.URL.Query().Get("vxid")
	if vxID == "" {
		http.Error(w, "missing vxid", http.StatusBadRequest)
		return
	}

	// Same policy as the transcription RPCs: admins always, volunteers only
	// while the asset is shared for editing.
	if err := h.transcription.authorize(getEmailFromHttp(r), vxID); err != nil {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	previewURL, err := h.transcription.cantemoClient.GetPreviewUrl(vxID)
	if err != nil || previewURL == "" {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("transcription: preview not available")
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
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("transcription: preview upstream failed")
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
