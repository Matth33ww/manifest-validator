package main

import (
	"fmt"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

type Manifest struct {
	Name           string `json:"name"`
	Owner          string `json:"owner"`
	URL            string `json:"url"`
	HealthEndpoint string `json:"healthEndpoint"`
	Version        string `json:"version"`     // optional
	Description    string `json:"description"` // optional
}

type Issue struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error" or "warning"
}

// A rule checks a manifest and returns the problems.
type rule func(Manifest) []Issue

// rules is the list of all checks. To add a new check, add it here.
var rules = []rule{
	checkRequired,
	checkName,
	checkOwnerEmail,
	checkURL,
	checkHealthEndpoint,
	checkVersion,
	checkDescription,
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

// namePattern means: lowercase letters and digits, joined by single hyphens.
var namePattern = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

const maxNameLength = 40

// Rule 2: name must use only lowercase letters, digits and single hyphens.
// It can be at most 40 characters long.
func checkName(m Manifest) []Issue {
	if strings.TrimSpace(m.Name) == "" {
		return nil // checkRequired already reports this
	}
	var issues []Issue
	if !namePattern.MatchString(m.Name) {
		issues = append(issues, Issue{
			Field:    "name",
			Message:  "name may only contain lowercase letters, digits and single hyphens (not at the start or end), got '" + m.Name + "'",
			Severity: "error",
		})
	}
	if n := utf8.RuneCountInString(m.Name); n > maxNameLength {
		issues = append(issues, Issue{
			Field:    "name",
			Message:  fmt.Sprintf("name must be at most %d characters, got %d", maxNameLength, n),
			Severity: "error",
		})
	}
	return issues
}

// Rule 3: owner must be a plain email address.
func checkOwnerEmail(m Manifest) []Issue {
	if strings.TrimSpace(m.Owner) == "" {
		return nil // checkRequired already reports this
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

// Rule 4: url must be a full HTTPS address, like https://example.com.
func checkURL(m Manifest) []Issue {
	if strings.TrimSpace(m.URL) == "" {
		return nil // checkRequired already reports this
	}
	u, err := url.Parse(m.URL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return []Issue{{
			Field:    "url",
			Message:  "url must be a valid HTTPS URL (https://...), got '" + m.URL + "'",
			Severity: "error",
		}}
	}
	return nil
}

// Rule 5: healthEndpoint must start with "/".
func checkHealthEndpoint(m Manifest) []Issue {
	if strings.TrimSpace(m.HealthEndpoint) == "" {
		return nil // checkRequired already reports this
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

// versionPattern is a semantic version: three numbers like 1.2.0.
// Numbers can not start with 0 (except 0 itself).
// It can also have an extra part, like 1.2.0-beta or 1.2.0+build5.
var versionPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\+[0-9A-Za-z.-]+)?$`)

// Rule 6: version is optional. If it is there, it must be a semantic version.
func checkVersion(m Manifest) []Issue {
	if strings.TrimSpace(m.Version) == "" {
		return nil // version is optional
	}
	if !versionPattern.MatchString(m.Version) {
		return []Issue{{
			Field:    "version",
			Message:  "version must follow semantic versioning (for example 1.2.0), got '" + m.Version + "'",
			Severity: "error",
		}}
	}
	return nil
}

const maxDescriptionLength = 200

// Rule 7: description is optional. If it is 200 characters or more, we warn.
// A warning does not make the manifest invalid.
func checkDescription(m Manifest) []Issue {
	if n := utf8.RuneCountInString(m.Description); n >= maxDescriptionLength {
		return []Issue{{
			Field:    "description",
			Message:  fmt.Sprintf("description should be under %d characters, got %d", maxDescriptionLength, n),
			Severity: "warning",
		}}
	}
	return nil
}
