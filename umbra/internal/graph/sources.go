package graph

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Source is one changed symbol: a light in the map.
type Source struct {
	Symbol string
	Name   string
	File   string
	Span   [2]int
	Change string
	Weight float64
	// Dependents is Graph's own count, kept as a cross-check against the BFS.
	Dependents int
	// OldSignature and NewSignature are present for a signature change.
	OldSignature string
	NewSignature string
}

// Change weights from ARCHITECTURE.md section 2.
var changeWeight = map[string]float64{
	"signature": 1.0,
	"removed":   1.0,
	"renamed":   0.8,
	"body":      0.6,
	"added":     0.3,
}

// Weight returns the weight for a change kind, defaulting to the body weight
// for anything unrecognised.
func Weight(change string) float64 {
	if w, ok := changeWeight[change]; ok {
		return w
	}
	return changeWeight["body"]
}

// commitJSON is the shape of `graph commit <rev> --json`.
type commitJSON struct {
	Base  string `json:"base"`
	Head  string `json:"head"`
	Files []struct {
		Path     string `json:"path"`
		Status   string `json:"status"`
		Language string `json:"language"`
		Changes  []struct {
			Type            string `json:"type"`
			Kind            string `json:"kind"`
			Name            string `json:"name"`
			OldSignature    string `json:"old_signature"`
			NewSignature    string `json:"new_signature"`
			BeforeStartLine int    `json:"before_start_line"`
			AfterStartLine  int    `json:"after_start_line"`
			DependentsCount int    `json:"dependents_count"`
		} `json:"changes"`
	} `json:"files"`
}

// SourceResult carries the sources and how they were obtained, so the header
// can name the route.
type SourceResult struct {
	Sources []Source
	Route   string
	Note    string
}

// LoadSources asks Graph what changed.
//
// `graph checkpoint <id>` is the documented bridge between Graph and
// Checkpoints and is tried first. The Step 0 probe found it answers
// "has no associated commit in this repository" for a checkpoint created by
// `entire import`, so `graph commit <sha> --json` is the fallback, exactly as
// the degradation table says.
func LoadSources(ctx context.Context, run runner.Runner, repo, checkpointID, commitish string) (*SourceResult, error) {
	if checkpointID != "" {
		stdout, _, exit, err := run.Run(ctx, "entire", []string{"graph", "checkpoint", checkpointID, "--json"}, nil)
		if err == nil && exit == 0 {
			if srcs, perr := ParseCommitJSON(stdout); perr == nil && len(srcs) > 0 {
				return &SourceResult{Sources: srcs, Route: "graph checkpoint"}, nil
			}
		}
	}

	args := []string{"graph", "commit", commitish, "--repo", repo, "--json"}
	stdout, stderr, exit, err := run.Run(ctx, "entire", args, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("entire graph commit exited %d: %s", exit, strings.TrimSpace(string(stderr)))
	}
	srcs, err := ParseCommitJSON(stdout)
	if err != nil {
		return nil, err
	}
	res := &SourceResult{Sources: srcs, Route: "graph commit"}
	if checkpointID != "" {
		res.Note = "graph checkpoint had no commit for this checkpoint, so the change came from graph commit"
	}
	return res, nil
}

