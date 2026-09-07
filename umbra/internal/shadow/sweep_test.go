package shadow

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// buildChain makes a call chain test -> mid... -> source so the forensics
// reasons can be exercised at a chosen depth.
func buildChain(depth int) (*graph.Field, *graph.RelationMap, string, string) {
	f := &graph.Field{
		Symbols: map[string]*graph.Symbol{},
		Out:     map[string][]graph.Edge{},
		In:      map[string][]graph.Edge{},
		ByFile:  map[string][]string{},
	}
	add := func(id, name, file string, isTest bool) {
		f.Symbols[id] = &graph.Symbol{ID: id, Name: name, File: file, Kind: "function", IsTest: isTest}
		f.ByFile[file] = append(f.ByFile[file], id)
	}
	link := func(from, to string) {
		e := graph.Edge{From: from, To: to, Relation: "CALLS"}
		f.Out[from] = append(f.Out[from], e)
		f.In[to] = append(f.In[to], e)
	}

	add("t", "test_receipt_total", "tests/test_report.py", true)
	prev := "t"
	for i := 1; i < depth; i++ {
		id := "mid" + string(rune('0'+i))
		add(id, "render_lines"+string(rune('0'+i)), "app/report.py", false)
		link(prev, id)
		prev = id
	}
	add("src", "compute_total", "app/service.py", false)
	link(prev, "src")

	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS", "DATA_FLOWS", "USES_TYPE"}})
	return f, rm, "t", "src"
}

// Reason 1: a path exists, but deeper than the selection searched.
func TestForensicsPathBeyondDepth(t *testing.T) {
	f, rm, _, src := buildChain(4)
	leaks := Forensics([]string{"tests/test_report.py::test_receipt_total"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{src}, MaxDepth: 2,
		Selected: map[string]bool{},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v", leaks)
	}
	if !strings.HasPrefix(leaks[0].Reason, "path exists at depth 4 via ") {
		t.Fatalf("reason = %q, want the depth and the first hop", leaks[0].Reason)
	}
}

// Reason 2: the test has no resolved outgoing calls at all.
func TestForensicsNoEdgeFromTheTestFile(t *testing.T) {
	f := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"t":   {ID: "t", Name: "test_dynamic", File: "tests/test_dyn.py", Kind: "function", IsTest: true},
			"src": {ID: "src", Name: "compute_total", File: "app/service.py", Kind: "function"},
		},
		Out:    map[string][]graph.Edge{},
		In:     map[string][]graph.Edge{},
		ByFile: map[string][]string{"tests/test_dyn.py": {"t"}, "app/service.py": {"src"}},
	}
	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS"}})

	leaks := Forensics([]string{"tests/test_dyn.py::test_dynamic"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{"src"}, MaxDepth: 2, Selected: map[string]bool{},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v", leaks)
	}
	if leaks[0].Reason != "no edge from the test file (dynamic dispatch or unparsed call)" {
		t.Fatalf("reason = %q", leaks[0].Reason)
	}
}

// Reason 3: the only path runs over a relation family the walk does not follow.
func TestForensicsReachesOnlyThroughAnotherFamily(t *testing.T) {
	f := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"t":   {ID: "t", Name: "test_typed", File: "tests/test_t.py", Kind: "function", IsTest: true},
			"mid": {ID: "mid", Name: "Holder", File: "app/models.py", Kind: "class"},
			"src": {ID: "src", Name: "compute_total", File: "app/service.py", Kind: "function"},
		},
		Out:    map[string][]graph.Edge{},
		In:     map[string][]graph.Edge{},
		ByFile: map[string][]string{"tests/test_t.py": {"t"}, "app/models.py": {"mid"}, "app/service.py": {"src"}},
	}
	link := func(from, to, rel string) {
		e := graph.Edge{From: from, To: to, Relation: rel}
		f.Out[from] = append(f.Out[from], e)
		f.In[to] = append(f.In[to], e)
	}
	// The test has a call edge, so reason 2 does not apply, but the only way
	// to the source is a type relation.
	link("t", "mid", "CALLS")
	link("mid", "src", "USES_TYPE")

	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS", "USES_TYPE"}})
	leaks := Forensics([]string{"tests/test_t.py::test_typed"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{"src"}, MaxDepth: 2, Selected: map[string]bool{},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v", leaks)
	}
	if !strings.HasPrefix(leaks[0].Reason, "reaches only through ") {
		t.Fatalf("reason = %q", leaks[0].Reason)
	}
}

