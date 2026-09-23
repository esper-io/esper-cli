package runtime

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

func TestHTTPClientRetriesAndAuthorizes(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		attempts++
		if request.Header.Get("Authorization") != "Bearer secret" {
			t.Errorf("Authorization = %q", request.Header.Get("Authorization"))
		}
		if request.URL.Query().Get("limit") != "10" {
			t.Errorf("limit = %q", request.URL.Query().Get("limit"))
		}
		if attempts < 3 {
			http.Error(writer, "temporary", http.StatusServiceUnavailable)
			return
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`{"ok":true}`))
	}))
	defer server.Close()

	client := &HTTPClient{
		BaseURL: server.URL,
		APIKey:  "secret",
		Client:  server.Client(),
		Retry: RetryPolicy{
			MaxAttempts: 4,
			InitialWait: time.Second,
			MaxWait:     8 * time.Second,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		},
	}
	data, err := client.Do(context.Background(), http.MethodGet, "/devices", url.Values{"limit": {"10"}}, nil)
	if err != nil {
		t.Fatalf("Do() error = %v", err)
	}
	if string(data) != `{"ok":true}` || attempts != 3 {
		t.Fatalf("Do() = %s after %d attempts", data, attempts)
	}
}

func TestHTTPClientDoesNotRetryPost(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(writer, "temporary", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &HTTPClient{
		BaseURL: server.URL,
		Client:  server.Client(),
		Retry: RetryPolicy{
			MaxAttempts: 4,
			InitialWait: time.Second,
			MaxWait:     time.Second,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		},
	}
	_, err := client.Do(context.Background(), http.MethodPost, "/devices", nil, []byte(`{}`))
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestHTTPClientDoesNotRetryDelete(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		attempts++
		http.Error(writer, "temporary", http.StatusServiceUnavailable)
	}))
	defer server.Close()
	client := &HTTPClient{
		BaseURL: server.URL,
		Client:  server.Client(),
		Retry: RetryPolicy{
			MaxAttempts: 4,
			Sleep:       func(context.Context, time.Duration) error { return nil },
		},
	}
	_, err := client.Do(context.Background(), http.MethodDelete, "/devices/1", nil, nil)
	var apiError *APIError
	if !errors.As(err, &apiError) || apiError.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("Do() error = %v", err)
	}
	if attempts != 1 {
		t.Fatalf("attempts = %d, want 1", attempts)
	}
}

func TestRetryAfterIsCapped(t *testing.T) {
	policy := RetryPolicy{InitialWait: time.Second, MaxWait: 8 * time.Second}
	for _, retryAfter := range []string{"86400", "9223372036854775807"} {
		response := &http.Response{Header: http.Header{"Retry-After": []string{retryAfter}}}
		if got := policy.delay(0, response); got != 8*time.Second {
			t.Fatalf("delay(%q) = %s, want 8s", retryAfter, got)
		}
	}
}

func TestHTTPClientUsesSelectedResponseMedia(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if got := request.Header.Get("Accept"); got != "application/x-pem-file" {
			t.Errorf("Accept = %q", got)
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := &HTTPClient{BaseURL: server.URL, Client: server.Client()}
	_, err := client.DoWithContentTypeAndHeaders(context.Background(), http.MethodGet, "/certificate", nil, nil, nil, "application/json", "application/x-pem-file")
	if err != nil {
		t.Fatal(err)
	}
}
