package report

import (
	"fmt"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// mkEvidenceAnalysis is one of each tier, so a renderer that drops a tier
// fails rather than passing on the two it kept.
func mkEvidenceAnalysis() *Analysis {
	a := mkAnalysis()
	a.TestRunner = "pytest -v"

	a.Nodes[0].Evidence, a.Nodes[0].EvidenceWhy = shadow.Confirmed, shadow.Confirmed.Meaning()
	a.Nodes[0].Resolution = graph.ResolutionExact
	a.Nodes[0].Confidence = 1

	a.Nodes[1].Evidence = shadow.NeedsVerification
	a.Nodes[1].EvidenceWhy = "the path to this node passes through render_lines, which the graph resolved as name_only rather than to a definition"
	a.Nodes[1].Resolution = graph.ResolutionNameOnly
	a.Nodes[1].HeuristicEdge, a.Nodes[1].HeuristicVia = true, "render_lines"
	a.Nodes[1].CallSite = 19

	a.Nodes[2].Evidence = shadow.Heuristic
	a.Nodes[2].EvidenceWhy = "the graph resolved this relation as type_inferred"
	a.Nodes[2].Resolution = graph.ResolutionTypeInferred
	a.Nodes[2].HeuristicEdge, a.Nodes[2].LastHopHeuristic = true, true

	a.Graph = GraphEvidence{
		Profile: "full", CompletenessLevel: "degraded", SawSummary: true,
		Files: 765, ParsedFiles: 762,
		PartialFailures: []GraphDiagnostic{{Code: "E_MINIFIED", File: "x.json"}},
		Warnings:        []GraphDiagnostic{{Code: "W_DATA_FLOW_EVIDENCE_UNMERGED"}},
	}
	return a
}

func TestTerminalPrintsAllThreeTiersAndTheirMeanings(t *testing.T) {
	out := render(t, mkEvidenceAnalysis(), TableOptions{UTF8: true, Width: 100, All: true})

	for _, want := range []string{
		"1 confirmed", "1 heuristic", "1 needs verification",
		shadow.Confirmed.Meaning(), shadow.Heuristic.Meaning(), shadow.NeedsVerification.Meaning(),
	} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal is missing %q", want)
		}
	}
	// The header must name the reason, not merely that there is one.
	for _, want := range []string{"may be incomplete", "degraded", "762 of 765", "E_MINIFIED", "W_DATA_FLOW_EVIDENCE_UNMERGED"} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal does not name the partial analysis: missing %q", want)
		}
	}
}

func TestTerminalPrintsTheExactVerificationCommand(t *testing.T) {
	out := render(t, mkEvidenceAnalysis(), TableOptions{UTF8: true, Width: 100, All: true})
	for _, want := range []string{
		"entire graph def --repo . --symbol apply_refund --file app/refunds.py",
		"app/refunds.py:19",
		"render_lines",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("terminal is missing the verification path element %q", want)
		}
	}
}

// The verification path names a test when a test reaches the node, with the
// runner this report actually used.
func TestVerifyPathNamesTheTestThatReachesIt(t *testing.T) {
	a := mkEvidenceAnalysis()
	a.Nodes[1].Tests = []shadow.TestRef{{ID: "tests/test_refunds.py::test_apply_refund_normal"}}
	v := VerifyFor(a, a.Nodes[1])
	if v.TestCommand != "pytest -v tests/test_refunds.py::test_apply_refund_normal" {
		t.Errorf("test command = %q", v.TestCommand)
	}
	if v.NoTest != "" {
		t.Error("a node with a test should not also carry the no-test explanation")
	}

	a.Nodes[1].Tests = nil
	v = VerifyFor(a, a.Nodes[1])
	if v.NoTest == "" {
		t.Error("a node with no test must say so rather than printing nothing")
	}
}

