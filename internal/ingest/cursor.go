package ingest

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/paths"
)

func discoverCursor(repo string) []string {
	root := paths.CursorProjectsDir()
	if root == "" {
		return nil
	}
	st, err := os.Stat(root)
	if err != nil || !st.IsDir() {
		return nil
	}
	ents, err := os.ReadDir(root)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() {
			continue
		}
		if !paths.MatchProjectDir(e.Name(), repo) {
			continue
		}
		at := filepath.Join(root, e.Name(), "agent-transcripts")
		out = append(out, listCursorTranscripts(at)...)
	}
	return out
}

func listCursorTranscripts(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		p := filepath.Join(dir, e.Name())
		if e.IsDir() {
			nested := filepath.Join(p, e.Name()+".jsonl")
			if st, err := os.Stat(nested); err == nil && !st.IsDir() {
				out = append(out, nested)
				continue
			}
			out = append(out, walkJSONL(p)...)
			continue
		}
		if strings.HasSuffix(strings.ToLower(e.Name()), ".jsonl") {
			out = append(out, p)
		}
	}
	return out
}

func parseCursorFile(path, repo string, mtime int64) (model.Session, bool) {
	evs, err := readEvents(path)
	if err != nil || len(evs) == 0 {
		return model.Session{}, false
	}
	id := sessionIDFrom(path, evs)
	if id == "" {
		id = filepath.Base(filepath.Dir(path))
	}
	sess := buildSession(model.AgentCursor, id, path, repo, mtime, evs)
	return sess, true
}
