package store

import (
	"encoding/json"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func (s *Store) Upsert(sess model.Session) error {
	files, err := json.Marshal(sess.Files)
	if err != nil {
		return err
	}
	risks, err := json.Marshal(sess.Risks)
	if err != nil {
		return err
	}
	status := sess.Status
	if status == "" {
		status = model.StatusUnseen
	}
	_, err = s.db.Exec(`
INSERT INTO sessions (id, agent, started_at, ended_at, cwd, repo, branch, intent, why, files, status, risks, source_path, source_mtime, added_lines, deleted_lines)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
	agent=excluded.agent,
	started_at=excluded.started_at,
	ended_at=excluded.ended_at,
	cwd=excluded.cwd,
	repo=excluded.repo,
	branch=excluded.branch,
	intent=excluded.intent,
	why=excluded.why,
	files=excluded.files,
	risks=excluded.risks,
	source_path=excluded.source_path,
	source_mtime=excluded.source_mtime,
	added_lines=excluded.added_lines,
	deleted_lines=excluded.deleted_lines
`, sess.ID, string(sess.Agent), fmtTime(sess.StartedAt), fmtTime(sess.EndedAt),
		sess.CWD, sess.Repo, sess.Branch, sess.Intent, sess.Why, string(files),
		status, string(risks), sess.SourcePath, sess.SourceMTime, sess.AddedLines, sess.DeletedLines)
	return err
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}
