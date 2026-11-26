package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// A manifest with no problems, used in many tests.
const goodManifest = "name: billing\nowner: a@b.com\nurl: https://x.com\nhealthEndpoint: /health\n"

// newServicesHandler makes the routes with a new, empty store.
// Each test gets its own store, so tests do not change each other.
func newServicesHandler() http.Handler {
	mux := http.NewServeMux()
	NewServicesAPI(NewStore()).Routes(mux)
	return mux
}

// call sends one fake request to the handler and returns the answer.
func call(h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestStoreRegisterAndGet(t *testing.T) {
	store := NewStore()

	first, ok := store.Register(Manifest{Name: "billing"})
	if !ok || first.ID != "1" {
		t.Fatalf("got %+v, %v, want id 1 and true", first, ok)
	}
	if _, ok := store.Register(Manifest{Name: "billing"}); ok {
		t.Error("same name twice: got true, want false")
	}
	if got, found := store.Get("1"); !found || got.Name != "billing" {
		t.Errorf("Get(1): got %+v, %v", got, found)
	}
	if _, found := store.Get("99"); found {
		t.Error("Get(99): got found, want not found")
	}
	if got := len(store.List()); got != 1 {
		t.Errorf("List: got %d services, want 1", got)
	}
}

func TestStoreAnnouncementsNewestFirst(t *testing.T) {
	store := NewStore()
	service, _ := store.Register(Manifest{Name: "billing"})

	store.AddAnnouncement(service.ID, "first", "")
	store.AddAnnouncement(service.ID, "second", "")

	got := store.Announcements(service.ID)
	if len(got) != 2 || got[0].Title != "second" || got[1].Title != "first" {
		t.Errorf("got %+v, want second then first", got)
	}
	if _, ok := store.AddAnnouncement("99", "x", ""); ok {
		t.Error("unknown service: got true, want false")
	}
}

func TestRegisterThenList(t *testing.T) {
	h := newServicesHandler()

	// The list is empty at the start. It must be "[]", not "null".
	rec := call(h, http.MethodGet, "/services", "")
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != "[]" {
		t.Fatalf("empty list: got %d %s, want 200 []", rec.Code, got)
	}

	rec = call(h, http.MethodPost, "/services", goodManifest)
	if rec.Code != http.StatusCreated {
		t.Fatalf("register: got %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	rec = call(h, http.MethodGet, "/services", "")
	var services []RegisteredService
	if err := json.Unmarshal(rec.Body.Bytes(), &services); err != nil {
		t.Fatalf("list is not JSON: %v", err)
	}
	if len(services) != 1 || services[0].Name != "billing" || services[0].ID != "1" {
		t.Errorf("got %+v, want one service named billing with id 1", services)
	}
}

func TestRegisterStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{"good manifest", goodManifest, http.StatusCreated},
		{"manifest with errors", "name: Billing Service\n", http.StatusUnprocessableEntity},
		{"empty body", "", http.StatusBadRequest},
		{"broken json", `{"name": `, http.StatusBadRequest},
		{"body over 64 KB", strings.Repeat("a", maxBodyBytes+1), http.StatusRequestEntityTooLarge},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			rec := call(newServicesHandler(), http.MethodPost, "/services", c.body)
			if rec.Code != c.wantStatus {
				t.Errorf("got %d, want %d (body: %s)", rec.Code, c.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRegisterSameNameTwice(t *testing.T) {
	h := newServicesHandler()
	call(h, http.MethodPost, "/services", goodManifest)

	rec := call(h, http.MethodPost, "/services", goodManifest)
	if rec.Code != http.StatusConflict {
		t.Errorf("got %d, want 409", rec.Code)
	}
}

func TestInvalidManifestIsNotSaved(t *testing.T) {
	h := newServicesHandler()
	rec := call(h, http.MethodPost, "/services", "name: Billing Service\n")

	// The answer has the same shape as /validate, so the page can show the issues.
	var resp ValidateResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("response is not JSON: %v", err)
	}
	if resp.Valid || len(resp.Issues) == 0 {
		t.Errorf("got %+v, want not valid with issues", resp)
	}

	rec = call(h, http.MethodGet, "/services", "")
	if got := strings.TrimSpace(rec.Body.String()); got != "[]" {
		t.Errorf("got %s, want [] because nothing should be saved", got)
	}
}

func TestAnnouncements(t *testing.T) {
	h := newServicesHandler()
	call(h, http.MethodPost, "/services", goodManifest)

	rec := call(h, http.MethodGet, "/services/1/announcements", "")
	if got := strings.TrimSpace(rec.Body.String()); rec.Code != http.StatusOK || got != "[]" {
		t.Fatalf("empty list: got %d %s, want 200 []", rec.Code, got)
	}

	rec = call(h, http.MethodPost, "/services/1/announcements", `{"title": "Maintenance", "body": "Sunday 10:00"}`)
	if rec.Code != http.StatusCreated {
		t.Fatalf("add: got %d, want 201 (body: %s)", rec.Code, rec.Body.String())
	}

	rec = call(h, http.MethodGet, "/services/1/announcements", "")
	var list []Announcement
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("list is not JSON: %v", err)
	}
	if len(list) != 1 || list[0].Title != "Maintenance" || list[0].ServiceID != "1" {
		t.Errorf("got %+v, want one announcement for service 1", list)
	}
	if list[0].CreatedAt.IsZero() {
		t.Error("createdAt is empty, want the time it was made")
	}
}

func TestAnnouncementStatusCodes(t *testing.T) {
	cases := []struct {
		name       string
		method     string
		path       string
		body       string
		wantStatus int
	}{
		{"list for unknown service", http.MethodGet, "/services/99/announcements", "", http.StatusNotFound},
		{"add to unknown service", http.MethodPost, "/services/99/announcements", `{"title": "x"}`, http.StatusNotFound},
		{"no title", http.MethodPost, "/services/1/announcements", `{"body": "x"}`, http.StatusBadRequest},
		{"title with only spaces", http.MethodPost, "/services/1/announcements", `{"title": "   "}`, http.StatusBadRequest},
		{"broken json", http.MethodPost, "/services/1/announcements", `{"title": `, http.StatusBadRequest},
		{"title of 100 letters", http.MethodPost, "/services/1/announcements", `{"title": "` + strings.Repeat("ā", 100) + `"}`, http.StatusCreated},
		{"title of 101 letters", http.MethodPost, "/services/1/announcements", `{"title": "` + strings.Repeat("ā", 101) + `"}`, http.StatusBadRequest},
		{"body too long", http.MethodPost, "/services/1/announcements", `{"title": "x", "body": "` + strings.Repeat("a", 1001) + `"}`, http.StatusBadRequest},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := newServicesHandler()
			call(h, http.MethodPost, "/services", goodManifest)

			rec := call(h, c.method, c.path, c.body)
			if rec.Code != c.wantStatus {
				t.Errorf("got %d, want %d (body: %s)", rec.Code, c.wantStatus, rec.Body.String())
			}
		})
	}
}
