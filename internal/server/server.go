package server

import (
	"encoding/json"
	"io/fs"
	"net/http"
	"sync"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/ingest"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/store"
	"github.com/BlakePeavy/AIChangeMonitor/ui"
)

type Server struct {
	Store  *store.Store
	Repo   string
	Poll   time.Duration
	mu     sync.Mutex
	gitPO  string
	lastN  int
}

func New(st *store.Store, repo string, poll time.Duration) *Server {
	return &Server{Store: st, Repo: repo, Poll: poll}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	static, err := fs.Sub(ui.FS, ".")
	if err != nil {
		static = ui.FS
	}
	mux.Handle("/", http.FileServer(http.FS(static)))
	mux.HandleFunc("/api/sessions", s.sessions)
	mux.HandleFunc("/api/session", s.session)
	mux.HandleFunc("/api/diff", s.diff)
	mux.HandleFunc("/api/review", s.review)
	mux.HandleFunc("/api/git", s.git)
	mux.HandleFunc("/api/scan", s.scanNow)
	return mux
}

func (s *Server) StartPoll() {
	if s.Poll <= 0 {
		return
	}
	go func() {
		t := time.NewTicker(s.Poll)
		defer t.Stop()
		for range t.C {
			s.scanOnce()
		}
	}()
}

func (s *Server) scanOnce() {
	n, _ := ingest.Scan(s.Store, s.Repo)
	po, _ := gitx.Status(s.Repo)
	s.mu.Lock()
	s.lastN = n
	s.gitPO = po
	s.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.List(s.Repo)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []model.Session{}
	}
	writeJSON(w, list)
}
