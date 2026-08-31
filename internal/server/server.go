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

const commitRefresh = 15 * time.Second

type Server struct {
	Store      *store.Store
	Repo       string
	Poll       time.Duration
	mu         sync.Mutex
	gitPO      string
	lastN      int
	lastHEAD   string
	lastCommit time.Time
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
	mux.HandleFunc("/api/repo", s.setRepo)
	mux.HandleFunc("/api/restore", s.restore)
	mux.HandleFunc("/api/reveal", s.reveal)
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
			s.scanOnce(false)
		}
	}()
}

func (s *Server) repo() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Repo
}

func (s *Server) bindRepo(root string) {
	s.mu.Lock()
	s.Repo = root
	s.gitPO = ""
	s.lastN = 0
	s.lastHEAD = ""
	s.lastCommit = time.Time{}
	s.mu.Unlock()
}

func (s *Server) scanOnce(forceCommits bool) {
	repo := s.repo()
	if repo == "" {
		return
	}
	head := gitx.HEAD(repo)
	s.mu.Lock()
	doCommits := forceCommits || head != s.lastHEAD || time.Since(s.lastCommit) >= commitRefresh
	if doCommits {
		s.lastHEAD = head
		s.lastCommit = time.Now()
	}
	s.mu.Unlock()
	n, _ := ingest.ScanPoll(s.Store, repo, doCommits)
	po, _ := gitx.Status(repo)
	s.mu.Lock()
	if s.Repo == repo {
		s.lastN = n
		s.gitPO = po
	}
	s.mu.Unlock()
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) sessions(w http.ResponseWriter, r *http.Request) {
	list, err := s.Store.List(s.repo())
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	if list == nil {
		list = []model.Session{}
	}
	writeJSON(w, list)
}
