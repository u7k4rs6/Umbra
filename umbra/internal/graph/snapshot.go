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
	ID     string
	Name   string
	Kind   string
	File   string
	Span   [2]int
	IsTest bool
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
}

// snapshot record shapes, exactly as the Step 0 probe observed them.
type snapRecord struct {
	RecordType string `json:"record_type"`

	// file
	Path string `json:"path"`

	// symbol
	ID        string `json:"id"`
	Kind      string `json:"kind"`
	Name      string `json:"name"`
	FilePath  string `json:"file_path"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	Signature string `json:"signature"`
	Language  string `json:"language"`

	// relation
	FromID   string         `json:"from_id"`
	ToID     string         `json:"to_id"`
	Type     string         `json:"type"`
	Evidence []snapEvidence `json:"evidence"`
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
		Symbols:   map[string]*Symbol{},
		In:        map[string][]Edge{},
		Out:       map[string][]Edge{},
		ByFile:    map[string][]string{},
		Relations: map[string]int{},
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
			continue
		}
		switch r.RecordType {
		case "file":
			if r.Path != "" {
				files[r.Path] = true
			}
		case "symbol":
			if r.ID == "" || r.FilePath == "" {
				continue
			}
			s := &Symbol{
				ID:        r.ID,
				Name:      r.Name,
				Kind:      r.Kind,
				File:      r.FilePath,
				Span:      [2]int{r.StartLine, r.EndLine},
				Signature: r.Signature,
				Language:  r.Language,
			}
			// The snapshot carries no test marking, which the Step 0 probe
			// confirmed, so IsTest comes from file and name conventions.
			s.IsTest = IsTestFile(s.File) && looksLikeTestSymbol(s)
			f.Symbols[s.ID] = s
			f.ByFile[s.File] = append(f.ByFile[s.File], s.ID)
			files[s.File] = true
		case "relation":
			if r.FromID == "" || r.ToID == "" {
				continue
			}
			e := Edge{From: r.FromID, To: r.ToID, Relation: r.Type}
			for _, ev := range r.Evidence {
				if ev.Kind == "call_site" && ev.StartLine > 0 {
					e.CallSite = ev.StartLine
					break
				}
			}
			f.Out[e.From] = append(f.Out[e.From], e)
			f.In[e.To] = append(f.In[e.To], e)
			f.Relations[e.Relation]++
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
