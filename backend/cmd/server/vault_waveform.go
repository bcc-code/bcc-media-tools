package main

import (
	"net/http"
	"regexp"
	"strconv"

	"github.com/bcc-code/mediabank-bridge/log"
	lru "github.com/hashicorp/golang-lru/v2"
)

// vaultWaveformHandler proxies Vidispine's pre-rendered waveform PNG endpoint
// (/item/{vxid}/waveform/image), with an LRU cache. Item shape analysis must
// have been performed upstream — items without analysis return upstream's
// error (typically 404).
// GET /vault/waveform?vxid=VX-123[&width=400&height=80&bgcolor=000000&fgcolor=ffffff].
type vaultWaveformHandler struct {
	vault *VaultAPI
	cache *lru.Cache[string, []byte]
}

func newVaultWaveformHandler(vault *VaultAPI) *vaultWaveformHandler {
	cache, _ := lru.New[string, []byte](2048)
	return &vaultWaveformHandler{vault: vault, cache: cache}
}

// hexColor is 6 lowercase hex chars; the # prefix is added before forwarding
// to Vidispine. Anything else is dropped so we don't pass garbage upstream.
var hexColor = regexp.MustCompile(`^[0-9a-fA-F]{6}$`)

func (h *vaultWaveformHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !PermissionsForEmail(getEmailFromHttp(r)).CanVault() {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	vxID := r.URL.Query().Get("vxid")
	if vxID == "" {
		http.Error(w, "missing vxid", http.StatusBadRequest)
		return
	}

	width := clampInt(r.URL.Query().Get("width"), 400, 1, 2000)
	height := clampInt(r.URL.Query().Get("height"), 80, 8, 400)
	bgColor := sanitizeColor(r.URL.Query().Get("bgcolor"))
	fgColor := sanitizeColor(r.URL.Query().Get("fgcolor"))

	cacheKey := vxID + "|" + strconv.Itoa(width) + "x" + strconv.Itoa(height) + "|" + bgColor + "|" + fgColor
	if data, ok := h.cache.Get(cacheKey); ok {
		writeWaveform(w, data, "HIT")
		return
	}

	data, err := h.vault.fetchWaveform(vxID, width, height, bgColor, fgColor)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault waveform: fetch failed")
		http.Error(w, "no waveform", http.StatusNotFound)
		return
	}

	h.cache.Add(cacheKey, data)
	writeWaveform(w, data, "MISS")
}

func sanitizeColor(s string) string {
	if !hexColor.MatchString(s) {
		return ""
	}
	return "#" + s
}

func writeWaveform(w http.ResponseWriter, data []byte, cacheStatus string) {
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "private, max-age=86400")
	w.Header().Set("X-Cache", cacheStatus)
	_, _ = w.Write(data)
}

func clampInt(s string, def, min, max int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	if n < min {
		return min
	}
	if n > max {
		return max
	}
	return n
}
