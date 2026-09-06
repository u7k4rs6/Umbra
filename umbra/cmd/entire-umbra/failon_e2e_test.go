package main

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// End to end tests for the exit codes.
//
// Everything else about --fail-on is unit tested against a built Analysis.
// Nothing drove the binary itself, so the argv path, the flag parsing and the
// exit code were only ever checked in pieces. These run the real binary and
// look at what a shell would see.
//
// They need entire and entire-graph on PATH, which ARCHITECTURE.md already
// names as the condition for an end to end test, and they read only commits
// that are recorded in this repository. No network, no agent, and no test here
// starts a session.

// buildBinary compiles the command under test once per run.
func buildBinary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "entire-umbra")
	cmd := exec.Command("go", "build", "-o", bin, ".")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("building the binary: %v\n%s", err, out)
	}
	return bin
}

// repoRoot is the repository this test is running inside, which is also the
// repository the recorded commits belong to.
func repoRoot(t *testing.T) string {
	t.Helper()
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".git")); err != nil {
		t.Skipf("not running inside the repository checkout: %v", err)
	}
	return root
}

func requireTooling(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"entire", "entire-graph", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not on PATH; the end to end tests need the real toolchain", tool)
		}
	}
}

// commits recorded in this repository, chosen for what their reports contain.
const (
	// A signature change on the fixture app. It has a field of dependents,
	// and running its probes cracks tests.
	commitWithShadow = "1c2cf29"
	// A commit that touches only Markdown. Its changed entities are in a
	// language the graph cannot resolve calls for, so they are not sources,
	// there is no field, and no node is in any state. That holds anywhere.
	commitWithNothing = "b8dc631"
)

type result struct {
	exit   int
	stdout string
	stderr string
}

func runUmbra(t *testing.T, bin, root string, args ...string) result {
	t.Helper()
	cmd := exec.Command(bin, args...)
	cmd.Dir = root
	var out, errb strings.Builder
	cmd.Stdout = &out
	cmd.Stderr = &errb
	err := cmd.Run()

	code := 0
	if err != nil {
		ee, ok := err.(*exec.ExitError)
		if !ok {
			t.Fatalf("running %v: %v", args, err)
		}
		code = ee.ExitCode()
	}
	return result{exit: code, stdout: out.String(), stderr: errb.String()}
}

// summary is the state counts from a real run.
type summary struct {
	Lit      int `json:"lit"`
	Penumbra int `json:"penumbra"`
	Umbra    int `json:"umbra"`
	Unknown  int `json:"unknown"`
}

// analyse runs the binary once and returns what the report actually says.
//
// The states cannot be hard coded. They come from the examined set, which
// comes from whichever checkpoint the reference pairs with, and the set of
// checkpoints present differs between this working copy and a clone of it:
// imported history is local and is not pushed. The same commit reports six
// umbra nodes here and none in a clone. So the condition to test with is
// derived from the run rather than assumed, which is what makes these tests
// mean the same thing everywhere.
//
// It also skips when the toolchain cannot analyse this checkout at all, which
// happens when the repository sits somewhere Graph refuses to run git under.
func analyse(t *testing.T, bin, root, ref string) summary {
	t.Helper()
	got := runUmbra(t, bin, root, ref, "--run", "none", "--no-audit", "--format", "json")
	if got.exit == ExitRuntime {
		t.Skipf("the toolchain cannot analyse this checkout: %s", strings.TrimSpace(got.stderr))
	}
	if got.exit != ExitOK {
		t.Fatalf("a plain run should complete: exit = %d\n%s", got.exit, got.stderr)
	}
	var report struct {
		Summary summary `json:"summary"`
	}
	if err := json.Unmarshal([]byte(got.stdout), &report); err != nil {
		t.Fatalf("reading the report: %v\n%s", err, got.stdout)
	}
	return report.Summary
}

// firingCondition returns a --fail-on value that must fire for this report,
// and whether there is one at all.
func firingCondition(s summary) (string, bool) {
	switch {
	case s.Umbra > 0:
		return "umbra", true
	case s.Penumbra > 0:
		// --fail-on penumbra holds when anything is penumbra or umbra.
		return "penumbra", true
	}
	return "", false
}

// A condition that holds must exit 2, and it must still print the report. An
// exit code with no output would tell a reader that something is wrong without
// telling them what.
func TestFailOnExitsTwoAndStillPrintsTheTable(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	cond, ok := firingCondition(analyse(t, bin, root, commitWithShadow))
	if !ok {
		t.Skip("nothing is in shadow in this checkout, so no condition can hold")
	}

	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit", "--fail-on", cond)
	if got.exit != ExitFailOn {
		t.Fatalf("--fail-on %s: exit = %d, want %d\nstdout:\n%s\nstderr:\n%s",
			cond, got.exit, ExitFailOn, got.stdout, got.stderr)
	}

	for _, want := range []string{"Umbra ", "coverage  ", "light  ", "compute_total"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the report is missing %q when --fail-on holds:\n%s", want, got.stdout)
		}
	}
}

