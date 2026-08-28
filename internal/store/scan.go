package store

import (
	"encoding/json"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

type rowScanner interface {
	Scan(dest ...any) error
}

func scanSession(row rowScanner) (model.Session, error) {
	var sess model.Session
	var agent, started, ended, filesJSON, status, risksJSON string
	err := row.Scan(
		&sess.ID, &agent, &started, &ended, &sess.CWD, &sess.Repo, &sess.Branch,
		&sess.Intent, &sess.Why, &filesJSON, &status, &risksJSON,
		&sess.SourcePath, &sess.SourceMTime, &sess.AddedLines, &sess.DeletedLines,
	)
	if err != nil {
		return model.Session{}, err
	}
	sess.Agent = model.Agent(agent)
	sess.Status = model.Status(status)
	sess.StartedAt = parseTime(started)
	sess.EndedAt = parseTime(ended)
	if filesJSON != "" {
		_ = json.Unmarshal([]byte(filesJSON), &sess.Files)
	}
	if risksJSON != "" {
		_ = json.Unmarshal([]byte(risksJSON), &sess.Risks)
	}
	return sess, nil
}

func parseTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
