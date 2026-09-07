package main

import (
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/report"
)

// The wiring half of the fix. The parser can read a count verify did not turn
// into names, and the renderers refuse to call that a clean audit, but neither
// matters unless the number reaches the report.

func TestNoteUnnamedCarriesTheCountOntoTheReport(t *testing.T) {
	a := &report.Analysis{}
	// 51 counted, 20 named, which is the shape of a capped verify list.
	v := graph.Verdict{FailingCount: 51, NewlyFailing: make([]string, 20)}
	noteUnnamed(a, v)
	if a.Execution.UnnamedFailures != 31 {
		t.Fatalf("unnamed failures = %d, want 31", a.Execution.UnnamedFailures)
	}
	if a.Execution.SweepNamedEverything() {
		t.Fatal("a report with 31 unnamed failures must not claim it named everything")
	}
}

// The probe run and the sweep both report. Neither may lower a count the other
// established, or a small probe verdict would erase what the full sweep found.
func TestNoteUnnamedKeepsTheLargerCount(t *testing.T) {
	a := &report.Analysis{}
	noteUnnamed(a, graph.Verdict{FailingCount: 51, NewlyFailing: make([]string, 20)})
	noteUnnamed(a, graph.Verdict{FailingCount: 3, NewlyFailing: make([]string, 3)})
	if a.Execution.UnnamedFailures != 31 {
		t.Fatalf("unnamed failures = %d, want the larger count 31", a.Execution.UnnamedFailures)
	}
}

func TestNoteUnnamedStaysZeroWhenEverythingWasNamed(t *testing.T) {
	a := &report.Analysis{}
	noteUnnamed(a, graph.Verdict{FailingCount: 15, NewlyFailing: make([]string, 15)})
	if a.Execution.UnnamedFailures != 0 {
		t.Fatalf("unnamed failures = %d, want 0", a.Execution.UnnamedFailures)
	}
	if !a.Execution.SweepNamedEverything() {
		t.Fatal("an untruncated run named everything")
	}
}

// Gating on the length of a list verify said was capped is the same mistake
// the sweep line made, so both conditions consult the count.
func TestFailOnFiresOnFailuresVerifyCouldNotName(t *testing.T) {
	for _, cond := range []string{"failure", "leak"} {
		a := &report.Analysis{}
		a.Execution.UnnamedFailures = 31
		if got := failOn(&Options{FailOn: cond}, a); got != ExitFailOn {
			t.Fatalf("--fail-on %s exited %d with 31 unnamed failures, want %d", cond, got, ExitFailOn)
		}
	}
}

// A suite-level pass with no per-test ids reports no failures at all. That is
// a degraded run, not a regression, and it must not exit 2.
func TestFailOnDoesNotFireOnADegradedRunWithNoFailures(t *testing.T) {
	for _, cond := range []string{"failure", "leak"} {
		a := &report.Analysis{}
		a.Execution.Degraded = true
		if got := failOn(&Options{FailOn: cond}, a); got != ExitOK {
			t.Fatalf("--fail-on %s exited %d on a degraded but passing run, want %d", cond, got, ExitOK)
		}
	}
}

// Defects 1 and 3, the parts that live in the pipeline rather than in the
// graph package.

func TestUnresolvedReasonAlwaysSaysSomething(t *testing.T) {
	if got := unresolvedReason(graph.Source{Unresolved: graph.UnresolvedNoMatch}); got != graph.UnresolvedNoMatch {
		t.Fatalf("reason = %q, want the recorded one", got)
	}
	// A source that reached here without a recorded reason is still named
	// rather than dropped, because a silent skip is the defect.
	if got := unresolvedReason(graph.Source{}); got == "" {
		t.Fatal("a source with no recorded reason must still produce a sentence")
	}
}

// A changed entity Graph could only name as a whole file has no symbol-level
// dependents to report. Counting it is what lets the report say so instead of
// printing the sentence reserved for "the graph looked and found nothing",
// which is defect 3 in the writeup.
func TestModuleGranularityCountsWholeFileChanges(t *testing.T) {
	sources := []graph.Source{
		{Name: "src/click/formatting.py", Kind: "module", File: "src/click/formatting.py"},
		{Name: "wrap_text", Kind: "function", File: "src/click/formatting.py"},
		{Name: "app/api.py", Kind: "file", File: "app/api.py"},
	}
	if got := moduleGranularity(sources); got != 2 {
		t.Fatalf("module granularity count = %d, want 2", got)
	}
	if got := moduleGranularity(sources[1:2]); got != 0 {
		t.Fatalf("a symbol-level source counted as %d, want 0", got)
	}
}

// The guard that decides between an inconclusive audit and an incomplete list.
// Without a test here, removing the guard from sweep() broke nothing, which is
// the definition of a check that is not yet a check.
func TestSweepIsInconclusiveOnlyWhenThereIsNothingToCount(t *testing.T) {
	cases := []struct {
		name    string
		v       graph.Verdict
		changed []string
		want    bool
	}{
		{"degraded, no ids, no count", graph.Verdict{Degraded: true}, nil, true},
		{"degraded, no ids, but a count is known",
			graph.Verdict{Degraded: true, FailingCount: 51}, nil, false},
		{"degraded, but ids came back", graph.Verdict{Degraded: true}, []string{"a"}, false},
		{"not degraded, nothing changed", graph.Verdict{}, nil, false},
		{"a clean run", graph.Verdict{}, []string{"a"}, false},
	}
	for _, c := range cases {
		if got := sweepIsInconclusive(c.v, c.changed); got != c.want {
			t.Fatalf("%s: sweepIsInconclusive = %v, want %v", c.name, got, c.want)
		}
	}
}
