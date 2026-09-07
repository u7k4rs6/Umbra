package report

import (
	"encoding/json"
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
	if err := Table(&b, Seal(a, nil), o); err != nil {
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

// A test the sweep turned up is a leak, not a cracked probe. Counting it as
// one names a test that was never among the probes, which the four-hop leak
// scenario makes happen for real.
func TestCrackedCountsOnlySelectedProbes(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"tests/test_service.py::test_rounding"}
	a.Execution.NewFailures = []string{
		"tests/test_service.py::test_rounding",
		"tests/test_report.py::test_receipt_total",
	}
	a.Execution.Sweep = true
	a.Execution.Leaks = []Leak{{
		Test:   "tests/test_report.py::test_receipt_total",
		Reason: "path exists at depth 4 via render_lines",
	}}

	if got := a.Execution.CrackedProbes(); len(got) != 1 || got[0] != "tests/test_service.py::test_rounding" {
		t.Fatalf("cracked probes = %v, want only the selected one", got)
	}
	if got := a.Execution.SweepOnlyFailures(); len(got) != 1 || got[0] != "tests/test_report.py::test_receipt_total" {
		t.Fatalf("sweep-only failures = %v", got)
	}

	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "1 selected  1 cracked") {
		t.Fatalf("expected one probe and one crack:\n%s", out)
	}
	// The leak must not be listed on the probes line as though it were a probe.
	probes := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "probes") {
			probes = line
		}
	}
	if strings.Contains(probes, "test_receipt_total") {
		t.Fatalf("a leak was listed as a cracked probe: %q", probes)
	}
	// It still has to be visible, as a leak with its reason.
	if !strings.Contains(out, "path exists at depth 4 via render_lines") {
		t.Fatalf("the leak and its reason must still appear:\n%s", out)
	}
	if !strings.Contains(out, "no probe covered") {
		t.Fatalf("the reader should be told a failure fell outside the probes:\n%s", out)
	}
}

// The kind was printed with " changed" appended, which is right for a body or
// a signature and wrong for the rest: a symbol that was added read as "added
// changed". One label, used by the table, the packet and the map.
func TestChangeKindReadsAsASentence(t *testing.T) {
	cases := []struct {
		change string
		want   string
	}{
		{"added", "added"},
		{"removed", "removed"},
		{"renamed", "renamed"},
		{"signature", "signature changed"},
		{"body", "body changed"},
	}
	for _, c := range cases {
		a := mkAnalysis()
		a.Sources = []graph.Source{{
			Name: "compute_total", File: "app/service.py", Span: [2]int{20, 30}, Change: c.change,
		}}
		out := render(t, a, TableOptions{UTF8: true})
		want := "compute_total  " + c.want + "  app/service.py:20"
		if !strings.Contains(out, want) {
			t.Errorf("change %q: table is missing %q\n%s", c.change, want, firstLines(out, 6))
		}
		if strings.Contains(out, c.change+" changed changed") {
			t.Errorf("change %q: doubled wording", c.change)
		}
	}
	// The one that used to be wrong, checked as a whole line.
	a := mkAnalysis()
	a.Sources = []graph.Source{{Name: "jsNode", File: "x.go", Span: [2]int{1, 2}, Change: "added"}}
	if strings.Contains(render(t, a, TableOptions{UTF8: true}), "added changed") {
		t.Error("an added symbol still reads as \"added changed\"")
	}
}

func firstLines(s string, n int) string {
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[:n]
	}
	return strings.Join(lines, "\n")
}

// The sweep on pallets/click reported "sweep full suite 0 leaks" on a change
// that broke 55 tests, because verify had counted 51 and named none of them in
// a line the parser read. A count it could not turn into names must never
// render as a clean audit, on any surface.
func TestSweepNeverReportsACleanAuditItCouldNotRead(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"tests/test_service.py::test_rounding"}
	a.Execution.Sweep = true
	a.Execution.UnnamedFailures = 31

	out := render(t, a, TableOptions{UTF8: true})
	if strings.Contains(out, "sweep   full suite  0 leaks\n") {
		t.Fatalf("a sweep that could not name 31 failures reported a clean audit:\n%s", out)
	}
	if !strings.Contains(out, "incomplete") {
		t.Fatalf("the sweep line must say the list is incomplete:\n%s", out)
	}
	if !strings.Contains(out, "31 further failing test(s)") {
		t.Fatalf("the count verify reported must be printed:\n%s", out)
	}
}

// The 15-failure case is the negative half: it still names every leak with its
// forensics and says nothing about being incomplete.
func TestSweepStillNamesEveryLeakWhenNothingWasTruncated(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"a"}
	a.Execution.Leaks = []Leak{
		{Test: "tests/test_formatting.py::test_basic_functionality", Reason: "test file not in the co-change set"},
		{Test: "tests/test_custom_classes.py::test_context_formatter_class", Reason: "reaches only through structure"},
	}
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "sweep   full suite  2 leaks\n") {
		t.Fatalf("an untruncated sweep prints a plain count:\n%s", out)
	}
	if strings.Contains(out, "incomplete") {
		t.Fatalf("nothing was truncated, so nothing should say incomplete:\n%s", out)
	}
	if !strings.Contains(out, "test file not in the co-change set") {
		t.Fatalf("forensics must still be printed:\n%s", out)
	}
}

// The same rule on the packet and on the map header, so the three surfaces
// cannot drift apart on it.
func TestPacketAndMapHeaderAlsoRefuseACleanAudit(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"a"}
	a.Execution.Sweep = true
	a.Execution.UnnamedFailures = 31

	var b strings.Builder
	if err := Packet(&b, Seal(a, nil)); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	if !strings.Contains(b.String(), "31 further failing test(s)") {
		t.Fatalf("the packet must carry the count:\n%s", b.String())
	}
	if !strings.Contains(b.String(), "incomplete") {
		t.Fatalf("the packet must say the list is incomplete:\n%s", b.String())
	}
	if got := sweepText(a); !strings.Contains(got, "incomplete") {
		t.Fatalf("map header = %q, want it to say the list is incomplete", got)
	}
}

