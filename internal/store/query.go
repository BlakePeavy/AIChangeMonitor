package store

import (
	"fmt"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

const sessionCols = `id, agent, started_at, ended_at, cwd, repo, branch, intent, why, files, status, risks, source_path, source_mtime, added_lines, deleted_lines, source`

func (s *Store) Get(id string) (model.Session, error) {
	row := s.db.QueryRow(`SELECT `+sessionCols+` FROM sessions WHERE id = ?`, id)
	return scanSession(row)
}

func (s *Store) Resolve(prefix string) (model.Session, error) {
	if sess, err := s.Get(prefix); err == nil {
		return sess, nil
	}
	rows, err := s.db.Query(`SELECT id FROM sessions WHERE id LIKE ?`, prefix+"%")
	if err != nil {
		return model.Session{}, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return model.Session{}, err
		}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return model.Session{}, fmt.Errorf("session %q not found", prefix)
	}
	if len(ids) > 1 {
		return model.Session{}, fmt.Errorf("session prefix %q is ambiguous (%d matches)", prefix, len(ids))
	}
	return s.Get(ids[0])
}

func (s *Store) List(repo string) ([]model.Session, error) {
	q := `SELECT ` + sessionCols + ` FROM sessions`
	var args []any
	if repo != "" {
		q += ` WHERE repo = ?`
		args = append(args, repo)
	}
	q += ` ORDER BY CASE WHEN COALESCE(source,'') = 'live' OR id LIKE 'live:%' THEN 0 ELSE 1 END, started_at DESC, id DESC`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Session
	for rows.Next() {
		sess, err := scanSession(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, sess)
	}
	return out, rows.Err()
}

func (s *Store) SetStatus(id string, st model.Status) error {
	res, err := s.db.Exec(`UPDATE sessions SET status = ? WHERE id = ?`, string(st), id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("session %q not found", id)
	}
	return nil
}

func (s *Store) SourceMTime(path string) int64 {
	var m int64
	_ = s.db.QueryRow(`SELECT mtime FROM sources WHERE path = ?`, path).Scan(&m)
	return m
}

func (s *Store) SetSourceMTime(path string, mtime int64) error {
	_, err := s.db.Exec(`INSERT INTO sources(path, mtime) VALUES(?, ?) ON CONFLICT(path) DO UPDATE SET mtime=excluded.mtime`, path, mtime)
	return err
}

func (s *Store) WhyForPath(repo, path string) ([]model.Session, error) {
	all, err := s.List(repo)
	if err != nil {
		return nil, err
	}
	want := strings.ToLower(strings.ReplaceAll(path, "\\", "/"))
	var out []model.Session
	for _, sess := range all {
		for _, f := range sess.Files {
			p := strings.ToLower(strings.ReplaceAll(f.Path, "\\", "/"))
			if p == want || strings.HasSuffix(p, "/"+want) || strings.HasSuffix(want, "/"+p) {
				out = append(out, sess)
				break
			}
		}
	}
	return out, nil
}
