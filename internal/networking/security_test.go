package networking

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"testing"
)

type fixtureTransport func(*http.Request) (*http.Response, error)

func (f fixtureTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func fixtureResponse(r *http.Request, code int, body string) *http.Response {
	return &http.Response{StatusCode: code, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(body)), ContentLength: -1, Request: r}
}
func TestSecurityPublicAddresses(t *testing.T) {
	for _, s := range []string{"127.0.0.1", "0.0.0.0", "10.0.0.1", "172.24.0.1", "192.168.0.1", "169.254.169.254", "100.64.0.1", "198.18.0.1", "224.0.0.1", "240.0.0.1", "::1", "::ffff:127.0.0.1", "fd7a:115c:a1e0::1", "fe80::1", "2001:db8::1", "64:ff9b::a00:1"} {
		if publicAddress(netip.MustParseAddr(s)) {
			t.Errorf("accepted %s", s)
		}
	}
	for _, s := range []string{"1.1.1.1", "8.8.8.8", "2606:4700:4700::1111"} {
		if !publicAddress(netip.MustParseAddr(s)) {
			t.Errorf("rejected %s", s)
		}
	}
}
func TestSecurityCookieScope(t *testing.T) {
	c := &http.Cookie{Name: "sessionid", Value: "fixture", Domain: ".instagram.com", Path: "/api", Secure: true}
	for raw, want := range map[string]bool{"https://www.instagram.com/api": true, "https://www.instagram.com/api/v1": true, "https://www.instagram.com/apifake": false, "http://www.instagram.com/api": false, "https://other.invalid/api": false, "https://instagram.com.evil.invalid/api": false} {
		u, _ := url.Parse(raw)
		if got := cookieMatches(c, u, false); got != want {
			t.Errorf("%s: %v", raw, got)
		}
	}
}
func TestSecurityNoCookieToForeignHost(t *testing.T) {
	c := DefaultHTTPClient(&NewHTTPClientOptions{Cookies: []*http.Cookie{{Name: "sessionid", Value: "fixture", Domain: ".instagram.com"}}})
	c.Client = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Cookie") != "" {
			t.Error("cookie leaked")
		}
		return fixtureResponse(r, 200, "ok"), nil
	})}
	resp, err := c.Fetch("GET", "https://other.invalid/a", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}
func TestSecurityRejectRedirectBeforeFollowingPrivateAddress(t *testing.T) {
	c := DefaultHTTPClient(nil)
	calls := 0
	c.Client = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		res := fixtureResponse(r, 302, "")
		res.Header.Set("Location", "http://127.0.0.1/secret")
		return res, nil
	})}
	_, err := c.Fetch("GET", "https://www.instagram.com/fixture", nil)
	if err == nil || !errors.Is(err, ErrUnsafeDestination) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}
func TestSecurityCookiePathRecheckedOnRedirect(t *testing.T) {
	c := DefaultHTTPClient(&NewHTTPClientOptions{Cookies: []*http.Cookie{{Name: "sessionid", Value: "fixture", Domain: ".instagram.com", Path: "/api"}}})
	calls := 0
	c.Client = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) {
		calls++
		res := fixtureResponse(r, 200, "ok")
		if calls == 1 {
			if r.Header.Get("Cookie") == "" {
				t.Error("missing scoped cookie")
			}
			res.StatusCode = 302
			res.Header.Set("Location", "https://www.instagram.com/unrelated")
		} else if r.Header.Get("Cookie") != "" {
			t.Error("path-limited cookie leaked")
		}
		return res, nil
	})}
	resp, err := c.Fetch("GET", "https://www.instagram.com/api", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if calls != 2 {
		t.Fatal(calls)
	}
}
func TestSecurityPrivateDialRejectedWithoutConnecting(t *testing.T) {
	if _, err := publicDialContext(context.Background(), "tcp", "127.0.0.1:443"); !errors.Is(err, ErrUnsafeDestination) {
		t.Fatal(err)
	}
}
func TestSecurityResponseLimitUnknownLength(t *testing.T) {
	for _, body := range []string{"12345678", "123456789"} {
		c := DefaultHTTPClient(nil)
		c.ResponseLimit = 8
		c.Client = &http.Client{Transport: fixtureTransport(func(r *http.Request) (*http.Response, error) { return fixtureResponse(r, 200, body), nil })}
		resp, err := c.Fetch("GET", "https://example.invalid/fixture", nil)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if len(data) > 8 || (len(body) > 8 && !errors.Is(err, ErrResponseTooLarge)) || (len(body) == 8 && err != nil) {
			t.Fatalf("len=%d err=%v", len(data), err)
		}
	}
}
func TestSecurityDownloadDropsSensitiveHeaders(t *testing.T) {
	c := DefaultHTTPClient(&NewHTTPClientOptions{Headers: map[string]string{"Authorization": "fixture", "Cookie": "fixture", "User-Agent": "fixture"}})
	d := c.AsDownloadClient()
	if d.Headers["Authorization"] != "" || d.Headers["Cookie"] != "" || d.Headers["User-Agent"] != "fixture" || c.Headers["Authorization"] != "fixture" {
		t.Fatal("header isolation failed")
	}
}
