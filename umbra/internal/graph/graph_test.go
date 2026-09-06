package graph

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

func read(t *testing.T, name string) []byte {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return blob
}

func loadField(t *testing.T) *Field {
	t.Helper()
	f, err := LoadSnapshot(read(t, "snapshot.ndjson"))
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	return f
}

func loadRelMap(t *testing.T) *RelationMap {
	t.Helper()
	c, err := ParseCapabilities(read(t, "capabilities.json"))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	return NewRelationMap(c)
}

func symbolID(t *testing.T, f *Field, file, name string) string {
	t.Helper()
	for _, id := range f.ByFile[file] {
		if f.Symbols[id].Name == name {
			return id
		}
	}
	t.Fatalf("no symbol %s in %s", name, file)
	return ""
}

func TestLoadSnapshotReadsSymbols(t *testing.T) {
	f := loadField(t)
	id := symbolID(t, f, "app/service.py", "compute_total")
	s := f.Symbols[id]
	if s.Kind != "function" {
		t.Fatalf("kind = %q", s.Kind)
	}
	// The span is asserted structurally rather than against fixed line
	// numbers: the fixture is a real file and symbols move when it is edited,
	// which is not what this test is about.
	if s.Span[0] <= 0 || s.Span[1] <= s.Span[0] {
		t.Fatalf("span = %v, want a real range with a positive start", s.Span)
	}
	if !strings.Contains(s.Signature, "def compute_total(") {
		t.Fatalf("signature = %q", s.Signature)
	}
	if s.IsTest {
		t.Fatal("compute_total is not a test")
	}
}

// The snapshot carries no test marking, so IsTest comes from conventions.
func TestIsTestFromConventions(t *testing.T) {
	f := loadField(t)
	id := symbolID(t, f, "tests/test_service.py", "test_rounding")
	if !f.Symbols[id].IsTest {
		t.Fatal("test_rounding should be marked as a test by convention")
	}
}

func TestIsTestFile(t *testing.T) {
	cases := map[string]bool{
		"tests/test_service.py": true,
		"app/test_helpers.py":   true,
		"pkg/thing_test.go":     true,
		"src/thing_test.py":     true,
		"web/button.test.tsx":   true,
		"web/button.spec.ts":    true,
		"tests/helpers.py":      true,
		"app/service.py":        false,
		"app/latest.py":         false,
		"":                      false,
	}
	for p, want := range cases {
		if got := IsTestFile(p); got != want {
			t.Errorf("IsTestFile(%q) = %v, want %v", p, got, want)
		}
	}
}

// A helper or an import inside a test file is not itself a runnable test,
// because a runner cannot name it.
func TestOnlyTestFunctionsCountAsTests(t *testing.T) {
	cases := []struct {
		sym  Symbol
		want bool
	}{
		{Symbol{Kind: "function", Name: "test_rounding"}, true},
		{Symbol{Kind: "function", Name: "TestRounding"}, true},
		{Symbol{Kind: "method", Name: "test_case"}, true},
		{Symbol{Kind: "function", Name: "make_order"}, false},
		{Symbol{Kind: "class", Name: "TestHelpers"}, false},
		{Symbol{Kind: "import", Name: "test_rounding"}, false},
	}
	for _, c := range cases {
		sym := c.sym
		if got := looksLikeTestSymbol(&sym); got != c.want {
			t.Errorf("looksLikeTestSymbol(%s %s) = %v, want %v", sym.Kind, sym.Name, got, c.want)
		}
	}
}

func TestLoadSnapshotReadsRelationsWithCallSites(t *testing.T) {
	f := loadField(t)
	target := symbolID(t, f, "app/service.py", "compute_total")
	edges := f.In[target]
	if len(edges) == 0 {
		t.Fatal("expected edges arriving at compute_total")
	}
	callers := map[string]bool{}
	for _, e := range edges {
		if e.Relation == "CALLS" {
			callers[f.Symbols[e.From].Name] = true
		}
	}
	for _, want := range []string{"handle_order", "quote", "apply_refund", "test_rounding"} {
		if !callers[want] {
			t.Fatalf("expected %s among callers, got %v", want, callers)
		}
	}
}

