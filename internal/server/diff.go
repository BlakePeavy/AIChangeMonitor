package server

import (
	"net/http"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func (s *Server) diff(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Query().Get("id")
	file := strings.TrimSpace(r.URL.Query().Get("file"))
	folder := strings.TrimSpace(r.URL.Query().Get("folder"))
	sess, err := s.Store.Resolve(id)
	if err != nil {
		http.Error(w, err.Error(), 404)
		return
	}
	paths := scopedDiffPaths(sess, file, folder)
	repo := s.repo()
	raw, err := sessionDiff(repo, sess, paths, file != "" || folder != "")
	if err != nil {
		raw = err.Error()
	}
	log, _ := gitx.LogSince(repo, sess.StartedAt, sess.FilePaths())
	writeJSON(w, map[string]any{
		"id":   sess.ID,
		"diff": AnnotateDiff(raw, sess),
		"log":  log,
	})
}

// scopedDiffPaths picks one file, a folder's files, or the whole session.
func scopedDiffPaths(sess model.Session, file, folder string) []string {
	if file != "" {
		return []string{file}
	}
	if folder != "" {
		return model.PathsInFolder(sess.Files, folder)
	}
	return sess.FilePaths()
}

// SessionDiff picks git show for commits, worktree+cached for live, worktree for transcripts.
func SessionDiff(repo string, sess model.Session) (string, error) {
	return sessionDiff(repo, sess, sess.FilePaths(), false)
}

func sessionDiff(repo string, sess model.Session, files []string, scoped bool) (string, error) {
	if scoped && len(files) == 0 {
		return "", nil
	}
	src := model.DeriveSource(sess.ID, sess.Source)
	switch src {
	case model.SourceCommit:
		sha := strings.TrimPrefix(sess.ID, "git:")
		return gitx.Show(repo, sha, files)
	case model.SourceLive:
		if scoped {
			return gitx.DiffHEAD(repo, files)
		}
		return gitx.DiffWorktree(repo, files)
	default:
		return gitx.Diff(repo, files)
	}
}
