package transcript

import "testing"

// A known-positive for the failed read.
//
// The parser records a Read when the tool call is made, which is before the
// result comes back. A read of a path that does not exist therefore used to
// land in the examined set exactly like a successful one, and the node it
// named came out lit. That is the worst direction for this tool to be wrong
// in: it reports code as inspected that the session never saw.
//
// The fixture holds one read that succeeded and one that returned
// is_error. Only the successful one may survive.

func TestFailedReadDoesNotCountAsInspected(t *testing.T) {
	s := load(t, "failed-read.jsonl")

	if _, ok := firstOf(s, Read, "app/service.py"); !ok {
		t.Error("the successful read was dropped; the fix must not remove real evidence")
	}
	if ev, ok := firstOf(s, Read, "app/missing.py"); ok {
		t.Errorf("a read that returned is_error was kept as evidence of inspection: %+v", ev)
	}

	reads := 0
	for _, e := range s.Events {
		if e.Kind == Read {
			reads++
		}
	}
	if reads != 1 {
		t.Errorf("expected exactly one surviving read, got %d", reads)
	}
}

// The same fixture, through the examined set, which is what the classifier
// actually consults. A node in the failed file must not be lit.
func TestFailedReadLeavesItsFileUnexamined(t *testing.T) {
	s := load(t, "failed-read.jsonl")

	var sawGood, sawBad bool
	for _, e := range s.Events {
		if e.Kind != Read {
			continue
		}
		switch e.Path {
		case "app/service.py":
			sawGood = true
		case "app/missing.py":
			sawBad = true
		}
	}
	if !sawGood {
		t.Error("app/service.py was read successfully and must be examined")
	}
	if sawBad {
		t.Error("app/missing.py failed to read and must stay in shadow")
	}
}
