package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "aichange: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	cmd := "serve"
	rest := args
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		cmd = args[0]
		rest = args[1:]
	}
	switch cmd {
	case "serve":
		return cmdServe(rest)
	case "sessions":
		return cmdSessions(rest)
	case "show":
		return cmdShow(rest)
	case "diff":
		return cmdDiff(rest)
	case "why":
		return cmdWhy(rest)
	case "review":
		return cmdReview(rest)
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	default:
		return fmt.Errorf("unknown command %q\n\n%s", cmd, usage)
	}
}

const usage = `aichange — review agent sessions against the dirty working tree

usage:
  aichange [serve] [--repo PATH] [--addr :7380] [--no-poll]
  aichange sessions [--json] [--repo PATH]
  aichange show [id]
  aichange diff [id]
  aichange why <path>
  aichange review <id> accept|flag|seen

Git is the ledger. This tool does not write the monitored repo.
`
