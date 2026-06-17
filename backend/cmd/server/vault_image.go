package main

import (
	"bytes"
	"image"
	_ "image/gif" // register decoder
	"image/jpeg"
	_ "image/png" // register decoder
	"net/http"
	"strconv"
	"strings"

	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/bcc-code/mediabank-bridge/log"
	lru "github.com/hashicorp/golang-lru/v2"
	"golang.org/x/image/draw"
)

// vaultImageHandler returns a resized JPEG version of an item's preview shape,
// for use as a grid thumbnail. The Cantemo preview shape for static images is
// the full-resolution file, which is too heavy for a 5-col grid of 50 cards;
// this handler decodes + downscales in-process and caches the result in
// memory (LRU). GET /vault/image?vxid=VX-123[&width=400].
type vaultImageHandler struct {
	cantemo      *cantemo.Client
	cantemoToken string
	cache        *lru.Cache[string, []byte]
}

func newVaultImageHandler(c *cantemo.Client, token string) *vaultImageHandler {
	// 1024 entries × ~50KB per JPEG ≈ ~50MB upper bound.
	cache, _ := lru.New[string, []byte](1024)
	return &vaultImageHandler{cantemo: c, cantemoToken: token, cache: cache}
}

func (h *vaultImageHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanVault() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vxID := r.URL.Query().Get("vxid")
	if vxID == "" {
		http.Error(w, "missing vxid", http.StatusBadRequest)
		return
	}

	width := 400
	if s := r.URL.Query().Get("width"); s != "" {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && n <= 1600 {
			width = n
		}
	}

	cacheKey := vxID + "|" + strconv.Itoa(width)
	if data, ok := h.cache.Get(cacheKey); ok {
		writeImage(w, data, "HIT")
		return
	}

	previewURL, err := h.cantemo.GetPreviewUrl(vxID)
	if err != nil || previewURL == "" {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault image: preview not available")
		http.Error(w, "no preview", http.StatusNotFound)
		return
	}

	req, err := http.NewRequestWithContext(r.Context(), http.MethodGet, previewURL, nil)
	if err != nil {
		http.Error(w, "bad upstream", http.StatusInternalServerError)
		return
	}
	if h.cantemoToken != "" {
		req.Header.Set("Auth-Token", h.cantemoToken)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault image: upstream failed")
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "upstream not OK", resp.StatusCode)
		return
	}
	if ct := resp.Header.Get("Content-Type"); !strings.HasPrefix(ct, "image/") {
		http.Error(w, "not an image", http.StatusUnsupportedMediaType)
		return
	}

	src, _, err := image.Decode(resp.Body)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault image: decode failed")
		http.Error(w, "decode failed", http.StatusInternalServerError)
		return
	}

	dst := scaleToWidth(src, width)
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dst, &jpeg.Options{Quality: 85}); err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}

	out := buf.Bytes()
	h.cache.Add(cacheKey, out)
	writeImage(w, out, "MISS")
}

func writeImage(w http.ResponseWriter, data []byte, cacheStatus string) {
	w.Header().Set("Content-Type", "image/jpeg")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Cache", cacheStatus)
	_, _ = w.Write(data)
}

// scaleToWidth proportionally resizes src to the given width, preserving aspect.
// If the source is already at or below the target width, returns src unchanged.
func scaleToWidth(src image.Image, targetW int) image.Image {
	b := src.Bounds()
	if b.Dx() <= targetW {
		return src
	}
	scale := float64(targetW) / float64(b.Dx())
	targetH := int(float64(b.Dy())*scale + 0.5)
	dst := image.NewRGBA(image.Rect(0, 0, targetW, targetH))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, b, draw.Over, nil)
	return dst
}
