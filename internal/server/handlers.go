package server

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/ingest"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	sess, err := s.Store.Resolve(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	ok, reason := s.sessionRestorePerm(sess)
	writeJSON(w, sessionDTO{Session: sess, RestoreAllowed: ok, RestoreReason: reason})
}

func (s *Server) review(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var body struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	st, ok := model.ParseStatus(body.Status)
	if !ok {
		http.Error(w, "status must be unseen|seen|accepted|flagged", 400)
		return
	}
	sess, err := s.Store.Resolve(body.ID)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	if err := s.Store.SetStatus(sess.ID, st); err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	sess.Status = st
	writeJSON(w, sess)
}

func (s *Server) git(w http.ResponseWriter, r *http.Request) {
	s.mu.Lock()
	po := s.gitPO
	n := s.lastN
	repo := s.Repo
	s.mu.Unlock()
	if po == "" {
		po, _ = gitx.Status(repo)
	}
	br := gitx.Branch(repo)
	writeJSON(w, map[string]any{"repo": repo, "branch": br, "status": po, "scanned": n})
}

func (s *Server) scanNow(w http.ResponseWriter, r *http.Request) {
	s.scanOnce(true)
	s.git(w, r)
}

func (s *Server) setRepo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	path := strings.TrimSpace(body.Path)
	if path == "" {
		http.Error(w, "path required", 400)
		return
	}
	root, err := gitx.RepoRoot(path)
	if err != nil {
		http.Error(w, "not a git repo", 400)
		return
	}
	root = filepath.Clean(root)
	s.bindRepo(root)
	_, _ = ingest.Scan(s.Store, root)
	writeJSON(w, map[string]any{"repo": root, "branch": gitx.Branch(root)})
}
