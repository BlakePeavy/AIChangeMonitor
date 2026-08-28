package risk

import "strings"

func isLock(base string) bool {
	if base == "go.sum" {
		return true
	}
	if strings.HasSuffix(base, ".lock") || strings.HasSuffix(base, ".lockb") {
		return true
	}
	if strings.Contains(base, "lock.yaml") {
		return true
	}
	return strings.Contains(base, "lock.json")
}
