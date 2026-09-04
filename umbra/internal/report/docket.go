package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// DocketRow is one row of the table. The docket is rendered by the Go template
// rather than by JavaScript, so it is present and readable with scripting
// turned off; the JavaScript only enhances it.
type DocketRow struct {
	ID        string
	Glyph     string
	State     string
	Cracked   bool
	Name      string
	Kind      string
	File      string
	Line      int
	Relation  string
	Depth     int
	Chips     []Chip
	Score     string
	Factors   string
	Probes    string
	Outcome   string
	Excerpt   string
	Beacon    string
	Pinned    bool
	Sentence  string
	Path      []string
	IsLeak    bool
	Reason    string
	Signature string
}

// Chip is a coined word with its plain meaning underneath, which is how
// FRONTEND_SPEC.md requires every coined word to appear.
type Chip struct {
	Word    string
	Meaning string
}

var chipMeanings = map[string]string{
	"glance":         "a partial read that missed it",
	"glimpse":        "only a search hit",
	"quoted":         "content seen in a tool result",
	"afterimage":     "read only before the change",
	"echo":           "named, never opened",
	"far field":      "across a package boundary",
	"fault line":     "call site inside error handling",
	"beacon":         "annotated call site",
	"scar":           "several recent fix commits",
	"test":           "a test function",
	"transitive":     "reached through another symbol",
	"co-change only": "no code path; changes together",
}

// BuildDocket turns the analysis into table rows, shadowed first in docket
// order, then the leaks the sweep found, which get rows of their own.
func BuildDocket(a *Analysis) []DocketRow {
	var rows []DocketRow
	for _, n := range a.Nodes {
		rows = append(rows, rowFor(a, n))
	}
	for _, l := range a.Execution.Leaks {
		rows = append(rows, DocketRow{
			ID:     "leak:" + l.Test,
			Glyph:  "!",
			State:  "leak",
			Name:   l.Test,
			IsLeak: true,
			Reason: l.Reason,
			Chips:  []Chip{{Word: "leak", Meaning: "the sweep found it; selection missed it"}},
		})
	}
	return rows
}

func rowFor(a *Analysis, n *shadow.Node) DocketRow {
	r := DocketRow{
		ID:       n.Symbol.ID,
		Glyph:    n.State.Glyph(),
		State:    n.State.String(),
		Name:     n.Symbol.Name,
		File:     n.Symbol.File,
		Line:     n.Symbol.Span[0],
		Relation: n.Relation,
		Depth:    n.Depth,
		Score:    fmt.Sprintf("%.1f", n.Score),
		Factors:  factorPopover(n),
		Sentence: shadow.StateSentence(n.State, n.Tier, n.Symbol.File),
		Path:     n.Path,
		Beacon:   n.Beacon,
		Pinned:   n.Pinned(),
		Cracked:  n.Result == shadow.OutcomeFail,
		Excerpt:  n.Failure,
	}
	switch n.Symbol.Kind {
	case "class", "type", "interface", "struct":
		r.Kind = n.Symbol.Kind
	}
	if a.Snippets {
		r.Signature = Cap(n.Symbol.Signature, 200)
	}
	for _, m := range n.Modifiers {
		r.Chips = append(r.Chips, Chip{Word: m, Meaning: chipMeanings[m]})
	}
	switch n.Result {
	case shadow.OutcomeFail:
		r.Outcome = "cracked"
	case shadow.OutcomePass:
		r.Outcome = "pass"
	}
	if len(n.Tests) > 0 {
		worst := "pass"
		for _, tr := range n.Tests {
			if tr.Outcome == shadow.OutcomeFail {
				worst = "cracked"
			}
		}
		r.Probes = fmt.Sprintf("%d %s", len(n.Tests), worst)
	}
	return r
}

// factorPopover is the arithmetic spelled out, so the score can be recomputed
// by hand from the row alone.
func factorPopover(n *shadow.Node) string {
	order := []string{"relation", "dependents", "state", "farfield", "faultline", "scar"}
	names := map[string]string{
		"relation": "relation", "dependents": "dependents", "state": "state",
		"farfield": "far field", "faultline": "fault line", "scar": "scar",
	}
	var parts []string
	for _, k := range order {
		if v, ok := n.Factors[k]; ok {
			parts = append(parts, fmt.Sprintf("%s %g", names[k], v))
		}
	}
	line := strings.Join(parts, " x ")
	if v, ok := n.Factors["test"]; ok && v > 0 {
		line += fmt.Sprintf(" + test %g", v)
	}
	if n.Pinned() {
		return line + " (pinned to the top by a beacon)"
	}
	return line
}

// FileCounts lists the files in the docket with how many rows each has, for
// the file filter.
func FileCounts(rows []DocketRow) []struct {
	File  string
	Count int
} {
	counts := map[string]int{}
	for _, r := range rows {
		if r.File != "" {
			counts[r.File]++
		}
	}
	out := make([]struct {
		File  string
		Count int
	}, 0, len(counts))
	for f, c := range counts {
		out = append(out, struct {
			File  string
			Count int
		}{f, c})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].File < out[j].File
	})
	return out
}
