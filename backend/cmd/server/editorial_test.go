package main

import (
	"bcc-media-tools/editorial"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatMillisTC(t *testing.T) {
	assert.Equal(t, "00:00:00", formatMillisTC(0))
	assert.Equal(t, "00:00:05", formatMillisTC(5000))
	assert.Equal(t, "00:01:30", formatMillisTC(90000))
	assert.Equal(t, "01:00:00", formatMillisTC(3600000))
	// Negative (e.g. end before start) clamps to zero.
	assert.Equal(t, "00:00:00", formatMillisTC(-500))
}

func TestEditorialExportFilename(t *testing.T) {
	assert.Equal(t, "Sunday-stream.csv", editorialExportFilename(&editorial.Session{Title: "Sunday stream"}))
	// Falls back to VXID when title is empty.
	assert.Equal(t, "VX-123.csv", editorialExportFilename(&editorial.Session{VXID: "VX-123"}))
	// Strips unsafe characters (æ/ø/å, slashes); spaces become hyphens.
	assert.Equal(t, "mte2026.csv", editorialExportFilename(&editorial.Session{Title: "møte/2026"}))
	// Empty everything falls back.
	assert.Equal(t, "editorial.csv", editorialExportFilename(&editorial.Session{}))
}

func TestEditorialSessionToCSV(t *testing.T) {
	sess := &editorial.Session{
		Title: "stream",
		Markers: []editorial.Marker{
			{Name: "Speaker A", Type: "appell", StartMS: 1000, EndMS: 5000, Publish: true},
			{Name: "Song", Type: "sang", StartMS: 6000, EndMS: 9000, Publish: false},
		},
	}

	out, err := editorialSessionToCSV(sess)
	require.NoError(t, err)

	// UTF-8 BOM prefix so Excel reads Norwegian characters.
	assert.Equal(t, []byte{0xEF, 0xBB, 0xBF}, out[:3])

	text := string(out[3:])
	lines := strings.Split(strings.TrimRight(text, "\r\n"), "\n")
	require.Len(t, lines, 3) // header + 2 markers

	assert.Equal(t, "Hvem eller hva;Type;Start;Slutt;Varighet;Publiseres", strings.TrimRight(lines[0], "\r"))
	assert.Equal(t, "Speaker A;appell;00:00:01;00:00:05;00:00:04;Ja", strings.TrimRight(lines[1], "\r"))
	assert.Equal(t, "Song;sang;00:00:06;00:00:09;00:00:03;Nei", strings.TrimRight(lines[2], "\r"))
}
