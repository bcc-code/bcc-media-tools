package playout

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"
	"time"
)

func TestGetManifest(t *testing.T) {
	for _, tt := range []struct {
		name  string
		types []string
		query url.Values
	}{
		{"no filter", nil, url.Values{}},
		{"empty filter", []string{}, url.Values{}},
		{"one type", []string{ContentTypeSpeaker}, url.Values{"type": {"SPEAKER"}}},
		{"multiple types", []string{ContentTypeSpeaker, ContentTypeScripture, ContentTypeSong}, url.Values{"type": {"SPEAKER,SCRIPTURE,SONG"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				checkRequest(t, r, "/api/manifest/tenant%2Fone%20%3F%23%25/event%2Fone%20%3F%23%25")
				if got := r.URL.Query(); !reflect.DeepEqual(got, tt.query) {
					t.Errorf("query = %v, want %v", got, tt.query)
				}
				w.Header().Set("Content-Type", "application/json")
				io.WriteString(w, `{
					"eventId": "event/one ?#%",
					"eventStart": "2026-07-26T12:00:00Z",
					"eventStartSource": "scheduled",
					"manifest": [{
						"type": "SPEAKER",
						"label": "Kåre J. Smith",
						"timestamp": "2026-07-26T12:06:31.123Z",
						"data": {"name":"Kåre J. Smith","id":42}
					}],
					"unknownField": "ignored"
				}`)
			}))
			defer server.Close()
			client := NewClient(server.URL+"/api///", "test-key", "tenant/one ?#%")
			manifest, err := client.GetManifest(context.Background(), "event/one ?#%", tt.types...)
			if err != nil {
				t.Fatalf("GetManifest: %v", err)
			}
			start := time.Date(2026, time.July, 26, 12, 0, 0, 0, time.UTC)
			source := "scheduled"
			want := &Manifest{
				EventID:          "event/one ?#%",
				EventStart:       &start,
				EventStartSource: &source,
				Entries: []ManifestEntry{{
					Type:      ContentTypeSpeaker,
					Label:     "Kåre J. Smith",
					Timestamp: time.Date(2026, time.July, 26, 12, 6, 31, 123000000, time.UTC),
					Data:      json.RawMessage(`{"name":"Kåre J. Smith","id":42}`),
				}},
			}
			if !reflect.DeepEqual(manifest, want) {
				t.Errorf("manifest = %+v, want %+v", manifest, want)
			}
		})
	}
}

func TestGetManifestNullableFields(t *testing.T) {
	for _, body := range []string{
		`{"eventId":"event-1","eventStart":null,"eventStartSource":null,"manifest":[]}`,
		`{"eventId":"event-1","manifest":[]}`,
	} {
		t.Run(body, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				io.WriteString(w, body)
			}))
			defer server.Close()
			manifest, err := NewClient(server.URL, "test-key", "tenant").GetManifest(context.Background(), "event-1")
			if err != nil {
				t.Fatalf("GetManifest: %v", err)
			}
			if manifest.EventID != "event-1" || manifest.EventStart != nil || manifest.EventStartSource != nil || len(manifest.Entries) != 0 {
				t.Errorf("unexpected manifest: %+v", manifest)
			}
		})
	}
}

func TestGetManifestRequiresEventID(t *testing.T) {
	client := NewClient("https://playout.example", "test-key", "tenant")
	client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		t.Error("missing event ID caused an HTTP request")
		return nil, errors.New("unexpected request")
	})}
	for _, eventID := range []string{"", " \t\n"} {
		manifest, err := client.GetManifest(context.Background(), eventID)
		if err == nil || err.Error() != "playout event id is required" || manifest != nil {
			t.Errorf("GetManifest(%q) = (%v, %v), want nil and missing event ID error", eventID, manifest, err)
		}
	}
}
