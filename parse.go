package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// ParseManifest reads a manifest from JSON or from flat YAML.
// If the text starts with "{", we read it as JSON. Otherwise we read it as YAML.
func ParseManifest(body []byte) (Manifest, error) {
	text := bytes.TrimSpace(body)
	if len(text) == 0 {
		return Manifest{}, errors.New("body is empty")
	}
	if text[0] == '{' {
		return parseJSON(text)
	}
	return parseYAML(string(text))
}

// parseJSON reads a JSON object into a manifest.
func parseJSON(text []byte) (Manifest, error) {
	var m Manifest
	if err := json.Unmarshal(text, &m); err != nil {
		return Manifest{}, fmt.Errorf("invalid JSON: %v", err)
	}
	return m, nil
}

// parseYAML reads simple "key: value" lines into a manifest.
// We only support flat YAML. Nested values and lists give an error.
func parseYAML(text string) (Manifest, error) {
	var m Manifest
	seen := map[string]bool{}

	lines := strings.Split(text, "\n")
	for i, line := range lines {
		lineNo := i + 1
		line = strings.TrimRight(line, "\r")

		// Skip empty lines, comments and the "---" start line.
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || trimmed == "---" {
			continue
		}

		// A line that starts with a space or a "-" is nested YAML or a list.
		if line[0] == ' ' || line[0] == '\t' || strings.HasPrefix(trimmed, "-") {
			return Manifest{}, fmt.Errorf("line %d: only flat 'key: value' lines are supported, nested values and lists are not", lineNo)
		}

		key, value, found := strings.Cut(line, ":")
		if !found {
			return Manifest{}, fmt.Errorf("line %d: expected 'key: value', got '%s'", lineNo, trimmed)
		}
		key = strings.TrimSpace(key)
		value = cleanValue(value)

		if seen[key] {
			return Manifest{}, fmt.Errorf("line %d: '%s' is written more than once", lineNo, key)
		}
		seen[key] = true

		switch key {
		case "name":
			m.Name = value
		case "owner":
			m.Owner = value
		case "url":
			m.URL = value
		case "healthEndpoint":
			m.HealthEndpoint = value
		case "version":
			m.Version = value
		case "description":
			m.Description = value
		default:
			// We ignore keys we do not know, like JSON does.
		}
	}
	return m, nil
}

// cleanValue removes spaces, quotes and a comment at the end of a value.
func cleanValue(value string) string {
	value = strings.TrimSpace(value)

	// A value in quotes: keep what is inside the quotes.
	if len(value) >= 2 {
		first, last := value[0], value[len(value)-1]
		if (first == '"' || first == '\'') && first == last {
			return value[1 : len(value)-1]
		}
	}

	// A value without quotes: " #" starts a comment.
	if i := strings.Index(value, " #"); i >= 0 {
		value = strings.TrimSpace(value[:i])
	}
	return value
}
