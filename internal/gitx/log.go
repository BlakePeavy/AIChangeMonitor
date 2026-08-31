package gitx

import (
	"strconv"
	"strings"
	"time"
)

const DefaultLogN = 40

// FileStat is one path from git numstat / porcelain.
type FileStat struct {
	Path    string
	Added   int
	Deleted int
	Delete  bool
}

// Commit is one git log --numstat record.
type Commit struct {
	Hash    string
	Author  string
	Time    time.Time
	Subject string
	Body    string
	Files   []FileStat
}

// LogNumstat returns the n most recent commits (default 40) with numstat files.
func LogNumstat(repo string, n int) ([]Commit, error) {
	if n <= 0 {
		n = DefaultLogN
	}
	out, err := run(repo, "log", "-n", strconv.Itoa(n),
		"--format=%x1e%H%x00%an%x00%aI%x00%s%x00%b%x00", "--numstat")
	if err != nil {
		return nil, err
	}
	return ParseLogNumstat(out), nil
}

// ParseLogNumstat turns `git log --format=RS+NUL --numstat` output into commits.
func ParseLogNumstat(out string) []Commit {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	var commits []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		rec = strings.TrimSpace(rec)
		if rec == "" {
			continue
		}
		parts := strings.SplitN(rec, "\x00", 6)
		if len(parts) < 5 {
			continue
		}
		c := Commit{
			Hash:    strings.TrimSpace(parts[0]),
			Author:  parts[1],
			Subject: parts[3],
			Body:    strings.TrimSpace(parts[4]),
		}
		if c.Hash == "" {
			continue
		}
		if t, err := time.Parse(time.RFC3339, parts[2]); err == nil {
			c.Time = t
		} else if t, err := time.Parse(time.RFC3339Nano, parts[2]); err == nil {
			c.Time = t
		}
		rest := ""
		if len(parts) >= 6 {
			rest = parts[5]
		}
		c.Files = ParseNumstat(rest)
		commits = append(commits, c)
	}
	return commits
}

// ParseNumstat parses `added\tdeleted\tpath` lines. Rename paths keep the new name.
func ParseNumstat(out string) []FileStat {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	var files []FileStat
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		f, ok := parseNumstatLine(line)
		if !ok {
			continue
		}
		files = append(files, f)
	}
	return files
}

func parseNumstatLine(line string) (FileStat, bool) {
	// added \t deleted \t path   (path may contain spaces)
	a, rest, ok := strings.Cut(line, "\t")
	if !ok {
		return FileStat{}, false
	}
	d, path, ok := strings.Cut(rest, "\t")
	if !ok {
		return FileStat{}, false
	}
	path = renameNew(strings.TrimSpace(path))
	if path == "" {
		return FileStat{}, false
	}
	add, okA := parseStatNum(a)
	del, okD := parseStatNum(d)
	if !okA || !okD {
		return FileStat{}, false
	}
	return FileStat{
		Path:    path,
		Added:   add,
		Deleted: del,
		Delete:  del > 0 && add == 0,
	}, true
}

func parseStatNum(s string) (int, bool) {
	s = strings.TrimSpace(s)
	if s == "-" {
		return 0, true
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	return n, true
}

func renameNew(path string) string {
	if i := strings.Index(path, " => "); i >= 0 {
		return strings.TrimSpace(path[i+4:])
	}
	return path
}
