package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

const testOrigin = "http://localhost:4200"

// send makes a fake request and runs it through our real routes.
// No real server or network is needed.
func send(method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	newHandler(testOrigin).ServeHTTP(rec, req)
	return rec
}

func TestAPIGoodManifest(t *testing.T) {
	body := "name: billing\nowner: a@b.com\nurl: https://x.com\nhealthEndpoint: /health\n"
	rec := send(http.MethodPost, "/validate", body)

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	var resp ValidateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if !resp.Valid || len(resp.Issues) != 0 {
		t.Errorf("got %+v, want valid with no issues", resp)
	}
}

func TestAPIIssuesIsEmptyListNotNull(t *testing.T) {
	body := `{"name":"billing","owner":"a@b.com","url":"https://x.com","healthEndpoint":"/h"}`
	rec := send(http.MethodPost, "/validate", body)

	if !strings.Contains(rec.Body.String(), `"issues":[]`) {
		t.Errorf("got %s, want issues to be []", rec.Body.String())
	}
}

func TestAPIBadManifest(t *testing.T) {
	rec := send(http.MethodPost, "/validate", "name: Billing Service\n")

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	var resp ValidateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if resp.Valid {
		t.Error("got valid true, want false")
	}
	// 3 missing fields + the bad name.
	if len(resp.Issues) != 4 {
		t.Errorf("got %d issues, want 4: %+v", len(resp.Issues), resp.Issues)
	}
}

func TestAPIOnlyWarningsIsStillValid(t *testing.T) {
	body := "name: billing\nowner: a@b.com\nurl: https://x.com\nhealthEndpoint: /h\ndescription: " + strings.Repeat("a", 250) + "\n"
	rec := send(http.MethodPost, "/validate", body)

	var resp ValidateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if !resp.Valid || len(resp.Issues) != 1 {
		t.Errorf("got %+v, want valid with one warning", resp)
	}
}

func TestAPIErrorStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"empty body", http.MethodPost, "/validate", "", http.StatusBadRequest},
		{"broken json", http.MethodPost, "/validate", `{"name": `, http.StatusBadRequest},
		{"nested yaml", http.MethodPost, "/validate", "owner:\n  email: a@b.com\n", http.StatusBadRequest},
		{"GET on validate", http.MethodGet, "/validate", "", http.StatusMethodNotAllowed},
		{"body over 64 KB", http.MethodPost, "/validate", strings.Repeat("a", maxBodyBytes+1), http.StatusRequestEntityTooLarge},
		{"unknown path", http.MethodGet, "/nothing", "", http.StatusNotFound},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := send(c.method, c.path, c.body)
			if rec.Code != c.wantStatus {
				t.Errorf("got status %d, want %d (body: %s)", rec.Code, c.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestAPIErrorsAreJSON(t *testing.T) {
	rec := send(http.MethodPost, "/validate", "")

	var resp map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("error response is not JSON: %v", err)
	}
	if resp["error"] == "" {
		t.Errorf("got %v, want an \"error\" message", resp)
	}
}

func TestAPIHealthz(t *testing.T) {
	rec := send(http.MethodGet, "/healthz", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("got status %d, want 200", rec.Code)
	}
	if got := strings.TrimSpace(rec.Body.String()); got != `{"status":"ok"}` {
		t.Errorf("got %s, want {\"status\":\"ok\"}", got)
	}
}

func TestAPICORS(t *testing.T) {
	// The browser asks first with OPTIONS. We answer 204 with the CORS headers.
	rec := send(http.MethodOptions, "/validate", "")

	if rec.Code != http.StatusNoContent {
		t.Errorf("got status %d, want 204", rec.Code)
	}
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("got allowed origin %q, want %q", got, testOrigin)
	}
}

func TestAPIServicesRoutesAreConnected(t *testing.T) {
	// Use one handler for all steps, so they share the same store.
	h := newHandler(testOrigin)

	// Register a service through the full server.
	req := httptest.NewRequest(http.MethodPost, "/services", strings.NewReader(goodManifest))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: got status %d, want 201", rec.Code)
	}

	// Now the list must have it.
	req = httptest.NewRequest(http.MethodGet, "/services", nil)
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("list: got status %d, want 200", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), `"name":"billing"`) {
		t.Errorf("list: got %s, want the billing service", rec.Body.String())
	}

	// The new routes also send the CORS header.
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != testOrigin {
		t.Errorf("got allowed origin %q, want %q", got, testOrigin)
	}
}

func TestAPICheckHealthOption(t *testing.T) {
	// A local server. We only use its address, which is on 127.0.0.1.
	server := httptest.NewServer(http.NotFoundHandler())
	defer server.Close()

	// Rule 4 needs https, so we use https with the same local address.
	// The safe client must block it. If it did not block it, the check would
	// fail in another way ("could not reach"), and the test below would see that.
	localHTTPS := "https://" + strings.TrimPrefix(server.URL, "http://")
	body := "name: billing\nowner: a@b.com\nurl: " + localHTTPS + "\nhealthEndpoint: /health\n"

	t.Run("without the option there is no health check", func(t *testing.T) {
		rec := send(http.MethodPost, "/validate", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", rec.Code)
		}
		if strings.Contains(rec.Body.String(), "health check") {
			t.Errorf("got %s, want no health warning", rec.Body.String())
		}
	})

	t.Run("with the option the blocked address gives a warning", func(t *testing.T) {
		rec := send(http.MethodPost, "/validate?checkHealth=true", body)
		if rec.Code != http.StatusOK {
			t.Fatalf("got status %d, want 200", rec.Code)
		}
		if !strings.Contains(rec.Body.String(), "private network") {
			t.Errorf("got %s, want a warning about a private network", rec.Body.String())
		}
		// A warning does not make the manifest invalid.
		if !strings.Contains(rec.Body.String(), `"valid":true`) {
			t.Errorf("got %s, want valid to stay true", rec.Body.String())
		}
	})

	t.Run("with errors the health check is skipped", func(t *testing.T) {
		// The owner is not an email, so the manifest has an error.
		bad := "name: billing\nowner: nope\nurl: " + localHTTPS + "\nhealthEndpoint: /health\n"
		rec := send(http.MethodPost, "/validate?checkHealth=true", bad)
		if strings.Contains(rec.Body.String(), "health check") {
			t.Errorf("got %s, want no health check when there are errors", rec.Body.String())
		}
	})
}
