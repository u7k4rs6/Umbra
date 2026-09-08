package report

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// The ninth scenario driven all the way through the renderers, from the same
// captured snapshot and the same transcript the classifier replay uses. This
// is where the requirement that the verification command is actually PRINTED
// is checked, rather than only that the tier was computed.
func partialAnalysis(t *testing.T) *Analysis {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "partial-snapshot.ndjson"))
	if err != nil {
		t.Fatalf("read partial snapshot: %v", err)
	}
	field, err := graph.LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	capsBlob, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "capabilities.json"))
	if err != nil {
		t.Fatalf("read capabilities: %v", err)
	}
	caps, err := graph.ParseCapabilities(capsBlob)
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	relMap := graph.NewRelationMap(caps)

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

	tr, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "recorded", "dynamic-dispatch", "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read scenario: %v", err)
	}
	session, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(tr)
	if err != nil {
		t.Fatalf("parse scenario: %v", err)
	}
	for i := range session.Events {
		session.Events[i].Path = strings.TrimPrefix(session.Events[i].Path, "umbra/fixtures/partial/")
		for j := range session.Events[i].Paths {
			session.Events[i].Paths[j] = strings.TrimPrefix(session.Events[i].Paths[j], "umbra/fixtures/partial/")
		}
	}
	session.ResolveMentions(shadow.Vocabulary(field))
	examined := shadow.BuildExamined(session, map[string]bool{"pricing.py": true})

	in := shadow.BuildInput{
		Field: field, RelMap: relMap, Sources: sources,
		Reach:    field.Dependents([]string{src.ID}, 4, relMap),
		Examined: examined, Impacts: map[string]*graph.Impact{}, Meta: field.Meta,
	}
	nodes := shadow.Build(in)
	shadow.SortDocket(nodes)

	a := &Analysis{
		Version: "0.1.0", CheckpointID: "partial00scenario", Commit: "aaaaaaaaaaaa",
		Adapter: "claude-code", Depth: 4, Run: "none", TestRunner: "pytest -v",
		Sources: sources, Nodes: nodes,
		RelationsHeuristic: relMap.Heuristic,
		Graph: GraphEvidence{
			Profile: field.Meta.Profile, SawSummary: field.Meta.SawSummary,
			CompletenessLevel: field.Meta.Stats.CompletenessLevel,
			Files:             field.Meta.Stats.Files, ParsedFiles: field.Meta.Stats.ParsedFiles,
		},
	}
	for _, pf := range field.Meta.PartialFailures {
		a.Graph.PartialFailures = append(a.Graph.PartialFailures,
			GraphDiagnostic{Code: pf.Code, Level: pf.Severity, File: pf.FilePath, Effect: pf.Effect})
	}
	a.Summary = shadow.Summarize(nodes)
	return a
}

func TestPartialScenarioTerminalPrintsTheVerificationCommand(t *testing.T) {
	a := partialAnalysis(t)
	out := render(t, a, TableOptions{UTF8: true, Width: 100, All: true})

	// The header names the partial analysis, with the provider's own code.
	if !strings.Contains(out, "this analysis may be incomplete") {
		t.Error("the header does not say the analysis may be incomplete")
	}
	if !strings.Contains(out, "E_MINIFIED") {
		t.Error("the header does not name the provider's own reason")
	}
	if !strings.Contains(out, "5 of 6 files parsed") {
		t.Error("the header does not say how much of the repository was read")
	}

	// The exact command, for a real node in the captured snapshot.
	for _, want := range []string{
		"entire graph def --repo . --symbol invoice --file billing.py",
		"entire graph def --repo . --symbol test_invoice_through_the_rate_card --file test_dispatch.py",
		"pytest -v test_dispatch.py::test_invoice_through_the_rate_card",
		"billing.py:",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the verification path is missing %q", want)
		}
	}

	// And it says which hop to check, not merely that there is one.
	if !strings.Contains(out, "quote") {
		t.Error("the reason does not name the hop that made the path heuristic")
	}
}

func TestPartialScenarioJSONAndPacketCarryTheSameThing(t *testing.T) {
	a := partialAnalysis(t)

	var j strings.Builder
	if err := WriteJSON(&j, Seal(a, nil)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	for _, want := range []string{
		`"evidence": "needs verification"`,
		`"evidence": "heuristic"`,
		`"evidence": "confirmed"`,
		`"resolution": "type_inferred"`,
		"entire graph def --repo . --symbol invoice --file billing.py",
		"pytest -v test_dispatch.py::test_invoice_through_the_rate_card",
		`"may_be_incomplete": true`,
	} {
		if !strings.Contains(j.String(), want) {
			t.Errorf("JSON is missing %q", want)
		}
	}

	var p strings.Builder
	if err := Packet(&p, Seal(a, nil)); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	for _, want := range []string{
		"## 6. What to verify",
		"entire graph def --repo . --symbol invoice --file billing.py",
		"This analysis may be incomplete",
	} {
		if !strings.Contains(p.String(), want) {
			t.Errorf("packet is missing %q", want)
		}
	}
}

// The dispatch table, getattr and importlib callers are absent from the graph,
// so no surface may name them at all, let alone as covered.
func TestPartialScenarioNeverClaimsTheDynamicCallersAreCovered(t *testing.T) {
	a := partialAnalysis(t)
	var b strings.Builder
	if err := WriteJSON(&b, Seal(a, nil)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	for _, absent := range []string{
		"invoice_total", "receipt_total", "dispatch_by_name", "dispatch_from_module",
	} {
		if strings.Contains(b.String(), `"name": "`+absent+`"`) {
			t.Errorf("%q reaches the change only through runtime dispatch and appears in the field", absent)
		}
	}
	// It follows that the report must say the field is a lower bound. That
	// sentence has been in the limitations from the first commit and this is
	// the run where it stops being boilerplate.
	a.Limitations = []string{"graph edges are incomplete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined"}
	b.Reset()
	if err := WriteJSON(&b, Seal(a, nil)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if !strings.Contains(b.String(), "dynamic dispatch") {
		t.Error("the report does not disclose that dynamic dispatch is invisible")
	}
}
