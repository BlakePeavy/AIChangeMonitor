package paths

import (
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestEncodeCWD(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"/Users/you/code/my-app", "-Users-you-code-my-app"},
		{"/home/blake/foo", "-home-blake-foo"},
		{`C:\Users\blake\source\vscode\AIChangeMonitor`, "C--Users-blake-source-vscode-AIChangeMonitor"},
		{"C:/Users/blake/source/vscode/AIChangeMonitor", "C--Users-blake-source-vscode-AIChangeMonitor"},
		{"", ""},
		{"/tmp/a_b", "-tmp-a-b"},
	}
	for _, tt := range tests {
		if got := EncodeCWD(tt.in); got != tt.want {
			t.Errorf("EncodeCWD(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMatchProjectDir(t *testing.T) {
	tests := []struct {
		folder, repo string
		want         bool
	}{
		{"-Users-you-code-my-app", "/Users/you/code/my-app", true},
		{"Users-you-code-my-app", "/Users/you/code/my-app", true},
		{"-home-blake-foo", "/home/blake/foo", true},
		{"C--Users-blake-source-vscode-AIChangeMonitor", `C:\Users\blake\source\vscode\AIChangeMonitor`, true},
		{"c--users-blake-source-vscode-aichangemonitor", `C:\Users\blake\source\vscode\AIChangeMonitor`, true},
		{"c-Users-blake-source-vscode-AIChangeMonitor", `C:\Users\blake\source\vscode\AIChangeMonitor`, true},
		{"-Users-you-code-other", "/Users/you/code/my-app", false},
		{"", "/Users/you/code/my-app", false},
		{"-Users-you-code-my-app", "", false},
	}
	for _, tt := range tests {
		if got := MatchProjectDir(tt.folder, tt.repo); got != tt.want {
			t.Errorf("MatchProjectDir(%q, %q) = %v, want %v", tt.folder, tt.repo, got, tt.want)
		}
	}
}

func TestMatchSessionCWD(t *testing.T) {
	repo := "/Users/you/code/my-app"
	if !MatchSessionCWD(repo, repo) {
		t.Fatal("same path should match")
	}
	if !MatchSessionCWD(repo+"/cmd/api", repo) {
		t.Fatal("subdir should match")
	}
	if MatchSessionCWD("/Users/you/code/other", repo) {
		t.Fatal("sibling should not match")
	}
	if MatchSessionCWD("", repo) {
		t.Fatal("empty cwd should not match")
	}
}

func TestSamePathAndUnder(t *testing.T) {
	if !Under("/repo/internal/x.go", "/repo") {
		t.Fatal("file under repo")
	}
	if Under("/repo-other/x", "/repo") {
		t.Fatal("prefix-not-boundary must not match")
	}
	if !SamePath("/tmp/foo", "/tmp/foo") {
		t.Fatal("same")
	}
}

func TestRelToRepo(t *testing.T) {
	got := RelToRepo("/Users/you/code/my-app/internal/x.go", "/Users/you/code/my-app")
	if got != "internal/x.go" {
		t.Fatalf("RelToRepo = %q", got)
	}
	got = RelToRepo("/elsewhere/a.go", "/Users/you/code/my-app")
	if !strings.Contains(got, "elsewhere") {
		t.Fatalf("outside path should stay, got %q", got)
	}
}

func TestIndexPathOverride(t *testing.T) {
	t.Setenv("AICHANGE_INDEX", filepath.Join(t.TempDir(), "idx.db"))
	p := IndexPath()
	if !strings.HasSuffix(p, "idx.db") {
		t.Fatalf("override ignored: %s", p)
	}
}

func TestIndexPathDefaultShape(t *testing.T) {
	t.Setenv("AICHANGE_INDEX", "")
	p := IndexPath()
	if runtime.GOOS == "windows" {
		if !strings.Contains(strings.ToLower(p), `aichange\index.db`) && !strings.Contains(p, "aichange/index.db") {
			t.Fatalf("windows index: %s", p)
		}
	} else {
		if !strings.HasSuffix(filepath.ToSlash(p), "aichange/index.db") {
			t.Fatalf("unix index: %s", p)
		}
	}
}

func TestClaudeCursorOverrides(t *testing.T) {
	t.Setenv("AICHANGE_CLAUDE_PROJECTS", "/tmp/claude-projects")
	t.Setenv("AICHANGE_CURSOR_PROJECTS", "/tmp/cursor-projects")
	got := ClaudeProjectsDirs()
	if len(got) != 1 || got[0] != "/tmp/claude-projects" {
		t.Fatalf("claude override: %v", got)
	}
	if CursorProjectsDir() != "/tmp/cursor-projects" {
		t.Fatalf("cursor override: %s", CursorProjectsDir())
	}
}
