package shadow

import (
	"fmt"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// LeakReason explains why the selection missed a test the full sweep found.
// The reasons are computed in the order ARCHITECTURE.md section 6 sets, and
// the first that applies is the one reported.
type LeakReason struct {
	Test   string
	Reason string
}

// ForensicsInput is what the sweep needs to explain itself.
type ForensicsInput struct {
	Field     *graph.Field
	RelMap    *graph.RelationMap
	SourceIDs []string
	// SourceFiles is used for the co-change check.
	CoChangeFiles map[string]bool
	// Selected holds the runner ids that were chosen.
	Selected map[string]bool
	// MaxDepth is the depth the selection searched, so a path found beyond it
	// can be named as such.
	MaxDepth    int
	RepoRelRoot string
}

// Forensics turns the tests the sweep found into leaks with reasons.
//
// A test that changed state and was selected is not a leak. Everything else
// gets the first reason that applies:
//
//  1. a path exists beyond the depth the selection searched
//  2. the test has no resolved outgoing call edges at all
//  3. a path exists only over a relation family Umbra does not traverse
//  4. no path and no co-change link
func Forensics(changed []string, in ForensicsInput) []LeakReason {
	sources := map[string]bool{}
	for _, id := range in.SourceIDs {
		sources[id] = true
	}

	var out []LeakReason
	for _, test := range changed {
		if in.Selected[test] {
			continue
		}
		out = append(out, LeakReason{Test: test, Reason: reasonFor(test, in, sources)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Test < out[j].Test })
	return out
}

func reasonFor(testID string, in ForensicsInput, sources map[string]bool) string {
	sym := findTestSymbol(in.Field, testID, in.RepoRelRoot)
	if sym == nil {
		return "no symbol in the graph matches this test id, so nothing could have selected it"
	}

	// 1. A path exists, but deeper than the selection looked.
	if depth, via, ok := in.Field.PathTo(sym.ID, sources, in.RelMap, 8); ok {
		if depth > in.MaxDepth {
			return fmt.Sprintf("path exists at depth %d via %s", depth, via)
		}
		// A path within the depth means selection should have found it. Say
		// so plainly rather than inventing a reason.
		return fmt.Sprintf("path exists at depth %d via %s but the test was not selected", depth, via)
	}

	// 2. No resolved calls at all: dynamic dispatch, a fixture, or a call
	// Graph could not resolve.
	if !in.Field.HasOutgoingCalls(sym.ID, in.RelMap) {
		return "no edge from the test file (dynamic dispatch or unparsed call)"
	}

	// 3. A path exists over a family Umbra does not traverse.
	if fam, ok := reachesOverOtherFamily(in.Field, in.RelMap, sym.ID, sources); ok {
		return "reaches only through " + string(fam)
	}

	// 4. Outside everything Umbra can see.
	if in.CoChangeFiles != nil && !in.CoChangeFiles[sym.File] {
		return "test file not in the co-change set"
	}
	return "no call path within the searched depth and no co-change link"
}

// reachesOverOtherFamily looks for a path that uses a relation family the
// selection does not follow, so the leak can name what it travelled over.
func reachesOverOtherFamily(field *graph.Field, relMap *graph.RelationMap, from string, targets map[string]bool) (graph.Family, bool) {
	seen := map[string]bool{from: true}
	type item struct {
		id  string
		fam graph.Family
	}
	queue := []item{{id: from}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		edges := append([]graph.Edge(nil), field.Out[cur.id]...)
		sort.Slice(edges, func(i, j int) bool { return edges[i].To < edges[j].To })
		for _, e := range edges {
			fam, ok := relMap.Family(e.Relation)
			if !ok || seen[e.To] {
				continue
			}
			seen[e.To] = true
			carried := cur.fam
			if fam != graph.FamilyCalls && carried == "" {
				carried = fam
			}
			if targets[e.To] && carried != "" {
				return carried, true
			}
			queue = append(queue, item{id: e.To, fam: carried})
		}
	}
	return "", false
}

// findTestSymbol maps a runner id back to the symbol it names.
func findTestSymbol(field *graph.Field, testID, repoRelRoot string) *graph.Symbol {
	file, name := splitTestID(testID)
	if name == "" {
		return nil
	}
	candidates := []string{file}
	if repoRelRoot != "" && file != "" {
		candidates = append(candidates, strings.TrimSuffix(repoRelRoot, "/")+"/"+file)
	}
	for _, f := range candidates {
		if f == "" {
			continue
		}
		for _, id := range field.ByFile[f] {
			if field.Symbols[id].Name == name {
				return field.Symbols[id]
			}
		}
	}
	// Fall back to a unique name match anywhere, which covers go test ids that
	// carry no path.
	var hit *graph.Symbol
	n := 0
	for _, s := range field.Symbols {
		if s.Name == name && s.IsTest {
			hit = s
			n++
		}
	}
	if n == 1 {
		return hit
	}
	return nil
}

// splitTestID takes path::Class::name or path::name apart.
func splitTestID(id string) (file, name string) {
	parts := strings.Split(id, "::")
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return "", parts[0]
	}
	return parts[0], parts[len(parts)-1]
}
