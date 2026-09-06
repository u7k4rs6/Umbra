package report

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

func mkNode(name, file string, line int, state shadow.State, tier shadow.Tier, score float64, mods ...string) *shadow.Node {
	return &shadow.Node{
		Symbol:    &graph.Symbol{ID: file + ":" + name, Name: name, File: file, Span: [2]int{line, line + 3}},
		State:     state,
		Tier:      tier,
		Relation:  "direct caller",
		Depth:     1,
		Score:     score,
		Modifiers: mods,
	}
}

func mkAnalysis() *Analysis {
	nodes := []*shadow.Node{
		mkNode("test_rounding", "tests/test_service.py", 7, shadow.Umbra, shadow.TierNone, 9.0, "test"),
		mkNode("apply_refund", "app/refunds.py", 17, shadow.Penumbra, shadow.TierEcho, 4.2, "echo"),
		mkNode("handle_order", "app/api.py", 14, shadow.Lit, shadow.TierNone, 0),
	}
	a := &Analysis{
		Version: "0.1.0", CheckpointID: "b20f84567474", Commit: "0063443c2bc2ac6",
		Adapter: "claude-code", Depth: 2, Run: "none", Audit: false,
		SessionSaid:     "Checked the callers and updated them. All tests pass.",
		SessionSaidFrom: "the checkpoint summary Entire stored",
		Sources: []graph.Source{{
			Name: "compute_total", File: "app/service.py", Span: [2]int{20, 30}, Change: "signature",
		}},
		Nodes: nodes,
	}
	a.Summary = shadow.Summarize(nodes)
	return a
}

func render(t *testing.T, a *Analysis, o TableOptions) string {
	t.Helper()
	var b strings.Builder
	if err := Table(&b, a, o); err != nil {
		t.Fatalf("Table: %v", err)
	}
	return b.String()
}

func TestTableHeader(t *testing.T) {
	out := render(t, mkAnalysis(), TableOptions{UTF8: true, Width: 100})
	for _, want := range []string{
		"Umbra  b20f84567474  0063443  claude-code  depth 2",
		`session said  "Checked the callers and updated them. All tests pass."`,
		"compute_total  signature changed  app/service.py:20",
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("table is missing %q\n%s", want, out)
		}
	}
}

func TestTableLightLineCountsAndFraction(t *testing.T) {
	out := render(t, mkAnalysis(), TableOptions{UTF8: true})
	if !strings.Contains(out, "1 lit  1 penumbra  1 umbra  0 unknown") {
		t.Fatalf("light line wrong:\n%s", out)
	}
	if !strings.Contains(out, "33% examined") {
		t.Fatalf("expected the illumination fraction:\n%s", out)
	}
	// The bar is one glyph per node, brightest first.
	if !strings.Contains(out, "●◐○") {
		t.Fatalf("expected the glyph bar:\n%s", out)
	}
}

// Everything unknown means the fraction is omitted rather than shown as zero.
func TestTableSaysWhenTheExaminedSetIsUnavailable(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("x", "app/a.py", 1, shadow.Unknown, shadow.TierNone, 3)}
	a.Summary = shadow.Summarize(a.Nodes)
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "examined set unavailable") {
		t.Fatalf("expected the unavailable notice:\n%s", out)
	}
	if strings.Contains(out, "% examined") {
		t.Fatalf("must not print a fraction when everything is unknown:\n%s", out)
	}
}

// Lit nodes are summarised as a count unless --all.
func TestTableHidesLitNodesByDefault(t *testing.T) {
	out := render(t, mkAnalysis(), TableOptions{UTF8: true})
	if strings.Contains(out, "handle_order") {
		t.Fatalf("lit nodes should be counted, not listed:\n%s", out)
	}
	if !strings.Contains(out, "1 lit node(s) not shown") {
		t.Fatalf("expected the lit count line:\n%s", out)
	}

	all := render(t, mkAnalysis(), TableOptions{UTF8: true, All: true})
	if !strings.Contains(all, "handle_order") {
		t.Fatalf("--all should list lit nodes:\n%s", all)
	}
}

func TestTableGlyphFallbackForNonUTF8(t *testing.T) {
	out := render(t, mkAnalysis(), TableOptions{UTF8: false})
	if strings.Contains(out, "○") || strings.Contains(out, "◐") {
		t.Fatalf("a non-UTF-8 terminal must not receive round glyphs:\n%s", out)
	}
	if !strings.Contains(out, "U") || !strings.Contains(out, "P") {
		t.Fatalf("expected the L, P, U fallback:\n%s", out)
	}
}

func TestTableNoColourWhenNotATerminal(t *testing.T) {
	out := render(t, mkAnalysis(), TableOptions{UTF8: true, Colour: false})
	if strings.Contains(out, "\x1b[") {
		t.Fatalf("no escape codes when stdout is not a terminal:\n%s", out)
	}
}

func TestTableColoursOnlyGlyphAndOutcome(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Result = shadow.OutcomeFail
	out := render(t, a, TableOptions{UTF8: true, Colour: true})
	if !strings.Contains(out, ansiRed+"○"+ansiReset) {
		t.Fatalf("expected a coloured glyph:\n%s", out)
	}
	if !strings.Contains(out, ansiRed+"fail"+ansiReset) {
		t.Fatalf("expected a coloured outcome word:\n%s", out)
	}
	// The symbol name itself is never painted.
	if strings.Contains(out, ansiRed+"test_rounding") {
		t.Fatalf("only the glyph and the outcome carry colour:\n%s", out)
	}
}

func TestTableShowsBeaconLineWhenPinned(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[1].Beacon = "# SAFETY: must not refund the full amount"
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "# SAFETY: must not refund the full amount") {
		t.Fatalf("expected the beacon comment line:\n%s", out)
	}
	if !strings.Contains(out, "!") {
		t.Fatalf("expected the pin marker:\n%s", out)
	}
}

func TestTableSaysWhenTheSweepWasSkipped(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = false
	a.Execution.Selected = []string{"tests/test_service.py::test_rounding"}
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "skipped (--no-audit)") {
		t.Fatalf("expected the skipped-sweep notice:\n%s", out)
	}
	if !strings.Contains(out, "unaudited") {
		t.Fatalf("the header must say the selection is unaudited:\n%s", out)
	}
}

// The leak count is printed even when it is zero.
func TestTablePrintsZeroLeaks(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"tests/test_service.py::test_rounding"}
	a.Execution.Sweep = true
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "sweep   full suite  0 leaks") {
		t.Fatalf("expected a zero leak count:\n%s", out)
	}
}

func TestTablePrintsLeaksWithReasons(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"a"}
	a.Execution.Leaks = []Leak{{Test: "tests/test_report.py::test_receipt_total", Reason: "path exists at depth 4 via render_lines"}}
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "1 leak\n") {
		t.Fatalf("expected a singular leak count:\n%s", out)
	}
	if !strings.Contains(out, "path exists at depth 4 via render_lines") {
		t.Fatalf("every leak must print its reason:\n%s", out)
	}
}

func TestTableSaysWhenNothingIsInShadow(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("handle_order", "app/api.py", 14, shadow.Lit, shadow.TierNone, 0)}
	a.Summary = shadow.Summarize(a.Nodes)
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "nothing is in shadow") {
		t.Fatalf("expected the all-clear line:\n%s", out)
	}
}

func TestTableWarnsWhenTheRunnerWasNotVerbose(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Execution.Selected = []string{"a"}
	a.Execution.Degraded = true
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "add -v") {
		t.Fatalf("expected the guidance to add -v:\n%s", out)
	}
}
