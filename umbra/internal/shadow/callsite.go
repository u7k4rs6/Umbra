package shadow

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// WindowRadius is how far either side of a call site the window looks. Four
// lines is what ARCHITECTURE.md specifies.
const WindowRadius = 4

// Error-handling openers, per language. This is a regex window, not a parser,
// and the limitation is disclosed in PRD.md.
var faultRE = regexp.MustCompile(`(?i)(^|\W)(except\b|catch\b|finally\b|rescue\b|defer\b|try\b|recover\b|if\s+err\s*!=\s*nil)`)

// Annotation markers that pin a call site to the top of the docket. They must
// sit inside a comment, so a symbol named CRITICAL in ordinary code does not
// become a beacon.
var beaconRE = regexp.MustCompile(`(?i)(SAFETY|CRITICAL|INVARIANT)`)

// commentStart matches the comment openers of the languages Graph parses
// semantically. A line whose comment marker precedes the annotation is a
// comment line.
var commentRE = regexp.MustCompile(`^\s*(#|//|/\*|\*|--|;|%|"""|''')`)

// Window is what the lines around a call site say.
type Window struct {
	FaultLine bool
	// Beacon is the annotation comment line, trimmed and capped. It reaches a
	// report only under --snippets; the modifier itself is always shown.
	Beacon string
}

// ReadWindow examines the lines around a call site in the head worktree.
//
// The window is matched in memory. Nothing but the single annotation comment
// line is ever retained, which is what SECURITY_AND_ACCESS.md requires.
func ReadWindow(root, file string, line int) Window {
	if line <= 0 || file == "" {
		return Window{}
	}
	lines, ok := readLines(filepath.Join(root, filepath.FromSlash(file)), line-WindowRadius, line+WindowRadius)
	if !ok {
		return Window{}
	}
	return ScanWindow(lines)
}

// ScanWindow applies the two matchers to a window of lines. It is separate
// from reading so it can be tested without a file system.
func ScanWindow(lines []string) Window {
	var w Window
	for _, ln := range lines {
		if faultRE.MatchString(stripComment(ln)) {
			w.FaultLine = true
		}
		if w.Beacon == "" && commentRE.MatchString(ln) && beaconRE.MatchString(ln) {
			w.Beacon = cap200(strings.TrimSpace(ln))
		}
	}
	return w
}

// stripComment removes a trailing comment so a word like "catch" inside prose
// does not register as error handling.
func stripComment(ln string) string {
	if commentRE.MatchString(ln) {
		return ""
	}
	for _, marker := range []string{"#", "//"} {
		if i := strings.Index(ln, marker); i >= 0 {
			return ln[:i]
		}
	}
	return ln
}

func readLines(path string, from, to int) ([]string, bool) {
	if from < 1 {
		from = 1
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()

	var out []string
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 1<<16), 1<<20)
	n := 0
	for sc.Scan() {
		n++
		if n > to {
			break
		}
		if n >= from {
			out = append(out, sc.Text())
		}
	}
	if sc.Err() != nil {
		return nil, false
	}
	return out, true
}

func cap200(s string) string {
	if len(s) <= 200 {
		return s
	}
	return s[:200] + "..."
}

// FarField reports whether a node sits in a different top-level package or
// directory than its source.
func FarField(nodeFile, sourceFile string) bool {
	return topLevel(nodeFile) != topLevel(sourceFile)
}

func topLevel(p string) string {
	p = strings.TrimPrefix(filepath.ToSlash(p), "./")
	if i := strings.Index(p, "/"); i >= 0 {
		return p[:i]
	}
	return "."
}
