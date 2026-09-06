package main

import (
	"fmt"
	"strings"
)

// Version is stamped at build time; the binary prints it on every run so a
// reader of a report can tell which build produced it.
var (
	Version = "0.1.0"
	Commit  = "dev"
)

// Options is the whole command surface from PRD.md.
type Options struct {
	Ref string

	Test     string
	Run      string // none, shadow, all
	NoAudit  bool
	History  bool
	Depth    int
	Format   string // table, json, html, packet
	Out      string
	FailOn   string // umbra, penumbra, failure, leak
	Adapter  string // auto, claude-code
	Snippets bool

	All           bool
	KeepWorktrees bool
	Repo          string
	// TestRoot is the repository-relative directory the runner runs in. It is
	// found automatically from the project files near the tests and is only
	// needed when that search picks the wrong one.
	TestRoot string
}

// Exit codes from PRD.md.
const (
	ExitOK      = 0
	ExitRuntime = 1
	ExitFailOn  = 2
)

// DefaultTestRunner is pytest in verbose mode.
//
// The Step 0 probe found that `entire graph verify` cannot parse `pytest -q`:
// quiet mode prints no per-test ids, so the baseline degrades to exit-code
// only and no test can be named. `-v` parses cleanly. PRD.md and the demo
// script still say `-q`; this constant is the correction.
const DefaultTestRunner = "pytest -v"

func validRun(s string) bool {
	switch s {
	case "none", "shadow", "all":
		return true
	}
	return false
}

func validFormat(s string) bool {
	switch s {
	case "table", "json", "html", "packet":
		return true
	}
	return false
}

func validFailOn(s string) bool {
	switch s {
	case "", "umbra", "penumbra", "failure", "leak":
		return true
	}
	return false
}

func validAdapter(s string) bool {
	switch s {
	case "auto", "claude-code":
		return true
	}
	return false
}

// Validate checks the flag combination and returns a plain message.
func (o *Options) Validate() error {
	if !validRun(o.Run) {
		return fmt.Errorf("--run must be none, shadow or all, not %q", o.Run)
	}
	if !validFormat(o.Format) {
		return fmt.Errorf("--format must be table, json, html or packet, not %q", o.Format)
	}
	if !validFailOn(o.FailOn) {
		return fmt.Errorf("--fail-on must be umbra, penumbra, failure or leak, not %q", o.FailOn)
	}
	if !validAdapter(o.Adapter) {
		return fmt.Errorf("--adapter must be auto or claude-code, not %q", o.Adapter)
	}
	if o.Depth < 1 || o.Depth > 3 {
		return fmt.Errorf("--depth must be 1, 2 or 3, not %d", o.Depth)
	}
	if o.Run != "none" && strings.TrimSpace(o.Test) == "" {
		return fmt.Errorf("--run %s needs a test runner: pass --test %q or --run none", o.Run, DefaultTestRunner)
	}
	if (o.Format == "html" || o.Format == "packet") && o.Out == "" {
		return fmt.Errorf("--format %s needs --out to say where to write", o.Format)
	}
	return nil
}

// SplitWords parses a runner prefix into argv exactly once, at flag parsing.
// Umbra never joins it back into a shell string, so nothing in the prefix is
// ever interpreted by a shell.
//
// It understands single quotes, double quotes and backslash escaping, which is
// enough for a test command a person types. Anything more exotic is a sign the
// user wants a shell, and Umbra deliberately does not give them one.
func SplitWords(s string) ([]string, error) {
	var out []string
	var cur strings.Builder
	inWord := false
	quote := byte(0)

	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			if c == quote {
				quote = 0
				continue
			}
			if c == '\\' && quote == '"' && i+1 < len(s) {
				i++
				cur.WriteByte(s[i])
				continue
			}
			cur.WriteByte(c)
		case c == '\'' || c == '"':
			quote = c
			inWord = true
		case c == '\\' && i+1 < len(s):
			i++
			cur.WriteByte(s[i])
			inWord = true
		case c == ' ' || c == '\t' || c == '\n':
			if inWord {
				out = append(out, cur.String())
				cur.Reset()
				inWord = false
			}
		default:
			cur.WriteByte(c)
			inWord = true
		}
	}
	if quote != 0 {
		return nil, fmt.Errorf("unbalanced %c quote in the test command", quote)
	}
	if inWord {
		out = append(out, cur.String())
	}
	return out, nil
}
