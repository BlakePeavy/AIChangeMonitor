package gitx

import "testing"

func TestParsePorcelain(t *testing.T) {
	in := "" +
		" M ui/app.js\n" +
		"M  internal/store/store.go\n" +
		" D gone.go\n" +
		"D  staged-gone.go\n" +
		"?? brand-new.md\n" +
		"R  old.txt -> new.txt\n" +
		`?? "file with spaces.txt"` + "\n"
	got := ParsePorcelain(in)
	if len(got) != 7 {
		t.Fatalf("n=%d %+v", len(got), got)
	}
	want := []DirtyPath{
		{Path: "ui/app.js"},
		{Path: "internal/store/store.go"},
		{Path: "gone.go", Delete: true},
		{Path: "staged-gone.go", Delete: true},
		{Path: "brand-new.md"},
		{Path: "new.txt"},
		{Path: "file with spaces.txt"},
	}
	for i, w := range want {
		if got[i] != w {
			t.Fatalf("%d: got %+v want %+v", i, got[i], w)
		}
	}
}

func TestMergeDirty(t *testing.T) {
	po := ParsePorcelain(" M a.go\n D b.go\n?? c.go\n")
	unstaged := ParseNumstat("4\t1\ta.go\n0\t9\tb.go\n")
	staged := ParseNumstat("2\t0\ta.go\n")
	got := MergeDirty(po, unstaged, staged)
	by := map[string]FileStat{}
	for _, f := range got {
		by[f.Path] = f
	}
	if by["a.go"].Added != 6 || by["a.go"].Deleted != 1 {
		t.Fatalf("a.go %+v", by["a.go"])
	}
	if !by["b.go"].Delete || by["b.go"].Deleted != 9 {
		t.Fatalf("b.go %+v", by["b.go"])
	}
	if _, ok := by["c.go"]; !ok {
		t.Fatalf("missing untracked: %+v", got)
	}
}
