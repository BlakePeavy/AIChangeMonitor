package main

import (
	"fmt"

	"github.com/BlakePeavy/AIChangeMonitor/internal/gitx"
	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
	"github.com/BlakePeavy/AIChangeMonitor/internal/server"
)

func cmdShow(args []string) error {
	fs, repoFlag := repoFlagSet("show")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	st, repo, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer st.Close()
	var sess model.Session
	if id == "" {
		list, err := st.List(repo)
		if err != nil {
			return err
		}
		if len(list) == 0 {
			return fmt.Errorf("no sessions")
		}
		sess = list[0]
	} else {
		sess, err = st.Resolve(id)
		if err != nil {
			return err
		}
	}
	printSession(sess)
	if log, err := gitx.LogSince(repo, sess.StartedAt, sess.FilePaths()); err == nil && log != "" {
		fmt.Printf("git log --since session:\n%s", indent(log))
	}
	return nil
}

func cmdDiff(args []string) error {
	fs, repoFlag := repoFlagSet("diff")
	if err := fs.Parse(args); err != nil {
		return err
	}
	id := fs.Arg(0)
	st, repo, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer st.Close()
	var files []string
	var sess model.Session
	if id != "" {
		sess, err = st.Resolve(id)
		if err != nil {
			return err
		}
		files = sess.FilePaths()
	}
	raw, err := gitx.Diff(repo, files)
	if err != nil {
		return err
	}
	fmt.Print(server.AnnotateDiff(raw, sess))
	return nil
}

func cmdWhy(args []string) error {
	fs, repoFlag := repoFlagSet("why")
	if err := fs.Parse(args); err != nil {
		return err
	}
	path := fs.Arg(0)
	if path == "" {
		return fmt.Errorf("usage: aichange why <path>")
	}
	st, repo, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer st.Close()
	list, err := st.WhyForPath(repo, path)
	if err != nil {
		return err
	}
	if len(list) == 0 {
		fmt.Println("no session touched that path")
		return nil
	}
	for _, s := range list {
		fmt.Printf("== %s  %s  %s  %s\n", shortID(s.ID), s.Agent, fmtTime(s.StartedAt), s.Status)
		fmt.Printf("intent: %s\n", s.Intent)
		fmt.Printf("why:\n%s\n", indent(s.Why))
	}
	return nil
}

func cmdReview(args []string) error {
	fs, repoFlag := repoFlagSet("review")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() < 2 {
		return fmt.Errorf("usage: aichange review <id> accept|flag|seen")
	}
	id, action := fs.Arg(0), fs.Arg(1)
	st, ok := parseReview(action)
	if !ok {
		return fmt.Errorf("status must be accept|flag|seen (or accepted|flagged|unseen)")
	}
	db, _, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer db.Close()
	sess, err := db.Resolve(id)
	if err != nil {
		return err
	}
	if err := db.SetStatus(sess.ID, st); err != nil {
		return err
	}
	sess.Status = st
	printSession(sess)
	return nil
}

func parseReview(a string) (model.Status, bool) {
	switch a {
	case "accept", "accepted":
		return model.StatusAccepted, true
	case "flag", "flagged":
		return model.StatusFlagged, true
	case "seen":
		return model.StatusSeen, true
	case "unseen":
		return model.StatusUnseen, true
	}
	return "", false
}
