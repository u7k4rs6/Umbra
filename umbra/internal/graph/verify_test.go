package graph

import (
	"context"
	"os"
	"path/filepath"
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

// Known-positive for the test-status parser.
//
// The failure this guards against is a check that looks for the wrong token:
// counting the lines that say ok rather than looking for the ones that say
// something failed. A green count and a red run are not the same question, and
// a checker that answers the first while claiming to answer the second reports
// a confident pass over a failing run.
func TestParseVerdictCatchesAPlantedFailure(t *testing.T) {
	out := `BASELINE RECORDED: /tmp/base.json (pytest; 14 passing, 0 failing, exit 0)
tests/test_service.py::test_rounding FAILED
NEWLY FAILING (1): tests/test_service.py::test_rounding
VERDICT: REGRESSION in 1 test: tests/test_service.py::test_rounding`

	v := ParseVerdict(out)
	if len(v.NewlyFailing) != 1 || v.NewlyFailing[0] != "tests/test_service.py::test_rounding" {
		t.Fatalf("newly failing = %v, want the one that failed", v.NewlyFailing)
	}
	if !strings.Contains(v.Line, "REGRESSION") {
		t.Fatalf("verdict line = %q, want the regression the run reported", v.Line)
	}
}

// And the other direction: output whose passing lines contain the word ok must
// not be miscounted. Every line here says ok somewhere and nothing failed.
func TestParseVerdictDoesNotMiscountTheWordOk(t *testing.T) {
	out := `BASELINE RECORDED: /tmp/base.json (pytest; 14 passing, 0 failing, exit 0)
tests/test_tokens.py::test_ok_response ok
tests/test_bookkeeping.py::test_ok ok
VERDICT: NO EFFECT the target tests behave exactly as before your edit.`

	v := ParseVerdict(out)
	if len(v.NewlyFailing) != 0 {
		t.Fatalf("newly failing = %v, want none", v.NewlyFailing)
	}
	if len(v.NewlyPassing) != 0 {
		t.Fatalf("newly passing = %v, want none; ok in a test name is not a status line", v.NewlyPassing)
	}
	if v.Degraded {
		t.Fatal("a parsed run is not degraded")
	}
	if !strings.HasPrefix(v.Line, "NO EFFECT") {
		t.Fatalf("verdict line = %q", v.Line)
	}
}

// The five files under testdata/verify are unedited stdout from real
// `entire graph verify` runs against a clone of pallets/click at 562e458.
// They exist because the first run on a repository that is not the fixture
// found the sweep reporting a clean audit on a change that broke 55 tests:
// verify's id list caps at twenty and its whole verdict caps in bytes, so the
// NEWLY FAILING line disappears entirely once the pair no longer fits. NOTES
// carries the shapes verbatim.

func recordedVerdict(t *testing.T, name string) Verdict {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "verify", name))
	if err != nil {
		t.Fatalf("reading recorded verify output: %v", err)
	}
	return ParseVerdict(string(b))
}

// The small case has to keep working exactly as it did, ids and all. It is the
// negative half of the pair: if this ever starts reporting a count without
// names, the fix has broken the case that was never broken.
func TestParseVerdictReadsTheCountAtFifteen(t *testing.T) {
	v := recordedVerdict(t, "15-both-lines.txt")
	if v.FailingCount != 15 {
		t.Fatalf("failing count = %d, want 15", v.FailingCount)
	}
	if len(v.NewlyFailing) == 0 {
		t.Fatal("the fifteen ids must still be named")
	}
	for _, want := range []string{
		"tests/test_arguments.py::test_argument_help",
		"tests/test_formatting.py::test_wrapping_long_options_strings",
	} {
		if !contains(v.NewlyFailing, want) {
			t.Fatalf("id %q missing from %v", want, v.NewlyFailing)
		}
	}
}

// Before the fix this returned no ids and no count, so the sweep reported zero
// leaks on seventeen newly failing tests.
func TestParseVerdictReadsTheCountAtSeventeen(t *testing.T) {
	v := recordedVerdict(t, "17-verdict-only.txt")
	if strings.Contains(readRecorded(t, "17-verdict-only.txt"), "NEWLY FAILING") {
		t.Fatal("this recording is meant to have no NEWLY FAILING line")
	}
	if v.FailingCount != 17 {
		t.Fatalf("failing count = %d, want 17", v.FailingCount)
	}
	if len(v.NewlyFailing) == 0 {
		t.Fatal("the verdict clause carries ids and none were read")
	}
	if v.UnnamedFailing() != 17-len(v.NewlyFailing) {
		t.Fatalf("unnamed = %d with %d named", v.UnnamedFailing(), len(v.NewlyFailing))
	}
}

