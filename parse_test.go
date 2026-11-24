package main

import (
	"strings"
	"testing"
)

func TestParseManifestGood(t *testing.T) {
	cases := []struct {
		name string
		body string
		want Manifest
	}{
		{
			"json",
			`{"name": "billing", "owner": "a@b.com", "url": "https://x.com", "healthEndpoint": "/health", "version": "1.2.0"}`,
			Manifest{Name: "billing", Owner: "a@b.com", URL: "https://x.com", HealthEndpoint: "/health", Version: "1.2.0"},
		},
		{
			"yaml",
			"name: billing\nowner: a@b.com\nurl: https://x.com\nhealthEndpoint: /health\nversion: 1.2.0\n",
			Manifest{Name: "billing", Owner: "a@b.com", URL: "https://x.com", HealthEndpoint: "/health", Version: "1.2.0"},
		},
		{
			"yaml with windows line endings",
			"name: billing\r\nowner: a@b.com\r\n",
			Manifest{Name: "billing", Owner: "a@b.com"},
		},
		{
			"yaml with comments, empty lines and ---",
			"---\n# my service\nname: billing\n\nowner: a@b.com # the team\n",
			Manifest{Name: "billing", Owner: "a@b.com"},
		},
		{
			"yaml with quotes",
			"name: \"billing\"\ndescription: 'Handles # invoices'\n",
			Manifest{Name: "billing", Description: "Handles # invoices"},
		},
		{
			"url with a port keeps all its colons",
			"url: https://x.com:8443/api\n",
			Manifest{URL: "https://x.com:8443/api"},
		},
		{
			"unknown keys are ignored",
			"name: billing\nteam: payments\n",
			Manifest{Name: "billing"},
		},
		{
			"the README example is read, even with bad values",
			"name: Billing Service\nowner: team payments\nurl: http://billing.example.com\nhealthEndpoint: health\nversion: 1.2\n",
			Manifest{Name: "Billing Service", Owner: "team payments", URL: "http://billing.example.com", HealthEndpoint: "health", Version: "1.2"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got, err := ParseManifest([]byte(c.body))
			if err != nil {
				t.Fatalf("got error %v, want none", err)
			}
			if got != c.want {
				t.Errorf("got %+v, want %+v", got, c.want)
			}
		})
	}
}

func TestParseManifestBad(t *testing.T) {
	cases := []struct {
		name      string
		body      string
		wantInErr string // a part of the error message we expect
	}{
		{"empty body", "", "body is empty"},
		{"only spaces", "   \n  ", "body is empty"},
		{"broken json", `{"name": `, "invalid JSON"},
		{"json with wrong type", `{"name": 5}`, "invalid JSON"},
		{"nested yaml", "owner:\n  email: a@b.com\n", "line 2"},
		{"yaml list", "- name: billing\n", "line 1"},
		{"line without colon", "name billing\n", "expected 'key: value'"},
		{"key written twice", "name: a\nname: b\n", "more than once"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseManifest([]byte(c.body))
			if err == nil {
				t.Fatalf("got no error, want one with %q", c.wantInErr)
			}
			if !strings.Contains(err.Error(), c.wantInErr) {
				t.Errorf("got error %q, want it to contain %q", err, c.wantInErr)
			}
		})
	}
}
