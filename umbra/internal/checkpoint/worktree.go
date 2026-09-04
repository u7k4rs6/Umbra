package checkpoint

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Pair is the two detached worktrees an analysis needs: the tree at the
// commit and the tree at its parent. Tests run in both so a failure that
// predates the change is not blamed on it.
type Pair struct {
	Head string
	Base string

	run  runner.Runner
	repo string
	dir  string
	keep bool
	made []string
}

// DataDir is where worktrees and baselines live. Entire sets
// ENTIRE_PLUGIN_DATA_DIR even for an unmanaged plugin, which the probe
// confirmed; XDG_CACHE_HOME is the fallback.
func DataDir() string {
	if d := os.Getenv("ENTIRE_PLUGIN_DATA_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CACHE_HOME"); d != "" {
		return filepath.Join(d, "umbra")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "umbra")
	}
	return filepath.Join(os.TempDir(), "umbra")
}

// AddWorktrees checks out head and base under dataDir keyed by the commit.
//
// A commit with no parent gets an empty Base and callers fall back to a
// state verdict rather than a delta.
func AddWorktrees(ctx context.Context, run runner.Runner, repo, dataDir string, res *Resolution, keep bool) (*Pair, error) {
	root := filepath.Join(dataDir, "wt", res.Commit)
	p := &Pair{run: run, repo: repo, dir: root, keep: keep}

	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}

	head := filepath.Join(root, "head")
	if err := p.add(ctx, head, res.Commit); err != nil {
		return nil, err
	}
	p.Head = head

	if res.Parent != "" {
		base := filepath.Join(root, "base")
		if err := p.add(ctx, base, res.Parent); err != nil {
			p.Remove(ctx)
			return nil, err
		}
		p.Base = base
	}
	return p, nil
}

func (p *Pair) add(ctx context.Context, path, sha string) error {
	// The worktree path is keyed by commit under a shared data directory, so
	// two checkouts of the same repository, or a run that died before its
	// cleanup, can leave one behind. Reusing it blindly is how a worktree
	// belonging to a deleted checkout poisons every later run: git can no
	// longer read its metadata and Graph refuses to run there. Reuse it only
	// when it is a healthy worktree sitting at the commit we want.
	if _, err := os.Stat(filepath.Join(path, ".git")); err == nil {
		if p.checkoutIsAt(ctx, path, sha) {
			p.made = append(p.made, path)
			return nil
		}
		if err := os.RemoveAll(path); err != nil {
			return err
		}
		// Drop the stale administrative entry so git will accept the path
		// again, whether or not it belonged to this repository.
		_, _, _, _ = p.run.Run(ctx, "git", []string{"-C", p.repo, "worktree", "prune"}, nil)
	}
	args := []string{"-C", p.repo, "worktree", "add", "--detach", path, sha}
	_, stderr, exit, err := p.run.Run(ctx, "git", args, nil)
	if err != nil {
		return err
	}
	if exit != 0 {
		return &runner.ExitError{
			Call:   runner.Call{Name: "git", Args: args},
			Exit:   exit,
			Stderr: string(stderr),
		}
	}
	p.made = append(p.made, path)
	return nil
}

// checkoutIsAt reports whether path is a working checkout of this repository
// sitting at sha.
func (p *Pair) checkoutIsAt(ctx context.Context, path, sha string) bool {
	stdout, _, exit, err := p.run.Run(ctx, "git", []string{"-C", path, "rev-parse", "HEAD"}, nil)
	if err != nil || exit != 0 {
		return false
	}
	return strings.TrimSpace(string(stdout)) == sha
}

// Remove tears the worktrees down unless the user asked to keep them.
func (p *Pair) Remove(ctx context.Context) error {
	if p.keep {
		return nil
	}
	var first error
	for i := len(p.made) - 1; i >= 0; i-- {
		args := []string{"-C", p.repo, "worktree", "remove", "--force", p.made[i]}
		if _, _, _, err := p.run.Run(ctx, "git", args, nil); err != nil && first == nil {
			first = err
		}
	}
	if err := os.RemoveAll(p.dir); err != nil && first == nil {
		first = err
	}
	return first
}

// Kept reports whether the worktrees survive the run, for the header.
func (p *Pair) Kept() bool { return p.keep }

// Dir is the directory holding both worktrees.
func (p *Pair) Dir() string { return p.dir }

// Describe is the one-line header form.
func (p *Pair) Describe() string {
	if p.Base == "" {
		return fmt.Sprintf("worktree %s (no parent, so there is no baseline)", p.Head)
	}
	return fmt.Sprintf("worktrees %s and %s", p.Head, p.Base)
}
