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
	Kind   string
	File   string
	Span   [2]int
	Change string
	Weight float64
	// Dependents is Graph's own count, kept as a cross-check against the BFS.
	Dependents int
	// OldSignature and NewSignature are present for a signature change.
	OldSignature string
	NewSignature string
	// Unresolved is why this changed entity could not be bound to a symbol in
	// the snapshot, and is empty when it was bound. It exists so the report
	// can distinguish a symbol whose dependents were never looked up from one
	// that was looked up and has none.
	Unresolved string
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
//
// It is only safe when the checkpoint and the commit are the same piece of
// work. A commit paired with a checkpoint by session time is not: the
// checkpoint belongs to a commit of its own, and asking Graph about it returns
// that commit's changes while the header still names the one the reader asked
// for. checkpointOwnsCommit says whether the pairing is tight enough to trust,
// and the caller decides it from how the reference resolved.
func LoadSources(ctx context.Context, run runner.Runner, repo, checkpointID, commitish string, checkpointOwnsCommit bool) (*SourceResult, error) {
	if checkpointID != "" && checkpointOwnsCommit {
		stdout, _, exit, err := run.Run(ctx, "entire", []string{"graph", "checkpoint", checkpointID, "--json"}, nil)
		if err == nil && exit == 0 {
			if srcs, perr := ParseCommitJSON(stdout); perr == nil && len(srcs) > 0 {
				return &SourceResult{Sources: FoldFields(srcs), Route: "graph checkpoint"}, nil
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
	srcs = FoldFields(srcs)
	res := &SourceResult{Sources: srcs, Route: "graph commit"}
	switch {
	case checkpointID != "" && !checkpointOwnsCommit:
		res.Note = "the checkpoint was paired with this commit by session time and owns a commit of its own, so the change came from graph commit"
	case checkpointID != "":
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
				Kind:         ch.Kind,
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

// KindLabel is how a change kind reads in a sentence.
//
// The kind was being printed with " changed" appended, which is right for a
// body or a signature and wrong for the rest: a symbol that was added reads as
// "added changed". The label is built in one place so the table, the packet
// and the map cannot word it differently.
func (s Source) KindLabel() string {
	switch s.Change {
	case "signature":
		return "signature changed"
	case "body":
		return "body changed"
	case "added", "removed", "renamed":
		return s.Change
	}
	if s.Change == "" {
		return "changed"
	}
	return s.Change
}

// FoldFields collapses a struct field into the type it belongs to.
//
// `graph commit` reports a field as a changed entity of its own, and a field's
// dependent count is every use of the type it sits in: adding one struct in a
// test file put jsNode.ID in a report as a source with 46 dependents, beside
// jsNode itself. A field is not a thing a caller calls. When its type is
// already a source the field is dropped, and when it is not the field is
// folded into one entry for the type so it is still counted, once.
func FoldFields(sources []Source) []Source {
	isSource := map[string]bool{}
	for _, s := range sources {
		if s.Kind != "field" {
			isSource[s.File+"\x00"+s.Name] = true
		}
	}

	var out []Source
	folded := map[string]bool{}
	for _, s := range sources {
		if s.Kind != "field" {
			out = append(out, s)
			continue
		}
		parent, ok := parentOf(s.Name)
		if !ok {
			// A field with no qualified name says nothing on its own.
			continue
		}
		key := s.File + "\x00" + parent
		if isSource[key] || folded[key] {
			continue
		}
		folded[key] = true

		// The type is not itself in the change list, so stand in for it once.
		p := s
		p.Name = parent
		p.Kind = "type"
		p.Symbol = ""
		out = append(out, p)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].File != out[j].File {
			return out[i].File < out[j].File
		}
		if out[i].Span[0] != out[j].Span[0] {
			return out[i].Span[0] < out[j].Span[0]
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// parentOf splits a qualified field name into the type that holds it.
func parentOf(name string) (string, bool) {
	i := strings.LastIndex(name, ".")
	if i <= 0 || i == len(name)-1 {
		return "", false
	}
	return name[:i], true
}

// Unresolved reasons, kept as constants so the renderers and the tests name
// the same thing.
const (
	// UnresolvedNoMatch means nothing in the file carries that name. A removed
	// symbol lands here legitimately: it has no entry in the head snapshot by
	// definition.
	UnresolvedNoMatch = "no symbol in the snapshot of this file carries that name"
	// UnresolvedAmbiguous means several symbols in the file carry the name and
	// the reported line did not settle which.
	UnresolvedAmbiguous = "several symbols in this file carry that name and the reported line did not settle which"
)

// Bind attaches each source to its symbol in the field, filling the identity
// and the real span.
//
// The name `graph commit` reports for a changed entity is the snapshot's
// qualified_name, which for a method is "TopClass.method" while the snapshot's
// Name field holds the bare "method". Binding on Name therefore never matched
// a changed method, in any repository, and the impact query for it was skipped
// as though it had no identity. Every changed symbol in fixtures/app is a
// module-level function, where the two forms are identical, so the whole of
// that is invisible to the fixture. NOTES carries the table this was measured
// from, across Python, Go and TypeScript.
//
// A source that cannot be bound keeps the line Graph reported and carries the
// reason it could not be bound, because a symbol that could not be resolved
// and a symbol with no dependents are different findings and the report has to
// say which one it has.
func Bind(sources []Source, f *Field) []Source {
	out := make([]Source, 0, len(sources))
	for _, s := range sources {
		id, sym, why := findSymbol(f, s.File, s.Name, s.Span[0])
		if sym != nil {
			s.Symbol = id
			s.Span = sym.Span
			s.Unresolved = ""
		} else {
			s.Unresolved = why
		}
		out = append(out, s)
	}
	return out
}

// findSymbol resolves one changed entity to one snapshot symbol.
//
// File, then qualified name, then span containment, and it has to land on
// exactly one symbol. A qualified name is not unique within a file: two
// functions in one module can each define a class `Helper` with a method
// `run`, and Graph reports both as `Helper.run`, distinguishing them only by
// the line. Where the line does not decide, this returns no match rather than
// the nearest one, because a guess that always answers is what put a symbol's
// dependents under the wrong heading in the first place.
func findSymbol(f *Field, file, name string, line int) (string, *Symbol, string) {
	var exact []string
	for _, id := range f.ByFile[file] {
		if symbolName(f.Symbols[id]) == name {
			exact = append(exact, id)
		}
	}
	switch len(exact) {
	case 0:
		return "", nil, UnresolvedNoMatch
	case 1:
		return exact[0], f.Symbols[exact[0]], ""
	}
	// Several carry the name. The reported line has to fall inside exactly one
	// of their spans, or this is ambiguous and nothing is bound.
	var containing []string
	for _, id := range exact {
		s := f.Symbols[id]
		if s.Span[0] <= line && line <= s.Span[1] {
			containing = append(containing, id)
		}
	}
	if len(containing) == 1 {
		return containing[0], f.Symbols[containing[0]], ""
	}
	return "", nil, UnresolvedAmbiguous
}

// symbolName is the name a changed entity is reported under: the provider's
// qualified_name where it published one, and the bare name otherwise, so a
// snapshot from a build that does not emit the field still binds module-level
// symbols exactly as it always did.
func symbolName(s *Symbol) string {
	if s == nil {
		return ""
	}
	if s.QualifiedName != "" {
		return s.QualifiedName
	}
	return s.Name
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
