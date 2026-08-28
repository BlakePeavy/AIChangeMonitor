package ingest

import (
	"strings"
)

func extractMessage(v any, ev *event) {
	switch m := v.(type) {
	case string:
		if m != "" {
			ev.Texts = append(ev.Texts, m)
			if ev.isUser() {
				ev.UserText = m
			}
		}
	case map[string]any:
		if ev.Role == "" {
			ev.Role = asString(m["role"])
		}
		extractContent(m["content"], ev)
	}
}

func extractContent(v any, ev *event) {
	switch c := v.(type) {
	case string:
		if c != "" {
			ev.Texts = append(ev.Texts, c)
		}
	case []any:
		for _, item := range c {
			extractBlock(item, ev)
		}
	}
}

func extractBlock(v any, ev *event) {
	m, ok := v.(map[string]any)
	if !ok {
		if s, ok := v.(string); ok && s != "" {
			ev.Texts = append(ev.Texts, s)
		}
		return
	}
	typ := asString(m["type"])
	switch typ {
	case "text", "":
		if t := asString(m["text"]); t != "" {
			ev.Texts = append(ev.Texts, t)
		}
	case "thinking":
		t := asString(m["thinking"])
		if t == "" {
			t = asString(m["text"])
		}
		if t != "" {
			ev.Thinking = append(ev.Thinking, t)
		}
	case "tool_use", "tool-use", "toolcall", "tool_call":
		ev.Tools = append(ev.Tools, parseTool(m))
	}
}

func looksHuman(ev event) bool {
	if ev.UserText == "" && len(ev.Texts) == 0 {
		return false
	}
	text := ev.UserText
	if text == "" {
		text = joinTexts(ev.Texts)
	}
	trim := strings.TrimSpace(text)
	if trim == "" {
		return false
	}
	if strings.HasPrefix(trim, "Caveat:") {
		return false
	}
	if strings.Contains(trim, "<tool_result") || strings.Contains(trim, "tool_result") && !strings.Contains(trim, "<user_query") {
		if !strings.Contains(trim, "<user_query") {
			return false
		}
	}
	return true
}
