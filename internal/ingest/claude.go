package ingest

import (
	"bufio"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/paths"
)

func discoverClaude(repo string) []string {
	var out []string
	for _, root := range paths.ClaudeProjectsDirs() {
		st, err := os.Stat(root)
		if err != nil || !st.IsDir() {
			continue
		}
		ents, err := os.ReadDir(root)
		if err != nil {
			continue
		}
		for _, e := range ents {
			if !e.IsDir() {
				continue
			}
			if !paths.MatchProjectDir(e.Name(), repo) {
				continue
			}
			proj := filepath.Join(root, e.Name())
			out = append(out, walkJSONL(proj)...)
		}
	}
	return out
}

func walkJSONL(root string) []string {
	var out []string
	_ = filepath.WalkDir(root, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := strings.ToLower(d.Name())
			if name == "subagents" || name == "subagent" {
				return fs.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(strings.ToLower(d.Name()), ".jsonl") {
			out = append(out, p)
		}
		return nil
	})
	return out
}

func readEvents(path string) ([]event, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var evs []event
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)
	for sc.Scan() {
		if ev, ok := parseLine(sc.Bytes()); ok {
			evs = append(evs, ev)
		}
	}
	return evs, sc.Err()
}

func sessionIDFrom(path string, evs []event) string {
	for _, ev := range evs {
		if ev.SessionID != "" {
			return ev.SessionID
		}
	}
	base := filepath.Base(path)
	return strings.TrimSuffix(base, filepath.Ext(base))
}

func parseClaudeFile(path, repo string, mtime int64) (model.Session, bool) {
	evs, err := readEvents(path)
	if err != nil || len(evs) == 0 {
		return model.Session{}, false
	}
	id := sessionIDFrom(path, evs)
	sess := buildSession(model.AgentClaudeCode, id, path, repo, mtime, evs)
	if sess.CWD != "" && !paths.MatchSessionCWD(sess.CWD, repo) && !paths.MatchProjectDir(filepath.Base(filepath.Dir(path)), repo) {
		// still accept: discover already matched the project folder
	}
	return sess, true
}