func TestJSONCarriesTheTiersInPlainWords(t *testing.T) {
	a := mkEvidenceAnalysis()
	var b strings.Builder
	if err := WriteJSON(&b, Seal(a, nil)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		`"evidence": "confirmed"`,
		`"evidence": "heuristic"`,
		`"evidence": "needs verification"`,
		`"evidence_meaning"`,
		`"evidence_why"`,
		`"may_be_incomplete": true`,
		`"needs_verification": 1`,
		`"dependents_are"`,
		`"resolution": "name_only"`,
		`"resolution_means"`,
		`"verify"`,
		`"evidence_legend"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("JSON is missing %q", want)
		}
	}
	// No coined abbreviation reaches the JSON.
	for _, bad := range []string{`"evidence": "="`, `"evidence": "~"`, `"evidence": "?"`} {
		if strings.Contains(out, bad) {
			t.Errorf("JSON carries a mark instead of the plain word: %q", bad)
		}
	}
}

func TestPacketCarriesTheTiersAndTheCommands(t *testing.T) {
	a := mkEvidenceAnalysis()
	var b strings.Builder
	if err := Packet(&b, Seal(a, nil)); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		"1 confirmed, 1 heuristic, 1 needs verification",
		"This analysis may be incomplete",
		"## 6. What to verify",
		"entire graph def --repo . --symbol apply_refund --file app/refunds.py",
		shadow.NeedsVerification.Meaning(),
		shadow.DependentCountIsHeuristic,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("packet is missing %q", want)
		}
	}

	// The per-node line, not only the summary. A packet that counts three
	// tiers in its header and then says nothing per node has given the reader
	// a number and withheld which row it belongs to.
	//
	// The header counts every dependent and the docket lists only the shadowed
	// ones, so a lit node's tier is counted above and has no row below. That is
	// the same population split the light line already uses, so the assertion
	// follows what the packet lists rather than what it counts.
	for _, n := range a.Nodes {
		if n.State == shadow.Lit {
			continue
		}
		want := fmt.Sprintf("- graph evidence: **%s**, meaning %s.", n.Evidence, n.Evidence.Meaning())
		if !strings.Contains(out, want) {
			t.Errorf("packet does not carry %s's own evidence line: missing %q", n.Symbol.Name, want)
		}
	}
}

func TestHTMLCarriesTheTiersAndTheirMeanings(t *testing.T) {
	a := mkEvidenceAnalysis()
	a.Layout = BuildLayout(a)
	var b strings.Builder
	if err := HTML(&b, Seal(a, nil)); err != nil {
		t.Fatalf("HTML: %v", err)
	}
	out := b.String()

	for _, want := range []string{
		"1 confirmed  1 heuristic  1 needs verification",
		"This analysis may be incomplete",
		`class="ev ev-needs-verification"`,
		`class="ev ev-confirmed"`,
		`class="ev ev-heuristic"`,
		"entire graph def --repo . --symbol apply_refund --file app/refunds.py",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("HTML is missing %q", want)
		}
	}
	// Every coined word beside its plain meaning, which is the page's rule.
	for _, e := range []shadow.Evidence{shadow.Confirmed, shadow.Heuristic, shadow.NeedsVerification} {
		if !strings.Contains(out, e.Meaning()) {
			t.Errorf("HTML shows %q without its plain meaning", string(e))
		}
	}
	// A class attribute must never be split into two by a space.
	if strings.Contains(out, `ev-needs verification`) {
		t.Error(`"needs verification" reached a class attribute unslugged`)
	}
}

// A fully resolved field must read exactly as it did before this axis existed.
func TestAFullyResolvedFieldSaysNothingAboutVerification(t *testing.T) {
	a := mkAnalysis()
	for _, n := range a.Nodes {
		n.Evidence, n.EvidenceWhy = shadow.Confirmed, shadow.Confirmed.Meaning()
		n.Resolution = graph.ResolutionExact
	}
	a.Graph = GraphEvidence{Profile: "full", CompletenessLevel: "complete", SawSummary: true}

	out := render(t, a, TableOptions{UTF8: true, Width: 100, All: true})
	if strings.Contains(out, "may be incomplete") {
		t.Error("a complete snapshot is being reported as incomplete")
	}
	if strings.Contains(out, "need verification.") {
		t.Error("a fully confirmed field printed a verification block")
	}
	if strings.Contains(out, shadow.NeedsVerification.Mark()+" apply_refund") {
		t.Error("a confirmed node carries a verification mark")
	}
	if !strings.Contains(out, "3 confirmed  0 heuristic  0 needs verification") {
		t.Error("the header does not report a fully confirmed field as fully confirmed")
	}
}
