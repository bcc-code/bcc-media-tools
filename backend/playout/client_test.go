package playout

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	for _, tt := range []struct {
		name    string
		baseURL string
		wantURL string
	}{
		{"default", "", DefaultBaseURL},
		{"whitespace uses default", " \t\n", DefaultBaseURL},
		{"custom URL", "https://playout.example/api", "https://playout.example/api"},
		{"trailing slashes", "https://playout.example/api///", "https://playout.example/api"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.baseURL, "test-key", "test-tenant")
			if client.baseURL != tt.wantURL {
				t.Errorf("base URL = %q, want %q", client.baseURL, tt.wantURL)
			}
			if client.apiKey != "test-key" || client.tenantID != "test-tenant" {
				t.Error("client did not retain its API key and tenant ID")
			}
			if client.httpClient != http.DefaultClient {
				t.Error("client should use http.DefaultClient")
			}
		})
	}
}

// Exercise the shared HTTP contract through both public methods.
var clientMethods = []struct {
	name    string
	request func(context.Context, *Client) error
}{
	{"GetManifest", func(ctx context.Context, client *Client) error {
		_, err := client.GetManifest(ctx, "event-1")
		return err
	}},
	{"ListEvents", func(ctx context.Context, client *Client) error {
		_, err := client.ListEvents(ctx)
		return err
	}},
}

func TestClientRequiresConfiguration(t *testing.T) {
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			for _, field := range []string{"base URL", "API key", "tenant ID"} {
				for _, value := range []string{"", " \t\n"} {
					t.Run(field+"="+value, func(t *testing.T) {
						client := NewClient("https://playout.example", "test-key", "tenant")
						switch field {
						case "base URL":
							client.baseURL = value
						case "API key":
							client.apiKey = value
						case "tenant ID":
							client.tenantID = value
						}
						client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
							t.Error("unconfigured client made an HTTP request")
							return nil, errors.New("unexpected request")
						})}
						err := method.request(context.Background(), client)
						if err == nil || err.Error() != "playout client is not configured" {
							t.Fatalf("error = %v, want configuration error", err)
						}
					})
				}
			}
		})
	}
}

func TestClientResponseErrors(t *testing.T) {
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			for _, tt := range []struct {
				name   string
				status int
				body   string
				want   string
			}{
				{"redirect without location", http.StatusMultipleChoices, `{}`, "unexpected status 300 Multiple Choices"},
				{"unauthorized", http.StatusUnauthorized, `{}`, "unexpected status 401 Unauthorized"},
				{"forbidden", http.StatusForbidden, `{}`, "unexpected status 403 Forbidden"},
				{"not found", http.StatusNotFound, `{}`, "unexpected status 404 Not Found"},
				{"rate limited", http.StatusTooManyRequests, `{}`, "unexpected status 429 Too Many Requests"},
				{"server error", http.StatusInternalServerError, `not JSON`, "unexpected status 500 Internal Server Error"},
				{"malformed JSON", http.StatusOK, `{`, "decode playout"},
				{"wrong JSON shape", http.StatusOK, `[]`, "decode playout"},
				{"empty body", http.StatusOK, "", "decode playout"},
			} {
				t.Run(tt.name, func(t *testing.T) {
					server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						w.WriteHeader(tt.status)
						io.WriteString(w, tt.body)
					}))
					defer server.Close()
					client := NewClient(server.URL, "test-key", "tenant")
					err := method.request(context.Background(), client)
					if err == nil || !strings.Contains(err.Error(), tt.want) {
						t.Fatalf("error = %v, want it to contain %q", err, tt.want)
					}
				})
			}
		})
	}
}

func TestClientInvalidURL(t *testing.T) {
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			err := method.request(context.Background(), NewClient("http://[invalid", "test-key", "tenant"))
			if err == nil || !strings.Contains(err.Error(), "build playout") {
				t.Fatalf("error = %v, want URL construction error", err)
			}
		})
	}
}

func TestClientTransportError(t *testing.T) {
	wantErr := errors.New("connection failed")
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			client := NewClient("https://playout.example", "test-key", "tenant")
			client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return nil, wantErr
			})}
			if err := method.request(context.Background(), client); !errors.Is(err, wantErr) {
				t.Fatalf("error = %v, want wrapped transport error %v", err, wantErr)
			}
		})
	}
}

func TestClientContextCancellation(t *testing.T) {
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Cancel only after the request arrives, so this exercises an
				// in-flight HTTP request rather than just a canceled context.
				cancel()
				<-r.Context().Done()
			}))
			defer server.Close()
			client := NewClient(server.URL, "test-key", "tenant")
			client.httpClient = &http.Client{Timeout: 5 * time.Second}
			if err := method.request(ctx, client); !errors.Is(err, context.Canceled) {
				t.Fatalf("error = %v, want context.Canceled", err)
			}
		})
	}
}

func TestClientClosesResponseBody(t *testing.T) {
	for _, method := range clientMethods {
		t.Run(method.name, func(t *testing.T) {
			for _, tt := range []struct {
				name   string
				status int
				body   string
			}{
				{"success", http.StatusOK, `{}`},
				{"HTTP error", http.StatusInternalServerError, `{}`},
				{"decode error", http.StatusOK, `{`},
			} {
				t.Run(tt.name, func(t *testing.T) {
					body := &trackedBody{ReadCloser: io.NopCloser(strings.NewReader(tt.body))}
					client := NewClient("https://playout.example", "test-key", "tenant")
					client.httpClient = &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
						return &http.Response{StatusCode: tt.status, Body: body}, nil
					})}
					method.request(context.Background(), client)
					if !body.closed {
						t.Fatal("response body was not closed")
					}
				})
			}
		})
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type trackedBody struct {
	io.ReadCloser
	closed bool
}

func (b *trackedBody) Close() error {
	b.closed = true
	return b.ReadCloser.Close()
}

func checkRequest(t *testing.T, r *http.Request, wantPath string) {
	t.Helper()
	if r.Method != http.MethodGet {
		t.Errorf("method = %q, want GET", r.Method)
	}
	if got := r.URL.EscapedPath(); got != wantPath {
		t.Errorf("path = %q, want %q", got, wantPath)
	}
	if got := r.Header.Get("X-Api-Key"); got != "test-key" {
		t.Errorf("X-Api-Key = %q, want test-key", got)
	}
	if got := r.Header.Get("Accept"); got != "application/json" {
		t.Errorf("Accept = %q, want application/json", got)
	}
}
