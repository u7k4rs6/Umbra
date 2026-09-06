package report

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

func packetOf(t *testing.T, a *Analysis) string {
	t.Helper()
	var b strings.Builder
	if err := Packet(&b, a); err != nil {
		t.Fatalf("Packet: %v", err)
	}
	return b.String()
}

// The packet is five sections a reviewer reads in a minute.
func TestPacketHasTheFiveSections(t *testing.T) {
	out := packetOf(t, mkAnalysis())
	for _, want := range []string{
		"## 1. What changed",
		"## 2. What was examined",
		"## 3. Ranked shadow",
		"## 4. Tests reaching the shadow",
		"## 5. Results",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the packet is missing %q", want)
		}
	}
}

// The header and the reproduce command top and tail it, so a reader can check
// any line without trusting the file.
func TestPacketCarriesTheReproduceCommand(t *testing.T) {
	a := mkAnalysis()
	a.TestRunner = "pytest -v"
	a.Run = "shadow"
	out := packetOf(t, a)
	if !strings.Contains(out, "## Reproduce") {
		t.Fatal("the packet must say how to reproduce it")
	}
	if !strings.Contains(out, "entire umbra b20f84567474") {
		t.Fatalf("the reproduce command must name the checkpoint:\n%s", out)
	}
	if !strings.Contains(out, `--test "pytest -v"`) {
		t.Fatal("the reproduce command must carry the runner")
	}
}

// Every coined word sits next to its plain meaning.
func TestPacketExplainsEveryCoinedWord(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Modifiers = []string{"echo", "far field", "fault line", "beacon", "scar", "test", "co-change only"}
	a.Nodes[0].State = shadow.Umbra
	out := packetOf(t, a)
	for _, pair := range [][2]string{
		{"echo", "named in the agent's own words"},
		{"far field", "across a package boundary"},
		{"fault line", "inside error handling"},
		{"beacon", "SAFETY, CRITICAL or INVARIANT"},
		{"scar", "recent fix commits"},
		{"co-change only", "historically changes with the source"},
	} {
		if !strings.Contains(out, pair[0]) {
			t.Errorf("the packet should mention %q", pair[0])
		}
		if !strings.Contains(out, pair[1]) {
			t.Errorf("%q appears without its plain meaning %q", pair[0], pair[1])
		}
	}
}

func TestPacketPrintsFactorsSoTheScoreCanBeRecomputed(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Factors = map[string]float64{
		"relation": 3, "dependents": 2, "state": 1, "farfield": 1.5, "faultline": 1, "test": 3,
	}
	out := packetOf(t, a)
	if !strings.Contains(out, "factors: relation 3, dependents 2, state 1, farfield 1.5, faultline 1, test 3") {
		t.Fatalf("the factor line is missing or reordered:\n%s", out)
	}
}

func TestPacketQuotesTheSessionButSaysItIsUnverified(t *testing.T) {
	out := packetOf(t, mkAnalysis())
	if !strings.Contains(out, "Checked the callers and updated them. All tests pass.") {
		t.Fatal("the session's own sentence should appear")
	}
	if !strings.Contains(out, "displayed, never checked") {
		t.Fatal("the packet must say the sentence is not verified")
	}
}

func TestPacketReportsLeaksWithReasons(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Execution.Sweep = true
	a.Execution.Selected = []string{"tests/a.py::test_one"}
	a.Execution.Leaks = []Leak{{Test: "tests/b.py::test_two", Reason: "path exists at depth 4 via render_lines"}}
	out := packetOf(t, a)
	if !strings.Contains(out, "**1 leak**") {
		t.Fatalf("expected a singular leak count:\n%s", out)
	}
	if !strings.Contains(out, "path exists at depth 4 via render_lines") {
		t.Fatal("every leak must carry its reason")
	}
}

func TestPacketSaysWhenTheSweepWasSkipped(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = false
	a.Execution.Sweep = false
	out := packetOf(t, a)
	if !strings.Contains(out, "sweep was skipped") {
		t.Fatalf("expected the unaudited notice:\n%s", out)
	}
}

func TestPacketSaysWhenNothingIsInShadow(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("handle_order", "app/api.py", 14, shadow.Lit, shadow.TierNone, 0)}
	a.Summary = shadow.Summarize(a.Nodes)
	out := packetOf(t, a)
	if !strings.Contains(out, "Nothing is in shadow") {
		t.Fatalf("expected the all-clear:\n%s", out)
	}
}

func TestPacketSaysWhenEverythingIsUnknown(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("x", "app/a.py", 1, shadow.Unknown, shadow.TierNone, 1)}
	a.Summary = shadow.Summarize(a.Nodes)
	out := packetOf(t, a)
	if !strings.Contains(out, "examined set is unavailable") {
		t.Fatalf("expected the unavailable notice:\n%s", out)
	}
}

// The packet lists at most twenty nodes and says how many it left out.
func TestPacketCapsTheShadowList(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = nil
	for i := 0; i < 25; i++ {
		a.Nodes = append(a.Nodes, mkNode("n", "app/a.py", i+1, shadow.Umbra, shadow.TierNone, float64(25-i)))
	}
	a.Summary = shadow.Summarize(a.Nodes)
	out := packetOf(t, a)
	if !strings.Contains(out, "and 5 more.") {
		t.Fatalf("expected the packet to say what it left out:\n%s", out)
	}
	if strings.Count(out, "### 21.") != 0 {
		t.Fatal("the packet must stop at twenty")
	}
}
