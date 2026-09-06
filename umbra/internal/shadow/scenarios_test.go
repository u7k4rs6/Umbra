package shadow

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// These tests replay the eight recorded scenarios from fixtures/recorded
// against the real graph snapshot of the fixture app. Nothing here starts a
// process, reaches the network, or needs an agent: the transcript is a file
// and the graph is a captured snapshot.

const fixturesDir = "../../fixtures/recorded"

// scenarioField loads the real snapshot captured from the fixture app. The
// paths inside it are repository relative, which is what the adapter produces.
func scenarioField(t *testing.T) (*graph.Field, *graph.RelationMap) {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "snapshot.ndjson"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
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

// The snapshot fixture was scrubbed to repository-relative paths without the
// umbra/fixtures/app prefix, while the scenario transcripts carry the full
// path as a real session would. This maps one onto the other.
const scenarioPrefix = "umbra/fixtures/app/"

func loadScenario(t *testing.T, name string) *transcript.Session {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join(fixturesDir, name, "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read scenario %s: %v", name, err)
	}
	s, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("parse scenario %s: %v", name, err)
	}
	// Strip the project prefix so the transcript paths line up with the
	// scrubbed snapshot.
	for i := range s.Events {
		s.Events[i].Path = stripPrefix(s.Events[i].Path)
		for j := range s.Events[i].Paths {
			s.Events[i].Paths[j] = stripPrefix(s.Events[i].Paths[j])
		}
	}
	return s
}

func stripPrefix(p string) string {
	if len(p) > len(scenarioPrefix) && p[:len(scenarioPrefix)] == scenarioPrefix {
		return p[len(scenarioPrefix):]
	}
	return p
}

type scenarioResult struct {
	nodes   []*Node
	byName  map[string]*Node
	summary Summary
	field   *graph.Field
	relMap  *graph.RelationMap
	sources []graph.Source
	srcIDs  []string
}

