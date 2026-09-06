package report

import (
	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// Analysis is everything a renderer needs. It is built once and handed to
// every output format, so the table, the JSON, the HTML and the packet can
// never disagree with each other.
type Analysis struct {
	Version string

	CheckpointID string
	Commit       string
	Parent       string
	Agent        string
	Adapter      string
	Route        string
	SessionIDs   []string
	Notes        []string

	// SessionSaid is the agent's own account of itself: Entire's stored
	// summary when there is one, otherwise the last assistant sentence before
	// the commit. It is displayed, never verified.
	SessionSaid string
	// SessionSaidFrom names where the sentence came from, so a reader knows.
	SessionSaidFrom string

	Depth            int
	TestRunner       string
	Run              string
	Audit            bool
	History          bool
	Snippets         bool
	RelationsUsed    []string
	RelationsIgnored []string
	Channels         map[string]bool

	Sources []graph.Source
	Nodes   []*shadow.Node
	Summary shadow.Summary

	Timeline []TimelineEvent
	Cut      int
	HasCut   bool
	// Coverage is how the session worked: how many of its tool events were
	// file reads and edits, and how many were shell commands.
	Coverage transcript.Coverage

	Execution   Execution
	Layout      *Layout
	Limitations []string
	Commands    []string
}

// TimelineEvent is one moment, reduced to what a report may carry: a kind, a
// sequence number, a timestamp, paths and symbol names. Never prose.
type TimelineEvent struct {
	Seq     int      `json:"seq"`
	TS      string   `json:"ts,omitempty"`
	Kind    string   `json:"kind"`
	Path    string   `json:"path,omitempty"`
	Range   *[2]int  `json:"range,omitempty"`
	Paths   []string `json:"paths,omitempty"`
	Symbols []string `json:"symbols,omitempty"`
	Cmd     string   `json:"cmd,omitempty"`
}

// Execution is what the test run found.
type Execution struct {
	Selected    []string `json:"selected"`
	NotRunnable []string `json:"not_runnable,omitempty"`
	NewFailures []string `json:"new_failures"`
	Fixed       []string `json:"fixed"`
	PreExisting []string `json:"pre_existing,omitempty"`
	Sweep       bool     `json:"sweep"`
	SweepCut    bool     `json:"sweep_cut_short,omitempty"`
	Leaks       []Leak   `json:"leaks"`
	// Verdict is the line verify printed, kept so the report can quote the
	// tool rather than paraphrase it.
	Verdict string `json:"verdict,omitempty"`
	// Degraded is set when verify could not parse per-test ids, which happens
	// when the runner is not verbose.
	Degraded bool `json:"degraded,omitempty"`
}

// Leak is a test the full sweep found that the selection missed.
type Leak struct {
	Test   string `json:"test"`
	Reason string `json:"reason"`
}

// CrackedProbes are the selected probes that newly failed.
//
// NewFailures holds every test the run found newly failing, including ones the
// sweep turned up that selection never chose. Those are leaks, and reporting
// one as a cracked probe names a test that was not among the probes. The
// distinction only shows when a leak actually fails, which is what the leak
// scenario now produces.
func (e Execution) CrackedProbes() []string {
	selected := map[string]bool{}
	for _, id := range e.Selected {
		selected[id] = true
	}
	var out []string
	for _, id := range e.NewFailures {
		if selected[id] {
			out = append(out, id)
		}
	}
	return out
}

// SweepOnlyFailures are the newly failing tests that selection never ran.
func (e Execution) SweepOnlyFailures() []string {
	selected := map[string]bool{}
	for _, id := range e.Selected {
		selected[id] = true
	}
	var out []string
	for _, id := range e.NewFailures {
		if !selected[id] {
			out = append(out, id)
		}
	}
	return out
}

// Shadowed returns the nodes that are not lit, in docket order.
func (a *Analysis) Shadowed() []*shadow.Node {
	var out []*shadow.Node
	for _, n := range a.Nodes {
		if n.Shadowed() {
			out = append(out, n)
		}
	}
	return out
}
