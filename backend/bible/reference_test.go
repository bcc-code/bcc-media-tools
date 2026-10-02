package bible

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseReferences(t *testing.T) {
	// The left column is what editors actually typed into Mediabanken.
	tests := []struct {
		in      string
		book    string
		chapter int
		ranges  []VerseRange
		display string
	}{
		{"1Kor 1,18+23-24", "1Cor", 1, []VerseRange{{18, 18}, {23, 24}}, "1. Korinterbrev 1,18+23–24"},
		{"1Tim 2,5", "1Tim", 2, []VerseRange{{5, 5}}, "1. Timoteus 2,5"},
		{"Hebr 2,14", "Heb", 2, []VerseRange{{14, 14}}, "Hebreerne 2,14"},
		{"Luk 9,23-24", "Lk", 9, []VerseRange{{23, 24}}, "Lukas 9,23–24"},
		{"Mat 7,24–27", "Mt", 7, []VerseRange{{24, 27}}, "Matteus 7,24–27"},
		{"Ef 1,19-20", "Eph", 1, []VerseRange{{19, 20}}, "Efeserne 1,19–20"},
		{"Åp 21,1", "Rev", 21, []VerseRange{{1, 1}}, "Åpenbaringen 21,1"},
		{"1. Mos 1,1-3", "Gen", 1, []VerseRange{{1, 3}}, "1. Mosebok 1,1–3"},
		// Shortened to the point of being unofficial, but unambiguous.
		{"2Kr 16:9", "2Chr", 16, []VerseRange{{9, 9}}, "2. Krønikebok 16,9"},
		{"Sal 51:8", "Ps", 51, []VerseRange{{8, 8}}, "Salmene 51,8"},
		{"Sal 23,1.4", "Ps", 23, []VerseRange{{1, 1}, {4, 4}}, "Salmene 23,1+4"},
		// Colon form and English abbreviations turn up too.
		{"John 3:16", "Jn", 3, []VerseRange{{16, 16}}, "Johannes 3,16"},
		{"Rom 8:1-4", "Rom", 8, []VerseRange{{1, 4}}, "Romerne 8,1–4"},
		// A whole chapter: no verse runs, still a valid reference.
		{"Sal 23", "Ps", 23, nil, "Salmene 23"},
	}

	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			ref, err := Parse(tt.in)
			require.NoError(t, err)
			assert.Equal(t, tt.book, ref.Book)
			assert.Equal(t, tt.chapter, ref.Chapter)
			assert.Equal(t, tt.ranges, ref.Ranges)
			assert.Equal(t, tt.display, ref.Display())
		})
	}
}

func TestParseRejectsNonReferences(t *testing.T) {
	for _, in := range []string{"", "   ", "ukjent bok 1,2", "Kommentar", "1Kor", "Luk kapittel", "Jo 3,16"} {
		t.Run(in, func(t *testing.T) {
			_, err := Parse(in)
			assert.ErrorIs(t, err, ErrUnparsable)
		})
	}
}

func TestParseIgnoresReversedRange(t *testing.T) {
	// "12-4" is a typo, not a range; keep the start verse rather than failing.
	ref, err := Parse("Joh 3,12-4")
	require.NoError(t, err)
	assert.Equal(t, []VerseRange{{12, 12}}, ref.Ranges)
}
