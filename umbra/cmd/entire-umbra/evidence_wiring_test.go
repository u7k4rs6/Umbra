package main

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// The evidence tiers are computed in shadow and rendered in report, and both
// sides have their own tests. Neither side notices if nothing joins them. This
// file tests the join: that the run's own completeness actually reaches the
// classifier, so a clean-looking node in a degraded run stops being confirmed.

func degradedAnalysis() *report.Analysis {
	return &report.Analysis{Graph: report.GraphEvidence{
		CompletenessLevel: "partial",
		SawSummary:        true,
		InScope:           true,
		Files:             10,
		ParsedFiles:       4,
	}}
}

func TestADegradedRunReachesTheClassifier(t *testing.T) {
	in := evidenceInput(degradedAnalysis())
	if !in.RunDegraded {
		t.Fatal("a degraded in-scope run did not mark its evidence input degraded")
	}
	if in.DegradedWhy == "" {
		t.Fatal("the run was marked degraded with no reason for a reader")
	}
	if !strings.Contains(in.DegradedWhy, "4 of 10 files parsed") {
		t.Fatalf("the reason does not say what was incomplete: %q", in.DegradedWhy)
	}

	// The node itself is clean: every hop resolved to a definition. Only the
	// run's own state should be able to demote it.
	n := &shadow.Node{Resolution: "exact"}
	ev, why := shadow.ClassifyEvidence(n, in)
	if ev != shadow.NeedsVerification {
		t.Fatalf("a clean node in a degraded run classified as %s, want %s", ev, shadow.NeedsVerification)
	}
	if why != in.DegradedWhy {
		t.Fatalf("the node's reason %q is not the run's reason %q", why, in.DegradedWhy)
	}
}

func TestDegradationOutsideTheChangedFilesDoesNotDemoteAnything(t *testing.T) {
	a := degradedAnalysis()
	a.Graph.InScope = false
	in := evidenceInput(a)
	if in.RunDegraded {
		t.Fatal("degradation the graph said cannot affect this field still demoted the run")
	}
	ev, _ := shadow.ClassifyEvidence(&shadow.Node{Resolution: "exact"}, in)
	if ev != shadow.Confirmed {
		t.Fatalf("out-of-scope degradation demoted a clean node to %s", ev)
	}
}

func TestAHealthyRunCarriesNoDoubt(t *testing.T) {
	a := &report.Analysis{Graph: report.GraphEvidence{
		CompletenessLevel: "complete", SawSummary: true, InScope: true,
		Files: 10, ParsedFiles: 10,
	}}
	in := evidenceInput(a)
	if in.RunDegraded || in.DegradedWhy != "" {
		t.Fatalf("a healthy run reported doubt: %+v", in)
	}
}
