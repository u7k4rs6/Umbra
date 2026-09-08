package graph

import "sort"

// Reach is one dependent found by the search, with how it was reached.
type Reach struct {
	ID string
	// Source is the changed symbol this dependent hangs off.
	Source string
	// Relation is the relation on the first hop away from the source, which
	// is the one the ranker weighs.
	Relation string
	Family   Family
	Depth    int
	// Path is the shortest chain from the source out to this node.
	Path []string
	// CallSite is the line of the call into the source, when the first hop
	// carried one.
	CallSite int
	// Heuristic is set when any hop was a relation type the provider documents
	// as heuristic, or was resolved by anything other than a parse.
	// LastHopHeuristic says whether the doubt is on this node's own edge or
	// inherited from further up the chain, and HeuristicVia names the first
	// hop that introduced it.
	Heuristic        bool
	LastHopHeuristic bool
	HeuristicVia     string
	// Warnings are every per-relation warning code the provider attached to
	// any hop on the path, deduplicated.
	Warnings []string
	// Weakest is the quality of the least resolved edge on the path.
	//
	// Relation, Family and CallSite above describe the FIRST hop, because that
	// is the hop the ranker weighs. Quality is carried the other way, from the
	// weakest hop, because a path is only as followable as its worst link and
	// a reader asking "can I trust this chain" wants the floor rather than the
	// entry. There was no weakest-hop rule in this walk before: nothing was
	// carried but the first hop, and resolution was not read at all.
	Weakest EdgeQuality
}

