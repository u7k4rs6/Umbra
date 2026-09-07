package shadow

import "github.com/u7k4rs6/Umbra/umbra/internal/graph"

// Evidence is how much of a dependent's presence in the field the graph can
// actually vouch for. It is a fourth axis beside state, tier and score: the
// four states say what the SESSION saw, and this says what the GRAPH knows.
//
// It exists because Graph is evidence and not an oracle. A relation the
// provider resolved to a definition and a relation it matched by unique name
// used to reach the reader looking identical, and a reviewer acting on the
// second one has no way to know they should check it first.
type Evidence string

const (
	// Confirmed: every hop the provider reported was resolved to a
	// definition, the relation type is not one it documents as heuristic, and
	// nothing about this run was degraded.
	Confirmed Evidence = "confirmed"
	// Heuristic: the relation attaching this node is one the provider itself
	// derives rather than parses. The reader can see the edge and judge it.
	Heuristic Evidence = "heuristic"
	// NeedsVerification: the node hangs off a link the reader cannot see, or
	// the provider said this analysis may be incomplete. The report prints the
	// exact command that settles it.
	NeedsVerification Evidence = "needs verification"
)

// Meaning is the plain sentence shown next to the word, everywhere the word
// appears. FRONTEND_SPEC.md requires this of every coined term, and these
// three are coined.
func (e Evidence) Meaning() string {
	switch e {
	case Confirmed:
		return "the graph resolved every relation on this path to a definition"
	case Heuristic:
		return "the graph derived this relation rather than parsing it, so it may be wrong"
	case NeedsVerification:
		return "reached through a relation the graph could not resolve, or found under a partial analysis; check it against the source"
	}
	return ""
}

// Mark is the terminal and map marker. It is deliberately not a circle: the
// four states own the circles, and confusing the two axes would be worse than
// having no marker at all.
func (e Evidence) Mark() string {
	switch e {
	case Confirmed:
		return "="
	case Heuristic:
		return "~"
	case NeedsVerification:
		return "?"
	}
	return " "
}

// Rank orders the three strongest first, for sorting and for a summary line.
func (e Evidence) Rank() int {
	switch e {
	case Confirmed:
		return 0
	case Heuristic:
		return 1
	case NeedsVerification:
		return 2
	}
	return 3
}

// EvidenceInput is what classification needs beyond the node itself.
type EvidenceInput struct {
	// RunDegraded is set when the provider said this whole analysis may be
	// incomplete: a degraded snapshot, a partial failure, a malformed record,
	// or an impact query it flagged for the changed symbol this node hangs
	// off.
	RunDegraded bool
	// DegradedWhy is the provider's own reason, quoted rather than
	// paraphrased.
	DegradedWhy string
}

// ClassifyEvidence returns the tier and the one sentence that explains it.
//
// The order below is the precedence, strongest claim last so it cannot be
// reached by accident: anything the provider flagged wins over anything
// derived from a single relation.
func ClassifyEvidence(n *Node, in EvidenceInput) (Evidence, string) {
	if n == nil {
		return NeedsVerification, "there is no node to judge"
	}

	// The provider said it could not fully analyse this node's own file.
	if n.FileIncomplete {
		return NeedsVerification, n.IncompleteWhy
	}

	// The provider said this run is incomplete. It cannot say which edges are
	// affected, which is exactly why the whole run inherits the doubt.
	if in.RunDegraded {
		why := in.DegradedWhy
		if why == "" {
			why = "the graph reported this analysis as incomplete"
		}
		return NeedsVerification, why
	}

	// Reached through a link the reader cannot see. The hop that attached this
	// node was fine; something further up the chain was not.
	if n.HeuristicEdge && !n.LastHopHeuristic {
		via := n.HeuristicVia
		if via == "" {
			via = "an earlier hop"
		}
		return NeedsVerification,
			"the path to this node passes through " + via +
				", which the graph resolved as " + resolutionWord(n.Resolution) +
				" rather than to a definition"
	}

	// The relation attaching this node is one the provider derives.
	if n.HeuristicEdge {
		return Heuristic,
			"the graph resolved this relation as " + resolutionWord(n.Resolution) +
				": " + graph.ResolutionMeaning(n.Resolution)
	}

	return Confirmed, "the graph resolved every hop on this path to a definition"
}

func resolutionWord(r string) string {
	if r == "" {
		return "unreported"
	}
	return r
}

// EvidenceSummary counts the three tiers, for the header line.
type EvidenceSummary struct {
	Confirmed         int
	Heuristic         int
	NeedsVerification int
}

// SummarizeEvidence counts the tiers across nodes.
func SummarizeEvidence(nodes []*Node) EvidenceSummary {
	var s EvidenceSummary
	for _, n := range nodes {
		switch n.Evidence {
		case Confirmed:
			s.Confirmed++
		case Heuristic:
			s.Heuristic++
		default:
			s.NeedsVerification++
		}
	}
	return s
}

// DependentCountIsHeuristic is stated wherever a dependent count is printed.
//
// It is not a property of one node: the provider derives every dependent count
// from the same mix of exact, name-only, package and pattern resolutions its
// relations carry, so no count anywhere in the report is compiler accurate.
const DependentCountIsHeuristic = "dependent counts are the graph's estimate, not a compiler's"
