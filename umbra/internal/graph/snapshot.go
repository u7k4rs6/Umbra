package graph

import (
	"bufio"
	"bytes"
	"encoding/json"
	"path"
	"sort"
	"strings"
)

// Symbol is one definition in the repository.
type Symbol struct {
	ID   string
	Name string
	// QualifiedName is the provider's own qualified form, one level deep:
	// "TopClass.method" for a method, and the bare name for anything at module
	// level. It is what `graph commit` names a changed entity by, so it is the
	// field a changed symbol is bound through. It is never reconstructed here;
	// a snapshot that omits it leaves this empty and binding falls back to
	// Name. NOTES carries the table this was measured from.
	QualifiedName string
	// ContainerID is the enclosing symbol, when there is one. It is not used
	// for binding, because QualifiedName already carries the container's name.
	ContainerID string
	Kind        string
	File        string
	Span        [2]int
	IsTest      bool
	// Signature is the declaration line Graph reported. It reaches a report
	// only under --snippets, capped and scrubbed.
	Signature string
	Language  string
}

// Edge is one relation between two symbols.
type Edge struct {
	From     string
	To       string
	Relation string
	// CallSite is the line Graph's evidence pointed at, when it gave one.
	// Snapshot evidence spans the calling function, so this is a starting
	// point that graph impact refines to the exact line.
	CallSite int
	// Quality is what Graph published about how it resolved this edge. Every
	// relation record carries all five fields; the snapshot loader used to
	// discard them, which is why a report could not tell an edge into the
	// repository from one leaving it. See NOTES phase 25 stage 1 for the
	// measured distributions.
	Quality EdgeQuality
}

// EdgeQuality is the provider's own account of one edge.
//
// Measured across three repositories and 21,000 edges: Scope is one of
// external, module, file or workspace and separates a same-repo target from an
// external one with no exceptions; TargetKind is one of symbol, external,
// file, config or route and names synthesised targets; Reason is a closed set
// of 87 phrases with nothing interpolated, so it is safe to render; Confidence
// is not derivable from Resolution.
type EdgeQuality struct {
	Confidence float64
	Reason     string
	Scope      string
	Resolution string
	TargetKind string
	// WarningCodes are the provider's own per-relation warnings, kept so a
	// report can quote the tool's code rather than paraphrase it.
	WarningCodes []string
}

// Scope and TargetKind values worth naming, since two of them are load bearing.
const (
	ScopeExternal      = "external"
	TargetKindExternal = "external"
	TargetKindSymbol   = "symbol"
)

// LeavesRepo reports whether this edge points outside the repository.
//
// Graph reports a call it could not resolve as external, the same as a call to
// the standard library, so this means "the target is not a symbol here" and
// not "resolution failed". Both are the same thing for a reader trying to
// follow the edge.
func (q EdgeQuality) LeavesRepo() bool {
	return q.Scope == ScopeExternal || q.TargetKind == TargetKindExternal
}

// Known reports whether the provider said anything about this edge, which is
// false for an edge built by a test fixture or an older snapshot.
func (q EdgeQuality) Known() bool {
	return q.Scope != "" || q.TargetKind != "" || q.Resolution != ""
}

// Field is the loaded graph: symbols, adjacency in both directions, and the
// symbols in each file ordered by position.
type Field struct {
	Symbols map[string]*Symbol
	In      map[string][]Edge
	Out     map[string][]Edge
	ByFile  map[string][]string
	Files   []string

	// Relations counts each relation name seen, for the header.
	Relations map[string]int
	// Resolutions counts each resolution seen, so the header can say how much
	// of the field the provider parsed and how much it guessed.
	Resolutions map[string]int
	// Meta is what the snapshot said about its own completeness: the profile,
	// the warnings, the partial failures and the records this loader could
	// not use. An empty field built from nothing readable used to look
	// exactly like a clean one.
	Meta *SnapshotMeta
}

