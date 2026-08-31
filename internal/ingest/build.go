package ingest

import (
	"strings"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/paths"
	"github.com/BlakePeavy/AIChangeMonitor/internal/redact"
)

func buildSession(agent model.Agent, id, source, repo string, mtime int64, evs []event) model.Session {
	sess := model.Session{
		ID:          id,
		Agent:       agent,
		Source:      model.SourceTranscript,
		SourcePath:  source,
		SourceMTime: mtime,
		Repo:        repo,
		Status:      model.StatusUnseen,
	}
	var whyParts []string
	var sawEdit bool
	var lastPrompt string
	var ops []model.File
	for _, ev := range evs {
		if !ev.Time.IsZero() {
			if sess.StartedAt.IsZero() || ev.Time.Before(sess.StartedAt) {
				sess.StartedAt = ev.Time
			}
			if ev.Time.After(sess.EndedAt) {
				sess.EndedAt = ev.Time
			}
		}
		if ev.CWD != "" && sess.CWD == "" {
			sess.CWD = ev.CWD
		}
		if ev.Branch != "" && sess.Branch == "" {
			sess.Branch = ev.Branch
		}
		if ev.Human {
			p := humanPrompt(ev.UserText)
			if p == "" {
				p = humanPrompt(joinTexts(ev.Texts))
			}
			if p != "" {
				lastPrompt = p
				if sess.Intent == "" {
					sess.Intent = p
				}
			}
		}
		if ev.isAssistant() && !sawEdit {
			whyParts = append(whyParts, ev.Thinking...)
			whyParts = append(whyParts, ev.Texts...)
		}
		for _, t := range ev.Tools {
			if isEditWrite(t.Name) {
				sawEdit = true
			}
			p := t.Path
			if p == "" {
				continue
			}
			if strings.EqualFold(t.Name, "Bash") || strings.EqualFold(t.Name, "Shell") {
				if !isMutateTool(t.Name) && p != "" {
					cmd := ""
					if t.Input != nil {
						cmd = asString(t.Input["command"])
					}
					del := t.Delete || bashIsDelete(cmd)
					ops = append(ops, model.File{Path: rel(p, repo), Tool: t.Name, Delete: del, Prompt: lastPrompt})
				}
				continue
			}
			if !isMutateTool(t.Name) && p != "" && !isEditWrite(t.Name) {
				continue
			}
			ops = append(ops, model.File{
				Path:   rel(p, repo),
				Tool:   t.Name,
				Delete: t.Delete,
				Prompt: lastPrompt,
			})
		}
	}
	if sess.EndedAt.IsZero() {
		sess.EndedAt = sess.StartedAt
	}
	sess.Why = whyExcerpt(whyParts)
	sess.Intent = redact.Redact(sess.Intent)
	sess.Files = joinFiles(ops)
	if sess.CWD == "" {
		sess.CWD = repo
	}
	return sess
}

func rel(p, repo string) string {
	return paths.RelToRepo(p, repo)
}
