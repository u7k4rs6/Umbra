package shadow

import (
	"math"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

func node(name, file string, fam graph.Family, depth int, state State, tier Tier, isTest bool) *Node {
	return &Node{
		Symbol: &graph.Symbol{ID: file + ":" + name, Name: name, File: file, Kind: "function", IsTest: isTest},
		Family: fam, Depth: depth, State: state, Tier: tier,
	}
}

func TestRelationWeights(t *testing.T) {
	cases := []struct {
		fam   graph.Family
		depth int
		want  float64
		name  string
	}{
		{graph.FamilyCalls, 1, 3, "direct caller"},
		{graph.FamilyCalls, 2, 1.5, "transitive caller"},
		{graph.FamilyDataFlow, 1, 2, "data flow"},
		{graph.FamilyTypeUse, 1, 2, "type consumer"},
		{graph.FamilyCoChange, 1, 1, "co-change file"},
	}
	for _, c := range cases {
		got, name := relationWeight(c.fam, c.depth)
		if got != c.want {
			t.Errorf("relationWeight(%v, %d) = %v, want %v", c.fam, c.depth, got, c.want)
		}
		if name != c.name {
			t.Errorf("relation name = %q, want %q", name, c.name)
		}
	}
}

// Weaker evidence scores higher: a node the agent only glimpsed is more
// suspicious than one it read.
func TestStateWeights(t *testing.T) {
	cases := []struct {
		state State
		tier  Tier
		want  float64
	}{
		{Lit, TierNone, 0},
		{Umbra, TierNone, 1.0},
		{Unknown, TierNone, 1.0},
		{Penumbra, TierGlance, 0.5},
		{Penumbra, TierGlimpse, 0.5},
		{Penumbra, TierQuoted, 0.5},
		{Penumbra, TierAfterimage, 0.6},
		{Penumbra, TierEcho, 0.7},
	}
	for _, c := range cases {
		if got := stateWeight(c.state, c.tier); got != c.want {
			t.Errorf("stateWeight(%v, %v) = %v, want %v", c.state, c.tier, got, c.want)
		}
	}
}

// A reader must be able to recompute the total by hand from the printed
// factors, so this checks the arithmetic end to end.
func TestScoreIsTheProductOfItsPrintedFactors(t *testing.T) {
	n := node("test_rounding", "tests/test_service.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	n.Dependents = 0
	addModifier(n, "far field")
	Score(n, RankInput{})

	f := n.Factors
	for _, key := range []string{"relation", "dependents", "state", "farfield", "faultline", "test"} {
		if _, ok := f[key]; !ok {
			t.Fatalf("factor %q is missing; every factor must be printed", key)
		}
	}
	want := f["relation"]*f["dependents"]*f["state"]*f["farfield"]*f["faultline"] + f["test"]
	if math.Abs(n.Score-round3(want)) > 1e-9 {
		t.Fatalf("score = %v, but the factors multiply to %v", n.Score, want)
	}
	// 3 * 1 * 1.0 * 1.5 * 1.0 + 3 = 7.5
	if n.Score != 7.5 {
		t.Fatalf("score = %v, want 7.5", n.Score)
	}
}

func TestScoreDependentsFactorUsesLog2(t *testing.T) {
	n := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	n.Dependents = 3
	Score(n, RankInput{})
	// 1 + log2(1 + 3) = 3
	if n.Factors["dependents"] != 3 {
		t.Fatalf("dependents factor = %v, want 3", n.Factors["dependents"])
	}
}

func TestScoreFaultLineMultiplies(t *testing.T) {
	plain := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(plain, RankInput{})

	fault := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	addModifier(fault, "fault line")
	Score(fault, RankInput{})

	if fault.Score != round3(plain.Score*1.5) {
		t.Fatalf("fault line score = %v, want %v", fault.Score, plain.Score*1.5)
	}
}

func TestScoreTestBonuses(t *testing.T) {
	plain := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(plain, RankInput{})

	isTest := node("test_f", "tests/test_a.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	Score(isTest, RankInput{})
	if isTest.Factors["test"] != 3 {
		t.Fatalf("a test function should add 3, got %v", isTest.Factors["test"])
	}

	reached := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	reached.Tests = []TestRef{{ID: "tests/test_a.py::test_f"}}
	Score(reached, RankInput{})
	if reached.Factors["test"] != 2 {
		t.Fatalf("a node a test reaches should add 2, got %v", reached.Factors["test"])
	}

	both := node("test_f", "tests/test_a.py", graph.FamilyCalls, 1, Umbra, TierNone, true)
	both.Tests = []TestRef{{ID: "x"}}
	Score(both, RankInput{})
	if both.Factors["test"] != 5 {
		t.Fatalf("both bonuses should add 5, got %v", both.Factors["test"])
	}
}

// The scar factor is off unless --history is on, and printed like the rest
// when it is on.
func TestScarFactorIsOptIn(t *testing.T) {
	off := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(off, RankInput{})
	if _, ok := off.Factors["scar"]; ok {
		t.Fatal("the scar factor must not appear when --history is off")
	}

	on := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(on, RankInput{Scars: map[string]int{"app/a.py": 3}})
	if on.Factors["scar"] != 1.6 {
		t.Fatalf("scar factor = %v, want 1 + 3/5", on.Factors["scar"])
	}
	if !hasModifier(on, "scar") {
		t.Fatal("a scarred file should carry the scar modifier")
	}
}

func TestScarFactorCapsAtFive(t *testing.T) {
	n := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(n, RankInput{Scars: map[string]int{"app/a.py": 40}})
	if n.Factors["scar"] != 2 {
		t.Fatalf("scar factor = %v, want it capped at 2", n.Factors["scar"])
	}
}

func TestScarFactorAbsentModifierWhenNoFixes(t *testing.T) {
	n := node("f", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	Score(n, RankInput{Scars: map[string]int{"app/a.py": 0}})
	if hasModifier(n, "scar") {
		t.Fatal("a file with no fixes should not be marked scarred")
	}
	if n.Factors["scar"] != 1 {
		t.Fatalf("scar factor = %v, want 1", n.Factors["scar"])
	}
}

// A lit node scores zero from the state factor, so it never crowds the docket.
func TestLitNodeScoresZeroBeforeTestBonuses(t *testing.T) {
	n := node("f", "app/a.py", graph.FamilyCalls, 1, Lit, TierNone, false)
	Score(n, RankInput{})
	if n.Score != 0 {
		t.Fatalf("score = %v, want 0 for a lit node with no test bonus", n.Score)
	}
}

// The beacon override pins an annotated call site to the top regardless of
// score.
func TestBeaconOverridePinsToTheTop(t *testing.T) {
	high := node("busy", "app/a.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
	high.Dependents = 30
	Score(high, RankInput{})

	pinned := node("guarded", "app/b.py", graph.FamilyCalls, 2, Penumbra, TierEcho, false)
	Score(pinned, RankInput{})
	pinned.Beacon = "# SAFETY: a pricing failure must not refund the full amount"

	if pinned.Score >= high.Score {
		t.Fatalf("the test needs the pinned node to score lower: %v vs %v", pinned.Score, high.Score)
	}

	nodes := []*Node{high, pinned}
	SortDocket(nodes)
	if nodes[0] != pinned {
		t.Fatal("a beacon must sort first even with a lower score")
	}
	if !pinned.Pinned() {
		t.Fatal("Pinned should report true")
	}
}

// Two runs on the same checkpoint must produce the same order.
func TestSortDocketIsDeterministic(t *testing.T) {
	mk := func() []*Node {
		a := node("b_same", "app/same.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
		b := node("a_same", "app/same.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
		c := node("c_other", "app/aaa.py", graph.FamilyCalls, 1, Umbra, TierNone, false)
		for _, n := range []*Node{a, b, c} {
			Score(n, RankInput{})
		}
		return []*Node{a, b, c}
	}
	first := mk()
	SortDocket(first)
	want := []string{"app/aaa.py", "app/same.py", "app/same.py"}
	gotNames := []string{"c_other", "a_same", "b_same"}
	for i := range first {
		if first[i].Symbol.File != want[i] || first[i].Symbol.Name != gotNames[i] {
			t.Fatalf("order[%d] = %s %s, want %s %s", i, first[i].Symbol.File, first[i].Symbol.Name, want[i], gotNames[i])
		}
	}
	for run := 0; run < 5; run++ {
		again := mk()
		SortDocket(again)
		for i := range again {
			if again[i].Symbol.Name != first[i].Symbol.Name {
				t.Fatalf("run %d differs at %d", run, i)
			}
		}
	}
}

func TestSummarizeAndIllumination(t *testing.T) {
	nodes := []*Node{
		node("a", "f.py", graph.FamilyCalls, 1, Lit, TierNone, false),
		node("b", "f.py", graph.FamilyCalls, 1, Lit, TierNone, false),
		node("c", "f.py", graph.FamilyCalls, 1, Lit, TierNone, false),
		node("d", "f.py", graph.FamilyCalls, 1, Penumbra, TierGlance, false),
		node("e", "f.py", graph.FamilyCalls, 1, Umbra, TierNone, false),
		node("f", "f.py", graph.FamilyCalls, 1, Umbra, TierNone, false),
		node("g", "f.py", graph.FamilyCalls, 1, Umbra, TierNone, false),
	}
	s := Summarize(nodes)
	if s.Lit != 3 || s.Penumbra != 1 || s.Umbra != 3 || s.Unknown != 0 {
		t.Fatalf("summary = %+v", s)
	}
	frac, ok := s.Illumination()
	if !ok {
		t.Fatal("illumination should be defined")
	}
	if math.Abs(frac-3.0/7.0) > 1e-9 {
		t.Fatalf("illumination = %v, want 3/7", frac)
	}
}

// When everything is unknown the fraction is omitted rather than printed as a
// misleading zero.
func TestIlluminationUndefinedWhenAllUnknown(t *testing.T) {
	s := Summarize([]*Node{node("a", "f.py", graph.FamilyCalls, 1, Unknown, TierNone, false)})
	if _, ok := s.Illumination(); ok {
		t.Fatal("illumination must be undefined when every node is unknown")
	}
}

func TestShadowedCountsUnknown(t *testing.T) {
	if !node("a", "f.py", graph.FamilyCalls, 1, Unknown, TierNone, false).Shadowed() {
		t.Fatal("unknown counts as shadowed because it cannot be shown to be examined")
	}
	if node("a", "f.py", graph.FamilyCalls, 1, Lit, TierNone, false).Shadowed() {
		t.Fatal("lit is not shadowed")
	}
}
