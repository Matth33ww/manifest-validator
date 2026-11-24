package main

import "testing"

// ruleCase is one test case: a manifest and how many issues the rule should report.
type ruleCase struct {
	name       string
	manifest   Manifest
	wantIssues int
}

// runRule runs one rule against a table of cases.
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

func TestCheckOwnerEmail(t *testing.T) {
	runRule(t, checkOwnerEmail, []ruleCase{
		{"valid email", Manifest{Owner: "team@example.com"}, 0},
		{"not an email", Manifest{Owner: "team payments"}, 1},
		{"display name form", Manifest{Owner: "Team <team@example.com>"}, 1},
		{"empty is skipped", Manifest{Owner: ""}, 0},
	})
}

func TestCheckHealthEndpoint(t *testing.T) {
	runRule(t, checkHealthEndpoint, []ruleCase{
		{"starts with slash", Manifest{HealthEndpoint: "/health"}, 0},
		{"missing slash", Manifest{HealthEndpoint: "health"}, 1},
		{"empty is skipped", Manifest{HealthEndpoint: ""}, 0},
	})
}

func TestValidateRunsAllRules(t *testing.T) {
	got := Validate(Manifest{Name: "billing", Owner: "team payments", URL: "https://x.com", HealthEndpoint: "health"})
	if len(got) != 2 {
		t.Errorf("got %d issues %v, want 2 (owner and healthEndpoint)", len(got), got)
	}
}
