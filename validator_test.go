package main

import "testing"

func TestCheckRequired(t *testing.T) {
	if got := checkRequired(Manifest{Name: "a", Owner: "a@b.com", URL: "https://x.com", HealthEndpoint: "/h"}); len(got) != 0 {
		t.Errorf("all fields present: got %v, want none", got)
	}
	if got := checkRequired(Manifest{}); len(got) != 4 {
		t.Errorf("empty manifest: got %d issues, want 4", len(got))
	}
}

func TestCheckOwnerEmail(t *testing.T) {
	if got := checkOwnerEmail(Manifest{Owner: "team@example.com"}); len(got) != 0 {
		t.Errorf("valid email: got %v, want none", got)
	}
	if got := checkOwnerEmail(Manifest{Owner: "team payments"}); len(got) != 1 {
		t.Errorf("invalid email: got %d issues, want 1", len(got))
	}
}
