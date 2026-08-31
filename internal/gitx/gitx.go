package gitx

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
)

var (
	gitOnce sync.Once
	gitPath string
)

// gitBin is "git" on PATH, or a well-known install if GUI PATH hid it.
func gitBin() string {
	gitOnce.Do(func() {
		if p, err := exec.LookPath("git"); err == nil {
			gitPath = p
			return
		}
		var candidates []string
		switch runtime.GOOS {
		case "windows":
			pf := os.Getenv("ProgramFiles")
			pf86 := os.Getenv("ProgramFiles(x86)")
			local := os.Getenv("LOCALAPPDATA")
			candidates = []string{
				filepath.Join(pf, "Git", "cmd", "git.exe"),
				filepath.Join(pf, "Git", "bin", "git.exe"),
				filepath.Join(pf86, "Git", "cmd", "git.exe"),
				filepath.Join(local, "Programs", "Git", "cmd", "git.exe"),
			}
		default:
			candidates = []string{
				"/usr/bin/git",
				"/usr/local/bin/git",
				"/opt/homebrew/bin/git",
			}
		}
		for _, c := range candidates {
			if c == "" || strings.Contains(c, string(filepath.Separator)+string(filepath.Separator)) {
				continue
			}
			if st, err := os.Stat(c); err == nil && !st.IsDir() {
				gitPath = c
				return
			}
		}
		gitPath = "git"
	})
	return gitPath
}

func run(repo string, args ...string) (string, error) {
	cmd := exec.Command(gitBin(), args...)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return stdout.String(), nil
}

func runExit(repo string, args ...string) (string, int, error) {
	cmd := exec.Command(gitBin(), args...)
	if repo != "" {
		cmd.Dir = repo
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := stdout.String()
	if err == nil {
		return out, 0, nil
	}
	msg := strings.TrimSpace(stderr.String())
	if msg == "" {
		msg = err.Error()
	}
	wrapped := fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	if ee, ok := err.(*exec.ExitError); ok {
		return out, ee.ExitCode(), wrapped
	}
	return out, -1, wrapped
}

func RepoRoot(dir string) (string, error) {
	out, err := run(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(out), nil
}

func Status(repo string) (string, error) {
	return run(repo, "status", "--porcelain")
}

func Branch(repo string) string {
	out, err := run(repo, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func HEAD(repo string) string {
	out, err := run(repo, "rev-parse", "HEAD")
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}
