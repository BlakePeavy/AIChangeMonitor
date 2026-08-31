package model

import "strings"

// FolderGroup is files rolled up by the first path segment (or "." for root).
type FolderGroup struct {
	Name    string `json:"name"`
	Count   int    `json:"count"`
	Added   int    `json:"added"`
	Deleted int    `json:"deleted"`
}

// GroupByFolder groups files by top-level folder. Order is first-seen.
func GroupByFolder(files []File) []FolderGroup {
	order := []string{}
	by := map[string]*FolderGroup{}
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		name := topFolder(f.Path)
		g, ok := by[name]
		if !ok {
			g = &FolderGroup{Name: name}
			by[name] = g
			order = append(order, name)
		}
		g.Count++
		g.Added += f.Added
		g.Deleted += f.Deleted
	}
	out := make([]FolderGroup, 0, len(order))
	for _, n := range order {
		out = append(out, *by[n])
	}
	return out
}

func topFolder(p string) string {
	p = strings.ReplaceAll(p, "\\", "/")
	p = strings.TrimPrefix(p, "./")
	if p == "" || p == "." {
		return "."
	}
	if i := strings.IndexByte(p, '/'); i >= 0 {
		if i == 0 {
			return topFolder(p[1:])
		}
		return p[:i]
	}
	return "."
}

// PathsInFolder returns paths whose top-level folder matches name (e.g. "ui" or ".").
func PathsInFolder(files []File, folder string) []string {
	var out []string
	for _, f := range files {
		if f.Path == "" {
			continue
		}
		if topFolder(f.Path) == folder {
			out = append(out, f.Path)
		}
	}
	return out
}
