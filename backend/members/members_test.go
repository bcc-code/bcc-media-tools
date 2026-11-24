package members

import (
	"cmp"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"reflect"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
)

// fakeCoreAPI serves Auth0's token endpoint, the Core API's persons search and
// its churches, counting the requests for those.
func fakeCoreAPI(t *testing.T, tokens, orgs *atomic.Int32, personsStatus int) *httptest.Server {
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
			want := url.Values{"search": {"ola nordmann"}, "fields": {personFields}, "limit": {"5"}}
			if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
				t.Errorf("query = %v, want %v", got, want)
			}
			w.WriteHeader(personsStatus)
			// Ola Nordmann's church is c3: c1 isn't active, c2 has ended and
			// org isn't a church.
			io.WriteString(w, `{"data":[{"displayName":"Ola Nordmann","uid":"00000000-0000-4000-8000-000000090001","birthDate":"2004-03-01","affiliations":[
				{"orgUid":"c1","isActive":false},
				{"orgUid":"c2","isActive":true,"validTo":"2020-01-01T00:00:00Z"},
				{"orgUid":"org","isActive":true},
				{"orgUid":"c3","isActive":true,"validFrom":"2010-01-01T00:00:00Z","validTo":null}
			]},{"displayName":"Ola Nordvik","uid":"00000000-0000-4000-8000-000000090003","birthDate":null}],"meta":{}}`)
		case "/v2/orgs":
			orgs.Add(1)
			want := url.Values{"filter": {`{"type":{"_eq":"Church"}}`}, "fields": {"uid,districtName"}, "limit": {"999"}}
			if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
				t.Errorf("orgs query = %v, want %v", got, want)
			}
			io.WriteString(w, `{"data":[{"uid":"c1","districtName":"Church One"},{"uid":"c2","districtName":"Church Two"},{"uid":"c3","districtName":"Church Three"}]}`)
		default:
			t.Errorf("unexpected request %s", r.URL.Path)
			http.NotFound(w, r)
		}
	}))
}

func TestClientSearch(t *testing.T) {
	var tokens, orgs atomic.Int32
	server := fakeCoreAPI(t, &tokens, &orgs, http.StatusOK)
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL + "/", AuthURL: server.URL, Audience: "api.bcc.no", ClientID: "id", ClientSecret: "secret"})

	for range 2 {
		got, err := c.Search(context.Background(), "ola nordmann", 5)
		if err != nil {
			t.Fatal(err)
		}
		want := []Person{
			{UID: "00000000-0000-4000-8000-000000090001", DisplayName: "Ola Nordmann", BirthDate: time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC), Church: "Church Three"},
			{UID: "00000000-0000-4000-8000-000000090003", DisplayName: "Ola Nordvik"},
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Search = %v, want %v", got, want)
		}
	}
	if n := tokens.Load(); n != 1 {
		t.Errorf("fetched the token %d times, want once", n)
	}
	if n := orgs.Load(); n != 1 {
		t.Errorf("read the churches %d times, want once", n)
	}
}

func TestBirthDateParsing(t *testing.T) {
	for in, want := range map[string]time.Time{
		"2004-03-01":           time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC),
		"2004-03-01T00:00:00Z": time.Date(2004, 3, 1, 0, 0, 0, 0, time.UTC),
		"":                     {},
		"01.03.2004":           {},
	} {
		if got := (corePerson{BirthDate: in}).person(nil).BirthDate; !got.Equal(want) {
			t.Errorf("birthDate %q = %v, want %v", in, got, want)
		}
	}
}

func TestClientSearchUnauthorizedRefetchesToken(t *testing.T) {
	var tokens, orgs atomic.Int32
	server := fakeCoreAPI(t, &tokens, &orgs, http.StatusUnauthorized)
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

func TestClientGetMany(t *testing.T) {
	var filters []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/token":
			io.WriteString(w, `{"access_token":"tok","expires_in":86400}`)
			return
		case "/v2/orgs":
			io.WriteString(w, `{"data":[]}`)
			return
		}
		q := r.URL.Query()
		if q.Get("fields") != personFields || q.Get("limit") != "999" {
			t.Errorf("query = %v", q)
		}
		filters = append(filters, q.Get("filter"))
		io.WriteString(w, `{"data":[{"displayName":"Ola Nordmann","uid":"00000000-0000-4000-8000-000000090001"}]}`)
	}))
	defer server.Close()
	c := NewClient(Config{APIURL: server.URL, AuthURL: server.URL})

	got, err := c.GetMany(context.Background(), []string{"00000000-0000-4000-8000-000000090001", "90001", "00000000-0000-4000-8000-000000090002"})
	if err != nil {
		t.Fatal(err)
	}
	if want := []Person{{UID: "00000000-0000-4000-8000-000000090001", DisplayName: "Ola Nordmann"}}; !reflect.DeepEqual(got, want) {
		t.Errorf("GetMany = %v, want %v", got, want)
	}
	if want := []string{`{"uid":{"_in":["00000000-0000-4000-8000-000000090001","00000000-0000-4000-8000-000000090002"]}}`}; !reflect.DeepEqual(filters, want) {
		t.Errorf("filters = %v, want %v (not a UUID left out, case normalized)", filters, want)
	}

	filters = nil
	ids := make([]string, 801)
	for i := range ids {
		ids[i] = uuid.NewString()
	}
	if _, err := c.GetMany(context.Background(), ids); err != nil {
		t.Fatal(err)
	}
	if len(filters) != 2 {
		t.Errorf("%d requests for 801 IDs, want 2", len(filters))
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
	// Only IDs in messages: no names, birth dates or churches in the output.
	dated, churched := 0, 0
	for _, p := range got {
		if uuid.Validate(p.UID) != nil || p.DisplayName == "" {
			t.Errorf("incomplete person %q", p.UID)
		}
		if !p.BirthDate.IsZero() {
			dated++
		}
		if p.Church != "" {
			churched++
		}
	}
	if dated == 0 {
		t.Error("no birth date parsed")
	}
	if churched == 0 {
		t.Error("no church found")
	}
	many, err := c.GetMany(context.Background(), []string{got[0].UID})
	if err != nil {
		t.Fatal(err)
	}
	if len(many) != 1 || many[0].UID != got[0].UID || !many[0].BirthDate.Equal(got[0].BirthDate) || many[0].Church != got[0].Church {
		t.Errorf("GetMany(%s) doesn't match the search result", got[0].UID)
	}
}
