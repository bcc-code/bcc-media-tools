package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bcc-code/bcc-media-flows/services/cantemo"
	"github.com/bcc-code/mediabank-bridge/log"
	lru "github.com/hashicorp/golang-lru/v2"
)

// vaultWaveformHandler renders a peak-waveform PNG of an item's audio
// preview. Audio is decoded by piping through ffmpeg, which handles every
// format we've seen (WAV, MP3, M4A/AAC, FLAC, OGG, ...). Result is cached.
// GET /vault/waveform?vxid=VX-123[&width=400][&height=80].
type vaultWaveformHandler struct {
	cantemo      *cantemo.Client
	cantemoToken string
	cache        *lru.Cache[string, []byte]
}

func newVaultWaveformHandler(c *cantemo.Client, token string) *vaultWaveformHandler {
	cache, _ := lru.New[string, []byte](2048)
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		log.L.Warn().Err(err).Msg("vault waveform: ffmpeg not found in PATH; /vault/waveform will return 500")
	}
	return &vaultWaveformHandler{cantemo: c, cantemoToken: token, cache: cache}
}

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

	cacheKey := vxID + "|" + strconv.Itoa(width) + "x" + strconv.Itoa(height)
	if data, ok := h.cache.Get(cacheKey); ok {
		writeWaveform(w, data, "HIT")
		return
	}

	previewURL, err := h.cantemo.GetPreviewUrl(vxID)
	if err != nil || previewURL == "" {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault waveform: preview not available")
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
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault waveform: upstream failed")
		http.Error(w, "upstream failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		http.Error(w, "upstream not OK", resp.StatusCode)
		return
	}
	log.L.Debug().
		Str("vxid", vxID).
		Str("upstream_content_type", resp.Header.Get("Content-Type")).
		Str("upstream_content_length", resp.Header.Get("Content-Length")).
		Msg("vault waveform: upstream fetched")

	peaks, err := decodePeaks(r.Context(), resp.Body, width, vxID)
	if err != nil {
		log.L.Debug().Err(err).Str("vxid", vxID).Msg("vault waveform: decode failed")
		http.Error(w, "decode failed", http.StatusUnsupportedMediaType)
		return
	}

	img := renderWaveform(peaks, width, height)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		http.Error(w, "encode failed", http.StatusInternalServerError)
		return
	}

	out := buf.Bytes()
	h.cache.Add(cacheKey, out)
	writeWaveform(w, out, "MISS")
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

// decodePeaks buffers the upstream audio to a temp file (some containers —
// notably MP4/M4A with moov at end-of-file — require seekable input that
// stdin pipes don't provide), then runs ffmpeg → mono 16-bit LE PCM at 8 kHz.
// 8 kHz is plenty for visual peaks and keeps the buffer small.
func decodePeaks(ctx context.Context, r io.Reader, width int, vxID string) ([]float64, error) {
	tmp, err := os.CreateTemp("", "vault-wave-*")
	if err != nil {
		return nil, fmt.Errorf("create temp: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)

	// Cap the buffer at 500 MB so a runaway upstream can't fill the disk.
	const maxBuffered = 500 << 20
	written, copyErr := io.Copy(tmp, io.LimitReader(r, maxBuffered))
	if cerr := tmp.Close(); cerr != nil && copyErr == nil {
		copyErr = cerr
	}
	if copyErr != nil {
		return nil, fmt.Errorf("buffer upstream: %w", copyErr)
	}
	if written == 0 {
		return nil, fmt.Errorf("upstream returned 0 bytes")
	}

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-hide_banner",
		"-nostdin",
		"-loglevel", "warning",
		"-fflags", "+discardcorrupt",
		"-i", tmpPath,
		"-ac", "1",
		"-ar", "8000",
		"-f", "s16le",
		"pipe:1",
	)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	pcm, err := cmd.Output()
	stderrStr := strings.TrimSpace(stderr.String())
	if err != nil {
		return nil, fmt.Errorf("ffmpeg: %w (stderr: %s)", err, stderrStr)
	}
	if len(pcm) == 0 {
		return nil, fmt.Errorf("ffmpeg produced no PCM output (stderr: %s)", stderrStr)
	}
	log.L.Debug().
		Str("vxid", vxID).
		Int64("upstream_bytes", written).
		Int("pcm_bytes", len(pcm)).
		Str("ffmpeg_stderr", stderrStr).
		Msg("vault waveform: ffmpeg decoded")
	return pcmPeaks(pcm, 1, width), nil
}

// pcmPeaks buckets interleaved 16-bit LE PCM samples into `width` columns
// and returns the max-abs peak per column, normalized to [0, 1] against the
// loudest column.
func pcmPeaks(data []byte, channels, width int) []float64 {
	if channels < 1 {
		channels = 1
	}
	bytesPerFrame := 2 * channels
	frameCount := len(data) / bytesPerFrame
	if frameCount == 0 {
		return make([]float64, width)
	}

	peaks := make([]float64, width)
	maxPeak := 0.0
	for col := 0; col < width; col++ {
		start := col * frameCount / width
		end := (col + 1) * frameCount / width
		if end <= start {
			end = start + 1
		}
		if end > frameCount {
			end = frameCount
		}
		var peak int32
		for i := start; i < end; i++ {
			base := i * bytesPerFrame
			for c := 0; c < channels; c++ {
				s := int32(int16(binary.LittleEndian.Uint16(data[base+c*2:])))
				if s < 0 {
					s = -s
				}
				if s > peak {
					peak = s
				}
			}
		}
		v := float64(peak) / 32768.0
		peaks[col] = v
		if v > maxPeak {
			maxPeak = v
		}
	}
	if maxPeak > 0 {
		scale := 1.0 / maxPeak
		for i := range peaks {
			peaks[i] *= scale
		}
	}
	return peaks
}

func renderWaveform(peaks []float64, width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	col := color.RGBA{R: 0x88, G: 0x88, B: 0x88, A: 0xff}
	mid := height / 2
	for x, p := range peaks {
		h := int(p*float64(height)/2 + 0.5)
		if h < 1 {
			h = 1
		}
		top := mid - h
		bot := mid + h
		if top < 0 {
			top = 0
		}
		if bot > height {
			bot = height
		}
		for y := top; y < bot; y++ {
			img.SetRGBA(x, y, col)
		}
	}
	return img
}
