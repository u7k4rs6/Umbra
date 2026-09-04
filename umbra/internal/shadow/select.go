package shadow

import (
	"path"
	"regexp"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// testIDRE is the validation from SECURITY_AND_ACCESS.md. An id that does not
// match is dropped and listed as not runnable rather than passed on.
var testIDRE = regexp.MustCompile(`^[A-Za-z0-9_./:\[\]\-]+$`)

// ValidTestID reports whether an id is safe to place in argv.
//
// Umbra never builds a shell string, so this is defence in depth rather than
// the only guard, but a name that cannot be a test id is a sign something is
// wrong and it is dropped loudly.
func ValidTestID(id string) bool {
	return id != "" && testIDRE.MatchString(id)
}

// Candidate is a test that reaches the change.
type Candidate struct {
	Node *Node
	ID   string
	// Group orders selection: 0 shadowed tests, 1 tests reaching shadow,
	// 2 everything else.
	Group int
}

// TestIDFor builds the runner id for a test symbol.
//
// pytest wants path::function, class qualified when the test sits in a class.
// go test wants -run '^Name$' with the package directory, which the caller
// assembles because it is two argv elements rather than one id.
func TestIDFor(sym *graph.Symbol, repoRelRoot string) (string, bool) {
	if sym == nil || !sym.IsTest {
		return "", false
	}
	file := sym.File
	if repoRelRoot != "" {
		trimmed := strings.TrimPrefix(file, strings.TrimSuffix(repoRelRoot, "/")+"/")
		if trimmed != file {
			file = trimmed
		}
	}
	switch strings.ToLower(path.Ext(sym.File)) {
	case ".py":
		id := file + "::" + sym.Name
		if !ValidTestID(id) {
			return "", false
		}
		return id, true
	case ".go":
		if !ValidTestID(sym.Name) {
			return "", false
		}
		return sym.Name, true
	}
	id := file + "::" + sym.Name
	if !ValidTestID(id) {
		return "", false
	}
	return id, true
}

// SelectTests picks the tests to run and the order to run them in.
//
// Selection order from PRD.md: tests that are themselves shadowed, then tests
// reaching shadowed nodes, then the rest. `--run shadow` runs the first two
// groups, `--run all` runs everything, `--run none` runs nothing.
func SelectTests(nodes []*Node, field *graph.Field, relMap *graph.RelationMap, sourceIDs []string, mode, repoRelRoot string) (selected []Candidate, notRunnable []string) {
	if mode == "none" {
		return nil, nil
	}

	shadowed := map[string]bool{}
	for _, n := range nodes {
		if n.Shadowed() {
			shadowed[n.Symbol.ID] = true
		}
	}
	sources := map[string]bool{}
	for _, id := range sourceIDs {
		sources[id] = true
	}

	seen := map[string]bool{}
	var out []Candidate

	for _, n := range nodes {
		if !n.IsTest() || seen[n.Symbol.ID] {
			continue
		}
		id, ok := TestIDFor(n.Symbol, repoRelRoot)
		if !ok {
			notRunnable = append(notRunnable, n.Symbol.File+"::"+n.Symbol.Name)
			continue
		}
		seen[n.Symbol.ID] = true

		group := 2
		switch {
		case n.Shadowed():
			group = 0
		case reachesShadow(field, relMap, n.Symbol.ID, shadowed, sources):
			group = 1
		}
		out = append(out, Candidate{Node: n, ID: id, Group: group})
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Group != out[j].Group {
			return out[i].Group < out[j].Group
		}
		if out[i].Node.Score != out[j].Node.Score {
			return out[i].Node.Score > out[j].Node.Score
		}
		return out[i].ID < out[j].ID
	})

	if mode == "shadow" {
		var kept []Candidate
		for _, c := range out {
			if c.Group <= 1 {
				kept = append(kept, c)
			}
		}
		out = kept
	}
	sort.Strings(notRunnable)
	return out, notRunnable
}

// reachesShadow reports whether a test's call chain passes through a shadowed
// node on its way to a source.
func reachesShadow(field *graph.Field, relMap *graph.RelationMap, testID string, shadowed, sources map[string]bool) bool {
	if field == nil {
		return false
	}
	seen := map[string]bool{testID: true}
	queue := []string{testID}
	depth := map[string]int{testID: 0}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if depth[cur] >= 3 {
			continue
		}
		for _, e := range field.Out[cur] {
			fam, ok := relMap.Family(e.Relation)
			if !ok || fam != graph.FamilyCalls || seen[e.To] {
				continue
			}
			seen[e.To] = true
			depth[e.To] = depth[cur] + 1
			if shadowed[e.To] {
				return true
			}
			if !sources[e.To] {
				queue = append(queue, e.To)
			}
		}
	}
	return false
}

// IDs returns the runner ids of a candidate list.
func IDs(cands []Candidate) []string {
	out := make([]string, 0, len(cands))
	for _, c := range cands {
		out = append(out, c.ID)
	}
	return out
}
