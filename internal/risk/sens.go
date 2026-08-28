package risk

import (
	"path/filepath"
	"strings"
)

func sensitivePath(p string) bool {
	low := strings.ToLower(filepath.ToSlash(p))
	base := filepath.Base(low)
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		return true
	}
	if strings.HasSuffix(base, ".pem") || strings.HasSuffix(base, ".p12") {
		return true
	}
	if strings.HasSuffix(base, ".pfx") || strings.HasSuffix(base, ".key") {
		return !strings.HasSuffix(base, ".go")
	}
	if strings.Contains(base, "secret") || strings.Contains(base, "credential") {
		return true
	}
	if strings.Contains(low, "/auth/") || strings.HasPrefix(base, "auth.") {
		return true
	}
	if base == "id_rsa" || base == "id_ed25519" {
		return true
	}
	return base == "credentials.json"
}
