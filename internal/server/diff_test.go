package server

import (
	"reflect"
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func TestScopedDiffPaths(t *testing.T) {
	sess := model.Session{Files: []model.File{
		{Path: "internal/store/store.go"},
		{Path: "internal/gitx/log.go"},
		{Path: "ui/app.js"},
		{Path: "README.md"},
	}}
	if got := scopedDiffPaths(sess, "ui/app.js", ""); !reflect.DeepEqual(got, []string{"ui/app.js"}) {
		t.Fatalf("file: %v", got)
	}
	if got := scopedDiffPaths(sess, "", "internal"); !reflect.DeepEqual(got, []string{"internal/store/store.go", "internal/gitx/log.go"}) {
		t.Fatalf("folder: %v", got)
	}
	if got := scopedDiffPaths(sess, "", "."); !reflect.DeepEqual(got, []string{"README.md"}) {
		t.Fatalf("root: %v", got)
	}
	if got := scopedDiffPaths(sess, "", "missing"); len(got) != 0 {
		t.Fatalf("missing: %v", got)
	}
	all := scopedDiffPaths(sess, "", "")
	if !reflect.DeepEqual(all, sess.FilePaths()) {
		t.Fatalf("all: %v", all)
	}
	// file wins over folder
	if got := scopedDiffPaths(sess, "ui/app.js", "internal"); !reflect.DeepEqual(got, []string{"ui/app.js"}) {
		t.Fatalf("file wins: %v", got)
	}
}
