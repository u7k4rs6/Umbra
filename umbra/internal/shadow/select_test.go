package shadow

import (
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// Test ids are built from graph data and reach a process as argv. The
// validation is defence in depth, because Umbra never builds a shell string,
// but an id that cannot be a test id is dropped loudly rather than passed on.
func TestValidTestID(t *testing.T) {
	valid := []string{
		"tests/test_service.py::test_rounding",
		"tests/test_api.py::TestOrders::test_total",
		"TestCompute",
		"tests/a-b_c.py::test_x[1]",
	}
	for _, id := range valid {
		if !ValidTestID(id) {
			t.Errorf("ValidTestID(%q) = false, want true", id)
		}
	}
	invalid := []string{
		"",
		"tests/a.py::test x",
		"tests/a.py::test;rm -rf /",
		"tests/a.py::test$(whoami)",
		"tests/a.py::test`id`",
		"tests/a.py::test&&echo",
		"tests/a.py::test|cat",
		"tests/a.py::test'quote",
		`tests/a.py::test"quote`,
		"tests/a.py::test\nnewline",
	}
	for _, id := range invalid {
		if ValidTestID(id) {
			t.Errorf("ValidTestID(%q) = true, want false", id)
		}
	}
}

func TestTestIDForPytest(t *testing.T) {
	sym := &graph.Symbol{Name: "test_rounding", File: "umbra/fixtures/app/tests/test_service.py", Kind: "function", IsTest: true}
	id, ok := TestIDFor(sym, "umbra/fixtures/app")
	if !ok {
		t.Fatal("expected a pytest id")
	}
	if id != "tests/test_service.py::test_rounding" {
		t.Fatalf("id = %q", id)
	}
}

func TestTestIDForGo(t *testing.T) {
	sym := &graph.Symbol{Name: "TestRounding", File: "internal/x/thing_test.go", Kind: "function", IsTest: true}
	id, ok := TestIDFor(sym, "")
	if !ok {
		t.Fatal("expected a go test name")
	}
	if id != "TestRounding" {
		t.Fatalf("id = %q", id)
	}
}

func TestTestIDForRejectsNonTest(t *testing.T) {
	sym := &graph.Symbol{Name: "helper", File: "tests/test_x.py", Kind: "function", IsTest: false}
	if _, ok := TestIDFor(sym, ""); ok {
		t.Fatal("a symbol that is not a test has no test id")
	}
	if _, ok := TestIDFor(nil, ""); ok {
		t.Fatal("nil has no test id")
	}
}

func TestTestIDForRejectsHostileName(t *testing.T) {
	sym := &graph.Symbol{Name: "test x; rm -rf /", File: "tests/test_x.py", Kind: "function", IsTest: true}
	if _, ok := TestIDFor(sym, ""); ok {
		t.Fatal("an id that fails validation must be dropped")
	}
}

// Selection order: shadowed tests first, then tests reaching shadow, then the
// rest. --run shadow keeps the first two groups.
func TestSelectTestsOrdersAndFilters(t *testing.T) {
	f := &graph.Field{
		Symbols: map[string]*graph.Symbol{},
		Out:     map[string][]graph.Edge{},
		In:      map[string][]graph.Edge{},
		ByFile:  map[string][]string{},
	}
	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS"}})

	shadowedTest := node("test_shadow", "tests/test_a.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	shadowedTest.Score = 9
	litTest := node("test_lit", "tests/test_b.py", graph.FamilyCalls, 1, Lit, TierNone, true)
	notATest := node("helper", "app/x.py", graph.FamilyCalls, 1, Umbra, TierNone, false)

	for _, n := range []*Node{shadowedTest, litTest, notATest} {
		f.Symbols[n.Symbol.ID] = n.Symbol
	}

	sel, notRunnable := SelectTests([]*Node{shadowedTest, litTest, notATest}, f, rm, nil, "shadow", "")
	if len(notRunnable) != 0 {
		t.Fatalf("notRunnable = %v", notRunnable)
	}
	// The lit test reaches nothing shadowed, so --run shadow drops it.
	if len(sel) != 1 {
		t.Fatalf("selected = %v, want only the shadowed test", IDs(sel))
	}
	if sel[0].ID != "tests/test_a.py::test_shadow" {
		t.Fatalf("selected = %q", sel[0].ID)
	}

	all, _ := SelectTests([]*Node{shadowedTest, litTest, notATest}, f, rm, nil, "all", "")
	if len(all) != 2 {
		t.Fatalf("--run all should keep both tests, got %v", IDs(all))
	}
	if all[0].ID != "tests/test_a.py::test_shadow" {
		t.Fatalf("the shadowed test must come first, got %v", IDs(all))
	}
}

func TestSelectTestsNoneSelectsNothing(t *testing.T) {
	n := node("test_a", "tests/test_a.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	sel, _ := SelectTests([]*Node{n}, nil, nil, nil, "none", "")
	if len(sel) != 0 {
		t.Fatalf("--run none must select nothing, got %v", IDs(sel))
	}
}

func TestSelectTestsReportsNotRunnable(t *testing.T) {
	n := node("test bad name", "tests/test_a.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	f := &graph.Field{Symbols: map[string]*graph.Symbol{n.Symbol.ID: n.Symbol}}
	rm := graph.NewRelationMap(&graph.Capabilities{SupportedRelationTypes: []string{"CALLS"}})

	sel, notRunnable := SelectTests([]*Node{n}, f, rm, nil, "shadow", "")
	if len(sel) != 0 {
		t.Fatalf("a hostile id must not be selected, got %v", IDs(sel))
	}
	if len(notRunnable) != 1 {
		t.Fatalf("notRunnable = %v, want the dropped id listed", notRunnable)
	}
}
