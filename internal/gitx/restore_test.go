package gitx

import "testing"

func TestCleanRelPath(t *testing.T) {
	ok, err := CleanRelPath(`ui\app.js`)
	if err != nil || ok != "ui/app.js" {
		t.Fatalf("slash: %q %v", ok, err)
	}
	ok, err = CleanRelPath("internal/store/store.go")
	if err != nil || ok != "internal/store/store.go" {
		t.Fatalf("rel: %q %v", ok, err)
	}
	if _, err := CleanRelPath(""); err == nil {
		t.Fatal("empty")
	}
	if _, err := CleanRelPath("/etc/passwd"); err == nil {
		t.Fatal("abs")
	}
	if _, err := CleanRelPath("../secret"); err == nil {
		t.Fatal("dotdot")
	}
	if _, err := CleanRelPath("foo/../../etc/passwd"); err == nil {
		t.Fatal("nested escape")
	}
	if _, err := CleanRelPath("C:\\Windows\\x"); err == nil {
		t.Fatal("windows abs")
	}
}
