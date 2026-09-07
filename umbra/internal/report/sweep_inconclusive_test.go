package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// A known-positive for the sweep's false zero.
//
// The sweep used to print "0 leaks" whether it had compared every test in the
// suite and found nothing, or had compared nothing at all because the runner
// printed no per-test ids. Those are opposite findings and they read
// identically, which makes the strongest claim the report can make out of the
// weakest evidence it has.
//
// Every renderer is checked, because the count reaches a reader through four
// surfaces and fixing one of them would leave the claim standing in the other
// three.

// inconclusiveAnalysis is a sweep that ran and could compare nothing.
func inconclusiveAnalysis() *Analysis {
	a := mkAnalysis()
	a.Run = "all"
	a.Audit = true
	a.Execution.Sweep = true
	a.Execution.Degraded = true
	a.Execution.SweepInconclusive = true
	a.Execution.SweepInconclusiveReason = "the runner printed no per-test ids, so the sweep could not compare any test against the baseline; add -v to the runner"
	a.Execution.Leaks = nil
	return a
}

// cleanSweepAnalysis is a sweep that compared ids and genuinely found nothing.
func cleanSweepAnalysis() *Analysis {
	a := mkAnalysis()
	a.Run = "all"
	a.Audit = true
	a.Execution.Sweep = true
	a.Execution.Degraded = false
	a.Execution.SweepInconclusive = false
	a.Execution.Leaks = nil
	return a
}

func renderAll(t *testing.T, a *Analysis) map[string]string {
	t.Helper()
	sd := Seal(a, nil)
	out := map[string]string{}

	var tb bytes.Buffer
	if err := Table(&tb, sd, TableOptions{UTF8: true}); err != nil {
		t.Fatalf("table: %v", err)
	}
	out["terminal"] = tb.String()

	var pk bytes.Buffer
	if err := Packet(&pk, sd); err != nil {
		t.Fatalf("packet: %v", err)
	}
	out["packet"] = pk.String()

	var hb bytes.Buffer
	if err := HTML(&hb, sd); err != nil {
		t.Fatalf("html: %v", err)
	}
	out["html"] = hb.String()

	var jb bytes.Buffer
	if err := WriteJSON(&jb, sd); err != nil {
		t.Fatalf("json: %v", err)
	}
	out["json"] = jb.String()
	return out
}

// The claim the fix exists to prevent: a zero leak count from a sweep that
// compared nothing.
var zeroLeakClaims = []string{"0 leaks", "0 leak", "**0 leaks**", "full sweep, 0"}

func TestInconclusiveSweepNeverClaimsZeroLeaks(t *testing.T) {
	for surface, text := range renderAll(t, inconclusiveAnalysis()) {
		if !strings.Contains(strings.ToLower(text), "inconclusive") {
			t.Errorf("%s: does not say the audit was inconclusive", surface)
		}
		for _, claim := range zeroLeakClaims {
			if strings.Contains(text, claim) {
				t.Errorf("%s: claims %q from a sweep that compared nothing", surface, claim)
			}
		}
	}
}

// The reason has to travel with the finding, or the reader is left guessing
// what went wrong.
func TestInconclusiveSweepCarriesItsReason(t *testing.T) {
	out := renderAll(t, inconclusiveAnalysis())
	for _, surface := range []string{"terminal", "packet", "json"} {
		if !strings.Contains(out[surface], "no per-test ids") {
			t.Errorf("%s: carries no reason for the inconclusive audit", surface)
		}
	}
	var doc struct {
		Execution struct {
			SweepInconclusive bool   `json:"sweep_inconclusive"`
			Reason            string `json:"sweep_inconclusive_reason"`
		} `json:"execution"`
	}
	if err := json.Unmarshal([]byte(out["json"]), &doc); err != nil {
		t.Fatalf("json: %v", err)
	}
	if !doc.Execution.SweepInconclusive {
		t.Error("json: sweep_inconclusive is not set, so a machine consumer cannot tell")
	}
	if doc.Execution.Reason == "" {
		t.Error("json: sweep_inconclusive_reason is empty")
	}
}

// The other half of the same proof. A sweep that really did compare every test
// and found nothing must still say zero, or the fix would have replaced one
// wrong answer with another.
func TestCleanSweepStillReportsZeroLeaks(t *testing.T) {
	out := renderAll(t, cleanSweepAnalysis())
	if !strings.Contains(out["terminal"], "0 leaks") {
		t.Errorf("terminal: a clean sweep must still report 0 leaks:\n%s", out["terminal"])
	}
	if !strings.Contains(out["packet"], "**0 leaks**") {
		t.Error("packet: a clean sweep must still report 0 leaks")
	}
	if !strings.Contains(out["html"], "full sweep, 0 leaks") {
		t.Error("html: a clean sweep must still report 0 leaks")
	}
	for surface, text := range out {
		if strings.Contains(strings.ToLower(text), "inconclusive") {
			t.Errorf("%s: a clean sweep must not be called inconclusive", surface)
		}
	}
}

// Inconclusive and incomplete are different findings and neither may swallow
// the other. Inconclusive means the sweep compared nothing at all. Incomplete
// means verify counted failures it could not name, which still has a number
// and must still print one.
func TestInconclusiveDoesNotSwallowAnIncompleteList(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"a"}
	a.Execution.Sweep = true
	// Verify gave a count and named none of them. Degraded, no ids, but the
	// count is real, so this is incomplete rather than inconclusive.
	a.Execution.Degraded = true
	a.Execution.UnnamedFailures = 51

	out := render(t, a, TableOptions{UTF8: true})
	if strings.Contains(out, "audit inconclusive") {
		t.Fatalf("a run with a known count of 51 is not inconclusive:\n%s", out)
	}
	if !strings.Contains(out, "51 further failing test(s)") {
		t.Fatalf("the count verify reported must still be printed:\n%s", out)
	}
	if strings.Contains(out, "sweep   full suite  0 leaks\n") {
		t.Fatalf("and it must not read as a clean audit either:\n%s", out)
	}
}

// The converse: degraded with no count at all is inconclusive, and prints no
// number of any kind.
func TestDegradedWithNoCountIsInconclusive(t *testing.T) {
	a := mkAnalysis()
	a.Run = "shadow"
	a.Audit = true
	a.Execution.Selected = []string{"a"}
	a.Execution.Sweep = true
	a.Execution.SweepInconclusive = true
	a.Execution.SweepInconclusiveReason = "the runner printed no per-test ids"

	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "audit inconclusive") {
		t.Fatalf("expected the inconclusive line:\n%s", out)
	}
	for _, forbidden := range []string{"0 leaks", "0 leak", "further failing test(s)"} {
		if strings.Contains(out, forbidden) {
			t.Fatalf("an inconclusive sweep printed %q:\n%s", forbidden, out)
		}
	}
}
