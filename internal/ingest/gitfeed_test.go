package ingest

import (
	"strings"
	"testing"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/risk"
)

func TestAgentFromTrailers(t *testing.T) {
	cases := []struct {
		subject, body, author string
		want                  model.Agent
	}{
		{"fix login", "Co-authored-by: Copilot <bot@github.com>", "Alice", model.AgentCopilot},
		{"wip", "Generated with Cursor", "Bob", model.AgentCursor},
		{"refactor", "Co-authored-by: Claude <claude@anthropic.com>", "", model.AgentClaudeCode},
		{"fmt", "Co-authored-by: aider", "", model.AgentAider},
		{"", "Generated with Codex", "", model.AgentCodex},
		{"", "Windsurf cascade", "", model.AgentWindsurf},
		{"chore", "Generated with an assistant", "", model.AgentUnknownAgent},
		{"plain fix", "no trailer here", "Jane", ""},
	}
	for _, tc := range cases {
		got := agentFromTrailers(tc.subject, tc.body, tc.author)
		if got != tc.want {
			t.Fatalf("agentFromTrailers(%q, %q, %q)=%q want %q", tc.subject, tc.body, tc.author, got, tc.want)
		}
	}
}

func TestSessionFromCommitWhyAndRisk(t *testing.T) {
	c := gitx.Commit{
		Hash:    "abc123def456abc123def456abc123def456abcd",
		Author:  "Alice",
		Time:    time.Date(2026, 8, 27, 10, 0, 0, 0, time.UTC),
		Subject: "rotate secrets",
		Body:    "Replace leaked .env values.",
		Files: []gitx.FileStat{
			{Path: ".env", Added: 2, Deleted: 2},
			{Path: "foo_test.go", Added: 0, Deleted: 20, Delete: true},
		},
	}
	sess := sessionFromCommit(c, "/repo", "main")
	if sess.ID != "git:"+c.Hash || sess.Source != model.SourceCommit {
		t.Fatalf("id/source %s %s", sess.ID, sess.Source)
	}
	if sess.Agent != model.AgentGit {
		t.Fatalf("agent %s", sess.Agent)
	}
	if sess.Intent != "rotate secrets" {
		t.Fatalf("intent %q", sess.Intent)
	}
	if !strings.Contains(sess.Why, "leaked") {
		t.Fatalf("why %q", sess.Why)
	}
	if sess.AddedLines != 2 || sess.DeletedLines != 22 {
		t.Fatalf("lines +%d -%d", sess.AddedLines, sess.DeletedLines)
	}
	if !has(sess.Risks, "secrets") || !has(sess.Risks, "tests-deleted") {
		t.Fatalf("risks %v", sess.Risks)
	}
	// same chips as calling risk on the git-derived list
	again := risk.Chips(sess.Files, sess.AddedLines, sess.DeletedLines)
	if len(again) == 0 {
		t.Fatal("risk.Chips empty on git files")
	}
}

func TestSessionFromCommitEmptyWhy(t *testing.T) {
	c := gitx.Commit{
		Hash:    "ffffffffffffffffffffffffffffffffffffffff",
		Subject: "typo",
		Files:   []gitx.FileStat{{Path: "a.go", Added: 1}},
	}
	sess := sessionFromCommit(c, "/repo", "main")
	if sess.Why != "" {
		t.Fatalf("expected blank why, got %q", sess.Why)
	}
}

func TestRepoFingerprintStable(t *testing.T) {
	a := repoFingerprint("/Users/you/code/app")
	b := repoFingerprint("/Users/you/code/app")
	if a == "" || a != b {
		t.Fatalf("%q %q", a, b)
	}
	if liveID("/r") != "live:"+repoFingerprint("/r") {
		t.Fatal("live id")
	}
}

func has(ss []string, w string) bool {
	for _, s := range ss {
		if s == w {
			return true
		}
	}
	return false
}
