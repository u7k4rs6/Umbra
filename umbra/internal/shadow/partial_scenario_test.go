package shadow

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// The ninth scenario. Its graph side is a separate captured snapshot, of a
// fixture built to contain analysis the provider cannot complete: Python
// dynamic dispatch through a table, through getattr and through importlib, a
// method call resolved by inferring a constructor's type, a file in a language
// the provider indexes inventory-only, and a minified file it declines to
// parse at all.
//
// Everything asserted below was measured from real provider output before it
// was written down. Nothing in the snapshot was edited except the repository
// key, which carried an absolute path.

const partialPrefix = "umbra/fixtures/partial/"

func partialField(t *testing.T) (*graph.Field, *graph.RelationMap) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "partial-snapshot.ndjson"))
	if err != nil {
		t.Fatalf("read partial snapshot: %v", err)
	}
	f, err := graph.LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	caps, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "capabilities.json"))
	if err != nil {
		t.Fatalf("read capabilities: %v", err)
	}
	c, err := graph.ParseCapabilities(caps)
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	return f, graph.NewRelationMap(c)
}

// runPartialScenario is the same classify and rank half the other eight
// scenarios drive, against the partial fixture and at depth 3, which is the
// depth at which a node hangs off a hop it did not make itself.
func runPartialScenario(t *testing.T, in EvidenceInput) scenarioResult {
	t.Helper()
	field, relMap := partialField(t)

	var src *graph.Symbol
	for _, id := range field.ByFile["pricing.py"] {
		if field.Symbols[id].Name == "compute_total" {
			src = field.Symbols[id]
		}
	}
	if src == nil {
		t.Fatal("compute_total missing from the partial snapshot")
	}
	sources := []graph.Source{{
		Symbol: src.ID, Name: src.Name, File: src.File, Span: src.Span,
		Change: "signature", Weight: graph.Weight("signature"), Dependents: 3,
	}}

	blob, err := os.ReadFile(filepath.Join(fixturesDir, "dynamic-dispatch", "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read scenario: %v", err)
	}
	session, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("parse scenario: %v", err)
	}
	for i := range session.Events {
		session.Events[i].Path = strings.TrimPrefix(session.Events[i].Path, partialPrefix)
		for j := range session.Events[i].Paths {
			session.Events[i].Paths[j] = strings.TrimPrefix(session.Events[i].Paths[j], partialPrefix)
		}
	}
	session.ResolveMentions(Vocabulary(field))
	examined := BuildExamined(session, map[string]bool{"pricing.py": true})

	// Depth 4, which is where the chain reaches its test. The other eight
	// scenarios run at 2; this one has a longer chain on purpose.
	reach := field.Dependents([]string{src.ID}, 4, relMap)
	bi := BuildInput{
		Field: field, RelMap: relMap, Sources: sources, Reach: reach,
		Examined: examined, Impacts: map[string]*graph.Impact{},
		Meta: field.Meta, Evidence: in,
	}
	nodes := Build(bi)
	SortDocket(nodes)

	byName := map[string]*Node{}
	for _, n := range nodes {
		byName[n.Symbol.Name] = n
	}
	return scenarioResult{
		nodes: nodes, byName: byName, summary: Summarize(nodes),
		field: field, relMap: relMap, sources: sources, srcIDs: []string{src.ID},
	}
}

//  1. The unresolved calls are not presented as confirmed. The provider draws
//     no edge for a dispatch table, for getattr or for importlib, so the four
//     tests that exercise those paths are not in the field at all, and nothing
//     in the report may claim they are covered.
func TestPartialScenarioDoesNotPresentAnUnresolvedCallAsConfirmed(t *testing.T) {
	r := runPartialScenario(t, EvidenceInput{})

	for _, name := range []string{
		"test_invoice_total",
		"test_receipt_total",
		"test_dispatch_table_reaches_compute_total",
		"test_dispatch_by_name_reaches_compute_total",
		"invoice_total",
		"receipt_total",
		"dispatch",
		"dispatch_by_name",
		"dispatch_from_module",
	} {
		if n, ok := r.byName[name]; ok {
			if n.Evidence == Confirmed {
				t.Errorf("%s reaches compute_total only through dynamic dispatch and is reported as confirmed", name)
			}
		}
	}
	// The graph draws no edge for any of them, so the correct report is that
	// they are absent, and the field is a lower bound.
	if _, ok := r.byName["invoice_total"]; ok {
		t.Error("the graph resolved a dispatch-table call; this fixture no longer demonstrates what it claims")
	}
}

// 2. The resolved parts of the same run classify exactly as before.
func TestPartialScenarioLeavesResolvedRelationsConfirmed(t *testing.T) {
	r := runPartialScenario(t, EvidenceInput{})

	for _, name := range []string{"apply", "line_total"} {
		n, ok := r.byName[name]
		if !ok {
			t.Fatalf("%s is not in the field", name)
		}
		if n.Evidence != Confirmed {
			t.Errorf("%s = %q, want confirmed: the provider resolved it exactly", name, n.Evidence)
		}
		if !graph.ResolutionIsStructural(n.Resolution) {
			t.Errorf("%s resolution = %q, want one the provider resolved to a definition",
				name, n.Resolution)
		}
	}

	// The states, the tiers and the scores are the session's answer about what
	// it looked at, and this change must not have touched them. line_total is
	// in the edited file, so it is lit; RateCard.apply is in billing.py, which
	// the session never opened, so it is full umbra.
	wantState(t, r, "line_total", Lit, TierNone)
	wantState(t, r, "apply", Umbra, TierNone)
	if r.byName["apply"].Score <= 0 {
		t.Error("the ranker stopped scoring")
	}
	// A confirmed node in shadow is the ordinary case and must read as one:
	// nothing to verify, and no reason attached beyond the plain sentence.
	if r.byName["apply"].EvidenceWhy != "the graph resolved every hop on this path to a definition" {
		t.Errorf("a confirmed node carries an unexpected reason: %q", r.byName["apply"].EvidenceWhy)
	}
}

