package server

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/store"
)

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=aichange",
			"GIT_AUTHOR_EMAIL=aichange@test",
			"GIT_COMMITTER_NAME=aichange",
			"GIT_COMMITTER_EMAIL=aichange@test",
			"GIT_CONFIG_NOSYSTEM=1",
			"GIT_TERMINAL_PROMPT=0",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("init")
	run("config", "user.email", "aichange@test")
	run("config", "user.name", "aichange")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	run("add", "README")
	run("commit", "-m", "init")
	root, err := gitx.RepoRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Clean(root)
}

func testStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "idx.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = st.Close() })
	return st
}

func postJSON(h http.Handler, path string, body any) *httptest.ResponseRecorder {
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	return rr
}

func TestSetRepo(t *testing.T) {
	a := initRepo(t)
	b := initRepo(t)
	st := testStore(t)
	s := New(st, a, 0)
	h := s.Handler()

	rr := postJSON(h, "/api/repo", map[string]string{"path": t.TempDir()})
	if rr.Code != 400 {
		t.Fatalf("nongit: %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "not a git repo") {
		t.Fatalf("nongit msg: %s", rr.Body.String())
	}

	rr = postJSON(h, "/api/repo", map[string]string{"path": ""})
	if rr.Code != 400 {
		t.Fatalf("empty: %d %s", rr.Code, rr.Body.String())
	}

	req := httptest.NewRequest(http.MethodGet, "/api/repo", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 405 {
		t.Fatalf("GET: %d %s", rr.Code, rr.Body.String())
	}

	if s.repo() != a {
		t.Fatalf("still on a before switch: %q", s.repo())
	}

	rr = postJSON(h, "/api/repo", map[string]string{"path": b})
	if rr.Code != 200 {
		t.Fatalf("switch: %d %s", rr.Code, rr.Body.String())
	}
	var got map[string]any
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["repo"] != b {
		t.Fatalf("repo %v want %s", got["repo"], b)
	}
	if got["branch"] == "" {
		t.Fatal("missing branch")
	}
	if s.repo() != b {
		t.Fatalf("Server.Repo %q want %q", s.repo(), b)
	}

	req = httptest.NewRequest(http.MethodGet, "/api/git", nil)
	rr = httptest.NewRecorder()
	h.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("git: %d %s", rr.Code, rr.Body.String())
	}
	got = map[string]any{}
	if err := json.Unmarshal(rr.Body.Bytes(), &got); err != nil {
		t.Fatal(err)
	}
	if got["repo"] != b {
		t.Fatalf("git repo %v want %s", got["repo"], b)
	}

	s.scanOnce(true)
	if s.repo() != b {
		t.Fatalf("poll still %q", s.repo())
	}
}