func TestRelationMapFromRealCapabilities(t *testing.T) {
	rm := loadRelMap(t)
	if fam, ok := rm.Family("CALLS"); !ok || fam != FamilyCalls {
		t.Fatalf("CALLS mapped to %q ok=%v", fam, ok)
	}
	if fam, ok := rm.Family("DATA_FLOWS"); !ok || fam != FamilyDataFlow {
		t.Fatalf("DATA_FLOWS mapped to %q", fam)
	}
	if fam, ok := rm.Family("PARAM_TYPE"); !ok || fam != FamilyTypeUse {
		t.Fatalf("PARAM_TYPE mapped to %q", fam)
	}
	if fam, ok := rm.Family("FILE_CHANGES_WITH"); !ok || fam != FamilyCoChange {
		t.Fatalf("FILE_CHANGES_WITH mapped to %q", fam)
	}
	if !rm.Traverses("CALLS") {
		t.Fatal("calls must be traversed")
	}
	if rm.Traverses("IMPORTS") {
		t.Fatal("imports are file level and must not add dependents")
	}
	if rm.Traverses("FILE_CHANGES_WITH") {
		t.Fatal("co-change enters through impact, not through the walk")
	}
}

// Relations Umbra does not traverse are listed rather than silently dropped.
func TestRelationMapListsIgnored(t *testing.T) {
	rm := loadRelMap(t)
	found := false
	for _, name := range rm.Ignored {
		if name == "HANDLES_ROUTE" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected HANDLES_ROUTE among ignored, got %v", rm.Ignored)
	}
}

// SECURITY_AND_ACCESS.md claims the default path is offline. Check it rather
// than assume it.
func TestInstalledGraphDeclaresNoNetwork(t *testing.T) {
	c, err := ParseCapabilities(read(t, "capabilities.json"))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	if !c.Offline() {
		t.Fatalf("expected no network features, got %v", c.NetworkFeatures)
	}
}

func TestDependentsFindsTheShadowedTest(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")

	reach := f.Dependents([]string{src}, 2, rm)
	byName := map[string]Reach{}
	for _, r := range reach {
		byName[f.Symbols[r.ID].Name] = r
	}

	// The direct callers, including the tests in the file the session never
	// opened, are what the whole product is about.
	for _, want := range []string{"handle_order", "quote", "apply_refund", "test_rounding", "test_negative"} {
		r, ok := byName[want]
		if !ok {
			t.Fatalf("expected %s among dependents", want)
		}
		if r.Depth != 1 {
			t.Fatalf("%s depth = %d, want 1", want, r.Depth)
		}
	}
	// A test that reaches compute_total only through handle_order is depth 2.
	if r, ok := byName["test_handle_order"]; !ok || r.Depth != 2 {
		t.Fatalf("test_handle_order = %+v, want depth 2", r)
	}
}

func TestDependentsRespectsDepth(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")

	at1 := f.Dependents([]string{src}, 1, rm)
	at2 := f.Dependents([]string{src}, 2, rm)
	if len(at1) >= len(at2) {
		t.Fatalf("depth 1 found %d, depth 2 found %d; depth 2 should reach further", len(at1), len(at2))
	}
	for _, r := range at1 {
		if r.Depth > 1 {
			t.Fatalf("depth 1 returned a node at depth %d", r.Depth)
		}
	}
}

func TestDependentsIsDeterministic(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")

	first := f.Dependents([]string{src}, 2, rm)
	for i := 0; i < 5; i++ {
		again := f.Dependents([]string{src}, 2, rm)
		if len(again) != len(first) {
			t.Fatalf("run %d found %d nodes, first found %d", i, len(again), len(first))
		}
		for j := range first {
			if again[j].ID != first[j].ID || again[j].Depth != first[j].Depth {
				t.Fatalf("run %d differs at %d: %v vs %v", i, j, again[j], first[j])
			}
		}
	}
}

func TestDependentsExcludesTheSourceItself(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")
	for _, r := range f.Dependents([]string{src}, 2, rm) {
		if r.ID == src {
			t.Fatal("a source must not be its own dependent")
		}
	}
}

