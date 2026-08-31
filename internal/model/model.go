package model

import "time"

type Agent string

const (
	AgentClaudeCode   Agent = "claude-code"
	AgentCursor       Agent = "cursor"
	AgentGit          Agent = "git"
	AgentLive         Agent = "live"
	AgentCopilot      Agent = "copilot"
	AgentAider        Agent = "aider"
	AgentCodex        Agent = "codex"
	AgentWindsurf     Agent = "windsurf"
	AgentUnknown      Agent = "unknown"
	AgentUnknownAgent Agent = "unknown-agent"
)

const (
	SourceTranscript = "transcript"
	SourceCommit     = "commit"
	SourceLive       = "live"
)

type Status string

const (
	StatusUnseen   Status = "unseen"
	StatusSeen     Status = "seen"
	StatusAccepted Status = "accepted"
	StatusFlagged  Status = "flagged"
)

func ParseStatus(s string) (Status, bool) {
	switch Status(s) {
	case StatusUnseen, StatusSeen, StatusAccepted, StatusFlagged:
		return Status(s), true
	default:
		return "", false
	}
}

// File is a path the agent wrote or deleted, joined from tool_use or git numstat.
type File struct {
	Path    string `json:"path"`
	Tool    string `json:"tool,omitempty"`
	Delete  bool   `json:"delete,omitempty"`
	Prompt  string `json:"prompt,omitempty"` // preceding user prompt
	Added   int    `json:"added,omitempty"`
	Deleted int    `json:"deleted,omitempty"`
}

// Session is the review unit: one conversation, commit, or dirty tree.
type Session struct {
	ID           string    `json:"id"`
	Agent        Agent     `json:"agent"`
	StartedAt    time.Time `json:"started_at"`
	EndedAt      time.Time `json:"ended_at"`
	CWD          string    `json:"cwd"`
	Repo         string    `json:"repo"`
	Branch       string    `json:"branch"`
	Intent       string    `json:"intent"`
	Why          string    `json:"why"`
	Files        []File    `json:"files"`
	Status       Status    `json:"status"`
	Risks        []string  `json:"risks"`
	Source       string    `json:"source,omitempty"`
	SourcePath   string    `json:"source_path"`
	SourceMTime  int64     `json:"source_mtime"`
	AddedLines   int       `json:"added_lines,omitempty"`
	DeletedLines int       `json:"deleted_lines,omitempty"`
}

func (s Session) FilePaths() []string {
	out := make([]string, 0, len(s.Files))
	for _, f := range s.Files {
		if f.Path != "" {
			out = append(out, f.Path)
		}
	}
	return out
}

func (s Session) DeletedPaths() []string {
	var out []string
	for _, f := range s.Files {
		if f.Delete && f.Path != "" {
			out = append(out, f.Path)
		}
	}
	return out
}

func (s Session) HighRisk() bool {
	for _, r := range s.Risks {
		if r == "blast-radius" || r == "secrets" || r == "tests-deleted" || r == "auth" {
			return true
		}
	}
	return false
}

// DeriveSource fills Source from the id prefix when the column is empty (old rows).
func DeriveSource(id, source string) string {
	if source != "" {
		return source
	}
	switch {
	case len(id) >= 5 && id[:5] == "live:":
		return SourceLive
	case len(id) >= 4 && id[:4] == "git:":
		return SourceCommit
	default:
		return SourceTranscript
	}
}
