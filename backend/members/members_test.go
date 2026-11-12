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
)

// fakeCoreAPI serves Auth0's token endpoint and the Core API's persons search.
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
			want := url.Values{"search": {"ola nordmann"}, "fields": {"personID,displayName"}, "limit": {"5"}}
			if got := r.URL.Query(); !reflect.DeepEqual(got, want) {
				t.Errorf("query = %v, want %v", got, want)
			}
			w.WriteHeader(personsStatus)
			io.WriteString(w, `{"data":[{"displayName":"Ola Nordmann","personID":90001},{"displayName":"Ola Nordvik","personID":90003}],"meta":{}}`)
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
		want := []Person{{"90001", "Ola Nordmann"}, {"90003", "Ola Nordvik"}}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("Search = %v, want %v", got, want)
		}
	}
	if n := tokens.Load(); n != 1 {
		t.Errorf("fetched the token %d times, want once", n)
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
	for _, p := range got {
		if p.PersonID == "" || p.PersonID == "0" || p.DisplayName == "" {
			t.Errorf("incomplete person %+v", p)
		}
	}
}
