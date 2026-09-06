package graph

import (
	"context"
	"regexp"
	"sort"
	"strconv"
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
// The first real repository added two more, and they are why this type carries
// counts as well as ids. `graph verify --help` says its id lists "cap at 20
// with a count", and that `--max-bytes` caps the rendered verdict while "the
// verdict clause always survives". So a large regression arrives either as
//
//	NEWLY FAILING (51): <20 ids>, … and 31 more
//	VERDICT: REGRESSION in 51 tests: <20 ids>, … and 31 more
//
// or, once the pair no longer fits the byte budget, as the verdict clause on
// its own with no NEWLY FAILING line at all. Reading ids from the list alone
// therefore reports nothing for the largest regressions, which is exactly the
// case the sweep exists to catch. The recorded shapes are in
// testdata/verify/ and NOTES.md carries them verbatim.
//
// The exit code is 0 even on a regression, so the verdict is read from the
// text and never from the status.
type Verdict struct {
	NewlyFailing []string
	NewlyPassing []string
	PreExisting  []string
	Line         string
	// FailingCount is how many tests verify said newly failed. It is
	// authoritative: the id list is capped and the whole verdict is capped in
	// bytes, so len(NewlyFailing) is a lower bound and this is not.
	FailingCount int
	// PassingCount is the same number for newly passing tests.
	PassingCount int
	// Degraded is set when verify could not parse per-test ids, which happens
	// when the runner is not verbose.
	Degraded bool
	// TimedOut is set when the invocation hit its timeout.
	TimedOut bool
}

// UnnamedFailing is how many newly failing tests verify counted but did not
// name. Zero means the id list is the whole story.
func (v Verdict) UnnamedFailing() int {
	if v.FailingCount <= len(v.NewlyFailing) {
		return 0
	}
	return v.FailingCount - len(v.NewlyFailing)
}

var (
	newlyFailingRE = regexp.MustCompile(`(?m)^NEWLY FAILING\s*\((\d+)\):\s*(.+)$`)
	newlyPassingRE = regexp.MustCompile(`(?m)^NEWLY PASSING\s*\((\d+)\):\s*(.+)$`)
	preExistingRE  = regexp.MustCompile(`(?m)^PRE-EXISTING[^:]*:\s*(.+)$`)
	verdictRE      = regexp.MustCompile(`(?m)^VERDICT:\s*(.+)$`)
	regressionRE   = regexp.MustCompile(`(?m)^VERDICT:\s*REGRESSION in (\d+) tests?(?::\s*(.*))?$`)
	degradedRE     = regexp.MustCompile(`(?i)output format not recognised|exit-code only`)
	// truncationRE matches the marker verify appends when it capped the list.
	// It appears as "… and 31 more" after a comma, and at a small byte budget
	// as a bare "…" separated from the last id by a space rather than a comma.
	truncationRE = regexp.MustCompile(`^(?:…|\.\.\.)?\s*(?:and\s+\d+\s+more)?$`)
	tailEllipsis = regexp.MustCompile(`\s+(?:…|\.\.\.)$`)
)

// ParseVerdict reads the verify output.
func ParseVerdict(text string) Verdict {
	var v Verdict
	v.FailingCount, v.NewlyFailing = countedIDs(newlyFailingRE, text)
	v.PassingCount, v.NewlyPassing = countedIDs(newlyPassingRE, text)
	if m := preExistingRE.FindStringSubmatch(text); m != nil {
		v.PreExisting = splitIDs(m[1])
	}
	if m := verdictRE.FindStringSubmatch(text); m != nil {
		v.Line = strings.TrimSpace(m[1])
	}

	// The verdict clause is the one verify guarantees to print, so it is the
	// only place a count can always be found. Above the rendering budget it is
	// also the only place any id survives.
	if m := regressionRE.FindStringSubmatch(text); m != nil {
		n, err := strconv.Atoi(m[1])
		if err == nil && n > v.FailingCount {
			v.FailingCount = n
		}
		if len(v.NewlyFailing) == 0 {
			v.NewlyFailing = splitIDs(m[2])
		}
	}

	v.Degraded = degradedRE.MatchString(text)
	return v
}

// countedIDs returns the count in the line's header and the ids it listed,
// which are not the same number once verify has capped the list.
func countedIDs(re *regexp.Regexp, text string) (int, []string) {
	m := re.FindStringSubmatch(text)
	if m == nil {
		return 0, nil
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		n = 0
	}
	return n, splitIDs(m[2])
}

func splitIDs(list string) []string {
	if strings.TrimSpace(list) == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(list, ",") {
		part = strings.TrimSpace(part)
		// verify caps its lists at twenty and appends a count. The marker is
		// its own comma-separated part when there was room for one, and is
		// otherwise glued to the last id by a space.
		if truncationRE.MatchString(part) {
			continue
		}
		part = strings.TrimSpace(tailEllipsis.ReplaceAllString(part, ""))
		if part == "" {
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