// Reason 4: outside everything Umbra can see.
func TestForensicsOutsideTheCoChangeSet(t *testing.T) {
	f := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"t":    {ID: "t", Name: "test_far", File: "tests/test_far.py", Kind: "function", IsTest: true},
			"othr": {ID: "othr", Name: "unrelated", File: "app/other.py", Kind: "function"},
			"src":  {ID: "src", Name: "compute_total", File: "app/service.py", Kind: "function"},
		},
		Out:    map[string][]graph.Edge{},
		In:     map[string][]graph.Edge{},
		ByFile: map[string][]string{"tests/test_far.py": {"t"}, "app/other.py": {"othr"}, "app/service.py": {"src"}},
	}
	e := graph.Edge{From: "t", To: "othr", Relation: "CALLS"}
	f.Out["t"] = append(f.Out["t"], e)
	f.In["othr"] = append(f.In["othr"], e)

	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS"}})
	leaks := Forensics([]string{"tests/test_far.py::test_far"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{"src"}, MaxDepth: 2,
		Selected: map[string]bool{}, CoChangeFiles: map[string]bool{"app/api.py": true},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v", leaks)
	}
	if leaks[0].Reason != "test file not in the co-change set" {
		t.Fatalf("reason = %q", leaks[0].Reason)
	}
}

// A test that was selected is not a leak, however it changed state.
func TestForensicsSkipsSelectedTests(t *testing.T) {
	f, rm, _, src := buildChain(4)
	leaks := Forensics([]string{"tests/test_report.py::test_receipt_total"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{src}, MaxDepth: 2,
		Selected: map[string]bool{"tests/test_report.py::test_receipt_total": true},
	})
	if len(leaks) != 0 {
		t.Fatalf("a selected test is not a leak, got %v", leaks)
	}
}

func TestForensicsUnknownTestID(t *testing.T) {
	f, rm, _, src := buildChain(2)
	leaks := Forensics([]string{"tests/nowhere.py::test_ghost"}, ForensicsInput{
		Field: f, RelMap: rm, SourceIDs: []string{src}, MaxDepth: 2, Selected: map[string]bool{},
	})
	if len(leaks) != 1 {
		t.Fatalf("leaks = %v", leaks)
	}
	if !strings.Contains(leaks[0].Reason, "no symbol in the graph") {
		t.Fatalf("reason = %q", leaks[0].Reason)
	}
}

func TestForensicsIsSortedForDeterminism(t *testing.T) {
	f, rm, _, src := buildChain(4)
	in := ForensicsInput{Field: f, RelMap: rm, SourceIDs: []string{src}, MaxDepth: 2, Selected: map[string]bool{}}
	changed := []string{"z::z", "a::a", "tests/test_report.py::test_receipt_total"}
	first := Forensics(changed, in)
	for i := 0; i < 3; i++ {
		again := Forensics(changed, in)
		for j := range first {
			if again[j].Test != first[j].Test {
				t.Fatalf("run %d differs at %d", i, j)
			}
		}
	}
	if first[0].Test != "a::a" {
		t.Fatalf("leaks should be sorted by test id, got %v", first)
	}
}

func TestSplitTestID(t *testing.T) {
	cases := []struct{ id, file, name string }{
		{"tests/a.py::test_one", "tests/a.py", "test_one"},
		{"tests/a.py::TestClass::test_one", "tests/a.py", "test_one"},
		{"TestGoName", "", "TestGoName"},
	}
	for _, c := range cases {
		file, name := splitTestID(c.id)
		if file != c.file || name != c.name {
			t.Errorf("splitTestID(%q) = %q, %q; want %q, %q", c.id, file, name, c.file, c.name)
		}
	}
}

// A test that makes calls, every one of which leaves the repository, used to
// fall through to reason 4 and be reported as not being in the co-change set.
// That describes the wrong thing: the calls are there and none can be followed.
// On pallets/click this was the common case, not the exception.
func TestLeakReasonNamesCallsThatLeaveTheRepository(t *testing.T) {
	external := graph.EdgeQuality{
		Confidence: 0.78, Resolution: "import_external",
		Scope: graph.ScopeExternal, TargetKind: graph.TargetKindExternal,
	}
	field := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"src":  {ID: "src", Name: "compute_total", File: "app/service.py", Span: [2]int{1, 5}},
			"test": {ID: "test", Name: "test_thing", File: "tests/test_thing.py", Span: [2]int{1, 4}, IsTest: true},
		},
		ByFile: map[string][]string{
			"app/service.py":      {"src"},
			"tests/test_thing.py": {"test"},
		},
		In: map[string][]graph.Edge{},
		Out: map[string][]graph.Edge{
			"test": {
				{From: "test", To: "external:symbol:click.command", Relation: "CALLS", Quality: external},
				{From: "test", To: "external:symbol:click.echo", Relation: "CALLS", Quality: external},
			},
		},
	}
	leaks := Forensics([]string{"tests/test_thing.py::test_thing"}, ForensicsInput{
		Field: field, RelMap: graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS", "DATA_FLOWS", "USES_TYPE"}}), SourceIDs: []string{"src"},
		Selected: map[string]bool{}, MaxDepth: 2,
	})
	if len(leaks) != 1 {
		t.Fatalf("expected one leak, got %d", len(leaks))
	}
	got := leaks[0].Reason
	if !strings.Contains(got, "leave the repository") {
		t.Fatalf("reason = %q, want it to say the calls leave the repository", got)
	}
	if !strings.Contains(got, "2 call") {
		t.Fatalf("reason = %q, want the count of unfollowable calls", got)
	}
	if strings.Contains(got, "co-change") {
		t.Fatalf("reason = %q, which is the sentence this replaces", got)
	}
}

