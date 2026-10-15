package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"bcc-media-tools/editorial"
	"context"
	"fmt"
	"strings"
	"time"
	"unicode"

	"connectrpc.com/connect"
	"github.com/xuri/excelize/v2"
)

// exportLabels are the workbook's words in one language. The editorial team
// works in Norwegian, so the frontend asks for its own UI language.
type exportLabels struct {
	Time, Start, End, Duration, Title, Contributors, BibleVerses, Type, Comment, PublishBmm, PublishBcc, From string
	Yes, No                                                                                                   string
	FromPlayout, FromYouFilled, FromYou                                                                       string
	Types                                                                                                     map[string]string
}

var exportLabelsByLanguage = map[string]exportLabels{
	"en": {
		Time: "Time", Start: "Start", End: "End", Duration: "Duration", Title: "Title", Contributors: "Contributors",
		BibleVerses: "Bible verses", Type: "Type", Comment: "Comment", PublishBmm: "BMM", PublishBcc: "BCC Media", From: "From",
		Yes: "Yes", No: "No",
		FromPlayout: "Playout", FromYouFilled: "You · filled by Playout", FromYou: "You",
		Types: map[string]string{
			"vitnesbyrd": "Testimony", "sang": "Song", "allsang": "Singsong", "tale": "Speech", "bønn": "Prayer",
			"tydning": "Interpretation", "programleder": "Host", "video": "Video", "intervju": "Interview", "annet": "Other",
		},
	},
	"nb": {
		Time: "Tid", Start: "Start", End: "Slutt", Duration: "Varighet", Title: "Tittel", Contributors: "Medvirkende",
		BibleVerses: "Bibelvers", Type: "Type", Comment: "Kommentar", PublishBmm: "BMM", PublishBcc: "BCC Media", From: "Fra",
		Yes: "Ja", No: "Nei",
		FromPlayout: "Playout", FromYouFilled: "Du · fylt ut av Playout", FromYou: "Du",
		Types: map[string]string{
			"vitnesbyrd": "Vitnesbyrd", "sang": "Sang", "allsang": "Allsang", "tale": "Tale", "bønn": "Bønn",
			"tydning": "Tydning", "programleder": "Programleder", "video": "Video", "intervju": "Intervju", "annet": "Annet",
		},
	},
}

func (e EditorialAPI) ExportEditorialSession(ctx context.Context, req *connect.Request[apiv1.ExportEditorialSessionRequest]) (*connect.Response[apiv1.ExportEditorialSessionResponse], error) {
	if _, err := requireEditorial(req, apiv1.EditorialReviewer); err != nil {
		return nil, err
	}
	sess, err := e.queries.GetSession(ctx, req.Msg.GetId())
	if err != nil {
		return nil, editorialErr(err)
	}
	markers, err := e.queries.ListMarkersForSession(ctx, sess.ID)
	if err != nil {
		return nil, editorialErr(fmt.Errorf("editorial: list markers: %w", err))
	}
	labels, ok := exportLabelsByLanguage[req.Msg.GetLanguage()]
	if !ok {
		labels = exportLabelsByLanguage["en"]
	}

	data, err := editorialWorkbook(sess, markers, labels)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}
	return connect.NewResponse(&apiv1.ExportEditorialSessionResponse{
		Data:     data,
		Filename: exportFilename(sess),
	}), nil
}

// editorialWorkbook lays the markers out like the tool's table. Times are real
// Excel times so the sheet can be sorted and summed: a live session's clock
// time of day (Oslo), or a recording's offsets.
func editorialWorkbook(sess editorial.Session, markers []editorial.Marker, l exportLabels) ([]byte, error) {
	live := sess.Kind == editorial.KindLive
	loc, err := time.LoadLocation(recordingTZ)
	if err != nil {
		return nil, fmt.Errorf("load %s: %w", recordingTZ, err)
	}

	var header []any
	if live {
		header = []any{l.Time}
	} else {
		header = []any{l.Start, l.End}
	}
	header = append(header, l.Duration, l.Title, l.Contributors, l.BibleVerses, l.Type, l.Comment, l.PublishBmm, l.PublishBcc)
	if live {
		header = append(header, l.From)
	}

	f := excelize.NewFile()
	defer f.Close()
	sheet := sheetName(sess)
	if err := f.SetSheetName(f.GetSheetName(0), sheet); err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	if err := f.SetSheetRow(sheet, "A1", &header); err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}

	yesNo := func(b bool) string {
		if b {
			return l.Yes
		}
		return l.No
	}
	for i, m := range markers {
		var row []any
		var durationMs int64
		if live {
			at := time.UnixMilli(sess.ClockStartMs + m.StartMs).In(loc)
			row = []any{osloWallClock(at)}
			if i+1 < len(markers) {
				durationMs = markers[i+1].StartMs - m.StartMs
			}
		} else {
			row = []any{excelDuration(m.StartMs), excelDuration(m.EndMs)}
			durationMs = m.EndMs - m.StartMs
		}
		// The last live row is still running: no duration yet.
		var duration any
		if durationMs > 0 {
			duration = excelDuration(durationMs)
		}
		row = append(row, duration, m.Name, m.Contributors, m.BibleVerses, typeLabel(m.Type, l), m.Comment,
			yesNo(m.PublishBmm), yesNo(m.PublishBcc))
		if live {
			row = append(row, sourceLabel(m, l))
		}
		cell, _ := excelize.CoordinatesToCellName(1, i+2)
		if err := f.SetSheetRow(sheet, cell, &row); err != nil {
			return nil, fmt.Errorf("xlsx: %w", err)
		}
	}

	if err := styleWorkbook(f, sheet, len(header), len(markers), live); err != nil {
		return nil, err
	}
	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, fmt.Errorf("xlsx: %w", err)
	}
	return buf.Bytes(), nil
}

