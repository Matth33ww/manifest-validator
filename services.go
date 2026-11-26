package main

import (
	"strconv"
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
