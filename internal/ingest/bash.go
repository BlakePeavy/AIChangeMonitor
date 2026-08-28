package ingest

import (
	"regexp"
	"strings"
)

var (
	reRm     = regexp.MustCompile(`(?i)\brm(?:\s+-[a-zA-Z]+)*\s+(?:"([^"]+)"|'([^']+)'|(\S+))`)
	reRedir  = regexp.MustCompile(`(?:>>?)\s*(?:"([^"]+)"|'([^']+)'|(\S+))`)
	rePathy  = regexp.MustCompile(`(?:^|[\s])((?:\./|/)[A-Za-z0-9._/-]+\.[A-Za-z0-9]{1,8})`)
)

func bashPath(cmd string) string {
	if cmd == "" {
		return ""
	}
	if m := reRm.FindStringSubmatch(cmd); m != nil {
		p := firstGroup(m)
		if looksFilePath(p) {
			return p
		}
	}
	if strings.Contains(cmd, "rm ") || strings.Contains(cmd, "rm\t") {
		if m := reRm.FindStringSubmatch(cmd); m != nil {
			return firstGroup(m)
		}
	}
	if m := reRedir.FindStringSubmatch(cmd); m != nil {
		p := firstGroup(m)
		if looksFilePath(p) {
			return p
		}
	}
	return ""
}

func firstGroup(m []string) string {
	for i := 1; i < len(m); i++ {
		if m[i] != "" {
			return m[i]
		}
	}
	return ""
}

func looksFilePath(p string) bool {
	if p == "" || p == "-" || strings.HasPrefix(p, "-") {
		return false
	}
	if strings.ContainsAny(p, "/\\") {
		return true
	}
	return strings.Contains(p, ".")
}

func bashIsDelete(cmd string) bool {
	c := strings.TrimSpace(cmd)
	return strings.HasPrefix(c, "rm ") || strings.HasPrefix(c, "rm\t") || strings.Contains(c, " rm ")
}
