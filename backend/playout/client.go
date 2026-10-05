package playout

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/go-resty/resty/v2"
)

const DefaultBaseURL = "https://api.playout.studio"

const (
	ContentTypeScripture = "SCRIPTURE"
	ContentTypeSpeaker   = "SPEAKER"
	ContentTypeSong      = "SONG"
	ContentTypeTwoLines  = "TWOLINES"
	// ContentTypeSlide is reserved for screen/slide changes.
	// The API accepts it as a filter but does not currently emit it
	ContentTypeSlide = "SLIDE"
)

type Manifest struct {
	EventID          string          `json:"eventId"`
	EventStart       *time.Time      `json:"eventStart"`
	EventStartSource *string         `json:"eventStartSource"`
	Entries          []ManifestEntry `json:"manifest"`
}

type ManifestEntry struct {
	Type      string          `json:"type"`
	Label     string          `json:"label"`
	Timestamp time.Time       `json:"timestamp"`
	Data      json.RawMessage `json:"data"`
}

// TwoLinesData is the Data payload of a TWOLINES manifest entry:
// An information or banner text shown on screen
type TwoLinesData struct {
	Text string `json:"text"`
}

// Event is one entry from the Event Discovery API. Date and ProductionUnit are
// nullable in the API and surface as empty strings here when absent.
type Event struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Date           string `json:"date"`
	Status         string `json:"status"`
	ProductionUnit string `json:"productionUnit"`
}

type eventsResponse struct {
	Data []Event `json:"data"`
}

type Client struct {
	baseURL  string
	apiKey   string
	tenantID string
	rest     *resty.Client
}

// Retry tuning; variables so tests can shorten the waits.
var (
	retryCount       = 2
	retryWaitTime    = 500 * time.Millisecond
	retryMaxWaitTime = 2 * time.Second
)

func NewClient(baseURL, apiKey, tenantID string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	rest := resty.New().
		SetTimeout(30 * time.Second).
		SetRetryCount(retryCount).
		SetRetryWaitTime(retryWaitTime).
		SetRetryMaxWaitTime(retryMaxWaitTime).
		// Replaces resty's default, so transport errors are retried here too.
		AddRetryCondition(func(resp *resty.Response, err error) bool {
			if err != nil {
				return true
			}
			return resp.StatusCode() == http.StatusTooManyRequests || resp.StatusCode() >= http.StatusInternalServerError
		}).
		SetDisableWarn(true)
	return &Client{
		baseURL:  strings.TrimRight(baseURL, "/"),
		apiKey:   apiKey,
		tenantID: tenantID,
		rest:     rest,
	}
}

func (c *Client) configured() bool {
	return strings.TrimSpace(c.baseURL) != "" && strings.TrimSpace(c.apiKey) != "" && strings.TrimSpace(c.tenantID) != ""
}

// request starts an authenticated JSON request.
func (c *Client) request(ctx context.Context) *resty.Request {
	return c.rest.R().
		SetContext(ctx).
		SetHeader("X-Api-Key", c.apiKey).
		SetHeader("Accept", "application/json").
		SetPathParam("tenant", c.tenantID)
}

// get sends req and decodes the JSON body into out. Not SetResult: it skips
// decoding when the Content-Type isn't JSON.
func (c *Client) get(req *resty.Request, path, action, what string, out any) error {
	if _, err := url.Parse(c.baseURL); err != nil {
		return fmt.Errorf("build playout %s URL: %w", what, err)
	}
	resp, err := req.Get(c.baseURL + path)
	if err != nil {
		return fmt.Errorf("%s: %w", action, err)
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("%s: unexpected status %s", action, resp.Status())
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return fmt.Errorf("decode playout %s: %w", what, err)
	}
	return nil
}

// GetManifest returns the timestamped content actions for an event. Types are
// optional; when supplied they are sent as the API's comma-separated filter.
func (c *Client) GetManifest(ctx context.Context, eventID string, types ...string) (*Manifest, error) {
	if !c.configured() {
		return nil, fmt.Errorf("playout client is not configured")
	}
	if strings.TrimSpace(eventID) == "" {
		return nil, fmt.Errorf("playout event id is required")
	}

	req := c.request(ctx).SetPathParam("event", eventID)
	if len(types) > 0 {
		req.SetQueryParam("type", strings.Join(types, ","))
	}
	var manifest Manifest
	if err := c.get(req, "/manifest/{tenant}/{event}", "get playout manifest", "manifest", &manifest); err != nil {
		return nil, err
	}
	return &manifest, nil
}

// eventsPageSize requests the maximum page size the Event Discovery API
// allows, minimizing the number of pages ListEvents has to fetch.
const eventsPageSize = 100

// ListEvents returns every event known to the tenant, fetching all pages.
// There is no server-side name search, so callers filter the returned list
// themselves.
func (c *Client) ListEvents(ctx context.Context) ([]Event, error) {
	if !c.configured() {
		return nil, fmt.Errorf("playout client is not configured")
	}

	var events []Event
	for page := 1; ; page++ {
		req := c.request(ctx).SetQueryParams(map[string]string{
			"page":     strconv.Itoa(page),
			"pageSize": strconv.Itoa(eventsPageSize),
		})
		var body eventsResponse
		if err := c.get(req, "/{tenant}/events", "list playout events", "events", &body); err != nil {
			return nil, err
		}

		events = append(events, body.Data...)
		// A page short of the requested size is the last one. This doesn't
		// rely on the response's totalPages/page fields being accurate.
		if len(body.Data) < eventsPageSize {
			break
		}
	}
	return events, nil
}
