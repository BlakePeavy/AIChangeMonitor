package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"unicode"
)

// EncodeCWD encodes an absolute working directory the way Claude Code names
// project folders: every non-alphanumeric rune becomes '-'.
//
//	/Users/you/code/my-app → -Users-you-code-my-app
//	C:\Users\blake\proj    → C--Users-blake-proj
func EncodeCWD(abs string) string {
	if abs == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(abs))
	for _, r := range abs {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		} else {
			b.WriteByte('-')
		}
	}
	return b.String()
}

// Normalize compares paths across OS slash and (on Windows) case differences.
func Normalize(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	p = filepath.Clean(p)
	p = strings.ReplaceAll(p, "\\", "/")
	if len(p) > 1 && strings.HasSuffix(p, "/") {
		p = strings.TrimSuffix(p, "/")
	}
	if runtime.GOOS == "windows" || looksWindows(p) {
		p = strings.ToLower(p)
	}
	return p
}

func looksWindows(p string) bool {
	if len(p) >= 2 && p[1] == ':' {
		return true
	}
	return strings.Contains(p, "\\")
}

// SamePath reports whether a and b name the same directory.
func SamePath(a, b string) bool {
	return Normalize(a) != "" && Normalize(a) == Normalize(b)
}

// Under reports whether child is the same as root or a subdirectory of it.
func Under(child, root string) bool {
	c, r := Normalize(child), Normalize(root)
	if c == "" || r == "" {
		return false
	}
	if c == r {
		return true
	}
	return strings.HasPrefix(c, r+"/")
}

// RelToRepo returns path relative to repo when it lives inside; otherwise the
// cleaned original (slash-separated).
func RelToRepo(path, repo string) string {
	if path == "" {
		return ""
	}
	if repo != "" && Under(path, repo) {
		c, r := Normalize(path), Normalize(repo)
		rel := strings.TrimPrefix(c, r+"/")
		if rel != c {
			return rel
		}
	}
	return filepath.ToSlash(filepath.Clean(path))
}

// EncodeVariants returns encodings we should accept when matching a repo root
// to a Claude/Cursor project folder name.
func EncodeVariants(repo string) []string {
	if repo == "" {
		return nil
	}
	seen := map[string]struct{}{}
	var out []string
	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		out = append(out, s)
		low := strings.ToLower(s)
		if low != s {
			if _, ok := seen[low]; !ok {
				seen[low] = struct{}{}
				out = append(out, low)
			}
		}
	}

	slash := strings.ReplaceAll(filepath.Clean(repo), "\\", "/")
	add(EncodeCWD(repo))
	add(EncodeCWD(slash))
	if strings.HasPrefix(slash, "/") {
		add(EncodeCWD(strings.TrimPrefix(slash, "/")))
	}
	// Windows drive: C:/Users/... and /c/Users/...
	if len(slash) >= 2 && slash[1] == ':' {
		drive := string(slash[0])
		rest := slash[2:]
		add(EncodeCWD(drive + ":" + rest))
		add(EncodeCWD(strings.ToLower(drive) + ":" + rest))
		add(EncodeCWD("/" + drive + rest))
		add(EncodeCWD("/" + strings.ToLower(drive) + rest))
		// Cursor often uses c-Users-... (colon dropped, not doubled).
		add(strings.ToLower(drive) + EncodeCWD(rest))
		add(strings.ToUpper(drive) + EncodeCWD(rest))
	}
	return out
}

// MatchProjectDir reports whether a Claude/Cursor project folder name belongs
// to repoRoot. Matching is by encoded-path equality (decoded path == repo root).
func MatchProjectDir(folder, repoRoot string) bool {
	if folder == "" || repoRoot == "" {
		return false
	}
	want := strings.ToLower(folder)
	for _, v := range EncodeVariants(repoRoot) {
		if strings.ToLower(v) == want {
			return true
		}
	}
	return false
}

// MatchSessionCWD reports whether a session's recorded cwd is this repo
// (same path, or a subdirectory — agents often start inside the tree).
func MatchSessionCWD(cwd, repoRoot string) bool {
	if cwd == "" || repoRoot == "" {
		return false
	}
	return Under(cwd, repoRoot) || Under(repoRoot, cwd)
}

// ClaudeProjectsDirs returns candidate Claude Code project roots.
func ClaudeProjectsDirs() []string {
	if env := strings.TrimSpace(os.Getenv("AICHANGE_CLAUDE_PROJECTS")); env != "" {
		return []string{filepath.Clean(env)}
	}
	var dirs []string
	if env := strings.TrimSpace(os.Getenv("CLAUDE_CONFIG_DIR")); env != "" {
		dirs = append(dirs, filepath.Join(env, "projects"))
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, ".claude", "projects"))
		dirs = append(dirs, filepath.Join(home, ".config", "claude", "projects"))
	}
	return unique(dirs)
}

// CursorProjectsDir is ~/.cursor/projects (or %USERPROFILE%\.cursor\projects).
func CursorProjectsDir() string {
	if env := strings.TrimSpace(os.Getenv("AICHANGE_CURSOR_PROJECTS")); env != "" {
		return filepath.Clean(env)
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".cursor", "projects")
	}
	return ""
}

// IndexPath is the SQLite file. Never inside the monitored repo.
//
//	Linux/mac: $XDG_STATE_HOME/aichange/index.db or ~/.local/state/aichange/index.db
//	Windows:   %LOCALAPPDATA%\aichange\index.db
//
// Override with AICHANGE_INDEX (tests, portable runs).
func IndexPath() string {
	if env := strings.TrimSpace(os.Getenv("AICHANGE_INDEX")); env != "" {
		return env
	}
	if runtime.GOOS == "windows" {
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			if home, err := os.UserHomeDir(); err == nil {
				base = filepath.Join(home, "AppData", "Local")
			}
		}
		return filepath.Join(base, "aichange", "index.db")
	}
	base := os.Getenv("XDG_STATE_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = "."
		}
		base = filepath.Join(home, ".local", "state")
	}
	return filepath.Join(base, "aichange", "index.db")
}

func unique(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, d := range in {
		n := filepath.Clean(d)
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