func TestPathToFindsDepthAndFirstHop(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")
	test := symbolID(t, f, "tests/test_api.py", "test_handle_order")

	depth, via, ok := f.PathTo(test, map[string]bool{src: true}, rm, 6)
	if !ok {
		t.Fatal("expected a path from test_handle_order to compute_total")
	}
	if depth != 2 {
		t.Fatalf("depth = %d, want 2", depth)
	}
	if via != "handle_order" {
		t.Fatalf("via = %q, want handle_order", via)
	}
}

func TestPathToReportsNoPath(t *testing.T) {
	f := loadField(t)
	rm := loadRelMap(t)
	src := symbolID(t, f, "app/service.py", "compute_total")
	unrelated := symbolID(t, f, "tests/test_models.py", "test_line_item_subtotal")

	if _, _, ok := f.PathTo(unrelated, map[string]bool{src: true}, rm, 6); ok {
		t.Fatal("test_line_item_subtotal does not reach compute_total")
	}
}

func TestParseImpactJSONCallSites(t *testing.T) {
	imp, err := ParseImpactJSON(read(t, "impact.json"))
	if err != nil {
		t.Fatalf("ParseImpactJSON: %v", err)
	}
	// The probe confirmed these exact call-site lines.
	if got := imp.CallSitesByName["app/api.py:handle_order"]; got != 16 {
		t.Fatalf("handle_order call site = %d, want 16", got)
	}
	if got := imp.CallSitesByName["app/refunds.py:apply_refund"]; got != 23 {
		t.Fatalf("apply_refund call site = %d, want 23", got)
	}
	if got := imp.CallSitesByName["tests/test_service.py:test_rounding"]; got != 9 {
		t.Fatalf("test_rounding call site = %d, want 9", got)
	}
	if imp.DirectCallers != 7 {
		t.Fatalf("direct callers = %d, want 7", imp.DirectCallers)
	}
}

// The text parser is the fallback for a Graph without --format json. It must
// read the documented shape "name (file:CALLSITE, def :DEFLINE)".
func TestParseImpactTextCallSites(t *testing.T) {
	imp := ParseImpactText(string(read(t, "impact.txt")))
	if got := imp.CallSitesByName["app/api.py:handle_order"]; got != 16 {
		t.Fatalf("handle_order call site = %d, want 16", got)
	}
	if got := imp.CallSitesByName["app/refunds.py:apply_refund"]; got != 23 {
		t.Fatalf("apply_refund call site = %d, want 23", got)
	}
	if imp.DirectCallers != 7 {
		t.Fatalf("direct callers = %d, want 7", imp.DirectCallers)
	}
	if imp.TransitiveCallers != 4 {
		t.Fatalf("transitive callers = %d, want 4", imp.TransitiveCallers)
	}
}

func TestParseImpactTextCoChange(t *testing.T) {
	imp := ParseImpactText(string(read(t, "impact.txt")))
	found := false
	for _, p := range imp.CoChange {
		if p == "app/api.py" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected app/api.py among co-change files, got %v", imp.CoChange)
	}
}

// graph commit distinguishes a signature change from a body change, which the
// source weights depend on.
func TestParseCommitJSONDistinguishesSignatureFromBody(t *testing.T) {
	srcs, err := ParseCommitJSON(read(t, "commit.json"))
	if err != nil {
		t.Fatalf("ParseCommitJSON: %v", err)
	}
	byName := map[string]Source{}
	for _, s := range srcs {
		byName[s.Name] = s
	}
	ct, ok := byName["compute_total"]
	if !ok {
		t.Fatalf("compute_total missing from %v", byName)
	}
	if ct.Change != "signature" {
		t.Fatalf("change = %q, want signature", ct.Change)
	}
	if ct.Weight != 1.0 {
		t.Fatalf("weight = %v, want 1.0", ct.Weight)
	}
	if ct.Dependents != 7 {
		t.Fatalf("dependents = %d, want 7", ct.Dependents)
	}
	if ct.OldSignature == "" || ct.NewSignature == "" {
		t.Fatal("a signature change should carry both signatures")
	}

	ho, ok := byName["handle_order"]
	if !ok {
		t.Fatal("handle_order missing")
	}
	if ho.Change != "body" {
		t.Fatalf("handle_order change = %q, want body", ho.Change)
	}
	if ho.Weight != 0.6 {
		t.Fatalf("handle_order weight = %v, want 0.6", ho.Weight)
	}
}