// Defect 3. "No dependents were found for the changed symbols" is a finding
// about the code. A commit whose symbols could not be resolved has no such
// finding, and printing that sentence for it tells the reader the graph was
// asked and answered when it was never asked at all.
func TestEmptyDocketDistinguishesUnresolvedFromNoDependents(t *testing.T) {
	base := func() *Analysis {
		a := mkAnalysis()
		a.Nodes = nil
		a.Summary = shadow.Summarize(nil)
		a.Sources = []graph.Source{{Name: "HelpFormatter.write_usage", File: "src/click/formatting.py"}}
		return a
	}

	asked := base()
	if got := asked.EmptyDocketReason(); !strings.Contains(got, "found no dependents") {
		t.Fatalf("a resolved commit with no dependents says %q", got)
	}

	never := base()
	never.Unresolved = []UnresolvedSource{{
		Name: "HelpFormatter.write_usage", File: "src/click/formatting.py", Line: 167,
		Reason: graph.UnresolvedNoMatch,
	}}
	got := never.EmptyDocketReason()
	if strings.Contains(got, "found no dependents") {
		t.Fatalf("an unresolved commit must not claim the graph found nothing: %q", got)
	}
	if !strings.Contains(got, "unresolved") {
		t.Fatalf("the sentence must name the gap: %q", got)
	}

	out := render(t, never, TableOptions{UTF8: true})
	if strings.Contains(out, "no dependents were found for the changed symbols") {
		t.Fatalf("the old sentence is still being printed:\n%s", out)
	}
	if !strings.Contains(out, "never looked up in the graph") {
		t.Fatalf("the table must say what was not looked up:\n%s", out)
	}
	if !strings.Contains(out, "HelpFormatter.write_usage") {
		t.Fatalf("an unresolved symbol must be named:\n%s", out)
	}
	if !strings.Contains(out, graph.UnresolvedNoMatch) {
		t.Fatalf("the reason must be printed:\n%s", out)
	}
}

// The packet and the map header carry the same distinction, so a reviewer
// reading either one is told the same thing.
func TestPacketNamesTheSymbolsTheGraphWasNeverAskedAbout(t *testing.T) {
	a := mkAnalysis()
	a.Unresolved = []UnresolvedSource{{
		Name: "Inner.inner_method", File: "app/models.py", Line: 40,
		Reason: graph.UnresolvedAmbiguous,
	}}
	var b strings.Builder
	if err := Packet(&b, Seal(a, nil)); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	for _, want := range []string{
		"never asked about",
		"Inner.inner_method",
		graph.UnresolvedAmbiguous,
		"not a finding of no dependents",
	} {
		if !strings.Contains(b.String(), want) {
			t.Fatalf("the packet is missing %q:\n%s", want, b.String())
		}
	}
}

// Nothing changes for a run where every symbol resolved.
func TestAResolvedRunSaysNothingAboutUnresolvedSymbols(t *testing.T) {
	a := mkAnalysis()
	out := render(t, a, TableOptions{UTF8: true})
	if strings.Contains(out, "never looked up") {
		t.Fatalf("a clean run must not mention unresolved symbols:\n%s", out)
	}
}

// The reach line. An edge leaving the repository and an absent dependent look
// identical in a docket, and on real Python the first is the common case, so
// every surface says how much of the local call graph could be followed.
func TestReachLineAppearsOnEverySurface(t *testing.T) {
	a := mkAnalysis()
	a.Reach = ReachSummary{Inside: 41, Leaving: 55}

	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "reach") {
		t.Fatalf("the terminal table has no reach line:\n%s", out)
	}
	if !strings.Contains(out, "41 of 96") || !strings.Contains(out, "55 leave it") {
		t.Fatalf("the reach line does not carry the counts:\n%s", out)
	}

	var b strings.Builder
	if err := Packet(&b, Seal(a, nil)); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	if !strings.Contains(b.String(), "41 of 96") {
		t.Fatalf("the packet has no reach line:\n%s", b.String())
	}

	var h strings.Builder
	if err := HTML(&h, Seal(a, nil)); err != nil {
		t.Fatalf("HTML: %v", err)
	}
	if !strings.Contains(h.String(), "41 of 96") {
		t.Fatal("the map header has no reach line")
	}

	blob, err := MarshalJSON(Seal(a, nil))
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	if !strings.Contains(string(blob), `"reach"`) {
		t.Fatal("the JSON record has no reach block")
	}
	var record struct {
		Reach ReachSummary `json:"reach"`
	}
	if err := json.Unmarshal(blob, &record); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if record.Reach.Inside != 41 || record.Reach.Leaving != 55 {
		t.Fatalf("the JSON reach block = %+v, want 41 inside and 55 leaving", record.Reach)
	}
}

// A run where every call resolved says so, and a run with no call edges at all
// says nothing rather than printing a zero.
func TestReachLineIsSilentWhenThereAreNoCallEdges(t *testing.T) {
	a := mkAnalysis()
	a.Reach = ReachSummary{}
	out := render(t, a, TableOptions{UTF8: true})
	if strings.Contains(out, "reach") {
		t.Fatalf("a run with no call edges must not print a reach line:\n%s", out)
	}

	a.Reach = ReachSummary{Inside: 7}
	out = render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "7 of 7") || !strings.Contains(out, "0 leave it") {
		t.Fatalf("a fully resolved run must still say so:\n%s", out)
	}
}
