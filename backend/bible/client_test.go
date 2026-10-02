package bible

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A real response from https://bibleapi.bcc.media/v1/nb-1930/Lk/9/23/24.
const sampleBody = `{"verses":[{"text":"Og han sa til alle: Vil nogen komme efter mig …","number":23},{"text":"For den som vil berge sitt liv, skal miste det …","number":24}],"bible_id":"nb-1930","book_id":"Lk","chapter":9}`

func TestDecodeVerses(t *testing.T) {
	verses, err := decodeVerses([]byte(sampleBody))
	require.NoError(t, err)
	require.Len(t, verses, 2)
	assert.Equal(t, int32(23), verses[0].Number)
	assert.Contains(t, verses[0].Text, "Vil nogen komme efter mig")
	assert.Equal(t, int32(24), verses[1].Number)
}

func TestVersesFetchesEveryRange(t *testing.T) {
	var paths []string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(sampleBody))
	}))
	defer srv.Close()

	c := NewClient(srv.URL, "nb-1930")
	ref, err := Parse("1Kor 1,18+23-24")
	require.NoError(t, err)

	verses, err := c.Verses(context.Background(), ref)
	require.NoError(t, err)
	// One request per verse run, and the results concatenated in order.
	assert.Equal(t, []string{"/nb-1930/1Cor/1/18/18", "/nb-1930/1Cor/1/23/24"}, paths)
	assert.Len(t, verses, 4)
}

func TestVersesMissingPassageIsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	ref, err := Parse("Joh 3,16")
	require.NoError(t, err)
	verses, err := NewClient(srv.URL, "nb-1930").Verses(context.Background(), ref)
	require.NoError(t, err)
	assert.Empty(t, verses)
}

func TestVersesWithoutConfiguredBible(t *testing.T) {
	ref, err := Parse("Joh 3,16")
	require.NoError(t, err)
	_, err = NewClient("", "").Verses(context.Background(), ref)
	assert.ErrorIs(t, err, ErrNotConfigured)
}

func TestTranslationLabel(t *testing.T) {
	assert.Equal(t, "NB 1930", NewClient("", "nb-1930").Translation())
	assert.Equal(t, "WEB", NewClient("", "web").Translation())
}
