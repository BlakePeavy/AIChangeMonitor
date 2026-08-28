package ingest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/paths"
)

func TestClaudeFixture(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "claude", "sample.jsonl")
	repo := "/Users/you/code/my-app"
	sess, ok := parseClaudeFile(p, repo, 1)
	if !ok {
		t.Fatal("parse failed")
	}
	if sess.Agent != model.AgentClaudeCode {
		t.Fatalf("agent %s", sess.Agent)
	}
	if sess.ID != "11111111-1111-1111-1111-111111111111" {
		t.Fatalf("id %s", sess.ID)
	}
	if sess.Branch != "main" {
		t.Fatalf("branch %s", sess.Branch)
	}
	if !strings.Contains(sess.Intent, "rate limiter") {
		t.Fatalf("intent %q", sess.Intent)
	}
	if strings.Contains(sess.Intent, "sk-abcdefghijklmnopqrstuvwxyz123456") {
		t.Fatalf("secret leaked in intent: %q", sess.Intent)
	}
	if !strings.Contains(sess.Why, "token-bucket") && !strings.Contains(sess.Why, "rate limiter") {
		t.Fatalf("why %q", sess.Why)
	}
	pathsGot := sess.FilePaths()
	if !contains(pathsGot, "internal/ratelimit/limiter.go") {
		t.Fatalf("files %v", pathsGot)
	}
	if !contains(pathsGot, "cmd/api/main.go") {
		t.Fatalf("files %v", pathsGot)
	}
	var sawDel bool
	for _, f := range sess.Files {
		if strings.Contains(f.Path, "old_test.go") && f.Delete {
			sawDel = true
		}
	}
	if !sawDel {
		t.Fatalf("expected bash rm join, files=%v", sess.Files)
	}
}

func contains(ss []string, want string) bool {
	for _, s := range ss {
		if s == want || strings.HasSuffix(s, want) {
			return true
		}
	}
	return false
}

func TestEncodeMatchesFixtureRepo(t *testing.T) {
	repo := "/Users/you/code/my-app"
	if !paths.MatchProjectDir(paths.EncodeCWD(repo), repo) {
		t.Fatal("encode/match")
	}
}

func TestReadMissingIsFalse(t *testing.T) {
	if _, ok := parseClaudeFile(filepath.Join(os.TempDir(), "nope.jsonl"), "/repo", 0); ok {
		t.Fatal("missing should fail")
	}
}
