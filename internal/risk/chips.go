package risk

import (
	"path/filepath"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func Chips(files []model.File, added, deleted int) []string {
	out := []string{}
	n := len(files)
	var sens, del, testdel, hasLock bool
	for _, f := range files {
		base := strings.ToLower(filepath.Base(f.Path))
		if isLock(base) {
			hasLock = true
		}
		if sensitivePath(f.Path) {
			sens = true
		}
		if f.Delete {
			del = true
			if isTestPath(f.Path) {
				testdel = true
			}
		}
	}
	if sens {
		out = append(out, "secrets")
	}
	if hasLock && n >= 5 {
		out = append(out, "lockfile")
	}
	if del {
		out = append(out, "deletes")
	}
	if added+deleted >= 800 || n >= 15 {
		out = append(out, "blast-radius")
	}
	if testdel {
		out = append(out, "tests-deleted")
	}
	return out
}
