package shadow

import (
	"math"
	"sort"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// TestOutcome is what happened to a test node.
type TestOutcome string

const (
	OutcomeNotRun TestOutcome = "notrun"
	OutcomePass   TestOutcome = "pass"
	OutcomeFail   TestOutcome = "fail"
)

// TestRef is one test that reaches a node.
type TestRef struct {
	ID      string
	Symbol  string
	Outcome TestOutcome
}

// Node is one dependent, classified, ranked and carrying its evidence.
type Node struct {
	Symbol  *graph.Symbol
	Sources []string
	// Relation is the strongest relation to any source.
	Relation string
	Family   graph.Family
	Depth    int
	Path     []string

	State     State
	Tier      Tier
	Modifiers []string
	Exposures []Exposure

	Dependents int
	CallSite   int
	Score      float64
	Factors    map[string]float64
	// Beacon is the annotation comment line when the node is pinned.
	Beacon string

	Tests   []TestRef
	Result  TestOutcome
	Failure string
}

// Pinned reports whether the beacon override applies.
func (n *Node) Pinned() bool { return n.Beacon != "" }

// IsTest reports whether the node is a test function.
func (n *Node) IsTest() bool { return n.Symbol != nil && n.Symbol.IsTest }

// Shadowed reports whether the node is anything other than lit. Unknown counts
// as shadowed because it cannot be shown to have been examined.
func (n *Node) Shadowed() bool { return n.State != Lit }

// relationWeight is factor 1 from PRD.md.
func relationWeight(fam graph.Family, depth int) (float64, string) {
	switch fam {
	case graph.FamilyCalls:
		if depth <= 1 {
			return 3, "direct caller"
		}
		return 1.5, "transitive caller"
	case graph.FamilyDataFlow:
		return 2, "data flow"
	case graph.FamilyTypeUse:
		return 2, "type consumer"
	case graph.FamilyCoChange:
		return 1, "co-change file"
	}
	return 1, "related"
}

// stateWeight is factor 3. Weaker evidence scores higher because a node the
// agent only glimpsed is more suspicious than one it read.
func stateWeight(s State, t Tier) float64 {
	switch s {
	case Lit:
		return 0
	case Umbra, Unknown:
		return 1.0
	case Penumbra:
		switch t {
		case TierAfterimage:
			return 0.6
		case TierEcho:
			return 0.7
		default:
			return 0.5
		}
	}
	return 1.0
}

// RankInput carries the settings the ranker needs.
type RankInput struct {
	// Scars maps a file to the number of recent fix commits, when --history
	// is on. A nil map means the factor is off.
	Scars map[string]int
}

// Score computes the six factors and fills them in on the node so the report
// can print every one and a reader can recompute the total by hand.
//
// The factors, in the order PRD.md lists them:
//
//  1. relation weight
//  2. multiplied by 1 + log2(1 + dependents)
//  3. multiplied by the state weight
//  4. multiplied by 1.5 in the far field
//  5. multiplied by 1.5 on a fault line
//  6. plus 3 for a test function, plus 2 when a test reaches the node
//
// The opt-in scar factor multiplies by 1 + min(fixes, 5) / 5.
func Score(n *Node, in RankInput) {
	f := map[string]float64{}

	rel, relName := relationWeight(n.Family, n.Depth)
	f["relation"] = rel
	n.Relation = relName

	dep := 1 + math.Log2(1+float64(n.Dependents))
	f["dependents"] = round3(dep)

	st := stateWeight(n.State, n.Tier)
	f["state"] = st

	far := 1.0
	if hasModifier(n, "far field") {
		far = 1.5
	}
	f["farfield"] = far

	fault := 1.0
	if hasModifier(n, "fault line") {
		fault = 1.5
	}
	f["faultline"] = fault

	testBonus := 0.0
	if n.IsTest() {
		testBonus += 3
	}
	if len(n.Tests) > 0 {
		testBonus += 2
	}
	f["test"] = testBonus

	score := rel * dep * st * far * fault

	if in.Scars != nil {
		fixes := in.Scars[n.Symbol.File]
		if fixes > 5 {
			fixes = 5
		}
		scar := 1 + float64(fixes)/5
		f["scar"] = round3(scar)
		score *= scar
		if fixes > 0 {
			addModifier(n, "scar")
		}
	}

	n.Score = round3(score + testBonus)
	n.Factors = f
}

func hasModifier(n *Node, m string) bool {
	for _, x := range n.Modifiers {
		if x == m {
			return true
		}
	}
	return false
}

func addModifier(n *Node, m string) {
	if !hasModifier(n, m) {
		n.Modifiers = append(n.Modifiers, m)
	}
}

func round3(v float64) float64 { return math.Round(v*1000) / 1000 }

// SortDocket orders the shadowed nodes for the docket and the table.
//
// Beacons come first, because an annotated call site is a hand-written note
// that this code matters. Everything else sorts by score. Ties break on file
// path then symbol name, so two runs on the same checkpoint produce the same
// order.
func SortDocket(nodes []*Node) {
	sort.SliceStable(nodes, func(i, j int) bool {
		a, b := nodes[i], nodes[j]
		if a.Pinned() != b.Pinned() {
			return a.Pinned()
		}
		if a.Score != b.Score {
			return a.Score > b.Score
		}
		if a.Symbol.File != b.Symbol.File {
			return a.Symbol.File < b.Symbol.File
		}
		return a.Symbol.Name < b.Symbol.Name
	})
}

// Summary counts the states for the header.
type Summary struct {
	Lit      int
	Penumbra int
	Umbra    int
	Unknown  int
}

// Illumination is lit over the nodes that could be judged. It is undefined
// when everything is unknown, which the caller reports rather than printing a
// misleading zero.
func (s Summary) Illumination() (float64, bool) {
	total := s.Lit + s.Penumbra + s.Umbra
	if total == 0 {
		return 0, false
	}
	return float64(s.Lit) / float64(total), true
}

// Summarize counts states across nodes.
func Summarize(nodes []*Node) Summary {
	var s Summary
	for _, n := range nodes {
		switch n.State {
		case Lit:
			s.Lit++
		case Penumbra:
			s.Penumbra++
		case Umbra:
			s.Umbra++
		default:
			s.Unknown++
		}
	}
	return s
}
