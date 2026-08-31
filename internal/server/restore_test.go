package server

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/ingest"
)

func TestRestoreAllowed(t *testing.T) {
	cases := []struct {
		name     string
		source   string
		id       string
		hasUp    bool
		ancestor bool
		ok       bool
		reason   string
	}{
		{name: "live source", source: "live", id: "live:abc", ok: true},
		{name: "live id prefix", source: "", id: "live:abc", ok: true},
		{name: "commit no upstream", source: "commit", id: "git:deadbeef", hasUp: false, ancestor: false, ok: true},
		{name: "commit no upstream ignore ancestor", source: "commit", id: "git:deadbeef", hasUp: false, ancestor: true, ok: true},
		{name: "commit unpushed", source: "commit", id: "git:deadbeef", hasUp: true, ancestor: false, ok: true},
		{name: "commit already remote", source: "commit", id: "git:deadbeef", hasUp: true, ancestor: true, ok: false, reason: errAlreadyRemote},
		{name: "commit id prefix unpushed", source: "", id: "git:abc", hasUp: true, ancestor: false, ok: true},
		{name: "transcript", source: "transcript", id: "sess-1", ok: false, reason: errNotRestorable},
		{name: "unknown id", source: "", id: "abc", ok: false, reason: errNotRestorable},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, reason := restoreAllowed(tc.source, tc.id, tc.hasUp, tc.ancestor)
			if ok != tc.ok || reason != tc.reason {
				t.Fatalf("restoreAllowed(%q,%q,up=%v,anc=%v)=(%v,%q) want (%v,%q)",
					tc.source, tc.id, tc.hasUp, tc.ancestor, ok, reason, tc.ok, tc.reason)
			}
		})
	}
}

func TestRestoreLiveFile(t *testing.T) {
	dir := initRepo(t)
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("dirty\n"), 0644); err != nil {
		t.Fatal(err)
	}
	st := testStore(t)
	if _, err := ingest.Scan(st, dir); err != nil {
		t.Fatal(err)
	}
	s := New(st, dir, 0)
	list, err := st.List(dir)
	if err != nil {
		t.Fatal(err)
	}
	liveID := ""
	for _, sess := range list {
		if strings.HasPrefix(sess.ID, "live:") {
			liveID = sess.ID
			break
		}
	}
	if liveID == "" {
		t.Fatal("no live session")
	}
	rr := postJSON(s.Handler(), "/api/restore", map[string]string{"id": liveID, "path": "README"})
	if rr.Code != 200 {
		t.Fatalf("live restore: %d %s", rr.Code, rr.Body.String())
	}
	b, err := os.ReadFile(filepath.Join(dir, "README"))
	if err != nil {
		t.Fatal(err)
	}
	if string(b) != "hi\n" {
		t.Fatalf("content %q", b)
	}
}

func TestRestoreCommitAlreadyOnRemote(t *testing.T) {
	dir := initRepo(t)
	bare := filepath.Join(t.TempDir(), "bare.git")
	cmd := exec.Command("git", "init", "--bare", bare)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("bare: %v\n%s", err, out)
	}
	run := func(args ...string) {
		t.Helper()
		c := exec.Command("git", args...)
		c.Dir = dir
		c.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
		if out, err := c.CombinedOutput(); err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
	}
	run("remote", "add", "origin", bare)
	run("branch", "-M", "main")
	run("push", "-u", "origin", "HEAD")
	st := testStore(t)
	if _, err := ingest.Scan(st, dir); err != nil {
		t.Fatal(err)
	}
	s := New(st, dir, 0)
	head := gitx.HEAD(dir)
	rr := postJSON(s.Handler(), "/api/restore", map[string]string{"id": "git:" + head, "path": "README"})
	if rr.Code != 409 {
		t.Fatalf("want 409, got %d %s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "already on the remote") {
		t.Fatalf("body %s", rr.Body.String())
	}
}