func TestWeights(t *testing.T) {
	cases := map[string]float64{
		"signature": 1.0, "removed": 1.0, "renamed": 0.8,
		"body": 0.6, "added": 0.3, "something else": 0.6,
	}
	for kind, want := range cases {
		if got := Weight(kind); got != want {
			t.Errorf("Weight(%q) = %v, want %v", kind, got, want)
		}
	}
}

func TestBindAttachesSymbolIdentity(t *testing.T) {
	f := loadField(t)
	srcs, err := ParseCommitJSON(read(t, "commit.json"))
	if err != nil {
		t.Fatalf("ParseCommitJSON: %v", err)
	}
	bound := Bind(srcs, f)
	for _, s := range bound {
		if s.Name == "compute_total" {
			if s.Symbol == "" {
				t.Fatal("compute_total should bind to a symbol id")
			}
			if s.Span[1] <= s.Span[0] {
				t.Fatalf("bound span = %v, want the real span from the snapshot", s.Span)
			}
			return
		}
	}
	t.Fatal("compute_total not found after binding")
}

func TestLoadSnapshotSkipsBadLines(t *testing.T) {
	blob := []byte("not json\n" +
		`{"record_type":"symbol","id":"a","kind":"function","name":"f","file_path":"x.py","start_line":1,"end_line":2}` + "\n")
	f, err := LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(f.Symbols) != 1 {
		t.Fatalf("symbols = %d, want 1", len(f.Symbols))
	}
}

func TestVocabulary(t *testing.T) {
	f := loadField(t)
	files, syms := f.Vocabulary()
	if len(files) == 0 || len(syms) == 0 {
		t.Fatalf("vocabulary = %d files, %d symbols", len(files), len(syms))
	}
	has := func(list []string, want string) bool {
		for _, x := range list {
			if x == want {
				return true
			}
		}
		return false
	}
	if !has(files, "app/service.py") {
		t.Fatal("expected app/service.py in the file vocabulary")
	}
	if !has(syms, "apply_refund") {
		t.Fatal("expected apply_refund in the symbol vocabulary")
	}
}

// graph commit reports Markdown headings and code fences as changed entities.
// They have no callers and never will, so they must not become light sources.
func TestFilterSourcesDropsInventoryOnlyLanguages(t *testing.T) {
	c, err := ParseCapabilities(read(t, "capabilities.json"))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	f := &Field{
		Symbols: map[string]*Symbol{
			"py":  {ID: "py", Name: "compute_total", Kind: "function", File: "app/service.py", Language: "Python"},
			"md":  {ID: "md", Name: "Build-notes", Kind: "section", File: "NOTES.md", Language: "Markdown"},
			"fen": {ID: "fen", Name: "code_fence_1_text", Kind: "code_fence", File: "NOTES.md", Language: "Markdown"},
		},
		ByFile: map[string][]string{"app/service.py": {"py"}, "NOTES.md": {"md", "fen"}},
	}
	in := []Source{
		{Symbol: "py", Name: "compute_total", File: "app/service.py", Change: "signature"},
		{Symbol: "md", Name: "Build-notes", File: "NOTES.md", Change: "added"},
		{Symbol: "fen", Name: "code_fence_1_text", File: "NOTES.md", Change: "added"},
	}
	kept, dropped := FilterSources(in, f, c)
	if len(kept) != 1 || kept[0].Name != "compute_total" {
		t.Fatalf("kept = %v, want only compute_total", kept)
	}
	if len(dropped) != 2 {
		t.Fatalf("dropped = %d, want 2", len(dropped))
	}
}

func TestSemanticLanguagesFromRealCapabilities(t *testing.T) {
	c, err := ParseCapabilities(read(t, "capabilities.json"))
	if err != nil {
		t.Fatalf("ParseCapabilities: %v", err)
	}
	sem := SemanticLanguages(c)
	for _, want := range []string{"Python", "Go", "TypeScript", "Rust"} {
		if !sem[want] {
			t.Errorf("%s should be semantic", want)
		}
	}
	for _, notWant := range []string{"Markdown", "JSON", "CSS"} {
		if sem[notWant] {
			t.Errorf("%s is inventory only and must not be semantic", notWant)
		}
	}
}

