package ingest

import (
	"os"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/risk"
	"github.com/BlakePeavy/AIChangeMonitor/internal/store"
)

func Scan(st *store.Store, repo string) (int, error) {
	files := append(discoverClaude(repo), discoverCursor(repo)...)
	n := 0
	for _, path := range files {
		info, err := os.Stat(path)
		if err != nil {
			continue
		}
		mtime := info.ModTime().UnixNano()
		if st.SourceMTime(path) == mtime {
			continue
		}
		var sess model.Session
		var ok bool
		if isCursorPath(path) {
			sess, ok = parseCursorFile(path, repo, mtime)
		} else {
			sess, ok = parseClaudeFile(path, repo, mtime)
		}
		if !ok {
			_ = st.SetSourceMTime(path, mtime)
			continue
		}
		if add, del, err := gitx.DiffStat(repo, sess.FilePaths()); err == nil {
			sess.AddedLines = add
			sess.DeletedLines = del
		}
		sess.Risks = risk.Chips(sess.Files, sess.AddedLines, sess.DeletedLines)
		if err := st.Upsert(sess); err != nil {
			return n, err
		}
		_ = st.SetSourceMTime(path, mtime)
		n++
	}
	return n, nil
}

func isCursorPath(p string) bool {
	return containsFold(p, "agent-transcripts")
}

func containsFold(p, sub string) bool {
	return len(p) >= len(sub) && (indexFold(p, sub) >= 0)
}

func indexFold(s, sub string) int {
	ls, lsub := []rune(s), []rune(sub)
	for i := 0; i <= len(ls)-len(lsub); i++ {
		ok := true
		for j := 0; j < len(lsub); j++ {
			a, b := ls[i+j], lsub[j]
			if a >= 'A' && a <= 'Z' {
				a += 'a' - 'A'
			}
			if b >= 'A' && b <= 'Z' {
				b += 'a' - 'A'
			}
			if a != b {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
}
