package server

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/ingest"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

const errAlreadyRemote = "already on the remote"
const errNotRestorable = "restore is only for uncommitted or unpushed commit files"

// restoreAllowed is the file-level policy with fake ancestor/upstream flags.
// Live sessions are always allowed. Commit sessions are allowed unless the
// commit is already an ancestor of upstream (pushed). No upstream → unpushed.
func restoreAllowed(source, id string, hasUpstream, ancestorOfUpstream bool) (ok bool, reason string) {
	switch model.DeriveSource(id, source) {
	case model.SourceLive:
		return true, ""
	case model.SourceCommit:
		if hasUpstream && ancestorOfUpstream {
			return false, errAlreadyRemote
		}
		return true, ""
	default:
		return false, errNotRestorable
	}
}

func (s *Server) sessionRestorePerm(sess model.Session) (bool, string) {
	src := model.DeriveSource(sess.ID, sess.Source)
	if src != model.SourceCommit {
		return restoreAllowed(sess.Source, sess.ID, false, false)
	}
	repo := s.repo()
	_, has := gitx.Upstream(repo)
	ancestor := false
	if has {
		sha := strings.TrimPrefix(sess.ID, "git:")
		ok, err := gitx.IsAncestor(repo, sha, "@{u}")
		if err == nil {
			ancestor = ok
		}
	}
	return restoreAllowed(sess.Source, sess.ID, has, ancestor)
}

type sessionDTO struct {
	model.Session
	RestoreAllowed bool   `json:"restore_allowed"`
	RestoreReason  string `json:"restore_reason,omitempty"`
}

func writeJSONStatus(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) restore(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var body struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, 400, map[string]string{"error": err.Error()})
		return
	}
	rel, err := gitx.CleanRelPath(body.Path)
	if err != nil {
		writeJSONStatus(w, 400, map[string]string{"error": err.Error()})
		return
	}
	sess, err := s.Store.Resolve(body.ID)
	if err != nil {
		writeJSONStatus(w, 404, map[string]string{"error": err.Error()})
		return
	}
	ok, reason := s.sessionRestorePerm(sess)
	if !ok {
		code := 400
		if reason == errAlreadyRemote {
			code = 409
		}
		writeJSONStatus(w, code, map[string]string{"error": reason})
		return
	}
	repo := s.repo()
	src := model.DeriveSource(sess.ID, sess.Source)
	var gitSrc string
	switch src {
	case model.SourceLive:
		gitSrc = ""
	case model.SourceCommit:
		sha := strings.TrimPrefix(sess.ID, "git:")
		if gitx.IsHEAD(repo, sha) {
			gitSrc = sha + "^"
		} else if _, has := gitx.Upstream(repo); has {
			gitSrc = "@{u}"
		} else {
			gitSrc = sha + "^"
		}
	default:
		writeJSONStatus(w, 400, map[string]string{"error": errNotRestorable})
		return
	}
	if err := gitx.Restore(repo, gitSrc, rel); err != nil {
		writeJSONStatus(w, 500, map[string]string{"error": err.Error()})
		return
	}
	_, _ = ingest.ScanGit(s.Store, repo, false)
	writeJSON(w, map[string]any{"ok": true, "path": rel})
}

func (s *Server) reveal(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", 405)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONStatus(w, 400, map[string]string{"error": err.Error()})
		return
	}
	rel, err := gitx.CleanRelPath(body.Path)
	if err != nil {
		writeJSONStatus(w, 400, map[string]string{"error": err.Error()})
		return
	}
	if err := gitx.RevealInFolder(s.repo(), rel); err != nil {
		writeJSONStatus(w, 500, map[string]any{"ok": false, "path": rel, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "path": rel})
}
