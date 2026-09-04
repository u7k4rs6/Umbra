package checkpoint

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

func TestDataDirPrefersPluginDir(t *testing.T) {
	t.Setenv("ENTIRE_PLUGIN_DATA_DIR", "/plugin/data/umbra")
	t.Setenv("XDG_CACHE_HOME", "/cache")
	if got := DataDir(); got != "/plugin/data/umbra" {
		t.Fatalf("DataDir = %q", got)
	}
}

func TestDataDirFallsBackToCache(t *testing.T) {
	t.Setenv("ENTIRE_PLUGIN_DATA_DIR", "")
	t.Setenv("XDG_CACHE_HOME", "/cache")
	if got := DataDir(); got != filepath.Join("/cache", "umbra") {
		t.Fatalf("DataDir = %q", got)
	}
}

func TestAddWorktreesCreatesHeadAndBase(t *testing.T) {
	f := runner.NewFake()
	data := t.TempDir()
	res := &Resolution{Commit: "abc123", Parent: "def456"}

	p, err := AddWorktrees(context.Background(), f, "/repo", data, res, false)
	if err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	if p.Head == "" || p.Base == "" {
		t.Fatalf("head = %q base = %q", p.Head, p.Base)
	}

	if len(f.Calls) != 2 {
		t.Fatalf("expected 2 git calls, got %d", len(f.Calls))
	}
	wantHead := []string{"-C", "/repo", "worktree", "add", "--detach", filepath.Join(data, "wt", "abc123", "head"), "abc123"}
	if f.Calls[0].Key() != (runner.Call{Name: "git", Args: wantHead}).Key() {
		t.Fatalf("head call = %v", f.Calls[0].Args)
	}
	wantBase := []string{"-C", "/repo", "worktree", "add", "--detach", filepath.Join(data, "wt", "abc123", "base"), "def456"}
	if f.Calls[1].Key() != (runner.Call{Name: "git", Args: wantBase}).Key() {
		t.Fatalf("base call = %v", f.Calls[1].Args)
	}
}

// A root commit has no parent, so there is no baseline and the caller must be
// told rather than handed a broken path.
func TestAddWorktreesRootCommitHasNoBase(t *testing.T) {
	f := runner.NewFake()
	res := &Resolution{Commit: "root01", Parent: ""}
	p, err := AddWorktrees(context.Background(), f, "/repo", t.TempDir(), res, false)
	if err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	if p.Base != "" {
		t.Fatalf("base = %q, want empty", p.Base)
	}
	if len(f.Calls) != 1 {
		t.Fatalf("expected 1 git call, got %d", len(f.Calls))
	}
}

func TestRemoveIsSkippedWhenKeeping(t *testing.T) {
	f := runner.NewFake()
	res := &Resolution{Commit: "abc123", Parent: "def456"}
	p, err := AddWorktrees(context.Background(), f, "/repo", t.TempDir(), res, true)
	if err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	before := len(f.Calls)
	if err := p.Remove(context.Background()); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(f.Calls) != before {
		t.Fatal("Remove must not run git when the user asked to keep worktrees")
	}
	if !p.Kept() {
		t.Fatal("Kept should report true")
	}
}

func TestRemoveTearsDownInReverseOrder(t *testing.T) {
	f := runner.NewFake()
	data := t.TempDir()
	res := &Resolution{Commit: "abc123", Parent: "def456"}
	p, _ := AddWorktrees(context.Background(), f, "/repo", data, res, false)

	if err := p.Remove(context.Background()); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	last := f.Calls[len(f.Calls)-1]
	if last.Args[len(last.Args)-1] != filepath.Join(data, "wt", "abc123", "head") {
		t.Fatalf("expected head removed last, got %v", last.Args)
	}
}

func TestAddWorktreesFailsLoudlyOnGitError(t *testing.T) {
	f := runner.NewFake()
	f.Default = runner.Result{Exit: 128, Stderr: "fatal: invalid reference"}
	res := &Resolution{Commit: "nope", Parent: "alsonope"}
	if _, err := AddWorktrees(context.Background(), f, "/repo", t.TempDir(), res, false); err == nil {
		t.Fatal("expected an error when git worktree add fails")
	}
}
