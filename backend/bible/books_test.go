package bible

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLookupBookAbbreviations(t *testing.T) {
	// Spellings seen in the editorial data, plus the ones prefix matching has
	// to get right on its own.
	tests := map[string]string{
		"2Kr":    "2Chr", // as typed in Mediabanken
		"2Krø":   "2Chr",
		"2Krøn":  "2Chr",
		"2 Kron": "2Chr",
		"1Kong":  "1Kgs",
		"2Kong":  "2Kgs",
		"2Kon":   "2Kgs",
		"1Kor":   "1Cor",
		"2Kor":   "2Cor",
		"Sal":    "Ps",
		"Salme":  "Ps",
		"1. Mos": "Gen",
		"5Mos":   "Deu",
		"Hebr":   "Heb",
		"Luk":    "Lk",
		"Mat":    "Mt",
		"Matt":   "Mt",
		"Ef":     "Eph",
		"Fil":    "Phil",
		"Filem":  "Phlm",
		"Jud":    "Jude",
		"Åp":     "Rev",
		"Apg":    "Acts",
		"Joh":    "Jn",
		"1Joh":   "1Jn",
		"3Joh":   "3Jn",
		"1Tim":   "1Tim",
		"1Tess":  "1Ths",
		"Jes":    "Isa",
		"Jer":    "Jer",
		"Jak":    "Jas",
		"1Pet":   "1Pet",
		"Åpenb":  "Rev",
		// English spellings turn up too.
		"John":         "Jn",
		"2 Chronicles": "2Chr",
		"Rev":          "Rev",
	}

	for in, want := range tests {
		t.Run(in, func(t *testing.T) {
			b, ok := lookupBook(in)
			assert.True(t, ok, "should resolve")
			assert.Equal(t, want, b.ID)
		})
	}
}

func TestLookupBookRejectsAmbiguous(t *testing.T) {
	// Too short to tell books apart, or no book at all — better to show the
	// reference as typed than to guess at scripture.
	for _, in := range []string{"J", "Jo", "K", "1K", "Es", "xyz", ""} {
		t.Run(in, func(t *testing.T) {
			_, ok := lookupBook(in)
			assert.False(t, ok)
		})
	}
}

func TestLookupBookOrdinalMustMatch(t *testing.T) {
	// A number on the token can only reach a numbered book.
	_, ok := lookupBook("2Jes")
	assert.False(t, ok)
	_, ok = lookupBook("Kor")
	assert.False(t, ok, "Korinterbrev is numbered; a bare Kor is ambiguous")
}
