package main

import (
	"flag"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/server"
)

func cmdServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	repoFlag := fs.String("repo", "", "git repository root")
	addr := fs.String("addr", ":7380", "listen address")
	noPoll := fs.Bool("no-poll", false, "scan once; do not poll transcript dirs")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, repo, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer st.Close()
	poll := 2 * time.Second
	if *noPoll {
		poll = 0
	}
	srv := server.New(st, repo, poll)
	srv.StartPoll()
	ln, err := net.Listen("tcp", *addr)
	if err != nil {
		return err
	}
	url := displayURL(*addr)
	fmt.Printf("aichange listening on %s\n", url)
	fmt.Printf("repo %s\n", repo)
	return http.Serve(ln, srv.Handler())
}

func displayURL(addr string) string {
	host := "127.0.0.1"
	port := addr
	if strings.HasPrefix(addr, ":") {
		port = addr
		return "http://" + host + port
	}
	return "http://" + addr
}
