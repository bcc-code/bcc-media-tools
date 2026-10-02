package bible

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultBaseURL is the public BCC bible server.
const DefaultBaseURL = "https://bibleapi.bcc.media/v1"

// ErrNotConfigured is returned when no bible ID is configured, so the server
// cannot know which translation to read.
var ErrNotConfigured = errors.New("bible: no bible ID configured")

// ErrWholeChapter is returned for a reference that names a chapter but no
// verses: the bible server only serves verse ranges.
var ErrWholeChapter = errors.New("bible: reference has no verses")

// Verse is a single verse of text.
type Verse struct {
	Number int32
	Text   string
}

// Client reads verse text from the BCC bible server.
// See https://bcc-code.gitbook.io/bible-server.
type Client struct {
	baseURL string
	bibleID string
	http    *http.Client
}

// NewClient builds a client. An empty baseURL falls back to DefaultBaseURL.
// The bible server takes no authentication.
func NewClient(baseURL, bibleID string) *Client {
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		bibleID: bibleID,
		http:    &http.Client{Timeout: 10 * time.Second},
	}
}

// Verses fetches the text for every verse run in the reference, in order.
func (c *Client) Verses(ctx context.Context, ref Reference) ([]Verse, error) {
	if c.bibleID == "" {
		return nil, ErrNotConfigured
	}
	if len(ref.Ranges) == 0 {
		return nil, ErrWholeChapter
	}

	var out []Verse
	for _, r := range ref.Ranges {
		verses, err := c.fetchRange(ctx, ref.Book, ref.Chapter, r)
		if err != nil {
			return nil, err
		}
		out = append(out, verses...)
	}
	return out, nil
}

func (c *Client) fetchRange(ctx context.Context, book string, chapter int, r VerseRange) ([]Verse, error) {
	endpoint := fmt.Sprintf("%s/%s/%s/%d/%d/%d",
		c.baseURL,
		url.PathEscape(c.bibleID),
		url.PathEscape(book),
		chapter, r.From, r.To,
	)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("bible: build request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	res, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bible: get %s: %w", endpoint, err)
	}
	defer func() { _ = res.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("bible: read response: %w", err)
	}
	if res.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("bible: %s returned %d: %s", endpoint, res.StatusCode, strings.TrimSpace(string(body)))
	}
	return decodeVerses(body)
}

// versesResponse is the bible server's verse payload, as served by
// src/http.go in bcc-code/bible-server (protobuf JSON, snake_case).
type versesResponse struct {
	BibleID string `json:"bible_id"`
	BookID  string `json:"book_id"`
	Chapter uint32 `json:"chapter"`
	Verses  []struct {
		Number int32  `json:"number"`
		Text   string `json:"text"`
	} `json:"verses"`
}

func decodeVerses(body []byte) ([]Verse, error) {
	var res versesResponse
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("bible: decode verses: %w", err)
	}

	out := make([]Verse, 0, len(res.Verses))
	for _, v := range res.Verses {
		if text := strings.TrimSpace(v.Text); text != "" {
			out = append(out, Verse{Number: v.Number, Text: text})
		}
	}
	return out, nil
}

// Translation labels the configured bible for the UI, e.g. "nb-1930" → "NB 1930".
func (c *Client) Translation() string {
	parts := strings.Split(c.bibleID, "-")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	parts[0] = strings.ToUpper(parts[0])
	return strings.Join(parts, " ")
}

// FormatRange renders a verse list as "18, 23–24" for display next to the text.
func FormatRange(ranges []VerseRange) string {
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		if r.From == r.To {
			parts = append(parts, strconv.Itoa(r.From))
			continue
		}
		parts = append(parts, strconv.Itoa(r.From)+"–"+strconv.Itoa(r.To))
	}
	return strings.Join(parts, ", ")
}
