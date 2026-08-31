package model

import "testing"

func TestGroupByFolder(t *testing.T) {
	files := []File{
		{Path: "internal/store/store.go", Added: 10, Deleted: 2},
		{Path: "internal/gitx/log.go", Added: 40, Deleted: 0},
		{Path: "ui/app.js", Added: 5, Deleted: 1},
		{Path: "README.md", Added: 3, Deleted: 0},
		{Path: "ui\\style.css", Added: 1, Deleted: 1},
	}
	got := GroupByFolder(files)
	if len(got) != 3 {
		t.Fatalf("groups %d: %+v", len(got), got)
	}
	if got[0].Name != "internal" || got[0].Count != 2 || got[0].Added != 50 || got[0].Deleted != 2 {
		t.Fatalf("internal %+v", got[0])
	}
	if got[1].Name != "ui" || got[1].Count != 2 || got[1].Added != 6 || got[1].Deleted != 2 {
		t.Fatalf("ui %+v", got[1])
	}
	if got[2].Name != "." || got[2].Count != 1 || got[2].Added != 3 {
		t.Fatalf("root %+v", got[2])
	}
}

func TestGroupByFolderEmpty(t *testing.T) {
	if got := GroupByFolder(nil); len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestPathsInFolder(t *testing.T) {
	files := []File{
		{Path: "internal/store/store.go"},
		{Path: "internal/gitx/log.go"},
		{Path: "ui/app.js"},
		{Path: "README.md"},
		{Path: "ui\\style.css"},
	}
	got := PathsInFolder(files, "internal")
	if len(got) != 2 || got[0] != "internal/store/store.go" || got[1] != "internal/gitx/log.go" {
		t.Fatalf("internal %v", got)
	}
	got = PathsInFolder(files, "ui")
	if len(got) != 2 || got[0] != "ui/app.js" || got[1] != "ui\\style.css" {
		t.Fatalf("ui %v", got)
	}
	got = PathsInFolder(files, ".")
	if len(got) != 1 || got[0] != "README.md" {
		t.Fatalf("root %v", got)
	}
	if got = PathsInFolder(files, "nope"); len(got) != 0 {
		t.Fatalf("missing %v", got)
	}
	if got = PathsInFolder(nil, "ui"); len(got) != 0 {
		t.Fatalf("nil %v", got)
	}
}
