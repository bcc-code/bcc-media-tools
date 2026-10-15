package main

import (
	"bytes"
	"context"
	"testing"

	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xuri/excelize/v2"
)

func openWorkbook(t *testing.T, data []byte) (*excelize.File, string) {
	t.Helper()
	f, err := excelize.OpenReader(bytes.NewReader(data))
	require.NoError(t, err)
	t.Cleanup(func() { _ = f.Close() })
	return f, f.GetSheetName(0)
}

// A live session exports Oslo clock times, the running last row has no
// duration, and labels follow the requested language.
func TestEditorialWorkbookLive(t *testing.T) {
	// 09:00:00 UTC is 11:00:00 in Oslo in summer.
	sess := editorial.Session{Title: "Søndag: formiddag", Kind: editorial.KindLive, ClockStartMs: ts(t, "2026-07-26T09:00:00Z").UnixMilli()}
	markers := []editorial.Marker{
		{StartMs: 5_000, Contributors: "Kåre J. Smith", BibleVerses: "John 3:16", Type: "tale", Source: editorial.SourceManual, PlayoutKey: "SPEAKER|x", PublishBmm: true},
		{StartMs: 760_000, Name: "HV 16", Type: "sang", Source: editorial.SourceImport},
	}
	data, err := editorialWorkbook(sess, markers, exportLabelsByLanguage["nb"])
	require.NoError(t, err)
	f, sheet := openWorkbook(t, data)

	assert.Equal(t, "Søndag- formiddag", sheet)
	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	require.Len(t, rows, 3)
	assert.Equal(t, []string{"Tid", "Varighet", "Tittel", "Medvirkende", "Bibelvers", "Type", "Kommentar", "BMM", "BCC Media", "Fra"}, rows[0])

	assert.Equal(t, "11:00:05", rows[1][0])
	assert.Equal(t, "0:12:35", rows[1][1])
	assert.Equal(t, "Kåre J. Smith", rows[1][3])
	assert.Equal(t, "Tale", rows[1][5])
	assert.Equal(t, "Ja", rows[1][7])
	assert.Equal(t, "Du · fylt ut av Playout", rows[1][9])

	assert.Equal(t, "11:12:40", rows[2][0])
	assert.Empty(t, rows[2][1], "the last row is still running")
	assert.Equal(t, "Playout", rows[2][9])
}

func TestEditorialWorkbookRecording(t *testing.T) {
	sess := editorial.Session{Vxid: "VX-1", Kind: editorial.KindRecording}
	markers := []editorial.Marker{{StartMs: 391_000, EndMs: 1_860_000, Name: "Tale", Type: "custom"}}
	data, err := editorialWorkbook(sess, markers, exportLabelsByLanguage["en"])
	require.NoError(t, err)
	f, sheet := openWorkbook(t, data)

	assert.Equal(t, "VX-1", sheet)
	rows, err := f.GetRows(sheet)
	require.NoError(t, err)
	require.Len(t, rows, 2)
	assert.Equal(t, "Start", rows[0][0])
	assert.Equal(t, []string{"0:06:31", "0:31:00", "0:24:29"}, rows[1][:3])
	assert.Equal(t, "Custom", rows[1][6], "unknown types are kept")
	assert.Equal(t, "No", rows[1][8])
}

func TestEditorialExportHandler(t *testing.T) {
	api := newTestEditorialAPI(t)
	sess := createTestSession(t, api, "VX-9", `Lørdag / kveld?`)
	saveTestSession(t, api, sess.Id, `Lørdag / kveld?`, []*apiv1.EditorialMarker{{Name: "Sang", StartMs: 0, EndMs: 60_000}})

	res, err := api.ExportEditorialSession(context.Background(), connect.NewRequest(&apiv1.ExportEditorialSessionRequest{Id: sess.Id, Language: "xx"}))
	require.NoError(t, err)
	assert.Equal(t, "Lørdag - kveld-.xlsx", res.Msg.Filename)
	f, sheet := openWorkbook(t, res.Msg.Data)
	header, err := f.GetCellValue(sheet, "A1")
	require.NoError(t, err)
	assert.Equal(t, "Start", header, "unknown languages fall back to English")

	_, err = api.ExportEditorialSession(context.Background(), connect.NewRequest(&apiv1.ExportEditorialSessionRequest{Id: "missing"}))
	requireNotFound(t, err)
}