// A test with a same-repo call keeps whatever reason it had. This is the
// negative half: the new reason must not swallow the existing ones.
func TestLeakReasonUnchangedWhenACallStaysInTheRepository(t *testing.T) {
	external := graph.EdgeQuality{Scope: graph.ScopeExternal, TargetKind: graph.TargetKindExternal}
	inside := graph.EdgeQuality{Scope: "file", TargetKind: graph.TargetKindSymbol, Confidence: 0.92}
	field := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"src":   {ID: "src", Name: "compute_total", File: "app/service.py", Span: [2]int{1, 5}},
			"other": {ID: "other", Name: "helper", File: "app/other.py", Span: [2]int{1, 3}},
			"test":  {ID: "test", Name: "test_thing", File: "tests/test_thing.py", Span: [2]int{1, 4}, IsTest: true},
		},
		ByFile: map[string][]string{
			"app/service.py":      {"src"},
			"app/other.py":        {"other"},
			"tests/test_thing.py": {"test"},
		},
		In: map[string][]graph.Edge{},
		Out: map[string][]graph.Edge{
			"test": {
				{From: "test", To: "external:symbol:click.echo", Relation: "CALLS", Quality: external},
				{From: "test", To: "other", Relation: "CALLS", Quality: inside},
			},
		},
	}
	leaks := Forensics([]string{"tests/test_thing.py::test_thing"}, ForensicsInput{
		Field: field, RelMap: graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS", "DATA_FLOWS", "USES_TYPE"}}), SourceIDs: []string{"src"},
		Selected: map[string]bool{}, MaxDepth: 2,
	})
	if len(leaks) != 1 {
		t.Fatalf("expected one leak, got %d", len(leaks))
	}
	if strings.Contains(leaks[0].Reason, "leave the repository") {
		t.Fatalf("reason = %q, but one call stays in the repository", leaks[0].Reason)
	}
}

// The new reason sits after the structural check. A test that genuinely does
// reach the change over a relation the selection does not traverse has a path,
// and naming the path is more useful than naming the calls that failed.
func TestStructuralPathOutranksTheCallsLeaveRepoReason(t *testing.T) {
	external := graph.EdgeQuality{Scope: graph.ScopeExternal, TargetKind: graph.TargetKindExternal}
	structural := graph.EdgeQuality{Scope: "file", TargetKind: graph.TargetKindSymbol, Confidence: 0.9}
	field := &graph.Field{
		Symbols: map[string]*graph.Symbol{
			"src":  {ID: "src", Name: "compute_total", File: "app/service.py", Span: [2]int{1, 5}},
			"test": {ID: "test", Name: "test_thing", File: "tests/test_thing.py", Span: [2]int{1, 4}, IsTest: true},
		},
		ByFile: map[string][]string{"app/service.py": {"src"}, "tests/test_thing.py": {"test"}},
		In:     map[string][]graph.Edge{},
		Out: map[string][]graph.Edge{
			"test": {
				// Every CALL leaves the repository ...
				{From: "test", To: "external:symbol:click.echo", Relation: "CALLS", Quality: external},
				// ... but a type edge really does reach the source.
				{From: "test", To: "src", Relation: "USES_TYPE", Quality: structural},
			},
		},
	}
	leaks := Forensics([]string{"tests/test_thing.py::test_thing"}, ForensicsInput{
		Field:     field,
		RelMap:    graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS", "DATA_FLOWS", "USES_TYPE"}}),
		SourceIDs: []string{"src"}, Selected: map[string]bool{}, MaxDepth: 2,
	})
	if len(leaks) != 1 {
		t.Fatalf("expected one leak, got %d", len(leaks))
	}
	if strings.Contains(leaks[0].Reason, "leave the repository") {
		t.Fatalf("reason = %q, but a real structural path exists and should be named", leaks[0].Reason)
	}
	if !strings.Contains(leaks[0].Reason, "reaches only through") {
		t.Fatalf("reason = %q, want the structural path", leaks[0].Reason)
	}
}