// The commit this was captured from is the one the first-real-repo experiment
// describes as breaking 55 tests. Fifty-five is pytest's count; fifty-one is
// verify's, and this is verify's own output.
func TestParseVerdictReadsTheCountAtFiftyOne(t *testing.T) {
	v := recordedVerdict(t, "51-verdict-only-truncated.txt")
	if v.FailingCount != 51 {
		t.Fatalf("failing count = %d, want 51", v.FailingCount)
	}
	if len(v.NewlyFailing) != 20 {
		t.Fatalf("verify caps its list at twenty, got %d: %v", len(v.NewlyFailing), v.NewlyFailing)
	}
	if v.UnnamedFailing() != 31 {
		t.Fatalf("unnamed = %d, want 31", v.UnnamedFailing())
	}
}

// "… and 31 more" is not a test id. Before the fix splitIDs only skipped a
// part beginning "and ", and the real marker begins with a Unicode ellipsis.
func TestParseVerdictDropsTheTruncationMarker(t *testing.T) {
	for _, name := range []string{
		"51-both-lines-truncated.txt",
		"51-verdict-only-truncated.txt",
		"51-verdict-only-bare-ellipsis.txt",
	} {
		v := recordedVerdict(t, name)
		for _, id := range v.NewlyFailing {
			if strings.Contains(id, "…") || strings.Contains(id, "more") || strings.HasPrefix(id, "and ") {
				t.Fatalf("%s: %q was parsed as a test id", name, id)
			}
			if !strings.Contains(id, "::") {
				t.Fatalf("%s: %q does not look like a test id", name, id)
			}
		}
	}
}

// The smallest budget verify will render: a count and almost no ids. This is
// the case the rule is written for, that a count with no usable list must
// still never read as a clean run.
func TestParseVerdictReadsACountWithNoUsableIDs(t *testing.T) {
	v := recordedVerdict(t, "51-verdict-only-bare-ellipsis.txt")
	if v.FailingCount != 51 {
		t.Fatalf("failing count = %d, want 51", v.FailingCount)
	}
	if v.UnnamedFailing() != 51-len(v.NewlyFailing) {
		t.Fatalf("unnamed = %d with %d named", v.UnnamedFailing(), len(v.NewlyFailing))
	}
	if v.UnnamedFailing() == 0 {
		t.Fatal("a truncated list must report the tests it could not name")
	}
}

// A run that really did break nothing must still report nothing, or the fix
// has traded a false clean for a false alarm.
func TestParseVerdictNoEffectCountsNothing(t *testing.T) {
	v := ParseVerdict("VERDICT: NO EFFECT the target tests behave exactly as before your edit.")
	if v.FailingCount != 0 || v.UnnamedFailing() != 0 {
		t.Fatalf("count = %d, unnamed = %d, want 0 and 0", v.FailingCount, v.UnnamedFailing())
	}
}

func readRecorded(t *testing.T, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("testdata", "verify", name))
	if err != nil {
		t.Fatalf("reading recorded verify output: %v", err)
	}
	return string(b)
}

func contains(ids []string, want string) bool {
	for _, id := range ids {
		if id == want {
			return true
		}
	}
	return false
}

// The header count is the only count there is when tests newly pass: the
// verdict clause for a fix reads "VERDICT: PASS ..." and carries no number, so
// nothing but "NEWLY PASSING (38)" says how many there were. Its list is
// capped at twenty like every other. This recording also fixes the wording of
// a passing verdict, which is not "REGRESSION in n tests" and must not be
// parsed as one.
func TestParseVerdictReadsTheHeaderCountWhenTheVerdictCarriesNone(t *testing.T) {
	name := "38-newly-passing-no-count-in-verdict.txt"
	raw := readRecorded(t, name)
	if strings.Contains(raw, "REGRESSION in") {
		t.Fatal("this recording is meant to be a passing verdict")
	}
	v := recordedVerdict(t, name)
	if v.PassingCount != 38 {
		t.Fatalf("passing count = %d, want 38", v.PassingCount)
	}
	if len(v.NewlyPassing) != 20 {
		t.Fatalf("verify caps its list at twenty, got %d", len(v.NewlyPassing))
	}
	if v.FailingCount != 0 || v.UnnamedFailing() != 0 {
		t.Fatalf("a passing verdict reports no failures, got count %d", v.FailingCount)
	}
}
