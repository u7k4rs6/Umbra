package report

import "testing"

// Known-positive for the cracked-probe count.
//
// The count once named a test that selection never ran. The distinction only
// shows when a leak actually fails, which is why it went unnoticed: every
// earlier scenario had newly failing tests that were all selected, so a
// version that ignored the selection entirely gave the same answer.
func TestCrackedProbesNeverCountsAFailureOutsideTheSelection(t *testing.T) {
	x := Execution{
		Selected: []string{
			"tests/test_service.py::test_rounding",
			"tests/test_service.py::test_currency",
		},
		NewFailures: []string{
			"tests/test_service.py::test_rounding",
			"tests/test_report.py::test_receipt_total",
		},
		Sweep: true,
		Leaks: []Leak{{
			Test:   "tests/test_report.py::test_receipt_total",
			Reason: "path exists at depth 4 via render_lines",
		}},
	}

	cracked := x.CrackedProbes()
	if len(cracked) != 1 || cracked[0] != "tests/test_service.py::test_rounding" {
		t.Fatalf("cracked probes = %v, want only the failure that was selected", cracked)
	}
	for _, id := range cracked {
		if id == "tests/test_report.py::test_receipt_total" {
			t.Error("a sweep-only failure was counted as a cracked probe")
		}
	}

	leaked := x.SweepOnlyFailures()
	if len(leaked) != 1 || leaked[0] != "tests/test_report.py::test_receipt_total" {
		t.Fatalf("sweep-only failures = %v, want the one selection never ran", leaked)
	}
}

// The reverse: when every failure was selected, none of them is a leak. A
// version that reported the whole failure list as leaks would pass the test
// above and fail this one.
func TestNoLeakWhenEveryFailureWasSelected(t *testing.T) {
	x := Execution{
		Selected:    []string{"a::x", "b::y"},
		NewFailures: []string{"a::x", "b::y"},
		Sweep:       true,
	}
	if got := x.CrackedProbes(); len(got) != 2 {
		t.Fatalf("cracked probes = %v, want both", got)
	}
	if got := x.SweepOnlyFailures(); len(got) != 0 {
		t.Fatalf("sweep-only failures = %v, want none", got)
	}
}