// runScenario is the classify and rank half of the pipeline, driven from a
// recorded transcript and a recorded snapshot.
func runScenario(t *testing.T, name string, headRoot string) scenarioResult {
	t.Helper()
	field, relMap := scenarioField(t)

	var src *graph.Symbol
	for _, id := range field.ByFile["app/service.py"] {
		if field.Symbols[id].Name == "compute_total" {
			src = field.Symbols[id]
		}
	}
	if src == nil {
		t.Fatal("compute_total missing from the snapshot")
	}
	sources := []graph.Source{{
		Symbol: src.ID, Name: src.Name, File: src.File, Span: src.Span,
		Change: "signature", Weight: graph.Weight("signature"), Dependents: 7,
	}}

	session := loadScenario(t, name)
	session.ResolveMentions(Vocabulary(field))
	examined := BuildExamined(session, map[string]bool{"app/service.py": true})

	reach := field.Dependents([]string{src.ID}, 2, relMap)
	in := BuildInput{
		Field: field, RelMap: relMap, Sources: sources, Reach: reach,
		Examined: examined, Impacts: map[string]*graph.Impact{}, HeadRoot: headRoot,
	}
	nodes := Build(in)
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

func wantState(t *testing.T, r scenarioResult, name string, state State, tier Tier) {
	t.Helper()
	n, ok := r.byName[name]
	if !ok {
		t.Fatalf("%s is not in the field", name)
	}
	if n.State != state {
		t.Fatalf("%s state = %v, want %v", name, n.State, state)
	}
	if n.Tier != tier {
		t.Fatalf("%s tier = %v, want %v", name, n.Tier, tier)
	}
}

// 1. Everything the change reaches was read in full after the cut.
func TestScenarioEverythingLit(t *testing.T) {
	r := runScenario(t, "everything-lit", "")
	for _, name := range []string{"handle_order", "quote", "apply_refund", "test_rounding", "test_handle_order"} {
		wantState(t, r, name, Lit, TierNone)
	}
	if r.summary.Umbra != 0 || r.summary.Penumbra != 0 {
		t.Fatalf("summary = %+v, want nothing in shadow", r.summary)
	}
	for _, n := range r.nodes {
		if n.Shadowed() {
			t.Fatalf("%s should not be shadowed", n.Symbol.Name)
		}
	}
}

// 2. The everyday case: one caller read, one only searched, one never opened.
func TestScenarioClassic(t *testing.T) {
	r := runScenario(t, "classic", "")
	wantState(t, r, "handle_order", Lit, TierNone)
	wantState(t, r, "apply_refund", Penumbra, TierGlimpse)
	wantState(t, r, "test_rounding", Umbra, TierNone)

	// The never-opened test outranks the caller that was at least searched.
	if r.byName["test_rounding"].Score <= r.byName["apply_refund"].Score {
		t.Fatalf("an unopened test should outrank a searched caller: %v vs %v",
			r.byName["test_rounding"].Score, r.byName["apply_refund"].Score)
	}
}

// 3. A read whose range stops before the caller's span.
func TestScenarioPartialRead(t *testing.T) {
	r := runScenario(t, "partial-read", "")
	wantState(t, r, "handle_order", Penumbra, TierGlance)
	// quote sits at line 26, also past the twelve line read.
	wantState(t, r, "quote", Penumbra, TierGlance)
	wantState(t, r, "test_rounding", Umbra, TierNone)
}

// 4. Read in full, but only before the change began.
func TestScenarioPreChange(t *testing.T) {
	r := runScenario(t, "pre-change", "")
	wantState(t, r, "apply_refund", Penumbra, TierAfterimage)
	// An afterimage scores higher than an ordinary penumbra, because a
	// picture of the old code is worse than a partial look at the new one.
	if r.byName["apply_refund"].Factors["state"] != 0.6 {
		t.Fatalf("afterimage state weight = %v, want 0.6", r.byName["apply_refund"].Factors["state"])
	}
}

// 5. No tool activity at all: every node is unknown, never an error.
func TestScenarioNoReads(t *testing.T) {
	r := runScenario(t, "no-reads", "")
	if len(r.nodes) == 0 {
		t.Fatal("the field should still be built without a transcript")
	}
	for _, n := range r.nodes {
		if n.State != Unknown {
			t.Fatalf("%s state = %v, want unknown", n.Symbol.Name, n.State)
		}
	}
	if _, ok := r.summary.Illumination(); ok {
		t.Fatal("illumination must be undefined when everything is unknown")
	}
}

// 6. The agent names a symbol it never opened.
func TestScenarioEcho(t *testing.T) {
	r := runScenario(t, "echo", "")
	wantState(t, r, "apply_refund", Penumbra, TierEcho)
	if r.byName["apply_refund"].Factors["state"] != 0.7 {
		t.Fatalf("echo state weight = %v, want 0.7", r.byName["apply_refund"].Factors["state"])
	}
	// A file it never named at all stays in full shadow.
	wantState(t, r, "test_rounding", Umbra, TierNone)
}

// 7. A test beyond the searched depth that the selection missed.
//
// This runs against the real chain in the captured snapshot, not a graph built
// for the test: tests/test_report.py::test_receipt_total calls render_lines,
// which calls line_text, which calls line_total, which calls compute_total.
// That is four hops, so a search capped at two never sees the test, the sweep
// finds it changing state, and the forensics have to say why.
func TestScenarioLeak(t *testing.T) {
	r := runScenario(t, "leak", "")

	const leaked = "tests/test_report.py::test_receipt_total"

	selected, _ := SelectTests(r.nodes, r.field, r.relMap, r.srcIDs, "shadow", "")
	sel := map[string]bool{}
	for _, id := range IDs(selected) {
		sel[id] = true
	}
	if len(selected) == 0 {
		t.Fatal("the leak scenario should still select the tests it can see")
	}
	// The whole point: the four-hop test is out of reach of the selection.
	if sel[leaked] {
		t.Fatalf("%s is four hops away and must not be selected at depth 2", leaked)
	}

	leaks := Forensics([]string{leaked}, ForensicsInput{
		Field: r.field, RelMap: r.relMap, SourceIDs: r.srcIDs,
		Selected: sel, MaxDepth: 2, CoChangeFiles: map[string]bool{"app/api.py": true},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v, want one", leaks)
	}
	// The reason names the depth and the first hop, from the real chain.
	want := "path exists at depth 4 via render_lines"
	if leaks[0].Reason != want {
		t.Fatalf("reason = %q, want %q", leaks[0].Reason, want)
	}
	if leaks[0].Test != leaked {
		t.Fatalf("leak names %q", leaks[0].Test)
	}

	// A test that was selected is never a leak, however it changed.
	notLeak := Forensics([]string{IDs(selected)[0]}, ForensicsInput{
		Field: r.field, RelMap: r.relMap, SourceIDs: r.srcIDs, Selected: sel, MaxDepth: 2,
	})
	if len(notLeak) != 0 {
		t.Fatalf("a selected test is not a leak, got %v", notLeak)
	}
}

// The chain the leak scenario depends on, asserted on its own so a change to
// the fixture that shortens it fails here with a clear reason rather than
// somewhere downstream.
func TestLeakChainIsFourHops(t *testing.T) {
	field, relMap := scenarioField(t)

	id := func(file, name string) string {
		t.Helper()
		for _, s := range field.ByFile[file] {
			if field.Symbols[s].Name == name {
				return s
			}
		}
		t.Fatalf("the fixture has no %s in %s", name, file)
		return ""
	}
	src := id("app/service.py", "compute_total")

	cases := []struct {
		file, name string
		depth      int
		via        string
	}{
		{"app/report.py", "line_total", 1, "compute_total"},
		{"app/report.py", "line_text", 2, "line_total"},
		{"app/report.py", "render_lines", 3, "line_text"},
		{"tests/test_report.py", "test_receipt_total", 4, "render_lines"},
	}
	for _, c := range cases {
		depth, via, ok := field.PathTo(id(c.file, c.name), map[string]bool{src: true}, relMap, 8)
		if !ok {
			t.Errorf("%s does not reach compute_total at all", c.name)
			continue
		}
		if depth != c.depth || via != c.via {
			t.Errorf("%s: depth %d via %s, want depth %d via %s", c.name, depth, via, c.depth, c.via)
		}
	}
}

// 8. A caller under a SAFETY comment inside an except block is pinned first.
func TestScenarioBeacon(t *testing.T) {
	// The window is read from the fixture app in the repository, which is a
	// committed file rather than a live checkout.
	root, err := filepath.Abs(filepath.Join("..", "..", "fixtures", "app"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "app", "refunds.py")); err != nil {
		t.Skipf("fixture app not present: %v", err)
	}

	field, relMap := scenarioField(t)
	var src *graph.Symbol
	for _, id := range field.ByFile["app/service.py"] {
		if field.Symbols[id].Name == "compute_total" {
			src = field.Symbols[id]
		}
	}
	sources := []graph.Source{{Symbol: src.ID, Name: src.Name, File: src.File, Span: src.Span, Change: "signature", Weight: 1}}

	session := loadScenario(t, "beacon")
	session.ResolveMentions(Vocabulary(field))
	examined := BuildExamined(session, map[string]bool{"app/service.py": true})

	// The call site of apply_refund into compute_total, from graph impact.
	impacts := map[string]*graph.Impact{
		src.ID: {CallSitesByName: map[string]int{"app/refunds.py:apply_refund": 23}, CallSites: map[string]int{}},
	}
	in := BuildInput{
		Field: field, RelMap: relMap, Sources: sources,
		Reach:    field.Dependents([]string{src.ID}, 2, relMap),
		Examined: examined, Impacts: impacts, HeadRoot: root,
	}
	nodes := Build(in)
	SortDocket(nodes)

	var refund *Node
	for _, n := range nodes {
		if n.Symbol.Name == "apply_refund" {
			refund = n
		}
	}
	if refund == nil {
		t.Fatal("apply_refund missing")
	}
	if !refund.Pinned() {
		t.Fatalf("apply_refund should be pinned by its beacon, modifiers %v", refund.Modifiers)
	}
	if !hasModifier(refund, "fault line") {
		t.Fatalf("the call site sits inside an except block, modifiers %v", refund.Modifiers)
	}
	if nodes[0] != refund {
		t.Fatalf("a beacon sorts first, got %s", nodes[0].Symbol.Name)
	}
	if refund.Beacon == "" {
		t.Fatal("the beacon comment line should be carried")
	}
}

// Every scenario directory must carry a transcript and a note saying what it
// shows, so a reader can tell what each fixture is for.
func TestEveryScenarioIsDocumented(t *testing.T) {
	want := []string{"everything-lit", "classic", "partial-read", "pre-change", "no-reads", "echo", "leak", "beacon"}
	for _, name := range want {
		if _, err := os.Stat(filepath.Join(fixturesDir, name, "transcript.jsonl")); err != nil {
			t.Errorf("scenario %s has no transcript: %v", name, err)
		}
		notes, err := os.ReadFile(filepath.Join(fixturesDir, name, "notes.txt"))
		if err != nil || len(notes) < 20 {
			t.Errorf("scenario %s has no useful notes", name)
		}
	}
}

// 9. The changed symbols are a method on a class and a function nested inside
// another function, which are the two definition shapes nothing else in this
// suite exercises.
//
// This scenario builds its sources the way the pipeline does, by handing
// graph.Bind the names `graph commit` would report, rather than looking a
// symbol up by its bare name the way the other scenarios do. That is the point
// of it: the method is named LineItem.subtotal and the snapshot's Name field
// holds the bare subtotal, and app/models.py holds a second symbol of that
// name on RefundLine.
func TestScenarioQualifiedNames(t *testing.T) {
	field, relMap := scenarioField(t)

	// Exactly what `graph commit --json` reports for this change.
	sources := graph.Bind([]graph.Source{
		{Name: "LineItem.subtotal", Kind: "method", File: "app/models.py",
			Span: [2]int{15, 15}, Change: "body", Weight: graph.Weight("body")},
		{Name: "pad", Kind: "function", File: "app/report.py",
			Span: [2]int{32, 32}, Change: "body", Weight: graph.Weight("body")},
	}, field)

	var ids []string
	for _, s := range sources {
		if s.Symbol == "" {
			t.Fatalf("%s did not bind: %s", s.Name, s.Unresolved)
		}
		ids = append(ids, s.Symbol)
	}

	// The method has to be LineItem's, not RefundLine's. Binding on the bare
	// name matches both, and either answer would be silently plausible.
	if got := field.Symbols[ids[0]].QualifiedName; got != "LineItem.subtotal" {
		t.Fatalf("the method bound to %q", got)
	}
	if got := field.Symbols[ids[1]]; got.ContainerID == "" {
		t.Fatal("pad is meant to be the nested function")
	}

	session := loadScenario(t, "qualified-names")
	session.ResolveMentions(Vocabulary(field))
	examined := BuildExamined(session, map[string]bool{
		"app/models.py": true, "app/report.py": true,
	})

	reach := field.Dependents(ids, 2, relMap)
	if len(reach) == 0 {
		t.Fatal("a changed method with callers must reach something")
	}
	nodes := Build(BuildInput{
		Field: field, RelMap: relMap, Sources: sources, Reach: reach,
		Examined: examined, Impacts: map[string]*graph.Impact{},
	})
	SortDocket(nodes)
	if len(nodes) == 0 {
		t.Fatal("the field is empty for a changed method that has dependents")
	}

	// The session read app/models.py lines 1 to 18, which covers
	// LineItem.subtotal and stops before RefundLine.subtotal. A run that bound
	// the wrong method would describe the wrong half of that file.
	byName := map[string]*Node{}
	for _, n := range nodes {
		byName[n.Symbol.Name] = n
	}
	if n, ok := byName["compute_total"]; ok && n.State == Unknown {
		t.Fatal("compute_total should have a state, not unknown")
	}
}
