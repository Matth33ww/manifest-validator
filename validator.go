package main

import (
	"net/mail"
	"strings"
)

type Manifest struct {
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	URL            string `json:"url"`
	HealthEndpoint string `json:"healthEndpoint"`
}

type Issue struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
}

// A rule checks one thing about a manifest and returns the problems it finds.
type rule func(Manifest) []Issue

// rules is every check Validate runs, in order. To add a rule, append it here.
var rules = []rule{
	checkRequired,
	checkOwnerEmail,
	checkHealthEndpoint,
}

// Validate runs every rule and returns all problems found.
func Validate(m Manifest) []Issue {
	var issues []Issue
	for _, r := range rules {
		issues = append(issues, r(m)...)
	}
	return issues
}

// Rule 1: required fields must not be empty.
func checkRequired(m Manifest) []Issue {
	fields := []struct {
		name  string
		value string
	}{
		{"name", m.Name},
		{"owner", m.Owner},
		{"url", m.URL},
		{"healthEndpoint", m.HealthEndpoint},
	}

	var issues []Issue
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			issues = append(issues, Issue{
				Field:    f.name,
				Message:  f.name + " is required",
				Severity: "error",
			})
		}
	}
	return issues
}

// Rule 2: owner must be a plain email address.
func checkOwnerEmail(m Manifest) []Issue {
	if strings.TrimSpace(m.Owner) == "" {
		return nil // already reported by checkRequired
	}
	addr, err := mail.ParseAddress(m.Owner)
	if err != nil || addr.Address != m.Owner {
		return []Issue{{
			Field:    "owner",
			Message:  "owner must be a valid email address, got '" + m.Owner + "'",
			Severity: "error",
		}}
	}
	return nil
}

// Rule 3: healthEndpoint must start with "/".
func checkHealthEndpoint(m Manifest) []Issue {
	if strings.TrimSpace(m.HealthEndpoint) == "" {
		return nil // already reported by checkRequired
	}
	if !strings.HasPrefix(m.HealthEndpoint, "/") {
		return []Issue{{
			Field:    "healthEndpoint",
			Message:  "healthEndpoint must start with '/', got '" + m.HealthEndpoint + "'",
			Severity: "error",
		}}
	}
	return nil
}
