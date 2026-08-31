package gitx

import (
	"strings"
	"time"
)

func Diff(repo string, files []string) (string, error) {
	args := []string{"diff", "--"}
	args = append(args, files...)
	if len(files) == 0 {
		args = []string{"diff"}
	}
	return run(repo, args...)
}

func DiffCached(repo string, files []string) (string, error) {
	args := []string{"diff", "--cached", "--"}
	args = append(args, files...)
	if len(files) == 0 {
		args = []string{"diff", "--cached"}
	}
	return run(repo, args...)
}

// DiffWorktree is unstaged + staged (live session).
func DiffWorktree(repo string, files []string) (string, error) {
	unstaged, err1 := Diff(repo, files)
	cached, err2 := DiffCached(repo, files)
	var b strings.Builder
	if strings.TrimSpace(unstaged) != "" {
		b.WriteString(unstaged)
		if !strings.HasSuffix(unstaged, "\n") {
			b.WriteByte('\n')
		}
	}
	if strings.TrimSpace(cached) != "" {
		b.WriteString(cached)
	}
	if b.Len() == 0 && err1 != nil {
		return "", err1
	}
	if b.Len() == 0 && err2 != nil {
		return "", err2
	}
	return b.String(), nil
}

// DiffHEAD is staged+unstaged vs HEAD as one patch (path-scoped live diffs).
func DiffHEAD(repo string, files []string) (string, error) {
	args := []string{"diff", "HEAD", "--"}
	args = append(args, files...)
	if len(files) == 0 {
		args = []string{"diff", "HEAD"}
	}
	return run(repo, args...)
}

// Show is the patch for one commit (no header).
func Show(repo, sha string, files []string) (string, error) {
	if sha == "" {
		return "", nil
	}
	args := []string{"show", "--format=", sha, "--"}
	args = append(args, files...)
	if len(files) == 0 {
		args = []string{"show", "--format=", sha}
	}
	return run(repo, args...)
}

func LogSince(repo string, since time.Time, files []string) (string, error) {
	if since.IsZero() {
		return "", nil
	}
	args := []string{"log", "--oneline", "--since=" + since.UTC().Format(time.RFC3339)}
	if len(files) > 0 {
		args = append(args, "--")
		args = append(args, files...)
	}
	return run(repo, args...)
}

func DiffStat(repo string, files []string) (added, deleted int, err error) {
	stats, err := NumstatFiles(repo, files)
	if err != nil {
		return 0, 0, err
	}
	for _, f := range stats {
		added += f.Added
		deleted += f.Deleted
	}
	return added, deleted, nil
}

func NumstatFiles(repo string, files []string) ([]FileStat, error) {
	if len(files) == 0 {
		return nil, nil
	}
	args := []string{"diff", "--numstat", "--"}
	args = append(args, files...)
	out, err := run(repo, args...)
	if err != nil {
		return nil, err
	}
	return ParseNumstat(out), nil
}