// styleWorkbook bolds and freezes the header, filters it, formats the time
// columns and sizes the columns for their content.
func styleWorkbook(f *excelize.File, sheet string, cols, rows int, live bool) error {
	lastCol, _ := excelize.ColumnNumberToName(cols)
	lastRow := rows + 1

	bold, err := f.NewStyle(&excelize.Style{Font: &excelize.Font{Bold: true}})
	if err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	clock := "hh:mm:ss"
	clockStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &clock})
	if err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	// [h] keeps counting past 24 hours instead of wrapping.
	elapsed := "[h]:mm:ss"
	elapsedStyle, err := f.NewStyle(&excelize.Style{CustomNumFmt: &elapsed})
	if err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}

	if err := f.SetCellStyle(sheet, "A1", lastCol+"1", bold); err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	if rows > 0 {
		// Live: A is clock time, B duration. Recording: A–C are start, end, duration.
		timeCols := []struct {
			col   string
			style int
		}{{"A", clockStyle}, {"B", elapsedStyle}}
		if !live {
			timeCols = []struct {
				col   string
				style int
			}{{"A", elapsedStyle}, {"B", elapsedStyle}, {"C", elapsedStyle}}
		}
		for _, c := range timeCols {
			if err := f.SetCellStyle(sheet, fmt.Sprintf("%s2", c.col), fmt.Sprintf("%s%d", c.col, lastRow), c.style); err != nil {
				return fmt.Errorf("xlsx: %w", err)
			}
		}
	}
	if err := f.SetPanes(sheet, &excelize.Panes{Freeze: true, YSplit: 1, TopLeftCell: "A2", ActivePane: "bottomLeft"}); err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}
	if err := f.AutoFilter(sheet, fmt.Sprintf("A1:%s%d", lastCol, lastRow), nil); err != nil {
		return fmt.Errorf("xlsx: %w", err)
	}

	// Narrow time and Yes/No columns, wide text columns.
	widths := []float64{10, 10, 36, 28, 22, 14, 30, 8, 11, 24}
	if !live {
		widths = []float64{10, 10, 10, 36, 28, 22, 14, 30, 8, 11}
	}
	for i, w := range widths {
		col, _ := excelize.ColumnNumberToName(i + 1)
		if err := f.SetColWidth(sheet, col, col, w); err != nil {
			return fmt.Errorf("xlsx: %w", err)
		}
	}
	return nil
}

// excelDuration is a span as Excel stores it: a fraction of a day.
func excelDuration(ms int64) float64 {
	return float64(ms) / float64(24*time.Hour/time.Millisecond)
}

// osloWallClock re-labels an Oslo time as UTC with the same wall clock:
// excelize stores the instant in UTC, and the sheet should read Oslo time.
func osloWallClock(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), 0, time.UTC)
}

func typeLabel(t string, l exportLabels) string {
	if label, ok := l.Types[t]; ok {
		return label
	}
	if t == "" {
		return ""
	}
	r := []rune(t)
	return string(unicode.ToUpper(r[0])) + string(r[1:])
}

// sourceLabel mirrors the "From" column of the live table.
func sourceLabel(m editorial.Marker, l exportLabels) string {
	switch {
	case m.Source == editorial.SourceImport:
		return l.FromPlayout
	case m.PlayoutKey != "":
		return l.FromYouFilled
	}
	return l.FromYou
}

// sheetName fits the session title to Excel's rules: at most 31 characters,
// none of : \ / ? * [ ].
func sheetName(sess editorial.Session) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`:\/?*[]`, r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(sessionLabel(sess)))
	if r := []rune(name); len(r) > 31 {
		name = strings.TrimSpace(string(r[:31]))
	}
	if name == "" {
		name = "Markers"
	}
	return name
}

// exportFilename is the session's name made safe for any file system.
func exportFilename(sess editorial.Session) string {
	name := strings.Map(func(r rune) rune {
		if strings.ContainsRune(`<>:"/\|?*`, r) || unicode.IsControl(r) {
			return '-'
		}
		return r
	}, strings.TrimSpace(sessionLabel(sess)))
	if name == "" {
		name = "editorial"
	}
	return name + ".xlsx"
}

func sessionLabel(sess editorial.Session) string {
	if sess.Title != "" {
		return sess.Title
	}
	return sess.Vxid
}
