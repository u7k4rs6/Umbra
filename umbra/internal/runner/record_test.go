package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCallKeyDistinguishesArgs(t *testing.T) {
	a := Call{Name: "git", Args: []string{"log", "-1"}}
	b := Call{Name: "git", Args: []string{"log", "-1", "HEAD"}}
	if a.Key() == b.Key() {
		t.Fatal("different argv must not share a key")
	}
	// A separator that cannot appear in argv keeps "a b" from colliding with
	// two arguments "a" and "b".
	c := Call{Name: "git", Args: []string{"log -1"}}
	if a.Key() == c.Key() {
		t.Fatal("joined arguments must not collide with separate ones")
	}
}

func TestReplayServesByNameAndArgs(t *testing.T) {
	rec := &Recording{Scenario: "t", Entries: []Entry{
		{Call: Call{Name: "entire", Args: []string{"graph", "capabilities", "--json"}}, Result: Result{Stdout: "caps"}},
		{Call: Call{Name: "git", Args: []string{"rev-parse", "HEAD"}}, Result: Result{Stdout: "abc\n"}},
	}}
	r := NewReplay(rec)

	out, _, exit, err := r.Run(context.Background(), "entire", []string{"graph", "capabilities", "--json"}, nil)
	if err != nil || exit != 0 || string(out) != "caps" {
		t.Fatalf("got %q exit %d err %v", out, exit, err)
	}
	out, _, _, err = r.Run(context.Background(), "git", []string{"rev-parse", "HEAD"}, nil)
	if err != nil || string(out) != "abc\n" {
		t.Fatalf("got %q err %v", out, err)
	}
}

// A baseline run and a verify run use the same argv but produce different
// output, so repeated calls must replay in the order they were recorded.
func TestReplayRepeatedCallsInOrder(t *testing.T) {
	args := []string{"graph", "verify", "--repo", "."}
	rec := &Recording{Entries: []Entry{
		{Call: Call{Name: "entire", Args: args}, Result: Result{Stdout: "BASELINE RECORDED"}},
		{Call: Call{Name: "entire", Args: args}, Result: Result{Stdout: "NEWLY FAILING (1): t"}},
	}}
	r := NewReplay(rec)

	first, _, _, _ := r.Run(context.Background(), "entire", args, nil)
	second, _, _, _ := r.Run(context.Background(), "entire", args, nil)
	third, _, _, _ := r.Run(context.Background(), "entire", args, nil)

	if string(first) != "BASELINE RECORDED" {
		t.Fatalf("first = %q", first)
	}
	if string(second) != "NEWLY FAILING (1): t" {
		t.Fatalf("second = %q", second)
	}
	// Past the end the last result repeats rather than failing.
	if string(third) != "NEWLY FAILING (1): t" {
		t.Fatalf("third = %q", third)
	}
}

func TestReplayStrictRejectsUnknownCall(t *testing.T) {
	r := NewReplay(&Recording{})
	_, _, _, err := r.Run(context.Background(), "git", []string{"push"}, nil)
	if err == nil {
		t.Fatal("expected an error for an unrecorded call")
	}
	if !strings.Contains(err.Error(), "git push") {
		t.Fatalf("error should name the call, got %v", err)
	}
}

func TestReplayNonStrictReturnsEmpty(t *testing.T) {
	r := NewReplay(&Recording{})
	r.Strict = false
	out, _, exit, err := r.Run(context.Background(), "git", []string{"push"}, nil)
	if err != nil || exit != 0 || len(out) != 0 {
		t.Fatalf("got %q exit %d err %v", out, exit, err)
	}
}

func TestReplayCarriesExitAndError(t *testing.T) {
	rec := &Recording{Entries: []Entry{
		{Call: Call{Name: "entire", Args: []string{"graph", "checkpoint", "x"}},
			Result: Result{Stdout: "checkpoint x has no associated commit in this repository", Exit: 1}},
	}}
	r := NewReplay(rec)
	out, _, exit, err := r.Run(context.Background(), "entire", []string{"graph", "checkpoint", "x"}, nil)
	if err != nil {
		t.Fatalf("unexpected error %v", err)
	}
	if exit != 1 {
		t.Fatalf("exit = %d, want 1", exit)
	}
	if !strings.Contains(string(out), "no associated commit") {
		t.Fatalf("stdout = %q", out)
	}
}

func TestRecorderRoundTripsThroughDisk(t *testing.T) {
	inner := NewFake()
	inner.Set("git", []string{"rev-parse", "HEAD"}, Result{Stdout: "/home/someone/repo abc\n"})

	scrub := func(s string) string { return strings.ReplaceAll(s, "/home/someone", "/home/user") }
	rec := NewRecorder(inner, scrub)
	if _, _, _, err := rec.Run(context.Background(), "git", []string{"rev-parse", "HEAD"}, nil); err != nil {
		t.Fatalf("Run: %v", err)
	}

	dir := t.TempDir()
	if err := rec.Save(dir, "unit", "written by a test"); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "recording.json")); err != nil {
		t.Fatalf("recording not written: %v", err)
	}

	loaded, err := LoadRecording(dir)
	if err != nil {
		t.Fatalf("LoadRecording: %v", err)
	}
	if loaded.Scenario != "unit" {
		t.Fatalf("scenario = %q", loaded.Scenario)
	}
	if len(loaded.Entries) != 1 {
		t.Fatalf("entries = %d", len(loaded.Entries))
	}
	if strings.Contains(loaded.Entries[0].Result.Stdout, "someone") {
		t.Fatalf("scrub did not apply: %q", loaded.Entries[0].Result.Stdout)
	}

	replay := NewReplay(loaded)
	out, _, _, _ := replay.Run(context.Background(), "git", []string{"rev-parse", "HEAD"}, nil)
	if !strings.Contains(string(out), "/home/user") {
		t.Fatalf("replayed %q", out)
	}
}

func TestExitErrorMessageNamesTheCall(t *testing.T) {
	e := &ExitError{Call: Call{Name: "git", Args: []string{"worktree", "add"}}, Exit: 128, Stderr: "fatal: bad"}
	if !strings.Contains(e.Error(), "git worktree add") {
		t.Fatalf("message = %q", e.Error())
	}
	if !strings.Contains(e.Error(), "fatal: bad") {
		t.Fatalf("message = %q", e.Error())
	}
}
