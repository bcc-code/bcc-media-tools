package main

import (
	apiv1 "bcc-media-tools/api/v1"
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

// Bible-verse marker autocomplete, backed by BCC's bible server
// (https://bibleapi.bcc.media, source: github.com/bcc-code/bible-server).
//
// The bible server has no search/autocomplete endpoint — only "list books" and
// "get verse". So the autocomplete logic (parse a free-text reference, prefix-
// match the book) lives here, driven by the authoritative book list we fetch
// and cache from the server. Using the server means the book names come from
// the actual translation (BCC broadcasts nb-1930) rather than a bundled table.
//
// A book's `id` is the canonical, translation-independent code ("Jn", "Gen")
// used both as the verse-endpoint path segment and in the canonical verse
// format ("Jn 3/16", "1Sam 2/18-34"). That canonical ref becomes the marker's
// entity_id, so a linked verse resolves the same regardless of display language.

const (
	defaultBibleBaseURL = "https://bibleapi.bcc.media/v1"
	// nb-1930 (Norwegian, Bibelen 1930) — BCC's broadcast translation.
	defaultBibleID = "nb-1930"
	booksCacheTTL  = 24 * time.Hour
)

type bibleBook struct {
	ID        string `json:"id"`
	Number    int    `json:"number"`
	LongName  string `json:"long_name"`
	ShortName string `json:"short_name"`

	// Normalized match forms, precomputed once when the list is cached so a
	// search doesn't re-normalize every book on every keystroke.
	normLong  string
	normWords []string
	normShort string
	normID    string
}

// BibleClient fetches and caches the book list for a single translation and
// resolves free-text references against it.
type BibleClient struct {
	bibleID string
	rest    *resty.Client

	mu          sync.Mutex
	books       []bibleBook
	booksExpiry time.Time
}

func NewBibleClient(baseURL, bibleID string) *BibleClient {
	if baseURL == "" {
		baseURL = defaultBibleBaseURL
	}
	if bibleID == "" {
		bibleID = defaultBibleID
	}
	rest := resty.New()
	rest.SetBaseURL(strings.TrimRight(baseURL, "/"))
	rest.SetHeader("Accept", "application/json")
	rest.SetTimeout(8 * time.Second)
	return &BibleClient{bibleID: bibleID, rest: rest}
}

// getBooks returns the cached book list, refreshing it from the server when stale.
func (c *BibleClient) getBooks(ctx context.Context) ([]bibleBook, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.books != nil && time.Now().Before(c.booksExpiry) {
		return c.books, nil
	}

	var books []bibleBook
	resp, err := c.rest.R().
		SetContext(ctx).
		SetResult(&books).
		Get(fmt.Sprintf("/%s/books", c.bibleID))
	if err != nil {
		return nil, err
	}
	if resp.IsError() {
		return nil, fmt.Errorf("bible server returned %s for %s", resp.Status(), resp.Request.URL)
	}

	for i := range books {
		b := &books[i]
		b.normLong = normalizeBibleQuery(b.LongName)
		b.normWords = strings.Fields(b.normLong)
		b.normShort = normalizeBibleQuery(b.ShortName)
		b.normID = normalizeBibleQuery(b.ID)
	}

	c.books = books
	c.booksExpiry = time.Now().Add(booksCacheTTL)
	return books, nil
}

var (
	bibleSpaceRe = regexp.MustCompile(`\s+`)
	// Splits a reference into its book part and a trailing chapter[:verse[-verse]].
	// The book part is non-greedy so a leading number ("1 Johannes") stays with
	// the book while the anchored trailing number is taken as the chapter. Both
	// ":" and "/" are accepted as the chapter/verse separator.
	bibleRefRe = regexp.MustCompile(`^(.*?)\s*(\d+)(?:[:/](\d+)(?:\s*-\s*(\d+))?)?$`)
)

func normalizeBibleQuery(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = strings.ReplaceAll(s, ".", "") // drop abbreviation dots ("Joh." -> "joh")
	return bibleSpaceRe.ReplaceAllString(s, " ")
}

// bookPrimaryMatch reports whether the query prefix names this book by its full
// display name, short name, or canonical id — the strong signals. A book whose
// name merely contains the query as a later word (e.g. "Johannes" in "Johannes’
// åpenbaring") is deliberately excluded.
func bookPrimaryMatch(b bibleBook, q string) bool {
	return strings.HasPrefix(b.normLong, q) ||
		strings.HasPrefix(b.normShort, q) ||
		strings.HasPrefix(b.normID, q)
}

// bookMatchTier scores how strongly the query names this book: 2 for an exact
// short-name/id/long-name hit, 1 for a prefix (primary) hit, 0 otherwise. The
// tiers let Resolve prefer the exact match — "joh" uniquely picks Johannes over
// "Johannes’ åpenbaring", which only shares the prefix.
func bookMatchTier(b bibleBook, q string) int {
	if q == b.normShort || q == b.normID || q == b.normLong {
		return 2
	}
	if bookPrimaryMatch(b, q) {
		return 1
	}
	return 0
}

// bookMatches is the looser search predicate: a primary match, or any word of
// the display name (so "genesis" hits "1 Mosebok Genesis"). All match forms are
// precomputed in getBooks.
func bookMatches(b bibleBook, q string) bool {
	if q == "" {
		return true
	}
	if bookPrimaryMatch(b, q) {
		return true
	}
	for _, word := range b.normWords {
		if strings.HasPrefix(word, q) {
			return true
		}
	}
	return false
}

// parsedRef is a bible reference split into its book part and numeric location.
type parsedRef struct {
	bookPart  string
	chapter   int
	verseFrom int
	verseTo   int
}

func parseBibleRef(normalizedQuery string) parsedRef {
	if m := bibleRefRe.FindStringSubmatch(normalizedQuery); m != nil {
		p := parsedRef{bookPart: strings.TrimSpace(m[1])}
		p.chapter, _ = strconv.Atoi(m[2])
		if m[3] != "" {
			p.verseFrom, _ = strconv.Atoi(m[3])
		}
		if m[4] != "" {
			p.verseTo, _ = strconv.Atoi(m[4])
		}
		return p
	}
	// No trailing number — the whole query is a book-name prefix.
	return parsedRef{bookPart: normalizedQuery}
}

// Search turns a free-text reference into canonical suggestions. Examples:
// "joh 3:16", "1 kor 13:4-7", "salme 23", "genesis", "jn 3/16".
//
// NOTE: chapter/verse numbers aren't range-validated here (the book list
// doesn't carry chapter counts); an out-of-range reference simply won't resolve
// when fetched. Book identity, however, is validated against the real list.
func (c *BibleClient) Search(ctx context.Context, query string, limit int) ([]*apiv1.Entity, error) {
	q := normalizeBibleQuery(query)
	if q == "" {
		return nil, nil
	}

	books, err := c.getBooks(ctx)
	if err != nil {
		return nil, err
	}

	ref := parseBibleRef(q)
	results := make([]*apiv1.Entity, 0, limit)
	for _, b := range books {
		if !bookMatches(b, ref.bookPart) {
			continue
		}
		results = append(results, bibleEntity(b, ref.chapter, ref.verseFrom, ref.verseTo))
		if len(results) >= limit {
			break
		}
	}
	return results, nil
}

// Resolve returns the single canonical entity a free-text reference maps to, or
// nil when it can't be resolved confidently. Confidence requires a chapter (so
// a bare book name won't auto-link a verse marker) and exactly one book at the
// strongest match tier (so an ambiguous prefix like "jo 3:16" is left for
// manual review, while an exact "joh 3:16" resolves).
func (c *BibleClient) Resolve(ctx context.Context, text string) (*apiv1.Entity, error) {
	q := normalizeBibleQuery(text)
	if q == "" {
		return nil, nil
	}

	books, err := c.getBooks(ctx)
	if err != nil {
		return nil, err
	}

	ref := parseBibleRef(q)
	if ref.chapter == 0 {
		return nil, nil
	}

	var match *bibleBook
	bestTier, tieCount := 0, 0
	for i := range books {
		tier := bookMatchTier(books[i], ref.bookPart)
		if tier == 0 {
			continue
		}
		if tier > bestTier {
			bestTier, tieCount, match = tier, 1, &books[i]
		} else if tier == bestTier {
			tieCount++
		}
	}
	if match == nil || tieCount != 1 {
		return nil, nil // no match, or ambiguous at the strongest tier
	}
	return bibleEntity(*match, ref.chapter, ref.verseFrom, ref.verseTo), nil
}

// bibleEntity builds the autocomplete entry. `label` is human display text
// (":" separator, translation name); `id` is the canonical machine ref ("/"
// separator, canonical book code) stored as Marker.entity_id.
func bibleEntity(b bibleBook, chapter, verseFrom, verseTo int) *apiv1.Entity {
	var id, label string
	switch {
	case chapter == 0:
		id = b.ID
		label = b.LongName
	case verseFrom == 0:
		id = fmt.Sprintf("%s %d", b.ID, chapter)
		label = fmt.Sprintf("%s %d", b.LongName, chapter)
	case verseTo > verseFrom:
		id = fmt.Sprintf("%s %d/%d-%d", b.ID, chapter, verseFrom, verseTo)
		label = fmt.Sprintf("%s %d:%d-%d", b.LongName, chapter, verseFrom, verseTo)
	default:
		id = fmt.Sprintf("%s %d/%d", b.ID, chapter, verseFrom)
		label = fmt.Sprintf("%s %d:%d", b.LongName, chapter, verseFrom)
	}
	return &apiv1.Entity{
		Id:     id,
		Source: "bible",
		Label:  label,
		Detail: id, // show the canonical ref as secondary text
	}
}