// Dependents walks the field outward from each source over incoming edges,
// because a caller points at what it calls: the dependents of a symbol are the
// symbols with an edge arriving at it.
//
// Calls are followed to depth. Type use and data flow are followed one hop, as
// specified: a type consumer of a type consumer is rarely affected by a
// signature change, and following it floods the map.
//
// The walk is deterministic: neighbours are visited in sorted order, and the
// first arrival at a node wins, so the recorded path is the shortest one and
// two runs on the same input produce the same field.
func (f *Field) Dependents(sources []string, depth int, rm *RelationMap) []Reach {
	if depth < 1 {
		depth = 1
	}
	best := map[string]Reach{}
	sourceSet := map[string]bool{}
	for _, s := range sources {
		sourceSet[s] = true
	}

	ordered := append([]string(nil), sources...)
	sort.Strings(ordered)

	for _, src := range ordered {
		type item struct {
			id           string
			depth        int
			relation     string
			family       Family
			path         []string
			callSite     int
			weakest      EdgeQuality
			heuristic    bool
			lastHop      bool
			heuristicVia string
			warnings     []string
		}
		seen := map[string]bool{src: true}
		queue := []item{{id: src, depth: 0, path: []string{src}}}

		for len(queue) > 0 {
			cur := queue[0]
			queue = queue[1:]

			edges := append([]Edge(nil), f.In[cur.id]...)
			sort.Slice(edges, func(i, j int) bool {
				if edges[i].From != edges[j].From {
					return edges[i].From < edges[j].From
				}
				return edges[i].Relation < edges[j].Relation
			})

			for _, e := range edges {
				fam, ok := rm.Family(e.Relation)
				if !ok || !rm.Traverses(e.Relation) {
					continue
				}
				// Type use and data flow contribute one hop only.
				if cur.depth >= 1 && fam != FamilyCalls {
					continue
				}
				if fam == FamilyCalls && cur.depth >= depth {
					continue
				}
				if fam != FamilyCalls && cur.depth >= 1 {
					continue
				}
				if seen[e.From] || sourceSet[e.From] {
					continue
				}
				if _, exists := f.Symbols[e.From]; !exists {
					continue
				}
				seen[e.From] = true

				relation, family, callSite := e.Relation, fam, e.CallSite
				if cur.depth > 0 {
					relation, family, callSite = cur.relation, cur.family, cur.callSite
				}
				next := item{
					id:       e.From,
					depth:    cur.depth + 1,
					relation: relation,
					family:   family,
					path:     append(append([]string(nil), cur.path...), e.From),
					callSite: callSite,
					weakest:  weaker(cur.weakest, e.Quality),
				}
				// A hop is heuristic when the provider calls the relation type
				// heuristic, or when it resolved the edge by something other
				// than a parse. Either way the chain is only as trustworthy as
				// that hop, so the flag travels and the first one to set it is
				// named.
				hopHeuristic := rm.IsHeuristic(e.Relation) ||
					(e.Quality.Known() && !ResolutionIsStructural(e.Quality.Resolution))
				next.lastHop = hopHeuristic
				next.heuristic = cur.heuristic || hopHeuristic
				next.heuristicVia = cur.heuristicVia
				if hopHeuristic && !cur.heuristic {
					// Name the hop, not the relation type: a reader chasing
					// this wants to know which link in the chain to check.
					next.heuristicVia = e.From
				}
				next.warnings = cur.warnings
				for _, w := range e.Quality.WarningCodes {
					if !containsString(next.warnings, w) {
						next.warnings = append(append([]string(nil), next.warnings...), w)
					}
				}
				queue = append(queue, next)

				r := Reach{
					ID: next.id, Source: src, Relation: next.relation, Family: next.family,
					Depth: next.depth, Path: next.path, CallSite: next.callSite,
					Weakest: next.weakest, Heuristic: next.heuristic,
					LastHopHeuristic: next.lastHop, HeuristicVia: next.heuristicVia,
					Warnings: next.warnings,
				}
				if prev, ok := best[next.id]; !ok || better(r, prev) {
					best[next.id] = r
				}
			}
		}
	}

	out := make([]Reach, 0, len(best))
	for _, r := range best {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Depth != out[j].Depth {
			return out[i].Depth < out[j].Depth
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// better prefers the shorter path, then the stronger relation, then the lower
// id, so the choice never depends on map iteration order.
func better(a, b Reach) bool {
	if a.Depth != b.Depth {
		return a.Depth < b.Depth
	}
	ra, rb := familyRank(a.Family), familyRank(b.Family)
	if ra != rb {
		return ra > rb
	}
	return a.ID < b.ID
}

func familyRank(f Family) int {
	switch f {
	case FamilyCalls:
		return 3
	case FamilyDataFlow:
		return 2
	case FamilyTypeUse:
		return 1
	}
	return 0
}

// PathTo finds a call path from one symbol out to any of the targets without a
// depth cap, and reports the depth and the first hop. The sweep's leak
// forensics uses it to say "path exists at depth 4 via <symbol>".
func (f *Field) PathTo(from string, targets map[string]bool, rm *RelationMap, maxDepth int) (depth int, via string, ok bool) {
	if targets[from] {
		return 0, "", true
	}
	type item struct {
		id    string
		depth int
		first string
	}
	seen := map[string]bool{from: true}
	queue := []item{{id: from}}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur.depth >= maxDepth {
			continue
		}
		edges := append([]Edge(nil), f.Out[cur.id]...)
		sort.Slice(edges, func(i, j int) bool { return edges[i].To < edges[j].To })

		for _, e := range edges {
			fam, known := rm.Family(e.Relation)
			if !known || fam != FamilyCalls {
				continue
			}
			if seen[e.To] {
				continue
			}
			seen[e.To] = true
			first := cur.first
			if first == "" {
				first = e.To
			}
			if targets[e.To] {
				name := first
				if s, ok := f.Symbols[first]; ok {
					name = s.Name
				}
				return cur.depth + 1, name, true
			}
			queue = append(queue, item{id: e.To, depth: cur.depth + 1, first: first})
		}
	}
	return 0, "", false
}

// HasOutgoingCalls reports whether a symbol has any resolved call edge at all.
// A test with none is invisible to selection, which is a leak reason.
func (f *Field) HasOutgoingCalls(id string, rm *RelationMap) bool {
	for _, e := range f.Out[id] {
		if fam, ok := rm.Family(e.Relation); ok && fam == FamilyCalls {
			return true
		}
	}
	return false
}

// CallSplit counts the outgoing calls of a symbol by whether the target is a
// symbol in this repository or something outside it.
//
// Unknown counts edges the provider published no quality for, which is what a
// graph assembled inside a test carries. Counting them separately keeps a
// fixture from being reported as if every one of its edges left the repository.
func (f *Field) CallSplit(id string, rm *RelationMap) (inside, leaving, unknown int) {
	for _, e := range f.Out[id] {
		fam, ok := rm.Family(e.Relation)
		if !ok || fam != FamilyCalls {
			continue
		}
		switch {
		case !e.Quality.Known():
			unknown++
		case e.Quality.LeavesRepo():
			leaving++
		default:
			inside++
		}
	}
	return inside, leaving, unknown
}

// CallsAllLeaveRepo reports whether a symbol makes calls and every one of them
// goes somewhere this repository cannot show you.
//
// This is the difference between "nothing depends on it" and "the calls are
// there and none of them can be followed", which the first real-repo run could
// not tell apart: 79 percent of the calls leaving the tests in two of three
// repositories measured resolve to a node outside the snapshot.
func (f *Field) CallsAllLeaveRepo(id string, rm *RelationMap) (int, bool) {
	inside, leaving, unknown := f.CallSplit(id, rm)
	if leaving == 0 || inside > 0 || unknown > 0 {
		return leaving, false
	}
	return leaving, true
}

// weaker returns whichever of two edge qualities a reader should be told
// about: the one the provider was least sure of.
//
// An unknown quality, which is what a fixture-built edge or an older snapshot
// carries, never displaces a known one, so a graph assembled in a test behaves
// as it always did.
func weaker(carried, next EdgeQuality) EdgeQuality {
	if !next.Known() {
		return carried
	}
	if !carried.Known() {
		return next
	}
	// Resolution first, confidence second. The provider's resolution is a
	// named category with an order the vocabulary in evidence.go defines;
	// confidence is a float it attaches on top and is not derivable from it.
	//
	// The reason the order matters this way round is that confidence is a
	// within-method quantity. It says how sure the provider is given the method
	// it used, so it is comparable between two name_only edges and between two
	// exact ones, and comparing it across resolution methods was never
	// meaningful. Ranking on it alone, which is what this did before, put a
	// name_only guess at 0.85 above an exact match at 0.8 and called the exact
	// match the weaker hop.
	if nr, cr := ResolutionRank(next.Resolution), ResolutionRank(carried.Resolution); nr != cr {
		if nr < cr {
			return next
		}
		return carried
	}
	if next.Confidence < carried.Confidence {
		return next
	}
	return carried
}

func containsString(list []string, want string) bool {
	for _, x := range list {
		if x == want {
			return true
		}
	}
	return false
}
