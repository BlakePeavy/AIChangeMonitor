package gitx

import (
	"strconv"
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
	args := []string{"diff", "--numstat", "--"}
	args = append(args, files...)
	if len(files) == 0 {
		return 0, 0, nil
	}
	out, err := run(repo, args...)
	if err != nil {
		return 0, 0, err
	}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f := strings.Fields(line)
		if len(f) < 2 {
			continue
		}
		if f[0] != "-" {
			if n, e := strconv.Atoi(f[0]); e == nil {
				added += n
			}
		}
		if f[1] != "-" {
			if n, e := strconv.Atoi(f[1]); e == nil {
				deleted += n
			}
		}
	}
	return added, deleted, nil
}
