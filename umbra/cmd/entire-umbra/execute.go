package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/checkpoint"
	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// execute selects tests, runs them through verify with a baseline, then sweeps
// the full suite and explains every test the selection missed.
func execute(ctx context.Context, o *Options, run runner.Runner, pair *checkpoint.Pair, a *report.Analysis, st *pipelineState) error {
	if o.Run == "none" {
		return nil
	}

	runnerArgv, err := SplitWords(o.Test)
	if err != nil {
		return err
	}
	if len(runnerArgv) == 0 {
		return fmt.Errorf("the test runner is empty")
	}

	// The runner is echoed before anything runs, which is the only guard
	// against a destructive --test and is stated as such in the README.
	fmt.Printf("running  %s\n", strings.Join(runnerArgv, " "))

	cands, notRunnable := shadow.SelectTests(a.Nodes, st.Field, st.RelMap, st.SourceIDs, o.Run, st.TestRoot)
	a.Execution.Selected = shadow.IDs(cands)
	a.Execution.NotRunnable = notRunnable

	if len(cands) == 0 {
		if a.Audit {
			return sweep(ctx, o, run, pair, a, st, runnerArgv, map[string]bool{})
		}
		return nil
	}

	baseRepo, headRepo := testRepo(pair.Base, st.TestRoot), testRepo(pair.Head, st.TestRoot)
	req := graph.VerifyRequest{
		BaseRepo:     baseRepo,
		HeadRepo:     headRepo,
		Runner:       runnerArgv,
		TestIDs:      a.Execution.Selected,
		BaselinePath: filepath.Join(pair.Dir(), "baseline.json"),
	}
	v, cmds, err := graph.Verify(ctx, run, req)
	if err != nil {
		return fmt.Errorf("running the selected tests: %w", err)
	}
	a.Commands = append(a.Commands, scrubAll(st, cmds)...)

	a.Execution.NewFailures = v.NewlyFailing
	a.Execution.Fixed = v.NewlyPassing
	a.Execution.PreExisting = v.PreExisting
	a.Execution.Verdict = v.Line
	a.Execution.Degraded = v.Degraded
	noteUnnamed(a, v)
	if v.TimedOut {
		a.Notes = append(a.Notes, "the selected tests were cut short by the ten minute timeout")
	}

	attachOutcomes(a, cands, v)

	if !a.Audit {
		return nil
	}
	selected := map[string]bool{}
	for _, id := range a.Execution.Selected {
		selected[id] = true
	}
	return sweep(ctx, o, run, pair, a, st, runnerArgv, selected)
}

// sweep runs the whole suite once and explains every changed test the
// selection did not choose.
func sweep(ctx context.Context, o *Options, run runner.Runner, pair *checkpoint.Pair, a *report.Analysis, st *pipelineState, runnerArgv []string, selected map[string]bool) error {
	req := graph.VerifyRequest{
		BaseRepo:     testRepo(pair.Base, st.TestRoot),
		HeadRepo:     testRepo(pair.Head, st.TestRoot),
		Runner:       runnerArgv,
		BaselinePath: filepath.Join(pair.Dir(), "sweep.json"),
		Timeout:      runner.SweepTimeout,
	}
	v, cmds, err := graph.Verify(ctx, run, req)
	if err != nil {
		return fmt.Errorf("running the full sweep: %w", err)
	}
	a.Commands = append(a.Commands, scrubAll(st, cmds)...)
	a.Execution.Sweep = true
	a.Execution.SweepCut = v.TimedOut
	noteUnnamed(a, v)

	changed := append(append([]string(nil), v.NewlyFailing...), v.NewlyPassing...)

	if sweepIsInconclusive(v, changed) {
		a.Execution.SweepInconclusive = true
		a.Execution.SweepInconclusiveReason = InconclusiveSweepReason
		return nil
	}

	leaks := shadow.Forensics(changed, shadow.ForensicsInput{
		Field:         st.Field,
		RelMap:        st.RelMap,
		SourceIDs:     st.SourceIDs,
		CoChangeFiles: st.CoChangeFiles,
		Selected:      selected,
		MaxDepth:      o.Depth,
		RepoRelRoot:   st.TestRoot,
	})
	for _, l := range leaks {
		a.Execution.Leaks = append(a.Execution.Leaks, report.Leak{Test: l.Test, Reason: l.Reason})
	}

	// The sweep also finds failures the selection never ran. Fold them in so
	// the report does not under-count what the change broke.
	known := map[string]bool{}
	for _, id := range a.Execution.NewFailures {
		known[id] = true
	}
	for _, id := range v.NewlyFailing {
		if !known[id] {
			a.Execution.NewFailures = append(a.Execution.NewFailures, id)
			known[id] = true
		}
	}
	return nil
}

