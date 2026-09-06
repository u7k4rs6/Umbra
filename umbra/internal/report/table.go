package report

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// TableOptions controls the terminal rendering.
type TableOptions struct {
	// All shows lit nodes as rows instead of a count.
	All bool
	// Colour is on only when stdout is a terminal, and only ever paints the
	// glyph and the outcome word.
	Colour bool
	// UTF8 chooses the round glyphs over the L, P, U and ? fallback.
	UTF8 bool
	// Width caps the symbol and file columns.
	Width int
}

// DetectTableOptions reads the environment the way a terminal program should.
func DetectTableOptions(all bool) TableOptions {
	o := TableOptions{All: all, UTF8: utf8Terminal(), Width: 100}
	if fi, err := os.Stdout.Stat(); err == nil {
		o.Colour = (fi.Mode() & os.ModeCharDevice) != 0
	}
	if os.Getenv("NO_COLOR") != "" {
		o.Colour = false
	}
	return o
}

func utf8Terminal() bool {
	for _, key := range []string{"LC_ALL", "LC_CTYPE", "LANG"} {
		v := os.Getenv(key)
		if v == "" {
			continue
		}
		return strings.Contains(strings.ToUpper(v), "UTF-8") || strings.Contains(strings.ToUpper(v), "UTF8")
	}
	return false
}

const (
	ansiReset  = "\x1b[0m"
	ansiDim    = "\x1b[2m"
	ansiYellow = "\x1b[33m"
	ansiRed    = "\x1b[31m"
	ansiGreen  = "\x1b[32m"
)

// Table writes the terminal rendering from FRONTEND_SPEC.md.
func Table(w io.Writer, a *Analysis, o TableOptions) error {
	b := &strings.Builder{}

	// Header.
	id := a.CheckpointID
	if id == "" {
		id = "(no checkpoint)"
	}
	fmt.Fprintf(b, "Umbra  %s  %s  %s  depth %d\n", id, shortSHA(a.Commit), a.Adapter, a.Depth)

	if a.SessionSaid != "" {
		fmt.Fprintf(b, "session said  %q\n", a.SessionSaid)
		if a.SessionSaidFrom != "" {
			fmt.Fprintf(b, "              %s\n", dim(o, "from "+a.SessionSaidFrom))
		}
	}
	for _, n := range a.Notes {
		fmt.Fprintf(b, "note   %s\n", n)
	}

	// Sources: the lights.
	for _, s := range a.Sources {
		fmt.Fprintf(b, "%s  %s changed  %s:%d\n", s.Name, s.Change, s.File, s.Span[0])
	}
	if len(a.Sources) == 0 {
		fmt.Fprintf(b, "%s\n", dim(o, "no changed entities were reported for this commit"))
	}

	// The light line.
	writeLightLine(b, a, o)
	b.WriteString("\n")

	// The docket.
	rows := a.Nodes
	if !o.All {
		rows = a.Shadowed()
	}
	if len(rows) == 0 {
		if a.Summary.Lit > 0 {
			fmt.Fprintf(b, " %s\n", dim(o, "nothing is in shadow: every dependent was examined"))
		} else {
			fmt.Fprintf(b, " %s\n", dim(o, "no dependents were found for the changed symbols"))
		}
	}
	for _, n := range rows {
		writeRow(b, n, o)
	}
	if !o.All && a.Summary.Lit > 0 {
		b.WriteString("\n")
		fmt.Fprintf(b, " %s\n", dim(o, fmt.Sprintf("%d lit node(s) not shown; pass --all to list them", a.Summary.Lit)))
	}

	// Execution and the sweep.
	writeExecution(b, a, o)

	_, err := io.WriteString(w, b.String())
	return err
}

func writeLightLine(b *strings.Builder, a *Analysis, o TableOptions) {
	s := a.Summary
	bar := strings.Repeat(glyph(shadow.Lit, o), s.Lit) +
		strings.Repeat(glyph(shadow.Penumbra, o), s.Penumbra) +
		strings.Repeat(glyph(shadow.Umbra, o), s.Umbra) +
		strings.Repeat(glyph(shadow.Unknown, o), s.Unknown)

	fmt.Fprintf(b, "light  %s   %d lit  %d penumbra  %d umbra  %d unknown",
		bar, s.Lit, s.Penumbra, s.Umbra, s.Unknown)
	if frac, ok := s.Illumination(); ok {
		fmt.Fprintf(b, "   %.0f%% examined", frac*100)
	} else {
		b.WriteString("   examined set unavailable")
	}
	b.WriteString("\n")
}

