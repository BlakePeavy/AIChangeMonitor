package gitx

import (
	"strings"
	"testing"
	"time"
)

const sampleLog = "" +
	"\x1eaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\x00Alice\x002026-08-27T10:00:00-05:00\x00fix login rate limit\x00Use a token bucket.\n\nCo-authored-by: Copilot <noreply>\x00\n" +
	"12\t3\tinternal/auth/login.go\n" +
	"0\t8\tinternal/auth/old.go\n" +
	"4\t0\tui/login.js\n" +
	"\n" +
	"\x1ebbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb\x00Bob\x002026-08-26T09:15:00Z\x00docs: tweak README\x00\x00\n" +
	"3\t1\tREADME.md\n" +
	"-\t-\tlogo.png\n"

func TestParseLogNumstat(t *testing.T) {
	got := ParseLogNumstat(sampleLog)
	if len(got) != 2 {
		t.Fatalf("commits %d: %+v", len(got), got)
	}
	a := got[0]
	if a.Hash != "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa" {
		t.Fatalf("hash %s", a.Hash)
	}
	if a.Author != "Alice" || a.Subject != "fix login rate limit" {
		t.Fatalf("meta %+v", a)
	}
	if !strings.Contains(a.Body, "token bucket") || !strings.Contains(a.Body, "Copilot") {
		t.Fatalf("body %q", a.Body)
	}
	if a.Time.Year() != 2026 || a.Time.Month() != time.August {
		t.Fatalf("time %v", a.Time)
	}
	if len(a.Files) != 3 {
		t.Fatalf("files %+v", a.Files)
	}
	if a.Files[0].Path != "internal/auth/login.go" || a.Files[0].Added != 12 || a.Files[0].Deleted != 3 {
		t.Fatalf("f0 %+v", a.Files[0])
	}
	if a.Files[1].Path != "internal/auth/old.go" || !a.Files[1].Delete || a.Files[1].Added != 0 {
		t.Fatalf("deleted %+v", a.Files[1])
	}
	b := got[1]
	if b.Subject != "docs: tweak README" || b.Body != "" {
		t.Fatalf("b %+v", b)
	}
	if len(b.Files) != 2 || b.Files[1].Path != "logo.png" {
		t.Fatalf("b files %+v", b.Files)
	}
}

func TestParseNumstatRename(t *testing.T) {
	got := ParseNumstat("10\t2\told.go => new.go\n")
	if len(got) != 1 || got[0].Path != "new.go" || got[0].Added != 10 {
		t.Fatalf("%+v", got)
	}
}
