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
	// Weakest is the quality of the least confident edge on the path.
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
			id       string
			depth    int
			relation string
			family   Family
			path     []string
			callSite int
			weakest  EdgeQuality
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
				queue = append(queue, next)

				r := Reach{
					ID: next.id, Source: src, Relation: next.relation, Family: next.family,
					Depth: next.depth, Path: next.path, CallSite: next.callSite,
					Weakest: next.weakest,
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
	if next.Confidence < carried.Confidence {
		return next
	}
	return carried
}