func writeRow(b *strings.Builder, n *shadow.Node, o TableOptions) {
	pin := " "
	if n.Pinned() {
		pin = "!"
	}
	loc := fmt.Sprintf("%s:%d", n.Symbol.File, n.Symbol.Span[0])

	mods := modifiersFor(n)
	outcome := ""
	switch n.Result {
	case shadow.OutcomeFail:
		outcome = colour(o, ansiRed, "fail")
	case shadow.OutcomePass:
		outcome = colour(o, ansiGreen, "pass")
	}

	fmt.Fprintf(b, "%s%s  %-24s %-40s %-18s depth %d  %-26s %6.1f  %s\n",
		pin,
		glyphColoured(n.State, o),
		trunc(n.Symbol.Name, 24),
		trunc(loc, 40),
		trunc(n.Relation, 18),
		n.Depth,
		trunc(strings.Join(mods, " "), 26),
		n.Score,
		outcome,
	)
	if n.Pinned() {
		fmt.Fprintf(b, "    %s\n", dim(o, n.Beacon))
	}
}

// modifiersFor drops the modifiers that are already visible elsewhere in the
// row, so the column carries only what the reader cannot otherwise see.
func modifiersFor(n *shadow.Node) []string {
	var out []string
	for _, m := range n.Modifiers {
		if m == "transitive" {
			continue // the depth column already says this
		}
		out = append(out, m)
	}
	return out
}

func writeExecution(b *strings.Builder, a *Analysis, o TableOptions) {
	e := a.Execution
	b.WriteString("\n")

	switch {
	case a.Run == "none":
		fmt.Fprintf(b, "probes  %s\n", dim(o, "not run (--run none)"))
	case len(e.Selected) == 0:
		fmt.Fprintf(b, "probes  %s\n", dim(o, "no test reaches the shadow"))
	default:
		cracked := len(e.NewFailures)
		line := fmt.Sprintf("probes  %d selected  %d cracked", len(e.Selected), cracked)
		if cracked > 0 {
			line += "  " + strings.Join(e.NewFailures, "  ")
		}
		b.WriteString(line + "\n")
		if e.Degraded {
			fmt.Fprintf(b, "        %s\n", dim(o, "the runner printed no per-test ids, so the verdict is suite level; add -v"))
		}
		if len(e.NotRunnable) > 0 {
			fmt.Fprintf(b, "        %s\n", dim(o, fmt.Sprintf("%d test id(s) not runnable and dropped", len(e.NotRunnable))))
		}
	}

	if !a.Audit {
		fmt.Fprintf(b, "sweep   %s\n", dim(o, "skipped (--no-audit), so the selection is unaudited"))
		return
	}
	if a.Run == "none" {
		return
	}
	word := "leaks"
	if len(e.Leaks) == 1 {
		word = "leak"
	}
	fmt.Fprintf(b, "sweep   full suite  %d %s\n", len(e.Leaks), word)
	for _, l := range e.Leaks {
		fmt.Fprintf(b, "        %s  %s\n", l.Test, dim(o, l.Reason))
	}
	if e.SweepCut {
		fmt.Fprintf(b, "        %s\n", dim(o, "the sweep was cut short by its timeout"))
	}
}

func glyph(s shadow.State, o TableOptions) string {
	if o.UTF8 {
		return s.Glyph()
	}
	return s.ASCII()
}

func glyphColoured(s shadow.State, o TableOptions) string {
	g := glyph(s, o)
	switch s {
	case shadow.Umbra:
		return colour(o, ansiRed, g)
	case shadow.Penumbra:
		return colour(o, ansiYellow, g)
	case shadow.Lit:
		return colour(o, ansiGreen, g)
	}
	return g
}

func colour(o TableOptions, code, s string) string {
	if !o.Colour {
		return s
	}
	return code + s + ansiReset
}

func dim(o TableOptions, s string) string { return colour(o, ansiDim, s) }

func trunc(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 3 {
		return string(r[:n])
	}
	return "..." + string(r[len(r)-(n-3):])
}

func shortSHA(s string) string {
	if len(s) > 7 {
		return s[:7]
	}
	if s == "" {
		return "(no commit)"
	}
	return s
}
