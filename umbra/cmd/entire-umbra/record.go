package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/u7k4rs6/Umbra/umbra/internal/checkpoint"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// runRecord implements `entire umbra record <ref> --out <dir>`.
//
// It performs a real analysis and stores every Runner call with its output, so
// a test can replay the whole pipeline with no Entire, no git and no agent.
// The scrub is applied to everything written, which is why this is the only
// command that persists transcript-derived data.
func runRecord(ctx context.Context, argv []string) (int, error) {
	fs := flag.NewFlagSet("umbra record", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	out := fs.String("out", "", "directory to write the recording into (required)")
	scenario := fs.String("scenario", "", "name for the scenario (default: the directory name)")
	notes := fs.String("notes", "", "one line describing what the scenario shows")
	test := fs.String("test", "", "test runner prefix")
	run := fs.String("run", "shadow", "which selected tests to run")
	depth := fs.Int("depth", 2, "dependency hops")
	noAudit := fs.Bool("no-audit", false, "skip the sweep")
	history := fs.Bool("history", false, "add the scar factor")
	repo := fs.String("repo", "", "repository to analyze")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage:\n  entire umbra record <ref> --out fixtures/recorded/<scenario>/\n\nFlags:\n")
		fs.PrintDefaults()
	}
	if err := fs.Parse(reorder(fs, argv)); err != nil {
		return ExitRuntime, err
	}
	if *out == "" {
		return ExitRuntime, fmt.Errorf("record needs --out to say where to write the recording")
	}

	o := &Options{
		Ref: fs.Arg(0), Test: *test, Run: *run, Depth: *depth,
		NoAudit: *noAudit, History: *history, Format: "table", Adapter: "auto",
		Repo: *repo,
	}
	if o.Ref == "" {
		o.Ref = "HEAD"
	}
	if o.Repo == "" {
		o.Repo = os.Getenv("ENTIRE_REPO_ROOT")
	}
	if o.Repo == "" {
		wd, err := os.Getwd()
		if err != nil {
			return ExitRuntime, err
		}
		o.Repo = wd
	}
	if err := o.Validate(); err != nil {
		return ExitRuntime, err
	}

	exec := runner.NewExec(o.Repo)

	// The same scrubber the analysis uses, derived once. There is no flag to
	// turn it off: a recording is committed, and an unscrubbed one is a leak
	// waiting for someone to forget the flag.
	rec := runner.NewRecorder(exec, o.Scrubber(ctx, exec).Clean)

	res, err := checkpoint.New(rec, o.Repo).Resolve(ctx, o.Ref)
	if err != nil {
		return ExitRuntime, err
	}
	pair, err := checkpoint.AddWorktrees(ctx, rec, o.Repo, checkpoint.DataDir(), res, false)
	if err != nil {
		return ExitRuntime, err
	}
	defer pair.Remove(context.WithoutCancel(ctx))

	a, st, err := pipeline(ctx, o, rec, res, pair)
	if err != nil {
		return ExitRuntime, err
	}
	if err := execute(ctx, o, rec, pair, a, st); err != nil {
		return ExitRuntime, err
	}

	name := *scenario
	if name == "" {
		name = baseName(*out)
	}
	if err := rec.Save(*out, name, *notes); err != nil {
		return ExitRuntime, err
	}
	fmt.Printf("recorded %s into %s\n", name, *out)
	return ExitOK, nil
}

func baseName(p string) string {
	for len(p) > 0 && p[len(p)-1] == '/' {
		p = p[:len(p)-1]
	}
	for i := len(p) - 1; i >= 0; i-- {
		if p[i] == '/' {
			return p[i+1:]
		}
	}
	return p
}
