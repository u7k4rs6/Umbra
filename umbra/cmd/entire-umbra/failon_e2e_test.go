package main

import (
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
	// A signature change on the fixture app. Its report has umbra nodes, and
	// running its probes cracks five tests.
	commitWithShadow = "0063443"
	// A commit that touches only Markdown. Its changed entities are in a
	// language the graph cannot resolve calls for, so they are not sources,
	// there is no field, and no node is in any state.
	commitWithNothing = "2e0aaca"
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

// A condition that holds must exit 2, and it must still print the report. An
// exit code with no output would tell a reader that something is wrong without
// telling them what.
func TestFailOnUmbraExitsTwoAndStillPrintsTheTable(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit", "--fail-on", "umbra")
	if got.exit != ExitFailOn {
		t.Fatalf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.exit, ExitFailOn, got.stdout, got.stderr)
	}

	// The table has to be there in full: the header, the light line with a
	// non-zero umbra count, and at least one docket row.
	for _, want := range []string{"Umbra ", "coverage  ", "light  ", "umbra"} {
		if !strings.Contains(got.stdout, want) {
			t.Errorf("the report is missing %q when --fail-on holds:\n%s", want, got.stdout)
		}
	}
	if !strings.Contains(got.stdout, "compute_total") {
		t.Errorf("expected the changed symbol in the report:\n%s", got.stdout)
	}
	if strings.TrimSpace(got.stdout) == "" {
		t.Fatal("exit 2 with no report tells the reader nothing")
	}
}

// The same run without the flag is a completed run, not a failure.
func TestWithoutFailOnTheSameRunExitsZero(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	got := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit")
	if got.exit != ExitOK {
		t.Fatalf("exit = %d, want %d\nstderr:\n%s", got.exit, ExitOK, got.stderr)
	}
	if !strings.Contains(got.stdout, "light  ") {
		t.Fatalf("expected the same report:\n%s", got.stdout)
	}
}

// A condition that does not hold exits 0 even though the flag is set.
func TestFailOnDoesNotFireWhenTheConditionIsAbsent(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	got := runUmbra(t, bin, root, commitWithNothing, "--run", "none", "--no-audit", "--fail-on", "umbra")
	if got.exit != ExitOK {
		t.Fatalf("exit = %d, want %d\nstdout:\n%s\nstderr:\n%s", got.exit, ExitOK, got.stdout, got.stderr)
	}
	if !strings.Contains(got.stdout, "0 umbra") {
		t.Fatalf("expected a report with nothing in shadow:\n%s", got.stdout)
	}
}

// The reference comes first in the documented surface, so a flag after it has
// to be honoured. The standard flag package stops at the first positional, and
// that broke this once already.
func TestFlagAfterTheReferenceIsHonoured(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)
	bin := buildBinary(t)

	after := runUmbra(t, bin, root, commitWithShadow, "--run", "none", "--no-audit", "--fail-on", "umbra")
	before := runUmbra(t, bin, root, "--fail-on", "umbra", "--run", "none", "--no-audit", commitWithShadow)

	if after.exit != ExitFailOn {
		t.Errorf("flag after the reference: exit = %d, want %d", after.exit, ExitFailOn)
	}
	if before.exit != ExitFailOn {
		t.Errorf("flag before the reference: exit = %d, want %d", before.exit, ExitFailOn)
	}
	if after.exit != before.exit {
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

// The condition the flag exists for: probes that cracked. This one needs the
// fixture virtualenv as well, because it actually runs the tests.
func TestFailOnFailureExitsTwoWhenProbesCrack(t *testing.T) {
	requireTooling(t)
	root := repoRoot(t)

	py := filepath.Join(root, "umbra", "fixtures", "app", ".venv", "bin", "python")
	if _, err := os.Stat(py); err != nil {
		t.Skip("the fixture virtualenv is not set up; run umbra/fixtures/app/setup.sh")
	}
	bin := buildBinary(t)
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
