// Command entire-umbra finds the code an agent's change affects but the agent
// never looked at.
//
// It is dispatched by the Entire CLI as `entire umbra` because any executable
// named entire-<name> on $PATH runs kubectl style.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/u7k4rs6/Umbra/umbra/internal/checkpoint"
	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

func main() {
	os.Exit(run())
}

func run() int {
	argv := os.Args[1:]

	// `entire umbra record <ref> --out <dir>` writes a Runner recording that a
	// replay test can drive the whole pipeline from.
	if len(argv) > 0 && argv[0] == "record" {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer stop()
		code, err := runRecord(ctx, argv[1:])
		if err != nil {
			fmt.Fprintf(os.Stderr, "umbra: %v\n", err)
		}
		return code
	}

	opts, err := parseFlags(argv)
	if err != nil {
		if err == flag.ErrHelp {
			return ExitOK
		}
		fmt.Fprintf(os.Stderr, "umbra: %v\n", err)
		return ExitRuntime
	}
	if opts == nil {
		return ExitOK
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	code, err := analyze(ctx, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "umbra: %v\n", err)
		return ExitRuntime
	}
	return code
}

func parseFlags(argv []string) (*Options, error) {
	fs := flag.NewFlagSet("entire-umbra", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	o := &Options{}
	fs.StringVar(&o.Test, "test", "", "test runner prefix, for example \"pytest -v\"; test ids are appended as separate arguments")
	fs.StringVar(&o.Run, "run", "shadow", "which selected tests to run: none, shadow or all")
	fs.BoolVar(&o.NoAudit, "no-audit", false, "skip the full-suite sweep that audits the selection")
	fs.BoolVar(&o.History, "history", false, "add the scar factor from git history")
	fs.IntVar(&o.Depth, "depth", 2, "dependency hops to follow: 1, 2 or 3")
	fs.StringVar(&o.Format, "format", "table", "table, json, html or packet")
	fs.StringVar(&o.Out, "out", "", "directory for the json, html and packet outputs")
	fs.StringVar(&o.FailOn, "fail-on", "", "exit 2 when the condition holds: umbra, penumbra, failure or leak")
	fs.StringVar(&o.Adapter, "adapter", "auto", "transcript adapter: auto or claude-code")
	fs.BoolVar(&o.Snippets, "snippets", false, "include declaration lines for nodes in the report")
	fs.BoolVar(&o.All, "all", false, "show lit nodes in the table instead of counting them")
	fs.BoolVar(&o.KeepWorktrees, "keep-worktrees", false, "leave the temporary worktrees in place")
	fs.StringVar(&o.Repo, "repo", "", "repository to analyze (default: ENTIRE_REPO_ROOT, else the current directory)")
	fs.StringVar(&o.TestRoot, "test-root", "", "directory the test runner runs in, relative to the repository (default: found from the project files near the tests)")
	showVersion := fs.Bool("version", false, "print the version and exit")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "umbra %s (%s)\n\n", Version, Commit)
		fmt.Fprintf(os.Stderr, "Finds the code a change affects that the session never looked at.\n\n")
		fmt.Fprintf(os.Stderr, "Usage:\n  entire umbra <checkpoint-id | commit-ish> [flags]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExit codes: 0 completed, 2 the --fail-on condition was met, 1 runtime error.\n")
	}

	// PRD.md puts the reference first: `entire umbra <ref> [flags]`. The
	// standard flag package stops parsing at the first non-flag argument, so
	// the flags after the reference would be dropped silently. Move
	// positionals to the end before parsing.
	if err := fs.Parse(reorder(fs, argv)); err != nil {
		return nil, err
	}
	if *showVersion {
		fmt.Printf("umbra %s (%s)\n", Version, Commit)
		return nil, nil
	}

	o.Ref = fs.Arg(0)
	if o.Ref == "" {
		o.Ref = "HEAD"
	}
	if o.Repo == "" {
		o.Repo = os.Getenv("ENTIRE_REPO_ROOT")
	}
	if o.Repo == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("cannot determine the working directory: %w", err)
		}
		o.Repo = wd
	}
	abs, err := filepath.Abs(o.Repo)
	if err != nil {
		return nil, err
	}
	o.Repo = abs

	if err := o.Validate(); err != nil {
		return nil, err
	}
	return o, nil
}