//  3. A relation the provider derived rather than parsed is heuristic, and a
//     node reached only THROUGH one needs verification. Both are real edges in
//     the captured snapshot, not constructed in the test.
func TestPartialScenarioSeparatesHeuristicFromNeedsVerification(t *testing.T) {
	r := runPartialScenario(t, EvidenceInput{})

	quote, ok := r.byName["quote"]
	if !ok {
		t.Fatal("quote is not in the field")
	}
	if quote.Evidence != Heuristic {
		t.Errorf("quote = %q, want heuristic: its own hop is the inferred one", quote.Evidence)
	}
	if quote.Resolution != graph.ResolutionTypeInferred {
		t.Errorf("quote resolution = %q, want type_inferred", quote.Resolution)
	}

	invoice, ok := r.byName["invoice"]
	if !ok {
		t.Fatal("invoice is not in the field")
	}
	if invoice.Evidence != NeedsVerification {
		t.Errorf("invoice = %q, want needs verification: its own hop is exact and the chain it hangs off is not",
			invoice.Evidence)
	}
	if !strings.Contains(invoice.EvidenceWhy, "quote") {
		t.Errorf("the reason does not name the hop to check: %q", invoice.EvidenceWhy)
	}
}

// 4. The snapshot's own account of itself reaches the field.
func TestPartialScenarioCarriesTheSnapshotsOwnDiagnostics(t *testing.T) {
	f, _ := partialField(t)
	m := f.Meta

	if !m.SawSummary {
		t.Fatal("the captured snapshot's summary record was not read")
	}
	if len(m.PartialFailures) != 1 || m.PartialFailures[0].Code != "E_MINIFIED" {
		t.Fatalf("partial failures = %+v, want one E_MINIFIED", m.PartialFailures)
	}
	if m.PartialFailures[0].FilePath != "rate_table.min.json" {
		t.Errorf("the partial failure names %q", m.PartialFailures[0].FilePath)
	}
	if !m.Degraded() {
		t.Error("a snapshot with a partial failure is not a complete reading")
	}
	if m.Stats.ParsedFiles != 5 || m.Stats.Files != 6 {
		t.Errorf("stats = %d of %d parsed, want 5 of 6", m.Stats.ParsedFiles, m.Stats.Files)
	}

	// Measured, not assumed: an inventory-only language on its own produces a
	// file, a symbol and no relations, and no partial failure of its own.
	if !m.InventoryOnly("COBOL") || !m.InventoryOnly("JSON") {
		t.Error("the snapshot's language tiers were not read")
	}
	if m.InventoryOnly("Python") {
		t.Error("Python is semantic in this snapshot")
	}

	// The provider's healthy level is "ok", not "complete". Assuming otherwise
	// put a warning on every clean snapshot.
	if m.Stats.CompletenessLevel != "ok" {
		t.Errorf("completeness level = %q, want the provider's own word", m.Stats.CompletenessLevel)
	}
	if !graph.CompletenessIsHealthy("ok") || graph.CompletenessIsHealthy("degraded") ||
		graph.CompletenessIsHealthy("unsafe") {
		t.Error("the healthy completeness levels are wrong")
	}
}

//  5. A node in a file the provider could not fully analyse needs verification,
//     whatever its relation said.
func TestPartialScenarioMarksNodesInFilesTheProviderSkipped(t *testing.T) {
	f, _ := partialField(t)
	n := &Node{
		Symbol:     &graph.Symbol{ID: "x", Name: "rates", File: "rate_table.min.json", Language: "JSON"},
		Resolution: graph.ResolutionExact,
	}
	bi := BuildInput{Meta: f.Meta}
	markIncompleteFile(n, bi.Meta)
	if !n.FileIncomplete {
		t.Fatal("a node in the minified file is not marked incomplete")
	}
	got, why := ClassifyEvidence(n, EvidenceInput{})
	if got != NeedsVerification {
		t.Errorf("= %q, want needs verification", got)
	}
	if !strings.Contains(why, "E_MINIFIED") {
		t.Errorf("the reason does not quote the provider's code: %q", why)
	}

	// A symbol in the inventory-only COBOL file is the other half: the file
	// parsed, no relations were extracted from it, so nothing in it can be
	// confirmed structurally.
	cob := &Node{Symbol: &graph.Symbol{ID: "c", Name: "legacy_rates",
		File: "legacy_rates.cob", Language: "COBOL"}, Resolution: graph.ResolutionExact}
	markIncompleteFile(cob, f.Meta)
	if !cob.FileIncomplete {
		t.Fatal("a symbol in an inventory-only language is not marked incomplete")
	}
	if got, _ := ClassifyEvidence(cob, EvidenceInput{}); got != NeedsVerification {
		t.Errorf("COBOL symbol = %q, want needs verification", got)
	}
}
