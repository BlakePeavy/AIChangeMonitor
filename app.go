package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/ingest"
	"github.com/BlakePeavy/AIChangeMonitor/internal/paths"
	"github.com/BlakePeavy/AIChangeMonitor/internal/store"
)

func openApp(repoFlag string) (st *store.Store, repo string, err error) {
	repo, err = resolveRepo(repoFlag)
	if err != nil {
		return nil, "", err
	}
	st, err = store.Open(paths.IndexPath())
	if err != nil {
		return nil, "", err
	}
	if _, err := ingest.Scan(st, repo); err != nil {
		st.Close()
		return nil, "", err
	}
	return st, repo, nil
}

func resolveRepo(repoFlag string) (string, error) {
	dir := repoFlag
	if dir == "" {
		wd, err := os.Getwd()
		if err != nil {
			return "", err
		}
		dir = wd
	}
	root, err := gitx.RepoRoot(dir)
	if err != nil {
		return "", fmt.Errorf("not a git repo (pass --repo): %w", err)
	}
	return filepath.Clean(root), nil
}

func repoFlagSet(name string) (*flag.FlagSet, *string) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	repo := fs.String("repo", "", "git repository root (default: current directory)")
	return fs, repo
}
