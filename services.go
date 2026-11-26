package main

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

// RegisteredService is a service that was saved after its manifest was valid.
// It has all the manifest fields, plus an id that the server gives it.
type RegisteredService struct {
	ID string `json:"id"`
	Manifest
}

// Announcement is a news item or a planned change for one service.
type Announcement struct {
	ID        string    `json:"id"`
	ServiceID string    `json:"serviceId"`
	Title     string    `json:"title"`
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"createdAt"`
}

// Store keeps registered services and their announcements in memory.
// The data is lost when the server stops. That is fine for this project.
type Store struct {
	mu            sync.Mutex // protects the fields below, because requests run at the same time
	services      []RegisteredService
	announcements []Announcement
	nextID        int
}

// NewStore makes an empty store.
func NewStore() *Store {
	return &Store{nextID: 1}
}

// Register saves a manifest as a new service and returns it with its id.
func (s *Store) Register(m Manifest) RegisteredService {
	s.mu.Lock()
	defer s.mu.Unlock()

	service := RegisteredService{ID: strconv.Itoa(s.nextID), Manifest: m}
	s.nextID++
	s.services = append(s.services, service)
	return service
}

// List returns all services, oldest first.
// It returns a copy, so the caller can not change the store by mistake.
func (s *Store) List() []RegisteredService {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := make([]RegisteredService, len(s.services))
	copy(list, s.services)
	return list
}

// Get finds one service by its id. The second value is false if there is none.
func (s *Store) Get(id string) (RegisteredService, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, service := range s.services {
		if service.ID == id {
			return service, true
		}
	}
	return RegisteredService{}, false
}

// AddAnnouncement saves a new announcement for a service.
// The second value is false if the service does not exist.
func (s *Store) AddAnnouncement(serviceID, title, body string) (Announcement, bool) {
	if _, found := s.Get(serviceID); !found {
		return Announcement{}, false
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	announcement := Announcement{
		ID:        strconv.Itoa(s.nextID),
		ServiceID: serviceID,
		Title:     title,
		Body:      body,
		CreatedAt: time.Now().UTC(),
	}
	s.nextID++
	s.announcements = append(s.announcements, announcement)
	return announcement, true
}

// Announcements returns the announcements of one service, newest first.
func (s *Store) Announcements(serviceID string) []Announcement {
	s.mu.Lock()
	defer s.mu.Unlock()

	list := []Announcement{}
	// Go from the end to the start, so the newest one comes first.
	for i := len(s.announcements) - 1; i >= 0; i-- {
		if s.announcements[i].ServiceID == serviceID {
			list = append(list, s.announcements[i])
		}
	}
	return list
}

// Limits for announcements, so nobody can save huge texts.
const (
	maxTitleLength = 100
	maxBodyLength  = 1000
)

// ServicesAPI is the HTTP part: it connects the store to the web routes.
type ServicesAPI struct {
	store *Store
}

// NewServicesAPI makes the handlers for a store.
func NewServicesAPI(store *Store) *ServicesAPI {
	return &ServicesAPI{store: store}
}

// Routes adds the routes of this API to a router.
// "{id}" is a placeholder for the service id in the URL.
func (a *ServicesAPI) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /services", a.handleList)
	mux.HandleFunc("POST /services", a.handleRegister)
	mux.HandleFunc("GET /services/{id}/announcements", a.handleListAnnouncements)
	mux.HandleFunc("POST /services/{id}/announcements", a.handleAddAnnouncement)
}

// handleList answers GET /services with all registered services.
func (a *ServicesAPI) handleList(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, a.store.List())
}

// handleRegister answers POST /services.
// It checks the manifest again, and saves it only if it has no errors.
func (a *ServicesAPI) handleRegister(w http.ResponseWriter, r *http.Request) {
	body, ok := readBody(w, r)
	if !ok {
		return
	}

	manifest, err := ParseManifest(body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	issues := Validate(manifest)
	if hasErrors(issues) {
		// 422 means: we understood the request, but the manifest is not valid.
		writeJSON(w, http.StatusUnprocessableEntity, ValidateResponse{Valid: false, Issues: issues})
		return
	}
	writeJSON(w, http.StatusCreated, a.store.Register(manifest))
}

// handleListAnnouncements answers GET /services/{id}/announcements.
func (a *ServicesAPI) handleListAnnouncements(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if _, found := a.store.Get(id); !found {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	writeJSON(w, http.StatusOK, a.store.Announcements(id))
}

// handleAddAnnouncement answers POST /services/{id}/announcements.
// The body is JSON: {"title": "...", "body": "..."}.
func (a *ServicesAPI) handleAddAnnouncement(w http.ResponseWriter, r *http.Request) {
	body, ok := readBody(w, r)
	if !ok {
		return
	}

	var input struct {
		Title string `json:"title"`
		Body  string `json:"body"`
	}
	if err := json.Unmarshal(body, &input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid JSON: "+err.Error())
		return
	}

	title := strings.TrimSpace(input.Title)
	if title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	if len(title) > maxTitleLength || len(input.Body) > maxBodyLength {
		writeError(w, http.StatusBadRequest, "title or body is too long")
		return
	}

	announcement, found := a.store.AddAnnouncement(r.PathValue("id"), title, input.Body)
	if !found {
		writeError(w, http.StatusNotFound, "service not found")
		return
	}
	writeJSON(w, http.StatusCreated, announcement)
}

// readBody reads the request body, up to 64 KB.
// If it fails, it already sent the error to the client, and returns false.
func readBody(w http.ResponseWriter, r *http.Request) ([]byte, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
	body, err := io.ReadAll(r.Body)
	if err != nil {
		var tooBig *http.MaxBytesError
		if errors.As(err, &tooBig) {
			writeError(w, http.StatusRequestEntityTooLarge, "body is larger than 64 KB")
		} else {
			writeError(w, http.StatusBadRequest, "could not read the body")
		}
		return nil, false
	}
	return body, true
}
