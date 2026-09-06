package report

import (
	"errors"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
)

// ErrUnsealed is what every renderer returns for a zero Sealed. It cannot
// happen through the exported API, since Seal is the only constructor, but a
// zero value inside this package would otherwise render an empty report and
// look like a working one.
var ErrUnsealed = errors.New("the analysis was never sealed, so nothing can be written")

// Sealed is an Analysis that has been through the scrubber.
//
// Every renderer in this package takes a Sealed rather than an *Analysis, and
// the only way to obtain one is Seal. The field is unexported, so a caller in
// another package cannot construct one, and a caller in this package cannot
// serialize an analysis by forgetting to scrub it. That is the whole point of
// the type.
//
// Five leaks in this project have had the same shape: an output path that
// wrote without going through the scrubber. Each was fixed where it was found,
// which fixed that path and left the next one open. A value that only the
// scrubbing path can produce closes the shape instead of the instance.
type Sealed struct {
	a *Analysis
}

// Analysis returns the scrubbed analysis. It is here for the renderers and for
// tests that need to look at what sealing did.
func (s Sealed) Analysis() *Analysis { return s.a }

// Valid reports whether the value came from Seal. A zero Sealed has no
// analysis, and every renderer refuses one.
func (s Sealed) Valid() bool { return s.a != nil }

// Seal scrubs every string in the analysis that could carry text from the
// machine the session ran on, then builds the layout from the scrubbed values
// so a node label cannot carry what the node name no longer does.
//
// A nil scrubber means the default one for this machine, never no scrubbing.
// There is deliberately no way to ask for less: the caps and replacements
// SECURITY_AND_ACCESS.md lists are the floor, not an option.
//
// Sealing the same analysis twice is safe. Every replacement is idempotent and
// every cap is applied to an already capped string.
func Seal(a *Analysis, s *Scrubber) Sealed {
	if a == nil {
		return Sealed{}
	}
	if s == nil {
		s = NewScrubber("")
	}

	a.Agent = s.Clean(a.Agent)
	a.Adapter = s.Clean(a.Adapter)
	a.Route = s.Clean(a.Route)
	// The runner prefix comes from a flag or from .umbra.json and is printed
	// in the reproduce section, so it carries a path whenever the user gave
	// one.
	a.TestRunner = s.Clean(a.TestRunner)
	a.SessionSaid = s.Sentence(a.SessionSaid)
	a.SessionSaidFrom = s.Clean(a.SessionSaidFrom)

	a.SessionIDs = cleanAll(s, a.SessionIDs)
	a.Notes = cleanAll(s, a.Notes)
	a.Limitations = cleanAll(s, a.Limitations)
	// Uncapped: SECURITY_AND_ACCESS.md says a report names the exact commands
	// run so a reader can reproduce a line without trusting the file. A capped
	// command cannot be run.
	a.Commands = cleanAll(s, a.Commands)
	a.RelationsUsed = cleanAll(s, a.RelationsUsed)
	a.RelationsIgnored = cleanAll(s, a.RelationsIgnored)

	for i := range a.Timeline {
		e := &a.Timeline[i]
		e.Kind = s.Clean(e.Kind)
		e.Path = s.Clean(e.Path)
		e.Paths = cleanAll(s, e.Paths)
		e.Symbols = cleanAll(s, e.Symbols)
		e.Cmd = s.Command(e.Cmd)
	}

	x := &a.Execution
	x.Selected = cleanAll(s, x.Selected)
	x.NotRunnable = cleanAll(s, x.NotRunnable)
	x.NewFailures = cleanAll(s, x.NewFailures)
	x.Fixed = cleanAll(s, x.Fixed)
	x.PreExisting = cleanAll(s, x.PreExisting)
	x.Verdict = s.Clean(x.Verdict)
	for i := range x.Leaks {
		x.Leaks[i].Test = s.Clean(x.Leaks[i].Test)
		x.Leaks[i].Reason = s.Clean(x.Leaks[i].Reason)
	}

	for i := range a.Sources {
		src := &a.Sources[i]
		src.Symbol = s.Clean(src.Symbol)
		src.Name = s.Clean(src.Name)
		src.File = s.Clean(src.File)
		src.Change = s.Clean(src.Change)
		src.OldSignature = s.Clean(src.OldSignature)
		src.NewSignature = s.Clean(src.NewSignature)
	}

	for _, n := range a.Nodes {
		if n == nil {
			continue
		}
		n.Symbol = cleanSymbol(s, n.Symbol)
		n.Sources = cleanAll(s, n.Sources)
		n.Relation = s.Clean(n.Relation)
		n.Path = cleanAll(s, n.Path)
		n.Modifiers = cleanAll(s, n.Modifiers)
		n.Beacon = s.Clean(n.Beacon)
		n.Failure = s.Excerpt(n.Failure)
		for j := range n.Tests {
			n.Tests[j].ID = s.Clean(n.Tests[j].ID)
		}
	}

	// The layout carries node names and file names as label text. Building it
	// after the scrub rather than before is the difference between a scrubbed
	// report and a scrubbed report with an unscrubbed picture on it.
	a.Layout = BuildLayout(a)
	return Sealed{a: a}
}

// cleanSymbol returns a scrubbed copy. The symbols come from the graph field,
// which other parts of the analysis still point at, so this does not write
// through the pointer.
func cleanSymbol(s *Scrubber, sym *graph.Symbol) *graph.Symbol {
	if sym == nil {
		return nil
	}
	out := *sym
	out.ID = s.Clean(out.ID)
	out.Name = s.Clean(out.Name)
	out.Kind = s.Clean(out.Kind)
	out.File = s.Clean(out.File)
	out.Signature = s.Clean(out.Signature)
	out.Language = s.Clean(out.Language)
	return &out
}

func cleanAll(s *Scrubber, list []string) []string {
	for i, v := range list {
		list[i] = s.Clean(v)
	}
	return list
}
