package ingest

import "time"

type event struct {
	Type      string
	Role      string
	Time      time.Time
	SessionID string
	CWD       string
	Branch    string
	Texts     []string
	Thinking  []string
	Tools     []toolUse
	UserText  string
	Human     bool
}

type toolUse struct {
	Name   string
	Input  map[string]any
	Path   string
	Delete bool
}

func (e event) isUser() bool {
	return e.Type == "user" || e.Role == "user"
}

func (e event) isAssistant() bool {
	return e.Type == "assistant" || e.Role == "assistant"
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case float64:
		return ""
	default:
		return ""
	}
}
