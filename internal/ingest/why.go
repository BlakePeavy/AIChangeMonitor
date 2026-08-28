package ingest

import (
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/redact"
)

func humanPrompt(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "<user_query>"); i >= 0 {
		rest := raw[i+len("<user_query>"):]
		if j := strings.Index(rest, "</user_query>"); j >= 0 {
			raw = rest[:j]
		} else {
			raw = rest
		}
	}
	return redact.Redact(strings.TrimSpace(raw))
}

func whyExcerpt(parts []string) string {
	var lines []string
	for _, p := range parts {
		for _, ln := range strings.Split(p, "\n") {
			ln = strings.TrimSpace(ln)
			if ln == "" {
				continue
			}
			lines = append(lines, ln)
			if len(lines) >= 8 {
				break
			}
		}
		if len(lines) >= 8 {
			break
		}
	}
	if len(lines) > 8 {
		lines = lines[:8]
	}
	if len(lines) < 3 && len(lines) > 0 {
		// keep what we have; 3-8 is a target, not a hard floor
	}
	return redact.Redact(strings.Join(lines, "\n"))
}

func joinFiles(ops []model.File) []model.File {
	order := []string{}
	by := map[string]model.File{}
	for _, op := range ops {
		if op.Path == "" {
			continue
		}
		if cur, ok := by[op.Path]; ok {
			if op.Delete {
				cur.Delete = true
			}
			if cur.Prompt == "" {
				cur.Prompt = op.Prompt
			}
			if cur.Tool == "" {
				cur.Tool = op.Tool
			}
			by[op.Path] = cur
			continue
		}
		order = append(order, op.Path)
		by[op.Path] = op
	}
	out := make([]model.File, 0, len(order))
	for _, p := range order {
		out = append(out, by[p])
	}
	return out
}
