package main

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"127.0.0.1", true},       // this computer
		{"::1", true},             // this computer, IPv6
		{"10.0.0.5", true},        // private network
		{"192.168.1.10", true},    // home network
		{"172.16.0.1", true},      // private network
		{"169.254.169.254", true}, // cloud server secrets
		{"0.0.0.0", true},
		{"::ffff:127.0.0.1", true}, // 127.0.0.1 written as IPv6
		{"8.8.8.8", false},         // a public address
		{"1.1.1.1", false},
	}
	for _, c := range cases {
		t.Run(c.ip, func(t *testing.T) {
			if got := isBlockedIP(net.ParseIP(c.ip)); got != c.want {
				t.Errorf("isBlockedIP(%s) = %v, want %v", c.ip, got, c.want)
			}
		})
	}
}

// newTestServer starts a local server. /health answers with the given status.
func newTestServer(t *testing.T, status int) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/health" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(status)
	}))
	t.Cleanup(server.Close)
	return server
}

func TestHealthCheckStatus(t *testing.T) {
	cases := []struct {
		name        string
		status      int
		wantWarning string // empty means no warning
	}{
		{"200 is fine", http.StatusOK, ""},
		{"204 is fine", http.StatusNoContent, ""},
		{"503 gives a warning", http.StatusServiceUnavailable, "returned status 503"},
		{"404 gives a warning", http.StatusNotFound, "returned status 404"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			server := newTestServer(t, c.status)
			// The test server runs on 127.0.0.1, which the safe client blocks.
			// So here we use the test server's own client.
			checker := &HealthChecker{client: server.Client()}

			issues := checker.Check(context.Background(), Manifest{URL: server.URL, HealthEndpoint: "/health"})

			if c.wantWarning == "" {
				if len(issues) != 0 {
					t.Fatalf("got %v, want no issues", issues)
				}
				return
			}
			if len(issues) != 1 || !strings.Contains(issues[0].Message, c.wantWarning) {
				t.Fatalf("got %v, want one warning with %q", issues, c.wantWarning)
			}
			if issues[0].Severity != "warning" || issues[0].Field != "healthEndpoint" {
				t.Errorf("got %+v, want a warning for healthEndpoint", issues[0])
			}
		})
	}
}

func TestHealthCheckBlocksLocalAddress(t *testing.T) {
	// This server works, but it is on 127.0.0.1. The safe client must not call it.
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
	}))
	defer server.Close()

	issues := NewHealthChecker().Check(context.Background(), Manifest{URL: server.URL, HealthEndpoint: "/health"})

	if called {
		t.Error("the safe client called a local address")
	}
	if len(issues) != 1 || !strings.Contains(issues[0].Message, "private network") {
		t.Errorf("got %v, want a warning about a private network", issues)
	}
}

func TestHealthCheckBlocksRedirectToLocalAddress(t *testing.T) {
	// The first server sends us on to 127.0.0.1. The redirect must be blocked too.
	// We call the first server with a client that may reach it, but that still
	// blocks every connection after that. This acts like a public server that redirects.
	inside := newTestServer(t, http.StatusOK)
	outside := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, inside.URL+"/health", http.StatusFound)
	}))
	defer outside.Close()

	client := newSafeClient()
	firstCall := true
	transport := client.Transport.(*http.Transport)
	safeDial := transport.DialContext
	transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
		if firstCall {
			firstCall = false
			return (&net.Dialer{}).DialContext(ctx, network, address)
		}
		return safeDial(ctx, network, address)
	}
	checker := &HealthChecker{client: client}

	issues := checker.Check(context.Background(), Manifest{URL: outside.URL, HealthEndpoint: "/start"})

	if len(issues) != 1 || !strings.Contains(issues[0].Message, "private network") {
		t.Errorf("got %v, want a warning about a private network", issues)
	}
}

func TestHealthCheckServerDown(t *testing.T) {
	// Start a server and stop it at once, so nothing answers on that address.
	server := httptest.NewServer(http.NotFoundHandler())
	address := server.URL
	server.Close()

	checker := &HealthChecker{client: http.DefaultClient}
	issues := checker.Check(context.Background(), Manifest{URL: address, HealthEndpoint: "/health"})

	if len(issues) != 1 || !strings.Contains(issues[0].Message, "could not reach") {
		t.Errorf("got %v, want a warning that the server could not be reached", issues)
	}
}