// With no capabilities available the filter must not silently drop real code.
func TestFilterSourcesKeepsEverythingWithoutCapabilities(t *testing.T) {
	f := &Field{
		Symbols: map[string]*Symbol{"a": {ID: "a", Kind: "function", File: "x.py", Language: "Python"}},
		ByFile:  map[string][]string{"x.py": {"a"}},
	}
	kept, dropped := FilterSources([]Source{{Symbol: "a", File: "x.py"}}, f, nil)
	if len(kept) != 1 || len(dropped) != 0 {
		t.Fatalf("kept %d dropped %d, want everything kept", len(kept), len(dropped))
	}
}

// graph checkpoint returns the changes of the commit the checkpoint belongs
// to. When a reference was paired with a checkpoint by session time, that is a
// different commit, and using it would describe one commit under another one's
// name. This came out of a fresh clone, where the pairing chose a checkpoint
// that owned a commit and the report described that commit instead.
func TestLoadSourcesIgnoresACheckpointThatDoesNotOwnTheCommit(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"graph", "checkpoint", "CK", "--json"},
		runner.Result{Stdout: `{"base":"x","head":"y","files":[{"path":"other/file.go","status":"M","language":"Go",
		 "changes":[{"type":"body_changed","kind":"function","name":"somethingElse","after_start_line":1,"dependents_count":0}]}]}`})
	f.Set("entire", []string{"graph", "commit", "SHA", "--repo", "/repo", "--json"},
		runner.Result{Stdout: `{"base":"p","head":"SHA","files":[{"path":"app/service.py","status":"M","language":"Python",
		 "changes":[{"type":"signature_changed","kind":"function","name":"compute_total","after_start_line":20,"dependents_count":7}]}]}`})

	// Not owned: the checkpoint must be ignored entirely.
	got, err := LoadSources(context.Background(), f, "/repo", "CK", "SHA", false)
	if err != nil {
		t.Fatalf("LoadSources: %v", err)
	}
	if got.Route != "graph commit" {
		t.Fatalf("route = %q, want the commit route", got.Route)
	}
	if len(got.Sources) != 1 || got.Sources[0].Name != "compute_total" {
		t.Fatalf("sources = %v, want the commit's own change", got.Sources)
	}
	if got.Note == "" || !strings.Contains(got.Note, "session time") {
		t.Fatalf("the header should say why, got %q", got.Note)
	}

	// Owned: the documented bridge is used.
	got, err = LoadSources(context.Background(), f, "/repo", "CK", "SHA", true)
	if err != nil {
		t.Fatalf("LoadSources: %v", err)
	}
	if got.Route != "graph checkpoint" {
		t.Fatalf("route = %q, want the checkpoint route", got.Route)
	}
	if len(got.Sources) != 1 || got.Sources[0].Name != "somethingElse" {
		t.Fatalf("sources = %v", got.Sources)
	}
}

// A field is not a thing a caller calls, and its dependent count is every use
// of the type it sits in. Adding one struct to a test file put jsNode.ID in a
// report as a source with 46 dependents, beside jsNode itself.
func TestFoldFieldsDropsAFieldWhoseTypeIsAlreadyASource(t *testing.T) {
	in := []Source{
		{Name: "jsNode", Kind: "type", File: "x_test.go", Span: [2]int{10, 10}, Change: "added", Dependents: 3},
		{Name: "jsNode.ID", Kind: "field", File: "x_test.go", Span: [2]int{11, 11}, Change: "added", Dependents: 46},
		{Name: "jsNode.Name", Kind: "field", File: "x_test.go", Span: [2]int{12, 12}, Change: "added", Dependents: 46},
		{Name: "compute_total", Kind: "function", File: "app/service.py", Span: [2]int{20, 20}, Change: "signature", Dependents: 7},
	}
	got := FoldFields(in)

	var names []string
	for _, s := range got {
		names = append(names, s.Name)
	}
	if len(got) != 2 {
		t.Fatalf("sources = %v, want the type and the function only", names)
	}
	for _, s := range got {
		if s.Kind == "field" {
			t.Errorf("a field survived as a source: %s", s.Name)
		}
	}
}

