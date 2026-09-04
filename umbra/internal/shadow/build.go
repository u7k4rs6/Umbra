package shadow

import (
	"context"
	"regexp"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// BuildInput is everything node construction needs.
type BuildInput struct {
	Field    *graph.Field
	RelMap   *graph.RelationMap
	Sources  []graph.Source
	Reach    []graph.Reach
	Examined *Examined
	Impacts  map[string]*graph.Impact
	// HeadRoot is the worktree the call-site windows are read from.
	HeadRoot string
	Scars    map[string]int
}

// Build turns reached symbols into classified, ranked nodes.
func Build(in BuildInput) []*Node {
	sourceFileOf := map[string]string{}
	for _, s := range in.Sources {
		if s.Symbol != "" {
			sourceFileOf[s.Symbol] = s.File
		}
	}

	var nodes []*Node
	for _, r := range in.Reach {
		sym := in.Field.Symbols[r.ID]
		if sym == nil {
			continue
		}
		n := &Node{
			Symbol:     sym,
			Sources:    []string{r.Source},
			Family:     r.Family,
			Depth:      r.Depth,
			Path:       r.Path,
			Dependents: len(in.Field.In[r.ID]),
			CallSite:   r.CallSite,
		}

		// The exact call line comes from impact; the snapshot's evidence only
		// spans the calling function.
		if imp := in.Impacts[r.Source]; imp != nil {
			if line := imp.CallSiteFor(sym.ID, sym.File, sym.Name); line > 0 {
				n.CallSite = line
			}
		}

		n.State, n.Tier = Classify(in.Examined, sym.File, sym.Name, sym.Span)
		n.Exposures = exposuresFor(in.Examined, sym.File, sym.Name)

		if n.Tier != TierNone {
			addModifier(n, string(n.Tier))
		}
		if sym.IsTest {
			addModifier(n, "test")
		}
		if r.Depth > 1 {
			addModifier(n, "transitive")
		}
		if srcFile, ok := sourceFileOf[r.Source]; ok && FarField(sym.File, srcFile) {
			addModifier(n, "far field")
		}

		// Two signals come from the call site rather than the session.
		if n.CallSite > 0 && in.HeadRoot != "" {
			w := ReadWindow(in.HeadRoot, sym.File, n.CallSite)
			if w.FaultLine {
				addModifier(n, "fault line")
			}
			if w.Beacon != "" {
				n.Beacon = w.Beacon
				addModifier(n, "beacon")
			}
		}

		Score(n, RankInput{Scars: in.Scars})
		nodes = append(nodes, n)
	}
	return nodes
}

// AddCoChange adds the symbols in files Graph says historically change with a
// source's file. They enter at the lowest weight and are marked so a reader
// knows they are not a code path.
func AddCoChange(nodes []*Node, in BuildInput) []*Node {
	have := map[string]bool{}
	for _, n := range nodes {
		have[n.Symbol.ID] = true
	}
	for _, s := range in.Sources {
		if s.Symbol == "" {
			continue
		}
		imp := in.Impacts[s.Symbol]
		if imp == nil {
			continue
		}
		for _, file := range imp.CoChange {
			if file == s.File {
				continue
			}
			for _, id := range in.Field.ByFile[file] {
				sym := in.Field.Symbols[id]
				if have[id] || sym == nil {
					continue
				}
				switch sym.Kind {
				case "function", "method", "class":
				default:
					continue
				}
				have[id] = true
				n := &Node{
					Symbol:     sym,
					Sources:    []string{s.Symbol},
					Family:     graph.FamilyCoChange,
					Depth:      1,
					Path:       []string{s.Symbol, id},
					Dependents: len(in.Field.In[id]),
				}
				n.State, n.Tier = Classify(in.Examined, sym.File, sym.Name, sym.Span)
				n.Exposures = exposuresFor(in.Examined, sym.File, sym.Name)
				if n.Tier != TierNone {
					addModifier(n, string(n.Tier))
				}
				if sym.IsTest {
					addModifier(n, "test")
				}
				addModifier(n, "co-change only")
				if FarField(sym.File, s.File) {
					addModifier(n, "far field")
				}
				Score(n, RankInput{Scars: in.Scars})
				nodes = append(nodes, n)
			}
		}
	}
	return nodes
}

var fixRE = regexp.MustCompile(`(?i)\bfix|\bbug`)

// LoadScars counts recent commits touching each file whose message mentions a
// fix or a bug. It is opt-in through --history, read-only, and printed like
// every other factor.
func LoadScars(ctx context.Context, run runner.Runner, root string, files []string) (map[string]int, error) {
	out := map[string]int{}
	for _, file := range files {
		args := []string{"-C", root, "log", "--since=6.months", "--format=%s", "--", file}
		stdout, _, exit, err := run.Run(ctx, "git", args, nil)
		if err != nil {
			return nil, err
		}
		if exit != 0 {
			continue
		}
		n := 0
		for _, line := range strings.Split(string(stdout), "\n") {
			if fixRE.MatchString(line) {
				n++
			}
		}
		out[file] = n
	}
	return out, nil
}

// exposuresFor is every piece of evidence about a node, in sequence order: the
// events on its file, and the mentions of its own name.
//
// Both are needed on the node itself because the report's JavaScript replays
// the state at each step from this list alone, and a node whose only evidence
// is a mention of its name would otherwise look untouched.
func exposuresFor(e *Examined, file, name string) []Exposure {
	if e == nil {
		return nil
	}
	out := append([]Exposure(nil), e.ByFile[file]...)
	out = append(out, e.BySymbol[name]...)
	if len(out) == 0 {
		return nil
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Seq < out[j].Seq })
	return out
}
