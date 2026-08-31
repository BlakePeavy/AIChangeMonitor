package ingest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/redact"
	"github.com/BlakePeavy/AIChangeMonitor/internal/risk"
	"github.com/BlakePeavy/AIChangeMonitor/internal/store"
)

const defaultLogN = gitx.DefaultLogN

// ScanGit upserts recent commits (optional) and one live dirty-tree session.
func ScanGit(st *store.Store, repo string, commits bool) (int, error) {
	n := 0
	if commits {
		nn, err := upsertCommits(st, repo)
		if err != nil {
			return n, err
		}
		n += nn
	}
	nn, err := upsertLive(st, repo)
	if err != nil {
		return n, err
	}
	return n + nn, nil
}

func upsertCommits(st *store.Store, repo string) (int, error) {
	list, err := gitx.LogNumstat(repo, defaultLogN)
	if err != nil {
		return 0, err
	}
	branch := gitx.Branch(repo)
	n := 0
	for _, c := range list {
		if c.Hash == "" {
			continue
		}
		sess := sessionFromCommit(c, repo, branch)
		if err := st.Upsert(sess); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func sessionFromCommit(c gitx.Commit, repo, branch string) model.Session {
	id := "git:" + c.Hash
	agent := agentFromTrailers(c.Subject, c.Body, c.Author)
	if agent == "" {
		agent = model.AgentGit
	}
	files := statsToFiles(c.Files)
	add, del := sumStats(c.Files)
	why := ""
	if strings.TrimSpace(c.Body) != "" {
		why = redact.Redact(strings.TrimSpace(c.Body))
	}
	return model.Session{
		ID:           id,
		Agent:        agent,
		StartedAt:    c.Time,
		EndedAt:      c.Time,
		CWD:          repo,
		Repo:         repo,
		Branch:       branch,
		Intent:       redact.Redact(c.Subject),
		Why:          why,
		Files:        files,
		Status:       model.StatusUnseen,
		Risks:        risk.Chips(files, add, del),
		Source:       model.SourceCommit,
		SourcePath:   id,
		AddedLines:   add,
		DeletedLines: del,
	}
}

func upsertLive(st *store.Store, repo string) (int, error) {
	id := liveID(repo)
	files, err := gitx.DirtyFiles(repo)
	if err != nil {
		return 0, err
	}
	if len(files) == 0 {
		_ = st.Delete(id)
		return 0, nil
	}
	mf := statsToFiles(files)
	add, del := sumStats(files)
	now := time.Now()
	started := now
	if existing, err := st.Get(id); err == nil && !existing.StartedAt.IsZero() {
		started = existing.StartedAt
	}
	sess := model.Session{
		ID:           id,
		Agent:        model.AgentLive,
		StartedAt:    started,
		EndedAt:      now,
		CWD:          repo,
		Repo:         repo,
		Branch:       gitx.Branch(repo),
		Intent:       fmt.Sprintf("Uncommitted (·%d files)", len(mf)),
		Why:          "",
		Files:        mf,
		Status:       model.StatusUnseen,
		Risks:        risk.Chips(mf, add, del),
		Source:       model.SourceLive,
		SourcePath:   id,
		AddedLines:   add,
		DeletedLines: del,
	}
	if err := st.Upsert(sess); err != nil {
		return 0, err
	}
	return 1, nil
}

func liveID(repo string) string {
	return "live:" + repoFingerprint(repo)
}

func repoFingerprint(repo string) string {
	sum := sha256.Sum256([]byte(filepath.ToSlash(filepath.Clean(repo))))
	return hex.EncodeToString(sum[:8])
}

func statsToFiles(stats []gitx.FileStat) []model.File {
	out := make([]model.File, 0, len(stats))
	for _, s := range stats {
		if s.Path == "" {
			continue
		}
		out = append(out, model.File{
			Path:    s.Path,
			Delete:  s.Delete,
			Added:   s.Added,
			Deleted: s.Deleted,
		})
	}
	return out
}

func sumStats(stats []gitx.FileStat) (add, del int) {
	for _, s := range stats {
		add += s.Added
		del += s.Deleted
	}
	return add, del
}

// agentFromTrailers maps obvious commit trailers to an agent. Empty means git.
// Do not treat the result as certainty — it is a badge heuristic.
func agentFromTrailers(subject, body, author string) model.Agent {
	blob := strings.ToLower(subject + "\n" + body + "\n" + author)
	switch {
	case strings.Contains(blob, "copilot"):
		return model.AgentCopilot
	case strings.Contains(blob, "cursor"):
		return model.AgentCursor
	case strings.Contains(blob, "claude"):
		return model.AgentClaudeCode
	case strings.Contains(blob, "aider"):
		return model.AgentAider
	case strings.Contains(blob, "codex"):
		return model.AgentCodex
	case strings.Contains(blob, "windsurf"):
		return model.AgentWindsurf
	case strings.Contains(blob, "generated with"):
		return model.AgentUnknownAgent
	default:
		return ""
	}
}
