package playout

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.playout.studio"

const (
	ContentTypeScripture = "SCRIPTURE"
	ContentTypeSpeaker   = "SPEAKER"
	ContentTypeSong      = "SONG"
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
	Data       []Event `json:"data"`
	Page       int     `json:"page"`
	TotalPages int     `json:"totalPages"`
}

type Client struct {
	baseURL    string
	apiKey     string
	tenantID   string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey, tenantID string) *Client {
	if strings.TrimSpace(baseURL) == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		tenantID:   tenantID,
		httpClient: http.DefaultClient,
	}
}

// GetManifest returns the timestamped content actions for an event. Types are
// optional; when supplied they are sent as the API's comma-separated filter.
func (c *Client) GetManifest(ctx context.Context, eventID string, types ...string) (*Manifest, error) {
	if strings.TrimSpace(c.baseURL) == "" || strings.TrimSpace(c.apiKey) == "" || strings.TrimSpace(c.tenantID) == "" {
		return nil, fmt.Errorf("playout client is not configured")
	}
	if strings.TrimSpace(eventID) == "" {
		return nil, fmt.Errorf("playout event id is required")
	}

	endpoint := fmt.Sprintf("%s/manifest/%s/%s", c.baseURL, url.PathEscape(c.tenantID), url.PathEscape(eventID))
	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("build playout manifest URL: %w", err)
	}
	if len(types) > 0 {
		query := u.Query()
		query.Set("type", strings.Join(types, ","))
		u.RawQuery = query.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, fmt.Errorf("create playout manifest request: %w", err)
	}
	req.Header.Set("X-Api-Key", c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("get playout manifest: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, fmt.Errorf("get playout manifest: unexpected status %s", resp.Status)
	}

	var manifest Manifest
	if err := json.NewDecoder(resp.Body).Decode(&manifest); err != nil {
		return nil, fmt.Errorf("decode playout manifest: %w", err)
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
	if strings.TrimSpace(c.baseURL) == "" || strings.TrimSpace(c.apiKey) == "" || strings.TrimSpace(c.tenantID) == "" {
		return nil, fmt.Errorf("playout client is not configured")
	}

	var events []Event
	for page := 1; ; page++ {
		endpoint := fmt.Sprintf("%s/%s/events", c.baseURL, url.PathEscape(c.tenantID))
		u, err := url.Parse(endpoint)
		if err != nil {
			return nil, fmt.Errorf("build playout events URL: %w", err)
		}
		query := u.Query()
		query.Set("page", fmt.Sprintf("%d", page))
		query.Set("pageSize", fmt.Sprintf("%d", eventsPageSize))
		u.RawQuery = query.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
		if err != nil {
			return nil, fmt.Errorf("create playout events request: %w", err)
		}
		req.Header.Set("X-Api-Key", c.apiKey)
		req.Header.Set("Accept", "application/json")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("list playout events: %w", err)
		}
		var body eventsResponse
		err = json.NewDecoder(resp.Body).Decode(&body)
		status := resp.StatusCode
		resp.Body.Close()
		if status < http.StatusOK || status >= http.StatusMultipleChoices {
			return nil, fmt.Errorf("list playout events: unexpected status %d", status)
		}
		if err != nil {
			return nil, fmt.Errorf("decode playout events: %w", err)
		}

		events = append(events, body.Data...)
		if page >= body.TotalPages || len(body.Data) == 0 {
			break
		}
	}
	return events, nil
}
