package graph

import (
	"context"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Every string below is real output captured from `entire graph verify` in
// the Step 0 probe.

func TestParseVerdictRegression(t *testing.T) {
	out := `NEWLY FAILING (3): tests/test_service.py::test_empty_is_zero, tests/test_service.py::test_negative, tests/test_service.py::test_rounding
VERDICT: REGRESSION in 3 tests: tests/test_service.py::test_empty_is_zero, tests/test_service.py::test_negative, tests/test_service.py::test_rounding`

	v := ParseVerdict(out)
	if len(v.NewlyFailing) != 3 {
		t.Fatalf("newly failing = %v", v.NewlyFailing)
	}
	if v.NewlyFailing[0] != "tests/test_service.py::test_empty_is_zero" {
		t.Fatalf("first id = %q", v.NewlyFailing[0])
	}
	if !strings.HasPrefix(v.Line, "REGRESSION in 3 tests") {
		t.Fatalf("verdict line = %q", v.Line)
	}
	if v.Degraded {
		t.Fatal("a parsed run is not degraded")
	}
}

func TestParseVerdictNoEffect(t *testing.T) {
	v := ParseVerdict("VERDICT: NO EFFECT the target tests behave exactly as before your edit.")
	if len(v.NewlyFailing) != 0 {
		t.Fatalf("newly failing = %v", v.NewlyFailing)
	}
	if !strings.HasPrefix(v.Line, "NO EFFECT") {
		t.Fatalf("verdict line = %q", v.Line)
	}
}

// pytest -q prints no per-test ids, so verify degrades to an exit code. The
// report must say so rather than claim every test passed.
func TestParseVerdictDegradedBaseline(t *testing.T) {
	out := "BASELINE RECORDED: /tmp/base.json (exit 0; output format not recognised, so the baseline is exit-code only)"
	v := ParseVerdict(out)
	if !v.Degraded {
		t.Fatal("expected the degraded flag")
	}
}

func TestParseVerdictParsedBaselineIsNotDegraded(t *testing.T) {
	out := "BASELINE RECORDED: /tmp/base.json (pytest; 14 passing, 0 failing, exit 0)"
	if ParseVerdict(out).Degraded {
		t.Fatal("a parsed baseline is not degraded")
	}
}

func TestParseVerdictNewlyPassingAndPreExisting(t *testing.T) {
	out := `NEWLY PASSING (1): tests/a.py::test_one
PRE-EXISTING FAILURES (2): tests/b.py::test_two, tests/b.py::test_three`
	v := ParseVerdict(out)
	if len(v.NewlyPassing) != 1 || v.NewlyPassing[0] != "tests/a.py::test_one" {
		t.Fatalf("newly passing = %v", v.NewlyPassing)
	}
	if len(v.PreExisting) != 2 {
		t.Fatalf("pre-existing = %v", v.PreExisting)
	}
}

func TestParseVerdictEmpty(t *testing.T) {
	v := ParseVerdict("")
	if len(v.NewlyFailing) != 0 || v.Line != "" {
		t.Fatalf("expected an empty verdict, got %+v", v)
	}
}

// The runner prefix and the ids are joined for verify's single --test string,
// and every id has already been validated so it cannot carry a space or a
// shell metacharacter.
func TestVerifyRequestCommand(t *testing.T) {
	req := VerifyRequest{
		Runner:  []string{"pytest", "-v"},
		TestIDs: []string{"tests/a.py::test_one", "tests/b.py::test_two"},
	}
	want := "pytest -v tests/a.py::test_one tests/b.py::test_two"
	if got := req.Command(); got != want {
		t.Fatalf("Command = %q, want %q", got, want)
	}
}

func TestVerifyRecordsBaselineThenAdjudicates(t *testing.T) {
	f := runner.NewFake()
	baseArgs := []string{"graph", "verify", "--repo", "/base", "--test", "pytest -v x", "--record-baseline", "/data/baseline.json"}
	headArgs := []string{"graph", "verify", "--repo", "/head", "--test", "pytest -v x", "--pre-edit-baseline", "/data/baseline.json"}
	f.Set("entire", baseArgs, runner.Result{Stdout: "BASELINE RECORDED: /data/baseline.json (pytest; 3 passing, 0 failing, exit 0)"})
	f.Set("entire", headArgs, runner.Result{Stdout: "NEWLY FAILING (1): x\nVERDICT: REGRESSION in 1 tests: x"})

	v, cmds, err := Verify(context.Background(), f, VerifyRequest{
		BaseRepo: "/base", HeadRepo: "/head",
		Runner: []string{"pytest", "-v"}, TestIDs: []string{"x"},
		BaselinePath: "/data/baseline.json",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(v.NewlyFailing) != 1 || v.NewlyFailing[0] != "x" {
		t.Fatalf("newly failing = %v", v.NewlyFailing)
	}
	if len(cmds) != 2 {
		t.Fatalf("expected both commands recorded, got %v", cmds)
	}
	if len(f.Calls) != 2 {
		t.Fatalf("expected two verify calls, got %d", len(f.Calls))
	}
	// The baseline must come first, on the parent tree.
	if f.Calls[0].Args[3] != "/base" {
		t.Fatalf("the baseline must be recorded on the parent tree, got %v", f.Calls[0].Args)
	}
}

// A degraded baseline must be carried forward even when the head run happens
// to print a parseable line.
func TestVerifyCarriesDegradedBaselineForward(t *testing.T) {
	f := runner.NewFake()
	f.Set("entire", []string{"graph", "verify", "--repo", "/base", "--test", "pytest -q", "--record-baseline", "/d/b.json"},
		runner.Result{Stdout: "BASELINE RECORDED: /d/b.json (exit 0; output format not recognised, so the baseline is exit-code only)"})
	f.Set("entire", []string{"graph", "verify", "--repo", "/head", "--test", "pytest -q", "--pre-edit-baseline", "/d/b.json"},
		runner.Result{Stdout: "VERDICT: NO EFFECT"})

	v, _, err := Verify(context.Background(), f, VerifyRequest{
		BaseRepo: "/base", HeadRepo: "/head", Runner: []string{"pytest", "-q"}, BaselinePath: "/d/b.json",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if !v.Degraded {
		t.Fatal("the degraded baseline must reach the report")
	}
}

// A root commit has no parent, so there is no baseline. The head run still
// happens and the verdict is a state rather than a delta.
func TestVerifyWithoutABaseline(t *testing.T) {
	f := runner.NewFake()
	f.Default = runner.Result{Stdout: "BASELINE RECORDED: /d/b.json.head (pytest; 3 passing, 0 failing, exit 0)"}
	_, cmds, err := Verify(context.Background(), f, VerifyRequest{
		HeadRepo: "/head", Runner: []string{"pytest", "-v"}, BaselinePath: "/d/b.json",
	})
	if err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if len(cmds) != 1 {
		t.Fatalf("expected one command with no parent, got %v", cmds)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("expected one call, got %d", len(f.Calls))
	}
}
