package gitx

import "strings"

// DirtyPath is one porcelain entry.
type DirtyPath struct {
	Path   string
	Delete bool
}

// DirtyFiles merges porcelain + unstaged/staged numstat into one live file list.
func DirtyFiles(repo string) ([]FileStat, error) {
	po, err := Status(repo)
	if err != nil {
		return nil, err
	}
	unstaged, _ := run(repo, "diff", "--numstat")
	staged, _ := run(repo, "diff", "--cached", "--numstat")
	return MergeDirty(ParsePorcelain(po), ParseNumstat(unstaged), ParseNumstat(staged)), nil
}

// MergeDirty combines porcelain paths with unstaged and staged numstat.
func MergeDirty(porcelain []DirtyPath, unstaged, staged []FileStat) []FileStat {
	order := []string{}
	by := map[string]*FileStat{}
	add := func(path string) *FileStat {
		if path == "" {
			return nil
		}
		if f, ok := by[path]; ok {
			return f
		}
		f := &FileStat{Path: path}
		by[path] = f
		order = append(order, path)
		return f
	}
	for _, f := range unstaged {
		cur := add(f.Path)
		cur.Added += f.Added
		cur.Deleted += f.Deleted
		cur.Delete = cur.Delete || f.Delete
	}
	for _, f := range staged {
		cur := add(f.Path)
		cur.Added += f.Added
		cur.Deleted += f.Deleted
		cur.Delete = cur.Delete || f.Delete
	}
	for _, p := range porcelain {
		cur := add(p.Path)
		if cur == nil {
			continue
		}
		if p.Delete {
			cur.Delete = true
		}
	}
	out := make([]FileStat, 0, len(order))
	for _, p := range order {
		out = append(out, *by[p])
	}
	return out
}

// ParsePorcelain parses `git status --porcelain` (XY + path, optional rename).
func ParsePorcelain(out string) []DirtyPath {
	out = strings.ReplaceAll(out, "\r\n", "\n")
	var files []DirtyPath
	for _, line := range strings.Split(out, "\n") {
		if line == "" {
			continue
		}
		if len(line) < 3 {
			continue
		}
		xy := line[:2]
		rest := ""
		if len(line) > 3 {
			rest = strings.TrimSpace(line[3:])
		} else {
			rest = strings.TrimSpace(line[2:])
		}
		path := porcelainPath(rest)
		if path == "" {
			continue
		}
		del := xy[0] == 'D' || xy[1] == 'D'
		files = append(files, DirtyPath{Path: path, Delete: del})
	}
	return files
}

func porcelainPath(rest string) string {
	rest = strings.TrimSpace(rest)
	if i := strings.Index(rest, " -> "); i >= 0 {
		rest = strings.TrimSpace(rest[i+4:])
	}
	return unquoteGitPath(rest)
}

func unquoteGitPath(p string) string {
	p = strings.TrimSpace(p)
	if len(p) >= 2 && p[0] == '"' && p[len(p)-1] == '"' {
		p = p[1 : len(p)-1]
		p = strings.ReplaceAll(p, `\"`, `"`)
	}
	return p
}
