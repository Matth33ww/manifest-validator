package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"syscall"
	"time"
)

// Rule 8 calls the service's health endpoint for real.
// That means our server opens a connection to an address the user typed.
// A bad user could type an address inside our own network, like
// http://127.0.0.1 or http://192.168.0.1, to reach things that are not public.
// So we block those addresses before we connect.

// errBlockedAddress is the error we get when the address is not allowed.
var errBlockedAddress = errors.New("this address is not allowed")

// isBlockedIP reports whether we must not connect to this IP address.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() || // this computer, like 127.0.0.1
		ip.IsPrivate() || // home and office networks, like 192.168.0.1 and 10.0.0.1
		ip.IsLinkLocalUnicast() || // like 169.254.169.254, used by cloud servers
		ip.IsLinkLocalMulticast() ||
		ip.IsInterfaceLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified() // 0.0.0.0
}

// checkAddress runs just before each connection, after the name is turned into an IP.
// Checking here (and not before) also covers redirects, and names that change their IP.
func checkAddress(network, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return err
	}
	ip := net.ParseIP(host)
	if ip == nil || isBlockedIP(ip) {
		return errBlockedAddress
	}
	return nil
}

// maxRedirects is how many times we follow a redirect before we stop.
const maxRedirects = 3

// newSafeClient makes an HTTP client for health checks.
// It blocks inside addresses, waits at most 5 seconds and follows at most 3 redirects.
func newSafeClient() *http.Client {
	dialer := &net.Dialer{
		Timeout: 3 * time.Second,
		Control: checkAddress,
	}
	transport := &http.Transport{
		// No proxy. We want to connect straight to the address we checked.
		Proxy:                 nil,
		DialContext:           dialer.DialContext,
		TLSHandshakeTimeout:   3 * time.Second,
		ResponseHeaderTimeout: 3 * time.Second,
	}
	return &http.Client{
		Timeout:   5 * time.Second,
		Transport: transport,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= maxRedirects {
				return errors.New("too many redirects")
			}
			return nil
		},
	}
}

// HealthChecker runs rule 8. It holds the HTTP client it uses.
// Tests can give it another client, for example one that may call a local test server.
type HealthChecker struct {
	client *http.Client
}

// NewHealthChecker makes a HealthChecker with the safe client.
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{client: newSafeClient()}
}

// Rule 8: the health endpoint should answer with a 2xx status, like 200 OK.
// If it does not, we warn. A warning does not make the manifest invalid,
// because the service may not be running yet.
// ctx stops the check early if the user's request is cancelled.
func (h *HealthChecker) Check(ctx context.Context, m Manifest) []Issue {
	// For example https://billing.example.com + /health.
	address := strings.TrimSuffix(m.URL, "/") + m.HealthEndpoint

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		return []Issue{healthWarning("health check could not start: " + err.Error())}
	}

	resp, err := h.client.Do(req)
	if errors.Is(err, errBlockedAddress) {
		return []Issue{healthWarning("health check skipped: the address points inside a private network")}
	}
	if err != nil {
		return []Issue{healthWarning("health check failed: could not reach " + address)}
	}
	defer resp.Body.Close()
	// Read a little of the body and throw it away, so the connection can be used again.
	io.Copy(io.Discard, io.LimitReader(resp.Body, 4096))

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return []Issue{healthWarning(fmt.Sprintf("health check returned status %d, want 2xx", resp.StatusCode))}
	}
	return nil
}

// healthWarning makes a warning for the healthEndpoint field.
func healthWarning(message string) Issue {
	return Issue{Field: "healthEndpoint", Message: message, Severity: "warning"}
}