// snapshot record shapes, exactly as the Step 0 probe observed them.
type snapRecord struct {
	RecordType string `json:"record_type"`

	// file
	Path string `json:"path"`

	// symbol
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Name          string `json:"name"`
	QualifiedName string `json:"qualified_name"`
	ContainerID   string `json:"container_id"`
	FilePath      string `json:"file_path"`
	StartLine     int    `json:"start_line"`
	EndLine       int    `json:"end_line"`
	Signature     string `json:"signature"`
	Language      string `json:"language"`

	// relation
	FromID       string         `json:"from_id"`
	ToID         string         `json:"to_id"`
	Type         string         `json:"type"`
	Evidence     []snapEvidence `json:"evidence"`
	Confidence   float64        `json:"confidence"`
	Reason       string         `json:"reason"`
	Scope        string         `json:"relation_scope"`
	Resolution   string         `json:"resolution"`
	TargetKind   string         `json:"target_kind"`
	WarningCodes []string       `json:"warning_codes"`

	// header and summary. The header line carries no record_type at all and
	// is identified by its schema_version; the summary carries record_type
	// "summary" and is the only record with the counts filled in.
	SchemaVersion   string            `json:"schema_version"`
	ProviderVersion string            `json:"provider_version"`
	Profile         string            `json:"profile"`
	Commit          string            `json:"commit"`
	LanguageTiers   map[string]string `json:"language_tiers"`
	Warnings        []Warning         `json:"warnings"`
	PartialFailures []PartialFailure  `json:"partial_failures"`
	Stats           Stats             `json:"stats"`
}

