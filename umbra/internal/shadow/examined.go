package shadow

import (
	"sort"

	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// Exposure is one moment the session touched a file.
type Exposure struct {
	Seq   int
	Kind  transcript.Kind
	Range *[2]int
}

// Examined is what the session looked at, indexed for classification.
type Examined struct {
	// ByFile lists every exposure on a file, in sequence order.
	ByFile map[string][]Exposure
	// BySymbol lists mentions of a symbol by name.
	BySymbol map[string][]Exposure
	// Cut is the sequence number of the first edit to a source file. Zero
	// means the session never edited a source.
	Cut int
	// HasCut reports whether the cut is real.
	HasCut bool
	// LastSourceTouch is the sequence of the last edit to a source file. The
	// agent's account of the change is whatever it said after that.
	LastSourceTouch int
	// Any reports whether the transcript carried evidence at all. When false
	// every node is unknown and the header says the examined set is
	// unavailable.
	Any bool
}

// BuildExamined turns a parsed session into the examined set.
//
// Mentions are indexed by both the paths and the symbol names they carried,
// because the agent may name either.
func BuildExamined(s *transcript.Session, sourceFiles map[string]bool) *Examined {
	e := &Examined{
		ByFile:   map[string][]Exposure{},
		BySymbol: map[string][]Exposure{},
	}
	if s == nil {
		return e
	}
	e.Any = s.HasExposure()
	if cut, ok := s.FirstEditSeq(sourceFiles); ok {
		e.Cut, e.HasCut = cut, true
	}
	for _, ev := range s.Events {
		if ev.Kind == transcript.Edit && sourceFiles[ev.Path] && ev.Seq > e.LastSourceTouch {
			e.LastSourceTouch = ev.Seq
		}
	}

	for _, ev := range s.Events {
		switch ev.Kind {
		case transcript.Read, transcript.Edit:
			if ev.Path == "" {
				continue
			}
			e.ByFile[ev.Path] = append(e.ByFile[ev.Path], Exposure{Seq: ev.Seq, Kind: ev.Kind, Range: ev.Range})
		case transcript.Grep, transcript.Glob, transcript.ResultFile, transcript.Mention:
			for _, p := range ev.Paths {
				e.ByFile[p] = append(e.ByFile[p], Exposure{Seq: ev.Seq, Kind: ev.Kind})
			}
			if ev.Kind == transcript.Mention {
				for _, sym := range ev.Symbols {
					e.BySymbol[sym] = append(e.BySymbol[sym], Exposure{Seq: ev.Seq, Kind: ev.Kind})
				}
			}
		}
	}
	for p := range e.ByFile {
		list := e.ByFile[p]
		sort.SliceStable(list, func(i, j int) bool { return list[i].Seq < list[j].Seq })
		e.ByFile[p] = list
	}
	return e
}

// Counts summarises the examined set for the header.
func (e *Examined) Counts() map[string]int {
	out := map[string]int{}
	for _, list := range e.ByFile {
		for _, x := range list {
			out[x.Kind.String()]++
		}
	}
	return out
}
