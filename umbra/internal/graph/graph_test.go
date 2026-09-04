package graph

import (
	"os"
	"path/filepath"
	"testing"
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
	if s.Span[0] != 20 {
		t.Fatalf("span = %v, want it to start at 20", s.Span)
	}
	if s.Span[1] <= s.Span[0] {
		t.Fatalf("span = %v, want a real range", s.Span)
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
		"tests/test_service.py":  true,
		"app/test_helpers.py":    true,
		"pkg/thing_test.go":      true,
		"src/thing_test.py":      true,
		"web/button.test.tsx":    true,
		"web/button.spec.ts":     true,
		"tests/helpers.py":       true,
		"app/service.py":         false,
		"app/latest.py":          false,
		"":                       false,
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
