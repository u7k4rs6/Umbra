package transcript

import (
	"regexp"
	"sort"
	"strings"
)

// Vocabulary is what a mention can name: the repository's files and the
// symbols in the field. Matching against a closed vocabulary is what keeps a
// mention from inventing evidence out of ordinary prose.
type Vocabulary struct {
	Files   []string
	Symbols []string
}

// tokenRE splits agent prose into things that could be a path or an
// identifier. Backticks, quotes and ordinary punctuation are separators.
var tokenRE = regexp.MustCompile(`[A-Za-z0-9_./-]+`)

// ResolveMentions appends Mention events for paths and symbol names that
// appear in the agent's own words with no tool event on that file.
//
// This is the weakest evidence Umbra has, and it only ever produces penumbra
// with the echo modifier. Path matching accepts a suffix so "service.py" in
// prose matches "app/service.py" in the repository, because an agent rarely
// writes the full path. Symbol matching is exact and case sensitive, so a
// common English word cannot be read as a symbol.
//
// Only the names are recorded. The sentence they came from is never stored.
func (s *Session) ResolveMentions(v Vocabulary) {
	if len(s.texts) == 0 {
		return
	}

	// Files the session actually touched through a tool. A mention adds
	// nothing for those, because a real event is always stronger evidence.
	touched := map[string]bool{}
	for _, e := range s.Events {
		if e.Path != "" {
			touched[e.Path] = true
		}
		for _, p := range e.Paths {
			touched[p] = true
		}
	}

	byBase := map[string][]string{}
	for _, f := range v.Files {
		byBase[baseOf(f)] = append(byBase[baseOf(f)], f)
	}
	symbols := map[string]bool{}
	for _, sym := range v.Symbols {
		if len(sym) >= 3 {
			symbols[sym] = true
		}
	}

	var added []Event
	for _, t := range s.texts {
		files := map[string]bool{}
		syms := map[string]bool{}

		for _, tok := range tokenRE.FindAllString(t.text, -1) {
			// A path mention: match the whole path, or the final segment when
			// it is unambiguous across the repository.
			if strings.Contains(tok, "/") || strings.Contains(tok, ".") {
				for _, f := range v.Files {
					if f == tok || strings.HasSuffix(f, "/"+tok) {
						files[f] = true
					}
				}
				if cands, ok := byBase[tok]; ok && len(cands) == 1 {
					files[cands[0]] = true
				}
			}
			if symbols[tok] {
				syms[tok] = true
			}
		}

		for f := range files {
			if touched[f] {
				delete(files, f)
			}
		}
		if len(files) == 0 && len(syms) == 0 {
			continue
		}
		added = append(added, Event{
			Seq:     t.seq,
			Kind:    Mention,
			Paths:   sortedKeys(files),
			Symbols: sortedKeys(syms),
		})
	}
	if len(added) == 0 {
		return
	}

	s.Events = append(s.Events, added...)
	sort.SliceStable(s.Events, func(i, j int) bool {
		if s.Events[i].Seq != s.Events[j].Seq {
			return s.Events[i].Seq < s.Events[j].Seq
		}
		// A mention shares the sequence number of the text that carried it and
		// sorts after it.
		return s.Events[i].Kind != Mention && s.Events[j].Kind == Mention
	})
}

func baseOf(p string) string {
	if i := strings.LastIndex(p, "/"); i >= 0 {
		return p[i+1:]
	}
	return p
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
