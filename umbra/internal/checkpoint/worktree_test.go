package checkpoint

import (
	"context"
	"os"
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

// The worktree path is keyed by commit under a shared data directory, so a run
// that died before its cleanup, or another checkout of the same repository,
// can leave one behind. Reusing a stale one is how a worktree belonging to a
// deleted checkout poisons every later run.
func TestAddWorktreesReusesAHealthyCheckout(t *testing.T) {
	data := t.TempDir()
	head := filepath.Join(data, "wt", "abc123", "head")
	if err := os.MkdirAll(head, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(head, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := runner.NewFake()
	f.Set("git", []string{"-C", head, "rev-parse", "HEAD"}, runner.Result{Stdout: "abc123\n"})

	res := &Resolution{Commit: "abc123"}
	if _, err := AddWorktrees(context.Background(), f, "/repo", data, res, false); err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	for _, c := range f.Calls {
		if len(c.Args) > 2 && c.Args[2] == "worktree" && c.Args[3] == "add" {
			t.Fatal("a healthy checkout at the right commit should be reused, not recreated")
		}
	}
}

func TestAddWorktreesReplacesAStaleCheckout(t *testing.T) {
	data := t.TempDir()
	head := filepath.Join(data, "wt", "abc123", "head")
	if err := os.MkdirAll(head, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(head, ".git"), []byte("gitdir: /gone\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	f := runner.NewFake()
	// The leftover cannot be read: it belonged to a checkout that is gone.
	f.Set("git", []string{"-C", head, "rev-parse", "HEAD"}, runner.Result{Exit: 128, Stderr: "not a git repository"})

	res := &Resolution{Commit: "abc123"}
	if _, err := AddWorktrees(context.Background(), f, "/repo", data, res, false); err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	var pruned, added bool
	for _, c := range f.Calls {
		if len(c.Args) > 3 && c.Args[2] == "worktree" && c.Args[3] == "prune" {
			pruned = true
		}
		if len(c.Args) > 3 && c.Args[2] == "worktree" && c.Args[3] == "add" {
			added = true
		}
	}
	if !pruned {
		t.Error("the stale administrative entry should be pruned")
	}
	if !added {
		t.Error("a stale checkout should be replaced with a fresh one")
	}
	if _, err := os.Stat(filepath.Join(head, ".git")); err == nil {
		t.Error("the stale directory should have been removed")
	}
}

// A checkout at the wrong commit is as useless as an unreadable one.
func TestAddWorktreesReplacesACheckoutAtTheWrongCommit(t *testing.T) {
	data := t.TempDir()
	head := filepath.Join(data, "wt", "abc123", "head")
	if err := os.MkdirAll(head, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(head, ".git"), []byte("gitdir: elsewhere\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	f := runner.NewFake()
	f.Set("git", []string{"-C", head, "rev-parse", "HEAD"}, runner.Result{Stdout: "deadbeef\n"})

	res := &Resolution{Commit: "abc123"}
	if _, err := AddWorktrees(context.Background(), f, "/repo", data, res, false); err != nil {
		t.Fatalf("AddWorktrees: %v", err)
	}
	added := false
	for _, c := range f.Calls {
		if len(c.Args) > 3 && c.Args[2] == "worktree" && c.Args[3] == "add" {
			added = true
		}
	}
	if !added {
		t.Error("a checkout at another commit should be replaced")
	}
}
