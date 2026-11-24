package main

import (
	"strings"
	"testing"
)

// ruleCase is one test: a manifest and how many problems we expect.
type ruleCase struct {
	name       string
	manifest   Manifest
	wantIssues int
}

// runRule runs one rule on each test case.
func runRule(t *testing.T, r rule, cases []ruleCase) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := r(c.manifest)
			if len(got) != c.wantIssues {
				t.Errorf("got %d issues %v, want %d", len(got), got, c.wantIssues)
			}
		})
	}
}

func TestCheckRequired(t *testing.T) {
	runRule(t, checkRequired, []ruleCase{
		{"all fields present", Manifest{Name: "a", Owner: "a@b.com", URL: "https://x.com", HealthEndpoint: "/h"}, 0},
		{"empty manifest", Manifest{}, 4},
		{"only spaces", Manifest{Name: " ", Owner: "a@b.com", URL: "https://x.com", HealthEndpoint: "/h"}, 1},
	})
}

func TestCheckName(t *testing.T) {
	runRule(t, checkName, []ruleCase{
		{"simple name", Manifest{Name: "billing"}, 0},
		{"with hyphen and digits", Manifest{Name: "billing-service-2"}, 0},
		{"exactly 40 characters", Manifest{Name: strings.Repeat("a", 40)}, 0},
		{"41 characters", Manifest{Name: strings.Repeat("a", 41)}, 1},
		{"capital letters and space", Manifest{Name: "Billing Service"}, 1},
		{"starts with hyphen", Manifest{Name: "-billing"}, 1},
		{"ends with hyphen", Manifest{Name: "billing-"}, 1},
		{"double hyphen", Manifest{Name: "billing--service"}, 1},
		{"underscore", Manifest{Name: "billing_service"}, 1},
		{"too long and wrong letters", Manifest{Name: strings.Repeat("A", 41)}, 2},
		{"empty is skipped", Manifest{Name: ""}, 0},
	})
}

func TestCheckOwnerEmail(t *testing.T) {
	runRule(t, checkOwnerEmail, []ruleCase{
		{"valid email", Manifest{Owner: "team@example.com"}, 0},
		{"not an email", Manifest{Owner: "team payments"}, 1},
		{"display name form", Manifest{Owner: "Team <team@example.com>"}, 1},
		{"empty is skipped", Manifest{Owner: ""}, 0},
	})
}

func TestCheckURL(t *testing.T) {
	runRule(t, checkURL, []ruleCase{
		{"https", Manifest{URL: "https://billing.example.com"}, 0},
		{"https with path", Manifest{URL: "https://example.com/billing"}, 0},
		{"http is not allowed", Manifest{URL: "http://billing.example.com"}, 1},
		{"no scheme", Manifest{URL: "billing.example.com"}, 1},
		{"no host", Manifest{URL: "https://"}, 1},
		{"not a url", Manifest{URL: "hello world"}, 1},
		{"empty is skipped", Manifest{URL: ""}, 0},
	})
}

func TestCheckHealthEndpoint(t *testing.T) {
	runRule(t, checkHealthEndpoint, []ruleCase{
		{"starts with slash", Manifest{HealthEndpoint: "/health"}, 0},
		{"missing slash", Manifest{HealthEndpoint: "health"}, 1},
		{"empty is skipped", Manifest{HealthEndpoint: ""}, 0},
	})
}

func TestCheckVersion(t *testing.T) {
	runRule(t, checkVersion, []ruleCase{
		{"simple version", Manifest{Version: "1.2.0"}, 0},
		{"zero version", Manifest{Version: "0.0.1"}, 0},
		{"with extra part", Manifest{Version: "1.2.0-beta.1"}, 0},
		{"with build part", Manifest{Version: "1.2.0+build5"}, 0},
		{"only two numbers", Manifest{Version: "1.2"}, 1},
		{"starts with v", Manifest{Version: "v1.2.0"}, 1},
		{"leading zero", Manifest{Version: "01.2.0"}, 1},
		{"letters", Manifest{Version: "latest"}, 1},
		{"empty is fine", Manifest{Version: ""}, 0},
	})
}

func TestCheckDescription(t *testing.T) {
	runRule(t, checkDescription, []ruleCase{
		{"short", Manifest{Description: "Handles invoices."}, 0},
		{"199 characters", Manifest{Description: strings.Repeat("a", 199)}, 0},
		{"200 characters", Manifest{Description: strings.Repeat("a", 200)}, 1},
		{"empty is fine", Manifest{Description: ""}, 0},
	})
}

func TestDescriptionIsOnlyAWarning(t *testing.T) {
	got := checkDescription(Manifest{Description: strings.Repeat("a", 300)})
	if len(got) != 1 || got[0].Severity != "warning" {
		t.Errorf("got %v, want one warning", got)
	}
}

func TestValidateRunsAllRules(t *testing.T) {
	// This is the example from the README. Every field has one problem.
	m := Manifest{
		Name:           "Billing Service",
		Owner:          "team payments",
		URL:            "http://billing.example.com",
		HealthEndpoint: "health",
		Version:        "1.2",
	}
	got := Validate(m)
	want := []string{"name", "owner", "url", "healthEndpoint", "version"}
	if len(got) != len(want) {
		t.Fatalf("got %d issues %v, want %d", len(got), got, len(want))
	}
	for i, field := range want {
		if got[i].Field != field {
			t.Errorf("issue %d: got field %q, want %q", i, got[i].Field, field)
		}
	}
}

func TestValidateGoodManifest(t *testing.T) {
	m := Manifest{
		Name:           "billing-service",
		Owner:          "payments@example.com",
		URL:            "https://billing.example.com",
		HealthEndpoint: "/health",
		Version:        "1.2.0",
		Description:    "Handles invoices.",
	}
	if got := Validate(m); len(got) != 0 {
		t.Errorf("got %v, want no issues", got)
	}
}
