package main

import (
	"context"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Scrubber returns the one scrubber for this run.
//
// There used to be two constructions of it, one in `umbra record` and one in
// the analysis pipeline, and only the recording path asked git for the author
// names. An ordinary run therefore left the repository owner's name in the
// timeline. The inputs are derived here, once, and every path that needs a
// scrubber gets this same value.
func (o *Options) Scrubber(ctx context.Context, run runner.Runner) *report.Scrubber {
	if o.scrub != nil {
		return o.scrub
	}
	s := report.NewScrubber(o.Repo)
	// A display name cannot be recognised by shape, so the scrubber has to be
	// told. A session's own shell commands carry the author name whenever it
	// ran git with an explicit identity, and those commands reach the
	// timeline.
	s.Names = gitNames(ctx, run, o.Repo)
	o.scrub = s
	return s
}

// gitNames asks git for the display names configured in this repository.
func gitNames(ctx context.Context, run runner.Runner, repo string) []string {
	var out []string
	for _, key := range []string{"user.name", "author.name", "committer.name"} {
		stdout, _, exit, err := run.Run(ctx, "git", []string{"-C", repo, "config", "--get", key}, nil)
		if err != nil || exit != 0 {
			continue
		}
		if name := strings.TrimSpace(string(stdout)); name != "" {
			out = append(out, name)
		}
	}
	return out
}
