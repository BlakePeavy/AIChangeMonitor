package ingest

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func TestCursorFixture(t *testing.T) {
	p := filepath.Join("..", "..", "testdata", "cursor", "sample.jsonl")
	repo := "/Users/you/code/my-app"
	sess, ok := parseCursorFile(p, repo, 2)
	if !ok {
		t.Fatal("parse failed")
	}
	if sess.Agent != model.AgentCursor {
		t.Fatalf("agent %s", sess.Agent)
	}
	if !strings.Contains(sess.Intent, "Rename the limiter") {
		t.Fatalf("intent %q", sess.Intent)
	}
	if strings.Contains(sess.Intent, "<user_query>") {
		t.Fatalf("user_query wrapper leaked: %q", sess.Intent)
	}
	if !strings.Contains(sess.Why, "rename") && !strings.Contains(strings.ToLower(sess.Why), "test") {
		t.Fatalf("why %q", sess.Why)
	}
	var sawEnv, sawTest bool
	for _, f := range sess.Files {
		if f.Path == ".env" && f.Delete {
			sawEnv = true
		}
		if strings.Contains(f.Path, "limiter_test.go") {
			sawTest = true
		}
	}
	if !sawEnv || !sawTest {
		t.Fatalf("files %+v", sess.Files)
	}
}

func TestWhyExcerptAndJoin(t *testing.T) {
	text := whyExcerpt([]string{"one\n", "two\n", "three\nfour\nfive"})
	if !strings.Contains(text, "one") || !strings.Contains(text, "three") {
		t.Fatalf("excerpt %q", text)
	}
	ops := []model.File{
		{Path: "a.go", Prompt: "first", Tool: "Write"},
		{Path: "a.go", Delete: true, Prompt: "second"},
		{Path: "b.go", Tool: "Edit"},
	}
	got := joinFiles(ops)
	if len(got) != 2 {
		t.Fatalf("len %d", len(got))
	}
	if got[0].Path != "a.go" || !got[0].Delete || got[0].Prompt != "first" {
		t.Fatalf("merge %+v", got[0])
	}
}

func TestMalformedLineSkipped(t *testing.T) {
	ev, ok := parseLine([]byte("not json"))
	if ok {
		t.Fatalf("wanted skip, got %+v", ev)
	}
	ev, ok = parseLine([]byte(`{"role":"user","message":{"content":[{"type":"text","text":"hi"}]}}`))
	if !ok || !ev.Human {
		t.Fatalf("wanted human user, got ok=%v %+v", ok, ev)
	}
}