// The same run without the flag is a completed run, not a failure.
func TestWithoutFailOnTheSameRunExitsZero(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)
	analyse(t, bin, root, commitWithShadow) // skips if the toolchain cannot run here

	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit")
	if got.exit != ExitOK {
		t.Fatalf("exit = %d, want %d\nstderr:\n%s", got.exit, ExitOK, got.stderr)
	}
	if !strings.Contains(got.stdout, "light  ") {
		t.Fatalf("expected the same report:\n%s", got.stdout)
	}
}

// A condition that does not hold exits 0 even though the flag is set.
//
// Two cases that hold in any checkout: a commit whose changes are all in a
// language the graph cannot follow has no nodes at all, and a run that never
// executed a test cannot have a failure.
func TestFailOnDoesNotFireWhenTheConditionIsAbsent(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	empty := analyse(t, bin, root, commitWithNothing)
	if empty.Umbra != 0 || empty.Penumbra != 0 || empty.Lit != 0 {
		t.Fatalf("expected a commit with no field at all, got %+v", empty)
	}
	for _, cond := range []string{"umbra", "penumbra"} {
		got := runUmbra(t, bin, root, commitWithNothing, "--run", "none", "--no-audit", "--fail-on", cond)
		if got.exit != ExitOK {
			t.Errorf("--fail-on %s on a commit with no field: exit = %d, want %d\n%s",
				cond, got.exit, ExitOK, got.stdout)
		}
	}

	// Nothing ran, so nothing can have failed. This uses the other commit, so
	// it needs the same guard: a checkout the toolchain cannot analyse has no
	// exit code worth asserting.
	analyse(t, bin, root, commitWithShadow)
	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit", "--fail-on", "failure")
	if got.exit != ExitOK {
		t.Fatalf("--fail-on failure with --run none: exit = %d, want %d\n%s", got.exit, ExitOK, got.stderr)
	}
}

// The reference comes first in the documented surface, so a flag after it has
// to be honoured. The standard flag package stops at the first positional, and
// that broke this once already.
func TestFlagAfterTheReferenceIsHonoured(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	cond, ok := firingCondition(analyse(t, bin, root, commitWithShadow))
	if !ok {
		t.Skip("nothing is in shadow in this checkout, so no condition can hold")
	}

	after := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit", "--fail-on", cond)
	before := runUmbra(t, bin, root, "--fail-on", cond, "--run", "none", "--no-audit", commitWithShadow)

	if after.exit != ExitFailOn {
		t.Errorf("flag after the reference: exit = %d, want %d", after.exit, ExitFailOn)
	}
	if before.exit != after.exit {
		t.Errorf("argument order changed the outcome: %d then %d", after.exit, before.exit)
	}
}

// A flag value that is not one of the four conditions is a usage error, which
// is exit 1 and not exit 2.
func TestUnknownFailOnConditionIsARuntimeError(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--fail-on", "everything")
	if got.exit != ExitRuntime {
		t.Fatalf("exit = %d, want %d", got.exit, ExitRuntime)
	}
	if !strings.Contains(got.stderr, "--fail-on") {
		t.Fatalf("the error should name the flag:\n%s", got.stderr)
	}
}

// The condition the flag exists for: probes that cracked. Which tests fail
// comes from running them against the two worktrees, so it does not depend on
// the examined set. It does need the fixture virtualenv.
func TestFailOnFailureExitsTwoWhenProbesCrack(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)

	py := filepath.Join(root, "umbra", "fixtures", "app", ".venv", "bin", "python")
	if _, err := os.Stat(py); err != nil {
		t.Skip("the fixture virtualenv is not set up; run umbra/fixtures/app/setup.sh")
	}
	bin := buildBinary(t)
	analyse(t, bin, root, commitWithShadow) // skips if the toolchain cannot run here
	runner := py + " -m pytest -v"

	got := runUmbra(t, bin, root, commitWithShadow, "--test", runner, "--no-audit", "--fail-on", "failure")
	if got.exit != ExitFailOn {
		t.Fatalf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.exit, ExitFailOn, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "cracked") {
		t.Fatalf("expected cracked probes in the report:\n%s", got.stdout)
	}

	same := runUmbra(t, bin, root, commitWithShadow, "--test", runner, "--no-audit")
	if same.exit != ExitOK {
		t.Fatalf("without --fail-on the same run should complete: exit = %d", same.exit)
	}
}
