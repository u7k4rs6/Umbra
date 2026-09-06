// Package scripts holds the tests for the shell scripts beside it.
//
// The scripts are the two prepared for a human decision: one grafts this code
// into a fork, the other reports what the checkpoint refs on a remote contain.
// Neither is run by the build. The audit is tested because it answers a
// question about what is public, and it has already answered it wrongly once.
package scripts

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// The planted values. plantedperson is not a user on any machine and
// example.invalid is reserved by RFC 2606, so neither can be real.
const (
	plantedHome  = "/home/plantedperson/work/notes.txt"
	plantedEmail = "planted.address@example.invalid"
)

// plantedRepo builds a repository with one checkpoint ref whose transcript
// carries a home path and an address, and returns its path.
func plantedRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	transcript := `{"seq":1,"cwd":"` + plantedHome + `"}
{"seq":2,"author":"` + plantedEmail + `"}
{"seq":3,"kind":"read","path":"app/service.py"}
`
	if err := os.WriteFile(filepath.Join(dir, "transcript.jsonl"), []byte(transcript), 0o644); err != nil {
		t.Fatal(err)
	}
	// A second file the audit must not count, so a passing run is not just
	// counting every line in the tree.
	if err := os.WriteFile(filepath.Join(dir, "clean.json"), []byte(`{"note":"nothing here"}`+"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}

	for _, args := range [][]string{
		{"init", "-q", "-b", "main"},
		{"config", "user.email", "committer@example.invalid"},
		{"config", "user.name", "Committer"},
		{"add", "-A"},
		{"commit", "-q", "-m", "planted"},
		{"update-ref", "refs/entire/checkpoints/planted", "HEAD"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
		}
	}
	return dir
}

// row is one line of the audit table: the ref name and its three counts.
type row struct {
	name   string
	bytes  int
	paths  int
	emails int
}

func runAudit(t *testing.T, workdir, remote string) []row {
	t.Helper()
	script, err := filepath.Abs("checkpoint-refs-audit.sh")
	if err != nil {
		t.Fatal(err)
	}
	// Read the script through the test process. Go's test cache keys on the
	// files the test binary opens, and bash opening it does not count: with
	// the script edited and nothing else touched, a cached "ok" is reported
	// for a version that was never run. This is the same class of mistake the
	// tests in this file exist to catch.
	if _, err := os.ReadFile(script); err != nil {
		t.Fatalf("reading the script: %v", err)
	}

	cmd := exec.Command("bash", script, remote)
	cmd.Dir = workdir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("the audit failed: %v\n%s", err, out)
	}

	var rows []row
	for _, line := range strings.Split(string(out), "\n") {
		f := strings.Fields(line)
		if len(f) != 4 || f[0] == "ref" {
			continue
		}
		b, err1 := strconv.Atoi(f[1])
		p, err2 := strconv.Atoi(f[2])
		m, err3 := strconv.Atoi(f[3])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		rows = append(rows, row{name: f[0], bytes: b, paths: p, emails: m})
	}
	if len(rows) == 0 {
		t.Fatalf("the audit reported no rows at all:\n%s", out)
	}
	return rows
}

func find(t *testing.T, rows []row, name string) row {
	t.Helper()
	for _, r := range rows {
		if r.name == name {
			return r
		}
	}
	t.Fatalf("no row for %q in %v", name, rows)
	return row{}
}

// Known-positive for the audit.
//
// An earlier version reported thirty refs of zero bytes carrying no paths and
// no addresses, which is a confident, precise all-clear on exactly the
// question the script exists to answer. It was wrong three separate ways: grep
// exiting non-zero on no match ended the run under pipefail, a ten megabyte
// archive in a shell variable came back empty, and git ls-tree is scoped to
// the working directory unless it is given --full-tree.
func TestAuditCatchesPlantedEmail(t *testing.T) {
	requireTools(t)
	dir := plantedRepo(t)

	r := find(t, runAudit(t, dir, dir), "planted")
	if r.bytes <= 0 {
		t.Errorf("bytes = %d, want the size of the planted transcript", r.bytes)
	}
	if r.emails < 1 {
		t.Errorf("emails = %d, want at least the planted address", r.emails)
	}
	if r.paths < 1 {
		t.Errorf("paths = %d, want at least the planted home path", r.paths)
	}
}

// The --full-tree regression. git ls-tree lists only what is under the working
// directory by default, and this script is run from wherever the reader
// happens to be. Run from a subdirectory it used to list nothing, and every
// count read zero.
func TestAuditReportsTheSameNumbersFromASubdirectory(t *testing.T) {
	requireTools(t)
	dir := plantedRepo(t)

	fromRoot := find(t, runAudit(t, dir, dir), "planted")
	fromSub := find(t, runAudit(t, filepath.Join(dir, "sub"), dir), "planted")

	if fromRoot != fromSub {
		t.Errorf("the audit disagrees with itself:\n  from the root %+v\n  from a subdirectory %+v", fromRoot, fromSub)
	}
}

func requireTools(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"bash", "git"} {
		if _, err := exec.LookPath(tool); err != nil {
			t.Skipf("%s is not on PATH", tool)
		}
	}
}
