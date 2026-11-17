package members

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
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

// Client finds the Core API's persons with an Auth0 client-credentials
// token. The token is fetched on first use and reused until shortly before it
// expires.
type Client struct {
	cfg  Config
	rest *resty.Client

	mu      sync.Mutex
	token   string
	expires time.Time

	churchMu    sync.Mutex
	churchNames map[string]string
	churchesAt  time.Time
}

func NewClient(cfg Config) *Client {
	cfg.APIURL = strings.TrimRight(cfg.APIURL, "/")
	cfg.AuthURL = strings.TrimRight(cfg.AuthURL, "/")
	return &Client{cfg: cfg, rest: resty.New().SetTimeout(5 * time.Second).SetDisableWarn(true)}
}

const personFields = "personID,displayName,birthDate,affiliations.orgUid,affiliations.isActive,affiliations.validFrom,affiliations.validTo"

type corePerson struct {
	PersonID    int    `json:"personID"`
	DisplayName string `json:"displayName"`
	// "2006-01-02"
	BirthDate    string `json:"birthDate"`
	Affiliations []struct {
		OrgUID    string     `json:"orgUid"`
		IsActive  bool       `json:"isActive"`
		ValidFrom *time.Time `json:"validFrom"`
		ValidTo   *time.Time `json:"validTo"`
	} `json:"affiliations"`
}

// person: churches maps church orgs' uids to their names.
func (p corePerson) person(churches map[string]string) Person {
	out := Person{PersonID: strconv.Itoa(p.PersonID), DisplayName: p.DisplayName}
	if len(p.BirthDate) >= len(time.DateOnly) {
		// zero, unknown, when it doesn't parse
		out.BirthDate, _ = time.Parse(time.DateOnly, p.BirthDate[:len(time.DateOnly)])
	}
	now := time.Now()
	for _, a := range p.Affiliations {
		active := a.IsActive && (a.ValidFrom == nil || !now.Before(*a.ValidFrom)) && (a.ValidTo == nil || now.Before(*a.ValidTo))
		if name := churches[a.OrgUID]; active && name != "" {
			out.Church = name
			break
		}
	}
	return out
}

// Search uses the Core API's fuzzy name search, which ignores accents and
// ranks by similarity.
func (c *Client) Search(ctx context.Context, query string, limit int) ([]Person, error) {
	people, err := c.persons(ctx, map[string]string{"search": query, "limit": strconv.Itoa(limit)})
	if err != nil {
		return nil, fmt.Errorf("search members: %w", err)
	}
	return people, nil
}

// GetMany looks people up by personID, 800 per request as Wayfarer does. IDs
// Members doesn't know, or that aren't numbers, are left out.
func (c *Client) GetMany(ctx context.Context, ids []string) ([]Person, error) {
	var nums []int
	for _, id := range ids {
		if n, err := strconv.Atoi(id); err == nil {
			nums = append(nums, n)
		}
	}
	var out []Person
	for chunk := range slices.Chunk(nums, 800) {
		filter, err := json.Marshal(map[string]any{"personID": map[string]any{"_in": chunk}})
		if err != nil {
			return nil, err
		}
		people, err := c.persons(ctx, map[string]string{"filter": string(filter), "limit": "999"})
		if err != nil {
			return nil, fmt.Errorf("get members: %w", err)
		}
		out = append(out, people...)
	}
	return out, nil
}

// persons lists /v2/persons with the given query parameters.
func (c *Client) persons(ctx context.Context, params map[string]string) ([]Person, error) {
	params["fields"] = personFields
	var body struct {
		Data []corePerson `json:"data"`
	}
	if err := c.get(ctx, "/v2/persons", params, &body); err != nil {
		return nil, err
	}
	churches := c.churches(ctx)
	out := make([]Person, 0, len(body.Data))
	for _, p := range body.Data {
		out = append(out, p.person(churches))
	}
	return out, nil
}

// churches maps church orgs' uids to their names, read once a day. When that
// fails the last ones are used, so people still come back, if without one.
func (c *Client) churches(ctx context.Context) map[string]string {
	c.churchMu.Lock()
	defer c.churchMu.Unlock()
	if time.Since(c.churchesAt) < 24*time.Hour {
		return c.churchNames
	}
	var body struct {
		Data []struct {
			UID  string `json:"uid"`
			Name string `json:"districtName"`
		} `json:"data"`
	}
	err := c.get(ctx, "/v2/orgs", map[string]string{
		"filter": `{"type":{"_eq":"Church"}}`,
		"fields": "uid,districtName",
		"limit":  "999",
	}, &body)
	if err != nil {
		return c.churchNames
	}
	c.churchNames = map[string]string{}
	for _, o := range body.Data {
		c.churchNames[o.UID] = o.Name
	}
	c.churchesAt = time.Now()
	return c.churchNames
}

// get reads path into out.
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
	if resp.StatusCode() == http.StatusUnauthorized {
		// revoked or rotated: fetch a new one next time
		c.mu.Lock()
		c.token = ""
		c.mu.Unlock()
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