// When the type itself did not change, the field folds into one entry for it
// rather than vanishing: something in that type did change.
func TestFoldFieldsFoldsIntoTheTypeWhenItIsNotASource(t *testing.T) {
	in := []Source{
		{Name: "Options.TestRoot", Kind: "field", File: "opt.go", Span: [2]int{30, 30}, Change: "added", Dependents: 12},
		{Name: "Options.Depth", Kind: "field", File: "opt.go", Span: [2]int{31, 31}, Change: "added", Dependents: 12},
	}
	got := FoldFields(in)
	if len(got) != 1 {
		t.Fatalf("got %d sources, want one entry for the type", len(got))
	}
	if got[0].Name != "Options" || got[0].Kind != "type" {
		t.Fatalf("folded into %s (%s), want Options (type)", got[0].Name, got[0].Kind)
	}
}

func TestFoldFieldsLeavesOrdinarySourcesAlone(t *testing.T) {
	in := []Source{
		{Name: "compute_total", Kind: "function", File: "a.py", Span: [2]int{1, 1}, Change: "signature"},
		{Name: "handle_order", Kind: "function", File: "b.py", Span: [2]int{2, 2}, Change: "body"},
	}
	if got := FoldFields(in); len(got) != 2 {
		t.Fatalf("got %d, want both kept", len(got))
	}
}

func TestKindLabel(t *testing.T) {
	cases := map[string]string{
		"added": "added", "removed": "removed", "renamed": "renamed",
		"signature": "signature changed", "body": "body changed",
	}
	for change, want := range cases {
		if got := (Source{Change: change}).KindLabel(); got != want {
			t.Errorf("KindLabel(%q) = %q, want %q", change, got, want)
		}
	}
}

// Binding a changed symbol
//
// `graph commit` names a changed entity by the snapshot's qualified_name.
// Bind compared against the snapshot's Name, which for a method is the bare
// form, so no changed method ever bound and no impact query was ever issued
// for one. Every changed symbol in fixtures/app is a module-level function,
// where the two forms are identical, which is why the fixture could not show
// it. NOTES carries the shape table this was measured from.

func bindField(t *testing.T) *Field {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", "snapshot.ndjson"))
	if err != nil {
		t.Fatalf("read snapshot: %v", err)
	}
	f, err := LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	return f
}

func bindOne(t *testing.T, f *Field, name, file string, line int) Source {
	t.Helper()
	out := Bind([]Source{{Name: name, File: file, Span: [2]int{line, line}}}, f)
	if len(out) != 1 {
		t.Fatalf("Bind returned %d sources", len(out))
	}
	return out[0]
}

// The case the whole defect is about. app/models.py holds two methods named
// subtotal, on LineItem and on RefundLine. Matching the bare Name finds both
// and the old code broke the tie by nearest line, so the answer depended on
// which line graph commit happened to report.
func TestBindResolvesAMethodByItsQualifiedName(t *testing.T) {
	f := bindField(t)

	for _, tc := range []struct{ name, wantSuffix string }{
		{"LineItem.subtotal", "method:LineItem.subtotal"},
		{"RefundLine.subtotal", "method:RefundLine.subtotal"},
		{"Order.line_count", "method:Order.line_count"},
	} {
		got := bindOne(t, f, tc.name, "app/models.py", 1)
		if got.Symbol == "" {
			t.Fatalf("%s did not bind: %s", tc.name, got.Unresolved)
		}
		if !strings.HasSuffix(got.Symbol, tc.wantSuffix) {
			t.Fatalf("%s bound to %s, want a symbol ending %s", tc.name, got.Symbol, tc.wantSuffix)
		}
		sym := f.Symbols[got.Symbol]
		if sym.QualifiedName != tc.name {
			t.Fatalf("%s bound to a symbol whose qualified name is %q", tc.name, sym.QualifiedName)
		}
		if got.Span != sym.Span {
			t.Fatalf("%s kept span %v, want the symbol's %v", tc.name, got.Span, sym.Span)
		}
	}
}

// The two methods must not resolve to the same symbol, which is what a bare
// name lookup would do for one of them.
func TestBindTellsTwoMethodsOfOneNameApart(t *testing.T) {
	f := bindField(t)
	a := bindOne(t, f, "LineItem.subtotal", "app/models.py", 15)
	b := bindOne(t, f, "RefundLine.subtotal", "app/models.py", 28)
	if a.Symbol == "" || b.Symbol == "" {
		t.Fatalf("both must bind, got %q and %q", a.Symbol, b.Symbol)
	}
	if a.Symbol == b.Symbol {
		t.Fatalf("both bound to the same symbol %s", a.Symbol)
	}
}

