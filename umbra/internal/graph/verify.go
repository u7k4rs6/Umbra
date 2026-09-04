package graph

import (
	"context"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Verdict is what `graph verify` reported.
//
// The Step 0 probe pinned these shapes:
//
//	BASELINE RECORDED: /path (pytest; 14 passing, 0 failing, exit 0)
//	BASELINE RECORDED: /path (exit 0; output format not recognised, so the baseline is exit-code only)
//	NEWLY FAILING (3): tests/a.py::x, tests/a.py::y, tests/a.py::z
//	VERDICT: REGRESSION in 3 tests: ...
//	VERDICT: NO EFFECT ...
//
// The exit code is 0 even on a regression, so the verdict is read from the
// text and never from the status.
type Verdict struct {
	NewlyFailing []string
	NewlyPassing []string
	PreExisting  []string
	Line         string
	// Degraded is set when verify could not parse per-test ids, which happens
	// when the runner is not verbose.
	Degraded bool
	// TimedOut is set when the invocation hit its timeout.
	TimedOut bool
}

var (
	newlyFailingRE = regexp.MustCompile(`(?m)^NEWLY FAILING\s*\(\d+\):\s*(.+)$`)
	newlyPassingRE = regexp.MustCompile(`(?m)^NEWLY PASSING\s*\(\d+\):\s*(.+)$`)
	preExistingRE  = regexp.MustCompile(`(?m)^PRE-EXISTING[^:]*:\s*(.+)$`)
	verdictRE      = regexp.MustCompile(`(?m)^VERDICT:\s*(.+)$`)
	degradedRE     = regexp.MustCompile(`(?i)output format not recognised|exit-code only`)
)

// ParseVerdict reads the verify output.
func ParseVerdict(text string) Verdict {
	var v Verdict
	v.NewlyFailing = splitIDs(newlyFailingRE, text)
	v.NewlyPassing = splitIDs(newlyPassingRE, text)
	v.PreExisting = splitIDs(preExistingRE, text)
	if m := verdictRE.FindStringSubmatch(text); m != nil {
		v.Line = strings.TrimSpace(m[1])
	}
	v.Degraded = degradedRE.MatchString(text)
	return v
}

func splitIDs(re *regexp.Regexp, text string) []string {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return nil
	}
	var out []string
	for _, part := range strings.Split(m[1], ",") {
		part = strings.TrimSpace(part)
		// verify caps its lists at twenty and appends a count.
		if part == "" || strings.HasPrefix(part, "and ") {
			continue
		}
		out = append(out, part)
	}
	sort.Strings(out)
	return out
}

// VerifyRequest is one pair of runs: record a baseline on the parent tree,
// then adjudicate the same command on the head tree.
type VerifyRequest struct {
	BaseRepo     string
	HeadRepo     string
	Runner       []string
	TestIDs      []string
	BaselinePath string
	Timeout      time.Duration
}

// Command returns the argv Umbra passes to verify's --test flag.
//
// The runner prefix was split into argv once, at flag parsing, and the test
// ids are appended as separate elements. Umbra never joins them back into a
// shell string itself; verify takes a single --test string, so the pieces are
// joined here with spaces and every id has already been validated against
// ^[A-Za-z0-9_./:\[\]\-]+$, which cannot contain a space, a quote or a shell
// metacharacter.
func (r VerifyRequest) Command() string {
	parts := append(append([]string(nil), r.Runner...), r.TestIDs...)
	return strings.Join(parts, " ")
}

// Verify runs the baseline and the adjudicated run.
//
// A missing base worktree, which happens for a root commit, means there is no
// baseline: the head run still happens and the verdict is a state rather than
// a delta. That is reported, not hidden.
func Verify(ctx context.Context, run runner.Runner, req VerifyRequest) (Verdict, []string, error) {
	var commands []string
	var baselineDegraded bool
	cmd := req.Command()

	timeout := req.Timeout
	if timeout <= 0 {
		timeout = runner.VerifyTimeout
	}

	if req.BaseRepo != "" {
		args := []string{"graph", "verify", "--repo", req.BaseRepo, "--test", cmd, "--record-baseline", req.BaselinePath}
		commands = append(commands, "entire "+strings.Join(args, " "))

		bctx, cancel := context.WithTimeout(ctx, timeout)
		stdout, _, _, err := run.Run(bctx, "entire", args, nil)
		cancel()
		if err != nil {
			if bctx.Err() != nil {
				return Verdict{TimedOut: true}, commands, nil
			}
			return Verdict{}, commands, err
		}
		// A baseline that could not name tests means the runner was not
		// verbose. Carry that forward so the report can say so even when the
		// head run happens to print a parseable line.
		baselineDegraded = degradedRE.MatchString(string(stdout))
	}

	args := []string{"graph", "verify", "--repo", req.HeadRepo, "--test", cmd}
	if req.BaseRepo != "" {
		args = append(args, "--pre-edit-baseline", req.BaselinePath)
	} else {
		args = append(args, "--record-baseline", req.BaselinePath+".head")
	}
	commands = append(commands, "entire "+strings.Join(args, " "))

	hctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	stdout, stderr, _, err := run.Run(hctx, "entire", args, nil)
	if err != nil {
		if hctx.Err() != nil {
			return Verdict{TimedOut: true}, commands, nil
		}
		return Verdict{}, commands, err
	}

	text := string(stdout)
	if strings.TrimSpace(text) == "" {
		text = string(stderr)
	}
	v := ParseVerdict(text)
	v.Degraded = v.Degraded || baselineDegraded
	return v, commands, nil
}
