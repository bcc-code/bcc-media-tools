package members

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
)

type Config struct {
	// e.g. https://api.bcc.no
	APIURL string
	// Auth0 tenant, e.g. https://login.bcc.no
	AuthURL      string
	Audience     string
	ClientID     string
	ClientSecret string
}

// Client reads the Core API's persons with an Auth0 client-credentials
// token. The token is fetched on first use and reused until shortly before it
// expires.
type Client struct {
	cfg  Config
	rest *resty.Client

	mu      sync.Mutex
	token   string
	expires time.Time
}

func NewClient(cfg Config) *Client {
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	cfg.AuthURL = strings.TrimRight(cfg.AuthURL, "/")
	return &Client{cfg: cfg, rest: resty.New().SetTimeout(5 * time.Second).SetDisableWarn(true)}
}

const personFields = "personID,displayName,birthDate"

type corePerson struct {
	PersonID    int    `json:"personID"`
	DisplayName string `json:"displayName"`
	// "2006-01-02"
	BirthDate string `json:"birthDate"`
}

func (p corePerson) person() Person {
	out := Person{PersonID: strconv.Itoa(p.PersonID), DisplayName: p.DisplayName}
	if len(p.BirthDate) >= len(time.DateOnly) {
		// zero, unknown, when it doesn't parse
		out.BirthDate, _ = time.Parse(time.DateOnly, p.BirthDate[:len(time.DateOnly)])
	}
	return out
}

// Search uses the Core API's fuzzy name search, which ignores accents and
// ranks by similarity.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Person, error) {
	var body struct {
		Data []corePerson `json:"data"`
	}
	err := c.get(ctx, "/v2/persons", map[string]string{
		"search": query,
		"fields": personFields,
		"limit":  strconv.Itoa(limit),
	}, &body)
	if err != nil {
		return nil, fmt.Errorf("search members: %w", err)
	}
	out := make([]Person, 0, len(body.Data))
	for _, p := range body.Data {
		out = append(out, p.person())
	}
	return out, nil
}

// Get looks one person up by personID; ErrNotFound when there's no such person.
func (c *Client) Get(ctx context.Context, personID string) (Person, error) {
	var body struct {
		Data corePerson `json:"data"`
	}
	err := c.get(ctx, "/v2/persons/"+url.PathEscape(personID), map[string]string{"fields": personFields}, &body)
	if err != nil {
		return Person{}, fmt.Errorf("get member %s: %w", personID, err)
	}
	return body.Data.person(), nil
}

func (c *Client) get(ctx context.Context, path string, params map[string]string, out any) error {
	token, err := c.accessToken(ctx)
	if err != nil {
		return err
	}
	resp, err := c.rest.R().
		SetContext(ctx).
		SetAuthToken(token).
		SetHeader("Accept", "application/json").
		SetQueryParams(params).
		Get(c.cfg.APIURL + path)
	if err != nil {
		return err
	}
	switch resp.StatusCode() {
	case http.StatusUnauthorized:
		// revoked or rotated: fetch a new one next time
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
	case http.StatusNotFound:
		return ErrNotFound
	}
	if !resp.IsSuccess() {
		return fmt.Errorf("unexpected status %s", resp.Status())
	}
	if err := json.Unmarshal(resp.Body(), out); err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	return nil
}

type tokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int    `json:"expires_in"`
}

// accessToken asks for no scope: Auth0 then grants every scope the client
// has, where asking for one it lacks fails the request.
func (c *Client) accessToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.token != "" && time.Now().Before(c.expires) {
		return c.token, nil
	}
	resp, err := c.rest.R().
		SetContext(ctx).
		SetBody(map[string]string{
			"client_id":     c.cfg.ClientID,
			"client_secret": c.cfg.ClientSecret,
			"audience":      c.cfg.Audience,
			"grant_type":    "client_credentials",
		}).
		Post(c.cfg.AuthURL + "/oauth/token")
	if err != nil {
		return "", fmt.Errorf("members token: %w", err)
	}
	var body tokenResponse
	if !resp.IsSuccess() || json.Unmarshal(resp.Body(), &body) != nil || body.AccessToken == "" {
		return "", fmt.Errorf("members token: unexpected response %s: %s", resp.Status(), resp.String())
	}
	c.token = body.AccessToken
	c.expires = time.Now().Add(time.Duration(body.ExpiresIn)*time.Second - time.Minute)
	return c.token, nil
}
