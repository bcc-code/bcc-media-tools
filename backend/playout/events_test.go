package playout

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestListEventsPagination(t *testing.T) {
	for _, tt := range []struct {
		name      string
		count     int
		wantPages int
	}{
		{"empty", 0, 1},
		{"short first page", 2, 1},
		{"exactly one full page", 100, 2},
		{"multiple pages with short final page", 205, 3},
		{"multiple full pages and empty final page", 200, 3},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var want []Event
			for i := 0; i < tt.count; i++ {
				want = append(want, Event{
					ID:             fmt.Sprintf("event-%d", i),
					Name:           fmt.Sprintf("Meeting %d", i),
					Date:           "2026-07-26",
					Status:         "published",
					ProductionUnit: "Oslo",
				})
			}
			pages := make(chan int, tt.wantPages+1)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				checkRequest(t, r, "/api/tenant%2Fone%20%3F%23%25/events")
				page := len(pages) + 1
				if page > tt.wantPages {
					t.Error("client requested a page after the final short page")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
				pages <- page
				wantQuery := url.Values{"page": {fmt.Sprint(page)}, "pageSize": {"100"}}
				if got := r.URL.Query(); !reflect.DeepEqual(got, wantQuery) {
					t.Errorf("query = %v, want %v", got, wantQuery)
				}
				start := (page - 1) * 100
				end := min(start+100, len(want))
				// Incorrect metadata must not cause early termination or an
				// extra request after a short page.
				json.NewEncoder(w).Encode(map[string]any{
					"data":       want[start:end],
					"page":       99,
					"totalPages": 1,
				})
			}))
			defer server.Close()
			client := NewClient(server.URL+"/api/", "test-key", "tenant/one ?#%")
			transport := server.Client().Transport
			var lastBody *trackedBody
			client.httpClient = &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
				if lastBody != nil && !lastBody.closed {
					t.Error("previous page's body was not closed before fetching the next page")
				}
				resp, err := transport.RoundTrip(r)
				if err == nil {
					lastBody = &trackedBody{ReadCloser: resp.Body}
					resp.Body = lastBody
				}
				return resp, err
			})}
			got, err := client.ListEvents(context.Background())
			if err != nil {
				t.Fatalf("ListEvents: %v", err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("events = %+v, want %+v", got, want)
			}
			if len(pages) != tt.wantPages {
				t.Errorf("requested %d pages, want %d", len(pages), tt.wantPages)
			}
			if lastBody == nil || !lastBody.closed {
				t.Error("final page's body was not closed")
			}
		})
	}
}

func TestListEventsNullableFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"data":[
			{"id":"event-1","name":"Undated","date":null,"status":"draft","productionUnit":null},
			{"id":"event-2","name":"Unscheduled","status":"draft"}
		]}`)
	}))
	defer server.Close()
	got, err := NewClient(server.URL, "test-key", "tenant").ListEvents(context.Background())
	if err != nil {
		t.Fatalf("ListEvents: %v", err)
	}
	want := []Event{
		{ID: "event-1", Name: "Undated", Status: "draft"},
		{ID: "event-2", Name: "Unscheduled", Status: "draft"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("events = %+v, want %+v", got, want)
	}
}

func TestListEventsLaterPageError(t *testing.T) {
	for _, tt := range []struct {
		name   string
		status int
		body   string
		want   string
	}{
		{"HTTP error", http.StatusBadGateway, "unavailable", "unexpected status 502 Bad Gateway"},
		{"decode error", http.StatusOK, `{`, "decode playout events"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			pages := make(chan string, 3)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				page := r.URL.Query().Get("page")
				select {
				case pages <- page:
				default:
					t.Error("unexpected extra request")
				}
				if page == "1" && len(pages) == 1 {
					events := make([]Event, 100)
					for i := range events {
						events[i] = Event{ID: fmt.Sprintf("event-%d", i)}
					}
					json.NewEncoder(w).Encode(map[string]any{"data": events})
					return
				}
				w.WriteHeader(tt.status)
				io.WriteString(w, tt.body)
			}))
			defer server.Close()
			got, err := NewClient(server.URL, "test-key", "tenant").ListEvents(context.Background())
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.want)
			}
			if got != nil {
				t.Errorf("returned %d partial events after a page failed", len(got))
			}
			close(pages)
			var requested []string
			for page := range pages {
				requested = append(requested, page)
			}
			if !reflect.DeepEqual(requested, []string{"1", "2"}) {
				t.Errorf("requested pages = %v, want [1 2]", requested)
			}
		})
	}
}
