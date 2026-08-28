package server

import (
	"bufio"
	"path/filepath"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/redact"
)

func AnnotateDiff(diff string, sess model.Session) string {
	if diff == "" {
		return diff
	}
	promptFor := map[string]string{}
	for _, f := range sess.Files {
		if f.Prompt == "" {
			continue
		}
		promptFor[filepath.ToSlash(f.Path)] = f.Prompt
		promptFor[filepath.Base(f.Path)] = f.Prompt
	}
	var b strings.Builder
	sc := bufio.NewScanner(strings.NewReader(diff))
	for sc.Scan() {
		line := sc.Text()
		if strings.HasPrefix(line, "diff --git ") {
			file := gitDiffPath(line)
			if p := lookupPrompt(promptFor, file); p != "" {
				b.WriteString("# prompt: ")
				b.WriteString(oneLine(p))
				b.WriteByte('\n')
			}
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return redact.Redact(b.String())
}

func gitDiffPath(line string) string {
	// diff --git a/foo.go b/foo.go
	fields := strings.Fields(line)
	if len(fields) >= 4 {
		p := fields[3]
		p = strings.TrimPrefix(p, "b/")
		return p
	}
	if len(fields) >= 3 {
		p := strings.TrimPrefix(fields[2], "a/")
		return p
	}
	return ""
}

func lookupPrompt(m map[string]string, file string) string {
	file = filepath.ToSlash(file)
	if p, ok := m[file]; ok {
		return p
	}
	if p, ok := m[filepath.Base(file)]; ok {
		return p
	}
	for k, p := range m {
		if strings.HasSuffix(file, k) || strings.HasSuffix(k, file) {
			return p
		}
	}
	return ""
}

func oneLine(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > 200 {
		return s[:200] + "…"
	}
	return s
}
