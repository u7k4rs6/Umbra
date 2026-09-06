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