// analyze is the pipeline. Phases land here one at a time; what is not built
// yet is stated plainly rather than faked.
func analyze(ctx context.Context, o *Options) (int, error) {
	exec := runner.NewExec(o.Repo)

	res, err := checkpoint.New(exec, o.Repo).Resolve(ctx, o.Ref)
	if err != nil {
		return ExitRuntime, err
	}

	data := checkpoint.DataDir()
	pair, err := checkpoint.AddWorktrees(ctx, exec, o.Repo, data, res, o.KeepWorktrees)
	if err != nil {
		return ExitRuntime, err
	}
	defer func() {
		if rmErr := pair.Remove(context.WithoutCancel(ctx)); rmErr != nil {
			fmt.Fprintf(os.Stderr, "umbra: could not remove worktrees: %v\n", rmErr)
		}
	}()

	a, st, err := pipeline(ctx, o, exec, res, pair)
	if err != nil {
		return ExitRuntime, err
	}

	if err := execute(ctx, o, exec, pair, a, st); err != nil {
		return ExitRuntime, err
	}

	// One choke point. Seal scrubs the analysis and builds the layout from
	// the scrubbed values; every renderer takes the sealed value and nothing
	// else can be written.
	sd := report.Seal(a, st.Scrub)
	if err := emit(o, sd); err != nil {
		return ExitRuntime, err
	}
	return failOn(o, a), nil
}

// failOn turns the requested condition into an exit code. Exit 2 means the
// condition held; it is not an error.
func failOn(o *Options, a *report.Analysis) int {
	switch o.FailOn {
	case "umbra":
		if a.Summary.Umbra > 0 {
			return ExitFailOn
		}
	case "penumbra":
		if a.Summary.Umbra > 0 || a.Summary.Penumbra > 0 {
			return ExitFailOn
		}
	// A failure verify counted but could not name is still a failure, and a
	// leak list verify told us was capped is still a leak. Gating on the
	// length of a truncated list is the same mistake the sweep itself made.
	// Degraded is deliberately not consulted here: a suite-level pass with no
	// per-test ids reports no failures, and that is not a condition to exit 2
	// on.
	case "failure":
		if len(a.Execution.NewFailures) > 0 || a.Execution.UnnamedFailures > 0 {
			return ExitFailOn
		}
	case "leak":
		if len(a.Execution.Leaks) > 0 || a.Execution.UnnamedFailures > 0 {
			return ExitFailOn
		}
	}
	return ExitOK
}

// reorder moves positional arguments after the flags so a reference typed
// before the flags still parses. A token starting with "-" is a flag; it
// consumes the next token as its value unless it is a boolean or already
// carries "=". Everything after a bare "--" is positional.
func reorder(fs *flag.FlagSet, argv []string) []string {
	var flags, positional []string
	for i := 0; i < len(argv); i++ {
		arg := argv[i]
		if arg == "--" {
			positional = append(positional, argv[i+1:]...)
			break
		}
		if len(arg) < 2 || arg[0] != '-' {
			positional = append(positional, arg)
			continue
		}
		flags = append(flags, arg)
		if strings.Contains(arg, "=") {
			continue
		}
		name := strings.TrimLeft(arg, "-")
		if isBoolFlag(fs, name) {
			continue
		}
		if i+1 < len(argv) {
			i++
			flags = append(flags, argv[i])
		}
	}
	return append(flags, positional...)
}

func isBoolFlag(fs *flag.FlagSet, name string) bool {
	f := fs.Lookup(name)
	if f == nil {
		// Unknown flag. Treat it as boolean so the next token stays
		// positional; flag.Parse reports the unknown flag itself.
		return true
	}
	bf, ok := f.Value.(interface{ IsBoolFlag() bool })
	return ok && bf.IsBoolFlag()
}
