package members

import (
	"cmp"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"
)

// fakeCoreAPI serves Auth0's token endpoint and the Core API's persons: search,
// and 90001 by ID.
func fakeCoreAPI(t *testing.T, tokens *atomic.Int32, personsStatus int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			tokens.Add(1)
			var body map[string]string
			b, _ := io.ReadAll(r.Body)
			_ = json.Unmarshal(b, &body)
			want := map[string]string{"client_id": "id", "client_secret": "secret", "audience": "api.bcc.no", "grant_type": "client_credentials"}
			if !reflect.DeepEqual(body, want) {
				t.Errorf("token request = %v, want %v (no scope)", body, want)
			}
			io.WriteString(w, `{"access_token":"tok","expires_in":86400,"token_type":"Bearer"}`)
		case "/v2/persons":
			if got := r.Header.Get("Authorization"); got != "Bearer tok" {
				t.Errorf("Authorization = %q", got)
			}
			want := url.Values{"search": {"ola nordmann"}, "fields": {"personID,displayName,birthDate"}, "limit": {"5"}}
			if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
				t.Errorf("query = %v, want %v", got, want)
			}
			w.WriteHeader(personsStatus)
			io.WriteString(w, `{"data":[{"displayName":"Ola Nordmann","personID":90001,"birthDate":"2004-03-01"},{"displayName":"Ola Nordvik","personID":90003,"birthDate":null}],"meta":{}}`)
		case "/v2/persons/90001":
			if got := r.URL.Query().Get("fields"); got != "personID,displayName,birthDate" {
				t.Errorf("fields = %q", got)
			}
			io.WriteString(w, `{"data":{"displayName":"Ola Nordmann","personID":90001,"birthDate":"2004-03-01"}}`)
		case "/v2/persons/90009":
			w.WriteHeader(http.StatusNotFound)
			io.WriteString(w, `{"error":{"code":"not-found","message":"not found"}}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestClientSearch(t *testing.T) {
	var tokens atomic.Int32
	server := fakeCoreAPI(t, &tokens, http.StatusOK)
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL + "/", AuthURL: server.URL, Audience: "api.bcc.no", ClientID: "id", ClientSecret: "secret"})

	for range 2 {
		got, err := c.Search(context.Background(), "ola nordmann", 5)
		if err != nil {
			t.Fatal(err)
		}
		want := []Person{
			{PersonID: "90001", DisplayName: "Ola Nordmann", BirthDate: time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC)},
			{PersonID: "90003", DisplayName: "Ola Nordvik"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Search = %v, want %v", got, want)
		}
	}
	if n := tokens.Load(); n != 1 {
		t.Errorf("fetched the token %d times, want once", n)
	}
}

func TestClientGet(t *testing.T) {
	var tokens atomic.Int32
	server := fakeCoreAPI(t, &tokens, http.StatusOK)
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL, AuthURL: server.URL, Audience: "api.bcc.no", ClientID: "id", ClientSecret: "secret"})

	got, err := c.Get(context.Background(), "90001")
	if err != nil {
		t.Fatal(err)
	}
	want := Person{PersonID: "90001", DisplayName: "Ola Nordmann", BirthDate: time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC)}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Get = %v, want %v", got, want)
	}
	if _, err := c.Get(context.Background(), "90009"); !errors.Is(err, ErrNotFound) {
		t.Errorf("Get unknown = %v, want ErrNotFound", err)
	}
}

func TestBirthDateParsing(t *testing.T) {
	for in, want := range map[string]time.Time{
		"2004-03-01":           time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC),
		"2004-03-01T00:00:00Z": time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC),
		"":                     {},
		"01.03.2004":           {},
	} {
		if got := (corePerson{BirthDate: in}).person().BirthDate; !got.Equal(want) {
			t.Errorf("birthDate %q = %v, want %v", in, got, want)
		}
	}
}

func TestClientSearchUnauthorizedRefetchesToken(t *testing.T) {
	var tokens atomic.Int32
	server := fakeCoreAPI(t, &tokens, http.StatusUnauthorized)
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL, AuthURL: server.URL, Audience: "api.bcc.no", ClientID: "id", ClientSecret: "secret"})

	for range 2 {
		if _, err := c.Search(context.Background(), "ola nordmann", 5); err == nil {
			t.Error("want an error for 401")
		}
	}
	if n := tokens.Load(); n != 2 {
		t.Errorf("fetched the token %d times, want a new one after a 401", n)
	}
}

func TestClientSearchTokenDenied(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		io.WriteString(w, `{"error":"access_denied"}`)
	}))
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL, AuthURL: server.URL})
	if _, err := c.Search(context.Background(), "ola", 5); err == nil {
		t.Error("want an error when Auth0 denies the token")
	}
}

// Searches the real Core API with the MEMBERS_* env vars, when set.
func TestClientSearchLive(t *testing.T) {
	if os.Getenv("MEMBERS_CLIENT_ID") == "" {
		t.Skip("MEMBERS_CLIENT_ID not set")
	}
	c := NewClient(Config{
		APIURL:       cmp.Or(os.Getenv("MEMBERS_API_URL"), "https://api.bcc.no"),
		AuthURL:      cmp.Or(os.Getenv("MEMBERS_AUTH0_BASE_URL"), "https://login.bcc.no"),
		Audience:     cmp.Or(os.Getenv("MEMBERS_AUDIENCE"), "api.bcc.no"),
		ClientID:     os.Getenv("MEMBERS_CLIENT_ID"),
		ClientSecret: os.Getenv("MEMBERS_CLIENT_SECRET"),
	})
	// a common surname, so the test names no one
	got, err := c.Search(context.Background(), cmp.Or(os.Getenv("MEMBERS_LIVE_QUERY"), "hansen"), 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) == 0 {
		t.Fatal("found no one")
	}
	// Only IDs in messages: no names or birth dates in the output.
	dated := 0
	for _, p := range got {
		if p.PersonID == "" || p.PersonID == "0" || p.DisplayName == "" {
			t.Errorf("incomplete person %q", p.PersonID)
		}
		if !p.BirthDate.IsZero() {
			dated++
		}
	}
	if dated == 0 {
		t.Error("no birth date parsed")
	}
	one, err := c.Get(context.Background(), got[0].PersonID)
	if err != nil {
		t.Fatal(err)
	}
	if one.PersonID != got[0].PersonID || !one.BirthDate.Equal(got[0].BirthDate) {
		t.Errorf("Get(%s) doesn't match the search result", got[0].PersonID)
	}
}
