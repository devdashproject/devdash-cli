package api

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ValidateEndpoint checks a user-selected API endpoint before any credential is sent.
// Plain HTTP is only useful for a local development server.
func ValidateEndpoint(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u == nil || u.Opaque != "" || u.Host == "" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || (u.Path != "" && u.Path != "/") {
		return fmt.Errorf("API URL must be an origin without credentials, path, query, or fragment")
	}
	if _, err := origin(u); err != nil {
		return err
	}
	return nil
}

func origin(u *url.URL) (string, error) {
	if u == nil || u.User != nil || u.Hostname() == "" {
		return "", fmt.Errorf("invalid API origin")
	}
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port != "" {
		n, err := strconv.Atoi(port)
		if err != nil || n < 1 || n > 65535 {
			return "", fmt.Errorf("invalid API port")
		}
	}
	switch scheme {
	case "https":
		if port == "" {
			port = "443"
		}
	case "http":
		ip := net.ParseIP(host)
		if !strings.EqualFold(host, "localhost") && (ip == nil || !ip.IsLoopback()) {
			return "", fmt.Errorf("plain HTTP is only allowed for loopback development")
		}
		if port == "" {
			port = "80"
		}
	default:
		return "", fmt.Errorf("API URL must use HTTPS or loopback HTTP")
	}
	return scheme + "://" + net.JoinHostPort(host, port), nil
}

// SameOriginRedirect prevents credentials from crossing an origin boundary.
// It is also used for admin-secret requests, which bypass the normal API client.
func SameOriginRedirect(req *http.Request, via []*http.Request) error {
	if len(via) == 0 || len(via) >= 10 {
		return fmt.Errorf("redirect rejected")
	}
	first, err := origin(via[0].URL)
	if err != nil {
		return fmt.Errorf("redirect rejected: %w", err)
	}
	next, err := origin(req.URL)
	if err != nil || next != first {
		return fmt.Errorf("cross-origin redirect rejected")
	}
	return nil
}
