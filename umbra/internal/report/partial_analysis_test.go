package report

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// A silently empty field currently looks like a clean one. Every one of these
// cases must produce a header that says the field may be incomplete and names
// the reason, in the graph's own codes.
func TestTheHeaderNamesEveryKindOfPartialAnalysis(t *testing.T) {
	cases := []struct {
		name string
		g    GraphEvidence
		want []string
	}{
		{
			name: "a degraded snapshot",
			g:    GraphEvidence{SawSummary: true, CompletenessLevel: "degraded", Files: 765, ParsedFiles: 762},
			want: []string{"degraded", "762 of 765 files parsed"},
		},
		{
			name: "partial failures",
			g: GraphEvidence{SawSummary: true, CompletenessLevel: "complete",
				PartialFailures: []GraphDiagnostic{{Code: "E_PARSE_ERROR", File: "a.h"}, {Code: "E_MINIFIED", File: "b.json"}}},
			want: []string{"2 file(s) the graph could not fully analyse", "E_PARSE_ERROR", "E_MINIFIED"},
		},
		{
			name: "the Python completeness warning captured this morning",
			g: GraphEvidence{SawSummary: true, CompletenessLevel: "degraded",
				Warnings: []GraphDiagnostic{{Code: "W_DATA_FLOW_EVIDENCE_UNMERGED", Level: "info"}}},
			want: []string{"W_DATA_FLOW_EVIDENCE_UNMERGED", "1 warning(s)"},
		},
		{
			name: "malformed snapshot records",
			g:    GraphEvidence{SawSummary: true, CompletenessLevel: "complete", Malformed: 3},
			want: []string{"3 snapshot record(s) could not be read at all"},
		},
		{
			name: "records that arrived without their ids",
			g:    GraphEvidence{SawSummary: true, CompletenessLevel: "complete", Dropped: 7},
			want: []string{"7 record(s) arrived without the ids they need"},
		},
		{
			name: "a snapshot that never reported on itself",
			g:    GraphEvidence{SawSummary: false},
			want: []string{"ended without its summary record"},
		},
		{
			name: "an impact answer the graph flagged",
			g: GraphEvidence{SawSummary: true, CompletenessLevel: "complete",
				ImpactDegraded: []string{"compute_total"}},
			want: []string{"flagged its own impact answer for compute_total"},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			a := mkAnalysis()
			a.Graph = c.g
			if !a.Graph.Degraded() {
				t.Fatal("this case is a partial analysis and Degraded reports otherwise")
			}
			note := GraphCompletenessNote(a)
			if note == "" {
				t.Fatal("the header says nothing about a partial analysis")
			}
			for _, want := range c.want {
				if !strings.Contains(note, want) {
					t.Errorf("the header does not name the reason: missing %q in %q", want, note)
				}
			}
			out := render(t, a, TableOptions{UTF8: true, Width: 100, All: true})
			if !strings.Contains(out, "this analysis may be incomplete") {
				t.Error("the terminal header does not say the field may be incomplete")
			}
		})
	}
}

// A complete snapshot must say nothing, or the warning means nothing.
func TestACompleteSnapshotSaysNothing(t *testing.T) {
	a := mkAnalysis()
	a.Graph = GraphEvidence{SawSummary: true, CompletenessLevel: "complete", Profile: "full",
		Files: 20, ParsedFiles: 20}
	if a.Graph.Degraded() {
		t.Fatal("a complete snapshot is being reported as degraded")
	}
	if note := GraphCompletenessNote(a); note != "" {
		t.Errorf("a complete snapshot produced a warning: %q", note)
	}
}

// The graph scopes its own diagnostics per query. A parse error in a vendored
// C header cannot affect an answer about a Python symbol, and reporting one as
// if it could would make the tier mean nothing.
func TestScopeIsReportedInBothDirections(t *testing.T) {
	a := mkAnalysis()
	a.Graph = GraphEvidence{SawSummary: true, CompletenessLevel: "degraded",
		PartialFailures: []GraphDiagnostic{{Code: "E_PARSE_ERROR", File: "vendor/array.h"}},
		OutOfScope:      []string{"E_PARSE_ERROR"}}
	note := GraphCompletenessNote(a)
	if !strings.Contains(note, "cannot affect this answer") {
		t.Errorf("an out-of-scope diagnostic is not reported as out of scope: %q", note)
	}

	a.Graph.InScope = true
	a.Graph.InScopeWhy = "the graph scopes W_DATA_FLOW_EVIDENCE_UNMERGED to Python, the language this field is in"
	note = GraphCompletenessNote(a)
	if !strings.Contains(note, "Python") {
		t.Errorf("an in-scope diagnostic is not reported as in scope: %q", note)
	}
}

// When the degradation is in scope, every node inherits the doubt, because the
// provider says it cannot identify which edges are affected.
func TestAnInScopeDegradationReachesEveryNode(t *testing.T) {
	n := &shadow.Node{
		Symbol:     mkNode("f", "a.py", 1, shadow.Umbra, shadow.TierNone, 1).Symbol,
		Resolution: "exact",
	}
	got, why := shadow.ClassifyEvidence(n, shadow.EvidenceInput{})
	if got != shadow.Confirmed {
		t.Fatalf("a fully resolved node in a clean run = %q, want confirmed", got)
	}

	got, why = shadow.ClassifyEvidence(n, shadow.EvidenceInput{
		RunDegraded: true,
		DegradedWhy: "the graph reports this snapshot as degraded",
	})
	if got != shadow.NeedsVerification {
		t.Fatalf("a fully resolved node under a degraded run = %q, want needs verification", got)
	}
	if !strings.Contains(why, "degraded") {
		t.Errorf("the reason does not quote the graph: %q", why)
	}
}
