package networking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHTTPClientFetch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Test-Header") != "test-value" {
			t.Errorf("expected X-Test-Header=test-value, got %q", r.Header.Get("X-Test-Header"))
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("test response"))
	}))
	defer server.Close()

	client := NewHTTPClient(nil)
	params := &RequestParams{
		Headers: map[string]string{
			"X-Test-Header": "test-value",
		},
	}

	resp, err := client.Fetch("GET", server.URL, params)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
}

func TestHTTPClientFetchWithContext(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(nil)

	t.Run("successful request", func(t *testing.T) {
		ctx := context.Background()
		resp, err := client.FetchWithContext(ctx, "GET", server.URL, nil)
		if err != nil {
			t.Fatalf("FetchWithContext failed: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status 200, got %d", resp.StatusCode)
		}
	})

	t.Run("cancelled context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		_, err := client.FetchWithContext(ctx, "GET", server.URL, nil)
		if err == nil {
			t.Error("expected error with cancelled context, got nil")
		}
	})

	t.Run("timeout context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := client.FetchWithContext(ctx, "GET", server.URL, nil)
		if err == nil {
			t.Error("expected timeout error, got nil")
		}
	})
}

func TestHTTPClientWithCookies(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("test-cookie")
		if err != nil {
			t.Errorf("expected cookie, got error: %v", err)
		}
		if cookie.Value != "test-value" {
			t.Errorf("expected cookie value test-value, got %q", cookie.Value)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(nil)
	params := &RequestParams{
		Cookies: []*http.Cookie{
			{Name: "test-cookie", Value: "test-value"},
		},
	}

	resp, err := client.Fetch("GET", server.URL, params)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer resp.Body.Close()
}

func TestHTTPClientUserAgent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua := r.Header.Get("User-Agent")
		if ua == "" {
			t.Error("expected User-Agent header, got empty string")
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewHTTPClient(nil)
	resp, err := client.Fetch("GET", server.URL, nil)
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	defer resp.Body.Close()
}
