package networking

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"sort"
	"strings"
	"time"
)

var ErrUnsafeDestination = errors.New("request destination is not public HTTP(S)")
var ErrResponseTooLarge = errors.New("HTTP response exceeded size limit")
var blockedNetworks = []netip.Prefix{
	netip.MustParsePrefix("0.0.0.0/8"), netip.MustParsePrefix("100.64.0.0/10"),
	netip.MustParsePrefix("192.0.0.0/24"), netip.MustParsePrefix("192.0.2.0/24"),
	netip.MustParsePrefix("198.18.0.0/15"), netip.MustParsePrefix("198.51.100.0/24"),
	netip.MustParsePrefix("203.0.113.0/24"), netip.MustParsePrefix("240.0.0.0/4"),
	netip.MustParsePrefix("64:ff9b::/96"), netip.MustParsePrefix("64:ff9b:1::/48"), netip.MustParsePrefix("2001:db8::/32"),
}

func publicAddress(ip netip.Addr) bool {
	ip = ip.Unmap()
	if !ip.IsValid() || !ip.IsGlobalUnicast() || ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
		return false
	}
	if ip == netip.MustParseAddr("168.63.129.16") {
		return false
	}
	for _, p := range blockedNetworks {
		if p.Contains(ip) {
			return false
		}
	}
	return true
}
func validatePublicURL(u *url.URL) error {
	if u == nil || (u.Scheme != "https" && u.Scheme != "http") || u.User != nil || u.Hostname() == "" || len(u.String()) > 4096 {
		return ErrUnsafeDestination
	}
	if p := u.Port(); p != "" && p != "80" && p != "443" {
		return ErrUnsafeDestination
	}
	host := strings.TrimSuffix(strings.ToLower(u.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".localhost") || strings.HasSuffix(host, ".local") || !strings.Contains(host, ".") {
		return ErrUnsafeDestination
	}
	if ip, err := netip.ParseAddr(host); err == nil && !publicAddress(ip) {
		return ErrUnsafeDestination
	}
	return nil
}

// Resolve and connect to the same checked numeric IP, avoiding a DNS rebinding gap.
func publicDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, ErrUnsafeDestination
	}
	if port != "80" && port != "443" {
		return nil, ErrUnsafeDestination
	}
	addresses, err := net.DefaultResolver.LookupNetIP(ctx, "ip", host)
	if err != nil {
		return nil, err
	}
	if len(addresses) == 0 {
		return nil, ErrUnsafeDestination
	}
	for _, ip := range addresses {
		if !publicAddress(ip) {
			return nil, ErrUnsafeDestination
		}
	}
	// Prefer IPv4 on Docker bridges without IPv6; still validate every DNS result.
	sort.SliceStable(addresses, func(i, j int) bool { return addresses[i].Is4() && !addresses[j].Is4() })
	dialer := &net.Dialer{Timeout: defaultTimeout, KeepAlive: defaultTimeout}
	var last error
	for i, ip := range addresses {
		if i >= 4 {
			break
		}
		conn, e := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if e == nil {
			return conn, nil
		}
		last = e
	}
	return nil, last
}
func checkRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= 8 {
		return fmt.Errorf("too many redirects")
	}
	if err := validatePublicURL(req.URL); err != nil {
		return err
	}
	if len(via) > 0 {
		previous := via[len(via)-1].URL
		if previous.Scheme == "https" && req.URL.Scheme != "https" {
			return ErrUnsafeDestination
		}
		if !strings.EqualFold(previous.Host, req.URL.Host) {
			req.Header.Del("Authorization")
			req.Header.Del("Proxy-Authorization")
			req.Header.Del("Cookie")
		}
	}
	return nil
}
func cookieMatches(c *http.Cookie, u *url.URL, explicit bool) bool {
	if c == nil || c.MaxAge < 0 || (!c.Expires.IsZero() && c.Expires.Before(time.Now())) || (c.Secure && u.Scheme != "https") {
		return false
	}
	domain := strings.TrimPrefix(strings.ToLower(c.Domain), ".")
	host := strings.ToLower(u.Hostname())
	if domain == "" {
		if !explicit {
			return false
		}
	} else if host != domain && !strings.HasSuffix(host, "."+domain) {
		return false
	}
	path := u.EscapedPath()
	if path == "" {
		path = "/"
	}
	cp := c.Path
	if cp == "" {
		cp = "/"
	}
	return path == cp || (strings.HasPrefix(path, cp) && (strings.HasSuffix(cp, "/") || (len(path) > len(cp) && path[len(cp)] == '/')))
}

type responseLimit struct {
	io.ReadCloser
	remaining int64
}

func (r *responseLimit) Read(p []byte) (int, error) {
	if r.remaining == 0 {
		var b [1]byte
		n, e := r.ReadCloser.Read(b[:])
		if n > 0 {
			return 0, ErrResponseTooLarge
		}
		return 0, e
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, e := r.ReadCloser.Read(p)
	r.remaining -= int64(n)
	return n, e
}