type snapEvidence struct {
	Kind      string `json:"kind"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Detail    string `json:"detail"`
}

// LoadSnapshot reads `graph snapshot --repo <path> --format ndjson`.
//
// Records Umbra does not use (externals, the capabilities header, the summary)
// are skipped. A record that does not parse is skipped rather than failing the
// load, so a snapshot with one bad line still produces a field.
func LoadSnapshot(ndjson []byte) (*Field, error) {
	f := &Field{
		Symbols:     map[string]*Symbol{},
		In:          map[string][]Edge{},
		Out:         map[string][]Edge{},
		ByFile:      map[string][]string{},
		Relations:   map[string]int{},
		Resolutions: map[string]int{},
		Meta:        &SnapshotMeta{},
	}
	files := map[string]bool{}

	sc := bufio.NewScanner(bytes.NewReader(ndjson))
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)

	for sc.Scan() {
		line := bytes.TrimSpace(sc.Bytes())
		if len(line) == 0 {
			continue
		}
		var r snapRecord
		if err := json.Unmarshal(line, &r); err != nil {
			// Counted, not silently skipped. A snapshot whose records all
			// failed to parse produced an empty field, and an empty field is
			// indistinguishable from a clean one.
			f.Meta.Malformed++
			continue
		}
		if r.RecordType == "" && r.SchemaVersion != "" {
			// The header line, which carries no record_type.
			f.Meta.SchemaVersion = r.SchemaVersion
			f.Meta.ProviderVersion = r.ProviderVersion
			f.Meta.Profile = r.Profile
			f.Meta.Commit = r.Commit
			mergeMeta(f.Meta, r)
			continue
		}
		switch r.RecordType {
		case "summary":
			f.Meta.SawSummary = true
			mergeMeta(f.Meta, r)
		case "file":
			if r.Path != "" {
				files[r.Path] = true
			}
		case "symbol":
			if r.ID == "" || r.FilePath == "" {
				f.Meta.Dropped++
				continue
			}
			s := &Symbol{
				ID:            r.ID,
				Name:          r.Name,
				QualifiedName: r.QualifiedName,
				ContainerID:   r.ContainerID,
				Kind:          r.Kind,
				File:          r.FilePath,
				Span:          [2]int{r.StartLine, r.EndLine},
				Signature:     r.Signature,
				Language:      r.Language,
			}
			// The snapshot carries no test marking, which the Step 0 probe
			// confirmed, so IsTest comes from file and name conventions.
			s.IsTest = IsTestFile(s.File) && looksLikeTestSymbol(s)
			f.Symbols[s.ID] = s
			f.ByFile[s.File] = append(f.ByFile[s.File], s.ID)
			files[s.File] = true
		case "relation":
			if r.FromID == "" || r.ToID == "" {
				f.Meta.Dropped++
				continue
			}
			e := Edge{From: r.FromID, To: r.ToID, Relation: r.Type, Quality: EdgeQuality{
				Confidence: r.Confidence, Reason: r.Reason, Scope: r.Scope,
				Resolution: r.Resolution, TargetKind: r.TargetKind,
				WarningCodes: r.WarningCodes,
			}}
			for _, ev := range r.Evidence {
				if ev.Kind == "call_site" && ev.StartLine > 0 {
					e.CallSite = ev.StartLine
					break
				}
			}
			f.Out[e.From] = append(f.Out[e.From], e)
			f.In[e.To] = append(f.In[e.To], e)
			f.Relations[e.Relation]++
			if e.Quality.Resolution != "" {
				f.Resolutions[e.Quality.Resolution]++
			}
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	for p := range files {
		f.Files = append(f.Files, p)
	}
	sort.Strings(f.Files)
	for file, ids := range f.ByFile {
		sorted := ids
		sort.Slice(sorted, func(i, j int) bool {
			a, b := f.Symbols[sorted[i]], f.Symbols[sorted[j]]
			if a.Span[0] != b.Span[0] {
				return a.Span[0] < b.Span[0]
			}
			return a.ID < b.ID
		})
		f.ByFile[file] = sorted
	}
	return f, nil
}

// IsTestFile applies the runner conventions from ARCHITECTURE.md. The snapshot
// does not mark test symbols, so this is the only signal available.
func IsTestFile(p string) bool {
	if p == "" {
		return false
	}
	base := path.Base(p)
	switch {
	case strings.HasPrefix(base, "test_") && strings.HasSuffix(base, ".py"):
		return true
	case strings.HasSuffix(base, "_test.py"):
		return true
	case strings.HasSuffix(base, "_test.go"):
		return true
	case strings.Contains(base, ".test."):
		return true
	case strings.Contains(base, ".spec."):
		return true
	}
	for _, seg := range strings.Split(path.Dir(p), "/") {
		if seg == "tests" || seg == "test" || seg == "__tests__" {
			return true
		}
	}
	return false
}

// looksLikeTestSymbol keeps helpers and fixtures in a test file from being
// treated as runnable tests. A test function is what a runner can name.
func looksLikeTestSymbol(s *Symbol) bool {
	switch s.Kind {
	case "function", "method":
	default:
		return false
	}
	if strings.HasPrefix(s.Name, "Test") || strings.HasPrefix(s.Name, "test_") {
		return true
	}
	return s.Name == "test"
}

// SymbolsAt returns the symbols in a file whose span contains the line.
func (f *Field) SymbolsAt(file string, line int) []*Symbol {
	var out []*Symbol
	for _, id := range f.ByFile[file] {
		s := f.Symbols[id]
		if s.Span[0] <= line && line <= s.Span[1] {
			out = append(out, s)
		}
	}
	return out
}

// RelationNames lists the relation names present, sorted, for the header.
func (f *Field) RelationNames() []string {
	out := make([]string, 0, len(f.Relations))
	for name := range f.Relations {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// Vocabulary returns the file and symbol names a mention may match.
func (f *Field) Vocabulary() ([]string, []string) {
	names := map[string]bool{}
	for _, s := range f.Symbols {
		if s.Name != "" {
			names[s.Name] = true
		}
	}
	syms := make([]string, 0, len(names))
	for n := range names {
		syms = append(syms, n)
	}
	sort.Strings(syms)
	return f.Files, syms
}

// mergeMeta takes the completeness fields from whichever record carried them.
// The header writes them empty because the counts are not known until the walk
// finishes; the summary writes them filled. Neither is allowed to erase what
// the other reported.
func mergeMeta(m *SnapshotMeta, r snapRecord) {
	if len(r.LanguageTiers) > 0 {
		m.LanguageTiers = r.LanguageTiers
	}
	if len(r.Warnings) > 0 {
		m.Warnings = append(m.Warnings, r.Warnings...)
	}
	if len(r.PartialFailures) > 0 {
		m.PartialFailures = append(m.PartialFailures, r.PartialFailures...)
	}
	if r.Stats.Files > 0 || r.Stats.Symbols > 0 || r.Stats.CompletenessLevel != "" {
		m.Stats = r.Stats
	}
}

// ResolutionNames lists the resolutions present, sorted, for the header.
func (f *Field) ResolutionNames() []string {
	out := make([]string, 0, len(f.Resolutions))
	for name := range f.Resolutions {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}