// ParseCommitJSON reads `graph commit --json` into sources.
func ParseCommitJSON(blob []byte) ([]Source, error) {
	trimmed := strings.TrimSpace(string(blob))
	if trimmed == "" {
		return nil, nil
	}
	i := strings.Index(trimmed, "{")
	if i < 0 {
		return nil, fmt.Errorf("graph commit produced no JSON")
	}
	var c commitJSON
	if err := json.Unmarshal([]byte(trimmed[i:]), &c); err != nil {
		return nil, fmt.Errorf("graph commit: %w", err)
	}

	var out []Source
	for _, f := range c.Files {
		for _, ch := range f.Changes {
			kind := normalizeChange(ch.Type)
			line := ch.AfterStartLine
			if line == 0 {
				line = ch.BeforeStartLine
			}
			out = append(out, Source{
				Name:         ch.Name,
				File:         f.Path,
				Span:         [2]int{line, line},
				Change:       kind,
				Weight:       Weight(kind),
				Dependents:   ch.DependentsCount,
				OldSignature: ch.OldSignature,
				NewSignature: ch.NewSignature,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		return out[i].Span[0] < out[j].Span[0]
	})
	return out, nil
}

func normalizeChange(t string) string {
	switch t {
	case "signature_changed":
		return "signature"
	case "body_changed":
		return "body"
	case "added":
		return "added"
	case "removed":
		return "removed"
	case "renamed":
		return "renamed"
	}
	return strings.TrimSuffix(t, "_changed")
}

// Bind attaches each source to its symbol in the field, filling the identity
// and the real span. A source with no symbol in the snapshot keeps the line
// Graph reported and is still a light, because a removed symbol has no entry
// in the head snapshot by definition.
func Bind(sources []Source, f *Field) []Source {
	out := make([]Source, 0, len(sources))
	for _, s := range sources {
		if id, sym := findSymbol(f, s.File, s.Name, s.Span[0]); sym != nil {
			s.Symbol = id
			s.Span = sym.Span
		}
		out = append(out, s)
	}
	return out
}

func findSymbol(f *Field, file, name string, line int) (string, *Symbol) {
	var exact []string
	for _, id := range f.ByFile[file] {
		if f.Symbols[id].Name == name {
			exact = append(exact, id)
		}
	}
	switch len(exact) {
	case 0:
		return "", nil
	case 1:
		return exact[0], f.Symbols[exact[0]]
	}
	// More than one symbol shares the name in the file; pick the one whose
	// span is nearest the reported line.
	bestID := exact[0]
	bestDist := -1
	for _, id := range exact {
		d := f.Symbols[id].Span[0] - line
		if d < 0 {
			d = -d
		}
		if bestDist < 0 || d < bestDist {
			bestDist, bestID = d, id
		}
	}
	return bestID, f.Symbols[bestID]
}

// SemanticLanguages returns the languages the installed Graph can resolve
// calls for. A language outside this set is inventory only: Graph reports its
// headings or blocks as symbols, but they have no callers, so a change to one
// cannot cast a shadow.
func SemanticLanguages(c *Capabilities) map[string]bool {
	out := map[string]bool{}
	if c == nil {
		return out
	}
	for lang, rels := range c.RelationByLanguage {
		for _, r := range rels {
			if f, ok := familyOf[r]; ok && (f == FamilyCalls || f == FamilyTypeUse || f == FamilyDataFlow) {
				out[lang] = true
				break
			}
		}
	}
	return out
}

// FilterSources drops changed entities that cannot have dependents.
//
// `graph commit` reports every changed entity, including Markdown headings and
// code fences, because they are symbols in the snapshot. They have no callers
// and never will, so treating them as light sources fills the map with
// documentation noise. A source is kept when its language can carry call, type
// or data-flow relations, which is read from capabilities rather than assumed.
//
// The dropped entities are returned so the header can say what was set aside.
func FilterSources(sources []Source, f *Field, c *Capabilities) (kept []Source, dropped []Source) {
	semantic := SemanticLanguages(c)
	for _, s := range sources {
		if isCodeSource(s, f, semantic) {
			kept = append(kept, s)
		} else {
			dropped = append(dropped, s)
		}
	}
	return kept, dropped
}

func isCodeSource(s Source, f *Field, semantic map[string]bool) bool {
	if s.Symbol != "" {
		if sym := f.Symbols[s.Symbol]; sym != nil {
			if len(semantic) > 0 && !semantic[sym.Language] {
				return false
			}
			switch sym.Kind {
			case "section", "code_fence", "document", "config", "import":
				return false
			}
			return true
		}
	}
	// A removed symbol has no entry in the head snapshot. Fall back to the
	// language of the file it lived in, judged from any symbol still there.
	if len(semantic) > 0 {
		for _, id := range f.ByFile[s.File] {
			return semantic[f.Symbols[id].Language]
		}
	}
	return true
}
