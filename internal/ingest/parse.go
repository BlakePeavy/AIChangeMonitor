package ingest

import (
	"bytes"
	"encoding/json"
)

func parseLine(line []byte) (event, bool) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 || line[0] != '{' {
		return event{}, false
	}
	var raw map[string]any
	if err := json.Unmarshal(line, &raw); err != nil {
		return event{}, false
	}
	ev := event{
		Type:      asString(raw["type"]),
		Role:      asString(raw["role"]),
		SessionID: asString(raw["sessionId"]),
		CWD:       asString(raw["cwd"]),
		Branch:    asString(raw["gitBranch"]),
	}
	if ev.SessionID == "" {
		ev.SessionID = asString(raw["session_id"])
	}
	if ev.Branch == "" {
		ev.Branch = asString(raw["git_branch"])
	}
	if ev.CWD == "" {
		ev.CWD = asString(raw["workspace"])
	}
	if ts := asString(raw["timestamp"]); ts != "" {
		ev.Time = parseTime(ts)
	}
	if ev.Time.IsZero() {
		if ts := asString(raw["createdAt"]); ts != "" {
			ev.Time = parseTime(ts)
		}
	}
	extractMessage(raw["message"], &ev)
	if len(ev.Texts) == 0 && len(ev.Tools) == 0 {
		extractContent(raw["content"], &ev)
	}
	if ev.UserText == "" && ev.isUser() {
		ev.UserText = joinTexts(ev.Texts)
	}
	ev.Human = ev.isUser() && looksHuman(ev)
	return ev, ev.Type != "" || ev.Role != "" || len(ev.Tools) > 0 || ev.UserText != ""
}

func joinTexts(ss []string) string {
	var b bytes.Buffer
	for _, s := range ss {
		s = string(bytes.TrimSpace([]byte(s)))
		if s == "" {
			continue
		}
		if b.Len() > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(s)
	}
	return b.String()
}
