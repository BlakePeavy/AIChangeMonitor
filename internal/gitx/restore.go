package gitx

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// CleanRelPath rejects empty, absolute, and repo-escaping paths. Slash-normalized.
func CleanRelPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	p = strings.ReplaceAll(p, "\\", "/")
	if p == "" {
		return "", fmt.Errorf("path required")
	}
	if filepath.IsAbs(p) || strings.HasPrefix(p, "/") || strings.HasPrefix(p, "~") {
		return "", fmt.Errorf("path must be relative")
	}
	if len(p) >= 2 && p[1] == ':' {
		return "", fmt.Errorf("path must be relative")
	}
	cleaned := path.Clean(p)
	cleaned = strings.TrimPrefix(cleaned, "./")
	if cleaned == "." || cleaned == "" {
		return "", fmt.Errorf("path required")
	}
	if cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", fmt.Errorf("path escapes repo")
	}
	return cleaned, nil
}

// AbsInRepo joins a relative path onto repo and checks it stays inside.
func AbsInRepo(repo, rel string) (string, error) {
	rel, err := CleanRelPath(rel)
	if err != nil {
		return "", err
	}
	root, err := filepath.Abs(repo)
	if err != nil {
		return "", err
	}
	abs := filepath.Join(root, filepath.FromSlash(rel))
	abs, err = filepath.Abs(abs)
	if err != nil {
		return "", err
	}
	sep := string(filepath.Separator)
	if abs != root && !strings.HasPrefix(abs, root+sep) {
		return "", fmt.Errorf("path escapes repo")
	}
	return abs, nil
}

// Upstream returns `git rev-parse --abbrev-ref @{u}`. ok is false when none.
func Upstream(repo string) (string, bool) {
	out, err := run(repo, "rev-parse", "--abbrev-ref", "@{u}")
	if err != nil {
		return "", false
	}
	s := strings.TrimSpace(out)
	return s, s != "" && s != "@{u}"
}

// IsAncestor is `git merge-base --is-ancestor commit rev` (exit 0/1).
func IsAncestor(repo, commit, rev string) (bool, error) {
	_, code, err := runExit(repo, "merge-base", "--is-ancestor", commit, rev)
	if code == 0 {
		return true, nil
	}
	if code == 1 {
		return false, nil
	}
	return false, err
}

// IsHEAD reports whether sha resolves to the repo HEAD.
func IsHEAD(repo, sha string) bool {
	h := HEAD(repo)
	if h == "" || sha == "" {
		return false
	}
	if h == sha {
		return true
	}
	full, err := run(repo, "rev-parse", sha)
	if err != nil {
		return false
	}
	return strings.TrimSpace(full) == h
}

// Restore runs `git restore [--source=src] --staged --worktree -- path`.
// Empty source means restore from HEAD (discard uncommitted changes).
func Restore(repo, source, relPath string) error {
	relPath, err := CleanRelPath(relPath)
	if err != nil {
		return err
	}
	args := []string{"restore"}
	if source != "" {
		args = append(args, "--source="+source)
	}
	args = append(args, "--staged", "--worktree", "--", relPath)
	_, err = run(repo, args...)
	return err
}

// RevealInFolder asks the OS to show the file (or its parent dir).
func RevealInFolder(repo, relPath string) error {
	abs, err := AbsInRepo(repo, relPath)
	if err != nil {
		return err
	}
	return revealAbs(abs)
}

func revealAbs(abs string) error {
	switch runtime.GOOS {
	case "windows":
		exe, err := exec.LookPath("explorer")
		if err != nil {
			return err
		}
		err = exec.Command(exe, "/select,"+abs).Run()
		if err == nil {
			return nil
		}
		if _, ok := err.(*exec.ExitError); ok {
			return nil // explorer often exits 1 even on success
		}
		return err
	case "darwin":
		return exec.Command("open", "-R", abs).Run()
	default:
		dir := filepath.Dir(abs)
		if fi, err := os.Stat(abs); err == nil && fi.IsDir() {
			dir = abs
		}
		cmd := exec.Command("xdg-open", dir)
		if err := cmd.Start(); err != nil {
			return err
		}
		go func() { _ = cmd.Wait() }()
		return nil
	}
}
