package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/BlakePeavy/AIChangeMonitor/internal/model"
)

func cmdSessions(args []string) error {
	fs, repoFlag := repoFlagSet("sessions")
	asJSON := fs.Bool("json", false, "print JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	st, repo, err := openApp(*repoFlag)
	if err != nil {
		return err
	}
	defer st.Close()
	list, err := st.List(repo)
	if err != nil {
		return err
	}
	if *asJSON {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(list)
	}
	w := tabwriter.NewWriter(os.Stdout, 0, 2, 2, ' ', 0)
	fmt.Fprintln(w, "ID\tTIME\tAGENT\tFILES\tRISK\tSTATUS\tINTENT")
	for _, s := range list {
		fmt.Fprintf(w, "%s\t%s\t%s\t%d\t%s\t%s\t%s\n",
			shortID(s.ID), fmtTime(s.StartedAt), s.Agent, len(s.Files),
			strings.Join(s.Risks, ","), s.Status, oneLine(s.Intent, 60))
	}
	return w.Flush()
}

func shortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func fmtTime(t time.Time) string {
	if t.IsZero() {
		return "—"
	}
	return t.Local().Format("2006-01-02 15:04")
}

func oneLine(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if len(s) > n {
		return s[:n] + "…"
	}
	return s
}

func printSession(s model.Session) {
	fmt.Printf("id:      %s\n", s.ID)
	fmt.Printf("agent:   %s\n", s.Agent)
	fmt.Printf("when:    %s\n", fmtTime(s.StartedAt))
	fmt.Printf("branch:  %s\n", s.Branch)
	fmt.Printf("cwd:     %s\n", s.CWD)
	fmt.Printf("status:  %s\n", s.Status)
	fmt.Printf("risks:   %s\n", strings.Join(s.Risks, ", "))
	fmt.Printf("intent:  %s\n", s.Intent)
	fmt.Printf("why:\n%s\n", indent(s.Why))
	fmt.Printf("files:\n")
	for _, f := range s.Files {
		mark := ""
		if f.Delete {
			mark = " (deleted)"
		}
		fmt.Printf("  - %s%s\n", f.Path, mark)
	}
}

func indent(s string) string {
	if s == "" {
		return "  (none)\n"
	}
	var b strings.Builder
	for _, ln := range strings.Split(s, "\n") {
		b.WriteString("  ")
		b.WriteString(ln)
		b.WriteByte('\n')
	}
	return b.String()
}
