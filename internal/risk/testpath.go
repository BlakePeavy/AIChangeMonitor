package risk

import (
	"path/filepath"
	"strings"
)

func isTestPath(p string) bool {
	low := strings.ToLower(filepath.ToSlash(p))
	base := filepath.Base(low)
	if strings.Contains(base, "_test.") || strings.Contains(base, ".test.") {
		return true
	}
	if strings.Contains(base, ".spec.") || strings.HasPrefix(base, "test_") {
		return true
	}
	if strings.Contains(low, "/test/") || strings.Contains(low, "/tests/") {
		return true
	}
	return strings.Contains(low, "/__tests__/")
}
