package redact

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	aws := "AKIA" + "IOSFODNN7EXAMPLE"
	pemBody := "MIIEowIBAAKCAQEA"
	openai := "sk-" + "abcdefghijklmnopqrstuvwxyz123456"
	gh := "ghp_" + "abcdefghijklmnopqrstuvwxyz1234"

	tests := []struct {
		name string
		in   string
		hide []string
		keep []string
	}{
		{
			name: "aws key",
			in:   "export key=" + aws + " and keep this",
			hide: []string{aws},
			keep: []string{"keep this", "[REDACTED]"},
		},
		{
			name: "pem",
			in:   "here\n-----BEGIN RSA PRIVATE KEY-----\n" + pemBody + "\n-----END RSA PRIVATE KEY-----\ndone",
			hide: []string{pemBody},
			keep: []string{"[REDACTED PEM]", "done"},
		},
		{
			name: "env assignment",
			in:   "OPENAI_API_KEY=" + openai + " and ok",
			hide: []string{openai},
			keep: []string{"OPENAI_API_KEY=", "ok"},
		},
		{
			name: "generic password",
			in:   "password=hunter2 leftover",
			hide: []string{"hunter2"},
			keep: []string{"password=", "leftover"},
		},
		{
			name: "github pat",
			in:   "token " + gh + " end",
			hide: []string{gh},
			keep: []string{"end"},
		},
		{
			name: "plain prose kept",
			in:   "Add a rate limiter to the API please",
			hide: nil,
			keep: []string{"Add a rate limiter to the API please"},
		},
		{
			name: "git sha kept",
			in:   "commit deadbeefdeadbeefdeadbeefdeadbeefdeadbeef on main",
			hide: nil,
			keep: []string{"deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Redact(tt.in)
			for _, h := range tt.hide {
				if strings.Contains(got, h) {
					t.Errorf("still contains %q: %q", h, got)
				}
			}
			for _, k := range tt.keep {
				if !strings.Contains(got, k) {
					t.Errorf("missing %q: %q", k, got)
				}
			}
		})
	}
}

func TestHighEntropy(t *testing.T) {
	tok := "aB3dE5fG7hI9jK1lM2nO3pQ4rS5tU6vW"
	got := Redact("hdr " + tok + " tail")
	if strings.Contains(got, tok) {
		t.Fatalf("entropy token survived: %q", got)
	}
	if !strings.Contains(got, "hdr ") || !strings.Contains(got, " tail") {
		t.Fatalf("surroundings lost: %q", got)
	}
}
