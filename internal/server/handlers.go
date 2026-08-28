package server

import (
	"encoding/json"
	"net/http"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func (s *Server) session(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	sess, err := s.Store.Resolve(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	writeJSON(w, sess)
}

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	sess, err := s.Store.Resolve(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	raw, err := gitx.Diff(s.Repo, sess.FilePaths())
	if err != nil {
		raw = err.Error()
	}
	log, _ := gitx.LogSince(s.Repo, sess.StartedAt, sess.FilePaths())
	writeJSON(w, map[string]any{
		"id":   sess.ID,
		"diff": AnnotateDiff(raw, sess),
		"log":  log,
	})
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
	s.mu.Unlock()
	if po == "" {
		po, _ = gitx.Status(s.Repo)
	}
	br := gitx.Branch(s.Repo)
	writeJSON(w, map[string]any{"repo": s.Repo, "branch": br, "status": po, "scanned": n})
}

func (s *Server) scanNow(w http.ResponseWriter, r *http.Request) {
	s.scanOnce()
	s.git(w, r)
}