// A nested function is the opposite trap. Its container exists, but Graph does
// not put it in the qualified name, so the lookup key is the bare name and
// anything that assembled "format_footer.pad" would find nothing.
func TestBindResolvesANestedFunctionByItsBareName(t *testing.T) {
	f := bindField(t)
	got := bindOne(t, f, "pad", "app/report.py", 32)
	if got.Symbol == "" {
		t.Fatalf("the nested function did not bind: %s", got.Unresolved)
	}
	sym := f.Symbols[got.Symbol]
	if sym.QualifiedName != "pad" {
		t.Fatalf("qualified name = %q, want the bare name", sym.QualifiedName)
	}
	if sym.ContainerID == "" {
		t.Fatal("the fixture's nested function is meant to have a container")
	}
	if bad := bindOne(t, f, "format_footer.pad", "app/report.py", 32); bad.Symbol != "" {
		t.Fatalf("a reconstructed qualified name must not resolve, got %s", bad.Symbol)
	}
}

// Module-level functions bound before this change and must still bind.
func TestBindStillResolvesAModuleLevelFunction(t *testing.T) {
	f := bindField(t)
	got := bindOne(t, f, "compute_total", "app/service.py", 20)
	if got.Symbol == "" {
		t.Fatalf("compute_total did not bind: %s", got.Unresolved)
	}
	if got.Unresolved != "" {
		t.Fatalf("a bound source must carry no reason, got %q", got.Unresolved)
	}
}

// A source that cannot be bound says why, so the report can tell "never asked"
// from "asked and found nothing".
func TestBindRecordsWhyASourceDidNotResolve(t *testing.T) {
	f := bindField(t)
	got := bindOne(t, f, "NoSuchClass.no_such_method", "app/models.py", 1)
	if got.Symbol != "" {
		t.Fatalf("an unknown name must not bind, got %s", got.Symbol)
	}
	if got.Unresolved != UnresolvedNoMatch {
		t.Fatalf("reason = %q, want %q", got.Unresolved, UnresolvedNoMatch)
	}
}

// Two symbols of one qualified name in one file, which Graph produces for two
// nested classes sharing a name. The line decides, and where it does not this
// must refuse rather than guess: the old code always answered with the nearest.
func TestBindRefusesAnAmbiguousMatchAndTakesTheLineWhenItDecides(t *testing.T) {
	f := &Field{
		Symbols: map[string]*Symbol{
			"a": {ID: "a", Name: "run", QualifiedName: "Helper.run", File: "collide.py", Span: [2]int{3, 4}},
			"b": {ID: "b", Name: "run", QualifiedName: "Helper.run", File: "collide.py", Span: [2]int{10, 11}},
		},
		ByFile: map[string][]string{"collide.py": {"a", "b"}},
	}
	if got := bindOne(t, f, "Helper.run", "collide.py", 3); got.Symbol != "a" {
		t.Fatalf("line 3 bound to %q, want a", got.Symbol)
	}
	if got := bindOne(t, f, "Helper.run", "collide.py", 10); got.Symbol != "b" {
		t.Fatalf("line 10 bound to %q, want b", got.Symbol)
	}
	got := bindOne(t, f, "Helper.run", "collide.py", 99)
	if got.Symbol != "" {
		t.Fatalf("a line inside neither span bound to %q, want no match", got.Symbol)
	}
	if got.Unresolved != UnresolvedAmbiguous {
		t.Fatalf("reason = %q, want %q", got.Unresolved, UnresolvedAmbiguous)
	}
}

// A snapshot from a build that publishes no qualified_name must still bind
// module-level symbols exactly as it always did.
func TestBindFallsBackToTheBareNameWithoutAQualifiedName(t *testing.T) {
	f := &Field{
		Symbols: map[string]*Symbol{"x": {ID: "x", Name: "helper", File: "a.py", Span: [2]int{1, 2}}},
		ByFile:  map[string][]string{"a.py": {"x"}},
	}
	if got := bindOne(t, f, "helper", "a.py", 1); got.Symbol != "x" {
		t.Fatalf("bound to %q, want x", got.Symbol)
	}
}
