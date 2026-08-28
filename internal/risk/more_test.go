package risk

import (
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func TestLockAndBlastFiles(t *testing.T) {
	files := []model.File{
		{Path: "go.sum"},
		{Path: "a.go"}, {Path: "b.go"}, {Path: "c.go"}, {Path: "d.go"},
	}
	got := Chips(files, 0, 0)
	if !has(got, "lockfile") {
		t.Fatalf("want lockfile, got %v", got)
	}
	small := []model.File{{Path: "go.sum"}, {Path: "a.go"}}
	got = Chips(small, 0, 0)
	if has(got, "lockfile") {
		t.Fatalf("did not want lockfile: %v", got)
	}
	many := make([]model.File, 15)
	for i := 0; i < 15; i++ {
		many[i] = model.File{Path: "f.go"}
		many[i].Path = "f" + string(rune('a'+i)) + ".go"
	}
	got = Chips(many, 0, 0)
	if !has(got, "blast-radius") {
		t.Fatalf("want blast-radius, got %v", got)
	}
}