// InconclusiveSweepReason is the one sentence a reader gets when the sweep
// compared nothing.
const InconclusiveSweepReason = "the runner printed no per-test ids, so the sweep could not compare any test against the baseline; add -v to the runner"

// sweepIsInconclusive reports whether the sweep audited nothing at all.
//
// Saying "0 leaks" for such a run would be indistinguishable from a clean
// audit, which is the strongest claim the report can make out of the weakest
// evidence it has.
//
// The count is what separates this from an incomplete list. When verify
// reports a total but drops the id list for exceeding its byte budget, changed
// is empty and yet the number of failures is known. That is incomplete, it
// still prints a number, and UnnamedFailures carries it. Only when there is
// nothing at all to count is the audit inconclusive.
func sweepIsInconclusive(v graph.Verdict, changed []string) bool {
	return v.Degraded && len(changed) == 0 && v.UnnamedFailing() == 0
}

// noteUnnamed records how many newly failing tests verify counted but could
// not name.
//
// Both runs report it and the larger wins. The sweep runs the whole suite and
// is a superset of the probes, so its number is normally the one that stands;
// taking the maximum means neither run can quietly lower a count the other
// already established. Without this the report takes the length of an id list
// verify explicitly told it was capped, which is how a change that broke 55
// tests came back as a clean audit.
func noteUnnamed(a *report.Analysis, v graph.Verdict) {
	if n := v.UnnamedFailing(); n > a.Execution.UnnamedFailures {
		a.Execution.UnnamedFailures = n
	}
}

// attachOutcomes marks each test node with what happened, and carries a
// failure onto every shadowed node on that test's path to a source.
func attachOutcomes(a *report.Analysis, cands []shadow.Candidate, v graph.Verdict) {
	failed := map[string]bool{}
	for _, id := range v.NewlyFailing {
		failed[id] = true
	}
	// A test that was already failing before the change is still failing. It
	// is not a crack, because this change did not break it, but calling it a
	// pass would be a lie.
	preExisting := map[string]bool{}
	for _, id := range v.PreExisting {
		preExisting[id] = true
	}
	for _, c := range cands {
		switch {
		case failed[c.ID], preExisting[c.ID]:
			c.Node.Result = shadow.OutcomeFail
		case v.Degraded:
			// Without per-test ids there is no per-test outcome to report.
			c.Node.Result = shadow.OutcomeNotRun
		default:
			c.Node.Result = shadow.OutcomePass
		}
	}

	// A failing probe is evidence about every shadowed node it passes through.
	for _, c := range cands {
		if !failed[c.ID] {
			continue
		}
		for _, n := range a.Nodes {
			if !n.Shadowed() || n.Symbol.ID == c.Node.Symbol.ID {
				continue
			}
			if onPath(n.Symbol.ID, c.Node.Path) {
				n.Tests = append(n.Tests, shadow.TestRef{ID: c.ID, Symbol: c.Node.Symbol.ID, Outcome: shadow.OutcomeFail})
			}
		}
	}
}

func onPath(id string, path []string) bool {
	for _, p := range path {
		if p == id {
			return true
		}
	}
	return false
}

// scrubAll puts the verify commands through the same scrubber as every other
// recorded command.
//
// They used to be appended raw, so a report carried the absolute path of the
// worktree the tests ran in. Every other command in the list was scrubbed,
// which is why it went unnoticed until a real report was committed and the
// artifacts test read it.
func scrubAll(st *pipelineState, cmds []string) []string {
	if st == nil || st.Scrub == nil {
		return cmds
	}
	out := make([]string, 0, len(cmds))
	for _, c := range cmds {
		out = append(out, st.Scrub.Clean(c))
	}
	return out
}

// testRepo returns the directory the runner should run in. The fixture app
// lives in a subdirectory, so tests run there rather than at the repository
// root.
func testRepo(worktree, relRoot string) string {
	if worktree == "" {
		return ""
	}
	if relRoot == "" {
		return worktree
	}
	return filepath.Join(worktree, filepath.FromSlash(relRoot))
}
