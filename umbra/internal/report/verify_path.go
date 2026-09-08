package report

import (
	"fmt"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// VerifyPath is the exact way to check one claim the graph could not confirm.
//
// It exists because "this may be wrong" is not useful on its own. A reader who
// is told a relation is heuristic and not told how to settle it will either
// ignore the warning or re-derive the check by hand, and both are worse than
// the report simply printing the command.
type VerifyPath struct {
	// Def is the graph lookup that shows the definition, disambiguated by
	// file. The file is not optional: `graph impact --symbol Dependents`
	// against this repository matches eight definitions and refuses to pick
	// one, and three of the eight are struct fields rather than the method.
	Def string `json:"def"`
	// Site is the file and line to open: the call site the graph pointed at
	// when it gave one, otherwise the definition.
	Site string `json:"site"`
	// Test is the test that reaches this node, and TestCommand is how to run
	// just that test with the runner this report used.
	Test        string `json:"test,omitempty"`
	TestCommand string `json:"test_command,omitempty"`
	// NoTest says why there is no test to name, when there is none. A missing
	// command with no explanation reads as an oversight.
	NoTest string `json:"no_test,omitempty"`
}

// Lines renders the path as the report prints it, one instruction per line.
func (v VerifyPath) Lines() []string {
	out := []string{"graph:  " + v.Def, "source: " + v.Site}
	switch {
	case v.TestCommand != "":
		out = append(out, "test:   "+v.TestCommand)
	case v.Test != "":
		out = append(out, "test:   "+v.Test)
	default:
		out = append(out, "test:   "+v.NoTest)
	}
	return out
}

// VerifyFor builds the verification path for one node.
//
// `--repo .` rather than the worktree path, deliberately: the worktree is a
// detached checkout under the plugin data directory, the path would be
// scrubbed out of the report anyway, and a reader runs this from their own
// checkout.
func VerifyFor(a *Analysis, n *shadow.Node) VerifyPath {
	v := VerifyPath{}
	if n == nil || n.Symbol == nil {
		return v
	}
	v.Def = fmt.Sprintf("entire graph def --repo . --symbol %s --file %s",
		n.Symbol.Name, n.Symbol.File)

	line := n.CallSite
	if line <= 0 {
		line = n.Symbol.Span[0]
	}
	v.Site = fmt.Sprintf("%s:%d", n.Symbol.File, line)

	switch {
	case len(n.Tests) > 0:
		v.Test = n.Tests[0].ID
	case n.IsTest():
		if id, ok := shadow.TestIDFor(n.Symbol, ""); ok {
			v.Test = id
		}
	}
	if v.Test != "" && a != nil && a.TestRunner != "" {
		v.TestCommand = a.TestRunner + " " + v.Test
	}
	if v.Test == "" {
		v.NoTest = "no test in this field reaches it; the full sweep is the only check that covers it"
	}
	return v
}

// EvidenceLegend is the three tiers with their plain meanings, printed
// wherever the coined words appear for the first time.
func EvidenceLegend() []string {
	var out []string
	for _, e := range []shadow.Evidence{shadow.Confirmed, shadow.Heuristic, shadow.NeedsVerification} {
		out = append(out, fmt.Sprintf("%s %-18s %s", e.Mark(), string(e), e.Meaning()))
	}
	return out
}

// GraphEvidenceLine is the one-line header account of how much of this field
// the graph could vouch for, and what it said about its own completeness.
//
// It is never omitted when the analysis is degraded: a field with no line is a
// claim that nothing went wrong.
func GraphEvidenceLine(a *Analysis) string {
	s := shadow.SummarizeEvidence(a.Nodes)
	line := fmt.Sprintf("%d confirmed  %d heuristic  %d needs verification",
		s.Confirmed, s.Heuristic, s.NeedsVerification)
	if a.Graph.Profile != "" {
		line += "   profile " + a.Graph.Profile
	}
	return line
}

// GraphCompletenessNote is the sentence naming why the analysis may be
// partial, in the provider's own codes. Empty when the provider reported a
// complete reading.
func GraphCompletenessNote(a *Analysis) string {
	g := a.Graph
	if !g.Degraded() {
		return ""
	}
	var parts []string
	if !g.SawSummary {
		parts = append(parts, "the snapshot ended without its summary record, so its own completeness was never reported")
	}
	if !graph.CompletenessIsHealthy(g.CompletenessLevel) {
		parts = append(parts, "the graph reports this snapshot as "+g.CompletenessLevel)
	}
	// How much of the repository was actually read, whatever the level. The
	// provider treats a deliberate skip as not dragging the level down, which
	// is a reasonable policy for the level and a poor reason to hide the count
	// from a reader.
	if g.Files > 0 && g.ParsedFiles > 0 && g.ParsedFiles < g.Files {
		parts = append(parts, fmt.Sprintf("%d of %d files parsed", g.ParsedFiles, g.Files))
	}
	if n := len(g.PartialFailures); n > 0 {
		parts = append(parts, fmt.Sprintf("%d file(s) the graph could not fully analyse: %s",
			n, strings.Join(diagnosticCodes(g.PartialFailures), ", ")))
	}
	if n := len(g.Warnings); n > 0 {
		parts = append(parts, fmt.Sprintf("%d warning(s): %s",
			n, strings.Join(diagnosticCodes(g.Warnings), ", ")))
	}
	if g.Malformed > 0 {
		parts = append(parts, fmt.Sprintf("%d snapshot record(s) could not be read at all", g.Malformed))
	}
	if g.Dropped > 0 {
		parts = append(parts, fmt.Sprintf("%d record(s) arrived without the ids they need and were not used", g.Dropped))
	}
	if len(g.ImpactDegraded) > 0 {
		parts = append(parts, "the graph flagged its own impact answer for "+strings.Join(g.ImpactDegraded, ", "))
	}
	if g.InScope {
		why := g.InScopeWhy
		if why == "" {
			why = "the graph scopes this to the languages in this field, so it can affect these dependents"
		}
		parts = append(parts, why)
	} else if len(g.OutOfScope) > 0 {
		parts = append(parts, "the graph scopes "+strings.Join(g.OutOfScope, ", ")+
			" to other languages, where it says they cannot affect this answer")
	}
	return strings.Join(parts, "; ")
}

func diagnosticCodes(list []GraphDiagnostic) []string {
	var out []string
	seen := map[string]bool{}
	for _, d := range list {
		if seen[d.Code] {
			continue
		}
		seen[d.Code] = true
		out = append(out, d.Code)
	}
	return out
}
