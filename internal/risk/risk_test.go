package risk

import (
	"testing"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func has(chips []string, name string) bool {
	for _, c := range chips {
		if c == name {
			return true
		}
	}
	return false
}

func TestChipsTable(t *testing.T) {
	cases := []struct {
		name  string
		files []model.File
		add   int
		del   int
		want  []string
		not   []string
	}{
		{"env", []model.File{{Path: ".env"}}, 0, 0, []string{"secrets"}, nil},
		{"pem", []model.File{{Path: "internal/auth/keys.pem"}}, 0, 0, []string{"secrets"}, nil},
		{"del", []model.File{{Path: "old.go", Delete: true}}, 0, 0, []string{"deletes"}, nil},
		{"tdel", []model.File{{Path: "foo_test.go", Delete: true}}, 0, 0, []string{"deletes", "tests-deleted"}, nil},
		{"blastL", []model.File{{Path: "big.go"}}, 500, 300, []string{"blast-radius"}, nil},
		{"author", []model.File{{Path: "internal/author.go"}}, 0, 0, nil, []string{"secrets", "auth"}},
		{"authdir", []model.File{{Path: "internal/auth/middleware.go"}}, 0, 0, []string{"auth"}, []string{"secrets"}},
		{"gitenv", []model.File{{Path: ".env", Added: 2, Deleted: 1}, {Path: "foo_test.go", Delete: true, Added: 0, Deleted: 20}}, 2, 21, []string{"secrets", "tests-deleted"}, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Chips(tc.files, tc.add, tc.del)
			for _, w := range tc.want {
				if !has(got, w) {
					t.Fatalf("missing %s in %v", w, got)
				}
			}
			for _, n := range tc.not {
				if has(got, n) {
					t.Fatalf("unexpected %s in %v", n, got)
				}
			}
		})
	}
}
