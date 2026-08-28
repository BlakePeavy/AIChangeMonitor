package store

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func TestUpsertPreservesStatus(t *testing.T) {
	p := filepath.Join(t.TempDir(), "index.db")
	st, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	s := model.Session{
		ID: "abc", Agent: model.AgentClaudeCode, Repo: "/repo",
		Intent: "first", Status: model.StatusUnseen,
		StartedAt: time.Unix(1, 0).UTC(),
	}
	if err := st.Upsert(s); err != nil {
		t.Fatal(err)
	}
	if err := st.SetStatus("abc", model.StatusFlagged); err != nil {
		t.Fatal(err)
	}
	s.Intent = "updated"
	s.Status = model.StatusUnseen
	if err := st.Upsert(s); err != nil {
		t.Fatal(err)
	}
	got, err := st.Get("abc")
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != model.StatusFlagged {
		t.Fatalf("status %s", got.Status)
	}
	if got.Intent != "updated" {
		t.Fatalf("intent %s", got.Intent)
	}
	list, err := st.List("/repo")
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
}
