# Umbra

**Umbra finds the code an AI agent's change affects but the agent never looked at.**

It is a plugin for the [Entire CLI](https://github.com/entireio/cli), run as
`entire umbra <checkpoint>`. It takes the dependents of every changed symbol
from [Entire Graph](https://github.com/entireio/entire-graph), subtracts what
the session actually examined according to the checkpoint's tool activity and
the agent's own words, ranks what is left in shadow, runs the tests that reach
it, and then sweeps the whole suite once so the selection audits itself.

The report is an eclipse map you can replay in the order the agent looked at
things, plus a one-page packet a reviewer reads in a minute.

The core is deterministic and offline. **There is no model anywhere in Umbra.**

## Who it is for

The reviewer of an agent-authored change, who wants to know where to look
first. A clean diff and a confident description tell you what changed. They do
not tell you what the agent never opened.

The everyday case: a signature change on `compute_total`, seven callers, three
of them tests in a file the session never opened, committed with a message that
says the callers were checked because only one test file ran.

```
Umbra  01M1PVD1WBK1J1J8GWDNJJR7XZ  0063443  claude-code  depth 2
session said  "Checked the callers and updated them; all tests pass."
compute_total  signature changed  app/service.py:20
light  ●●◐◐◐○○○○○○   2 lit  3 penumbra  6 umbra  0 unknown   18% examined

!◐  apply_refund        app/refunds.py:17      direct caller  depth 1  echo fault line beacon  10.5
    # SAFETY: a pricing failure must not refund the full amount paid.
 ○  test_rounding       tests/test_service.py:7  direct caller  depth 1  test far field   12.0  fail
 ○  test_negative       tests/test_service.py:13 direct caller  depth 1  test far field   12.0  fail

probes  8 selected  5 cracked
sweep   full suite  0 leaks
```

## The four states

| State | Glyph | Meaning |
|---|---|---|
| Lit | ● | Read with a range covering the symbol, at or after the change began; or edited |
| Penumbra | ◐ | Half seen: `glance`, `glimpse`, `quoted`, `afterimage` or `echo` |
| Umbra | ○ | Nothing in the session touched or mentioned the file |
| Unknown | ? | The transcript carries no tool events and no mentions |

Missing data is a state, never an error. A transcript with no tool records
gives every node the unknown state, which is the correct answer rather than a
guess, and the selection falls back to the graph alone.

## Install

```
git clone -b umbra-buildathon https://github.com/u7k4rs6/entire-graph
cd entire-graph/umbra
go build -o "$HOME/.local/bin/entire-umbra" ./cmd/entire-umbra
```

**`go install` does not work for this module, and the README used to say it
did.** Two things are wrong with the obvious command. `@main` names a branch
that has no `umbra/` directory, because `main` tracks upstream
`entireio/entire-graph`. And naming a branch or a commit that does have it
still fails, because the module carries no tagged version:

```
$ go install github.com/u7k4rs6/entire-graph/umbra/cmd/entire-umbra@umbra-buildathon
go: ...loading deprecation for github.com/u7k4rs6/entire-graph/umbra:
    no matching versions for query "latest"
```

Both forms were run before this was written. The clone and build above is the
path every run in this project has actually used.

Any executable named `entire-<name>` on `$PATH` is dispatched by the Entire CLI
kubectl style, so this is all the installation there is. Requires the Entire
CLI with Checkpoints enabled and the entire-graph plugin.

## Run

```
entire umbra 6dea614c --test "pytest -v" --out ./umbra-out
open ./umbra-out/umbra.html
```

Use `pytest -v`, not `pytest -q`. Quiet mode prints no per-test names, so
`entire graph verify` cannot tell you which test broke and the verdict degrades
to a suite-level pass or fail. Umbra says so in the header when that happens.

```
entire umbra <checkpoint-id | commit-ish> [flags]

  --test CMD        test runner prefix; validated test ids are appended as argv
  --run none|shadow|all    which selected tests to run (default shadow)
  --no-audit        skip the full-suite sweep, which is on by default
  --history         add the opt-in scar factor from git history
  --depth 1|2|3     dependency hops (default 2)
  --format table|json|html|packet
  --out PATH        write umbra.json, umbra.html and umbra.packet.md
  --fail-on umbra|penumbra|failure|leak    exit 2 when the condition holds
  --snippets        include declaration lines in the report (off by default)
  --all             list lit nodes in the table instead of counting them
  --test-root PATH  where the runner runs, when the project is nested
```

Exit codes: 0 completed, 2 the `--fail-on` condition was met, 1 runtime error.

## Reproduce the demo

```
git clone entire://aws-ap-south-1.entire.io/gh/u7k4rs6/entire-graph
cd entire-graph
(cd umbra && go install ./cmd/entire-umbra)
cd umbra/fixtures/app && ./setup.sh && cd ../../..
. umbra/fixtures/app/.venv/bin/activate
entire umbra "$(git log --format='%H %s' | grep -m1 'umbra: seed the fixture signature change' | cut -d' ' -f1)" --test "pytest -v"
```

That reports 8 probes selected and 5 cracked, naming `test_empty_is_zero`,
`test_negative` and `test_rounding` in `tests/test_service.py` and both refund
tests in `tests/test_refunds.py`, with a full sweep and 0 leaks.

The commit is looked up by subject rather than written down as a hash. The hash
this block used to carry, `0063443`, belonged to the repository Umbra was built
in, and that history was deliberately not carried across the graft, so the
block could not run here at all. The subject is stable across a rebase and is
the same in every clone. To see it before running:

```
git log --format='%H %s' | grep 'umbra: seed the fixture signature change'
```

Two commits matter and they are adjacent. The one above is the change. Its
parent, `umbra: fixture baseline before the seeded signature change`, is the
state with all 16 fixture tests green, which is what the probe run records as
its baseline so that a test that was already red is never reported as newly
broken.

**The runner has to be on `PATH`, which is why the venv is activated.** Umbra
runs the tests in a detached worktree of the commit, not in your working tree,
so a runner given as a path relative to the repository root does not resolve
from there, and the virtualenv is not in the worktree at all because it is not
committed. A runner that cannot start produces no output, and `verify` then has
nothing to parse: the verdict degrades to a suite-level pass or fail and no
test is named. That looks exactly like the `pytest -q` failure and has a
different cause. Activating the venv, or giving an absolute path to the
interpreter, avoids both.

The `go install` line is what puts `entire-umbra` on `$PATH`, which is all the
installation there is: the Entire CLI dispatches any `entire-<name>` executable
it finds there. It installs from the clone rather than from a module path so
that what runs is the code in the checkout you just made. It runs inside a
subshell that changes directory first, because `umbra/` is its own Go module:
`go install ./umbra/cmd/entire-umbra` from the repository root fails with
`main module (github.com/entireio/entire-graph) does not contain package ...`,
which is what running this block from a fresh clone turned up. The clone also
has to sit somewhere Graph will run: Graph refuses git subprocesses
against repository metadata it considers unsafe, which in practice means a
clone under a shared temporary directory fails before any analysis happens.
A clone under your home directory works. This was checked from a fresh clone
rather than assumed, and both readings are in the limitations below.

The fixture is seeded. It is built to produce lit, penumbra and umbra nodes and
a signature change that breaks tests in two files the session never opened. The
report says so and so does this README.

The states you get will not match the example at the top of this README, and
that is correct. Node states come from the examined set, which comes from
whichever checkpoint the reference pairs with, and checkpoint history is local
to a machine and is not pushed. The probes, the cracks and the sweep are the
same everywhere; the states are not.

The run that seeded this commit is a good illustration and an unflattering one.
It reported `0 lit  9 penumbra  4 umbra  0 unknown` and `0% examined`, above a
coverage line reading `0 file reads, 0 edits, 0 searches, 33 shell commands`.
The session that made the change worked through the shell rather than through
file tools, so it left no read events for Umbra to subtract, and almost
everything came back unexamined. That is the first limitation below, visible in
the one report this README tells you to produce. The coverage line exists so
that a thin report of this kind cannot be mistaken for a real finding.

## How the ranking works

Six factors, all printed next to the node so the number can be recomputed by
hand:

1. **Relation weight.** Direct caller 3, data flow 2, type consumer 2,
   transitive caller 1.5, co-change file 1.
2. **What depends on it.** Multiplied by `1 + log2(1 + dependents)`.
3. **How weakly it was seen.** Multiplied by 1.0 umbra, 0.7 echo,
   0.6 afterimage, 0.5 other penumbra tiers, 0 lit.
4. **Far field.** Multiplied by 1.5 across a package boundary.
5. **Fault line.** Multiplied by 1.5 when the call site sits inside error
   handling.
6. **Tests.** Plus 3 for a test function, plus 2 when a test reaches the node.

**The beacon override**: a call site within four lines of a `SAFETY`,
`CRITICAL` or `INVARIANT` comment is pinned to the top of the docket whatever
it scored, with the comment shown. Someone wrote that note by hand; it outranks
a heuristic.

**The scar factor** is opt in with `--history` and multiplies by up to 2 for a
file with several recent fix commits. It is off by default because it is noisy.

Ties break by file path then symbol name, so two runs on one checkpoint produce
the same order.

## The sweep

After the selected tests, the whole suite runs once by default. Every test that
changed state but was not selected is reported as a leak with the reason the
selection missed it: a path deeper than the search went, no resolved call edges
from the test file, a path only over a relation Umbra does not traverse, or
nothing the graph can see. The count is printed even when it is zero, because a
selection that audits itself is worth more than one that does not. `--no-audit`
skips it and the header says the selection is unaudited.

## Security

- Umbra runs as the invoking user and asks for nothing more. It sees exactly
  what you can see through git and the Entire CLI.
- **No transcript prose reaches any output**, with one named exception: the
  "session said" sentence, taken from Entire's stored checkpoint summary or, if
  there is none, the agent's own last sentence. It is scrubbed and capped at
  200 characters, and it is displayed, never verified.
- **Umbra never runs a command it found in a transcript.** Test ids are built
  from graph data, validated against `^[A-Za-z0-9_./:\[\]\-]+$`, and passed as
  separate argv elements. There is no `sh -c` anywhere in Umbra.
- The runner comes only from `--test`. Umbra echoes the full command before
  running it, and everything runs in detached worktrees, never in your working
  tree.
- The report is one self-contained file with a Content-Security-Policy that
  forbids external scripts, styles, images and connections. It makes no
  requests. The drop-in viewer on the landing page reads files in your browser
  and uploads nothing.
- Reports name symbols, files and spans inside the blast radius. Share one with
  the same care you would share a diff.

## Documented but not built

The four planning documents in `docs/` are the original design and stay as
written, per the standing rule of this build. That means they describe some
things the binary does not do. Rather than quietly edit the design to match the
code, every gap is listed here with the manual alternative where one exists.

| Described | Where | Reality | What to do instead |
|---|---|---|---|
| `.umbra.json`, a committed file supplying the runner prefix | `docs/SECURITY_AND_ACCESS.md` lines 16 and 25 | Never implemented. The runner comes only from `--test`. A comment in `internal/report/seal.go` still refers to it | Pass `--test "pytest -v"` on every run |
| `umbra clean`, which "removes all" worktrees | `docs/SECURITY_AND_ACCESS.md` line 28 | No `clean` subcommand exists | Delete by hand: `rm -rf "$ENTIRE_PLUGIN_DATA_DIR/wt"` under a managed install, otherwise `rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/umbra/wt"` |
| `umbra record --scrub` | `docs/SECURITY_AND_ACCESS.md` line 55 | There is no `--scrub` flag. Scrubbing is not optional and always runs; the `--no-scrub` escape hatch that once existed was removed | Nothing. `umbra record` already scrubs everything it writes |
| A 60 second timeout on any single Graph call, and a 30 second snapshot warning | `docs/SECURITY_AND_ACCESS.md`, execution policy | Not implemented per call. The timeouts that exist are the 10 minute verify invocation and the 20 minute sweep | Interrupt a long Graph call yourself. A slow snapshot will not warn |

### Language support is narrower than the design implies

**Test selection and execution work for Python, through pytest.** That is the
path the fixture exercises and the only one with end to end coverage.

**Go test selection does not work yet**, and the reason is specific. A test id
is built as `file + "::" + name` in `internal/shadow/select.go`, which is
pytest's form. `VerifyRequest.Command` then appends the ids to the runner
prefix as plain arguments. For pytest that is correct. For `go test` a non-flag
argument is a **package pattern**, not a filter, so an id like
`internal/shadow/select_test.go::TestSelectTests` is read as a package name and
does not select anything. Making it work needs the id translated into
`-run '^TestName$'` plus the package path, which is not done.

Umbra will still map and rank shadow in a Go repository. It is the probe
selection and execution step that does not apply. Run with `--run none` there,
and read the map and the docket rather than the probes.

### Flags that are narrower than they look

`--adapter` accepts `auto` and `claude-code` and **rejects anything else with a
clear message**, which was verified rather than assumed:

```
$ entire umbra HEAD --adapter codex --run none
umbra: --adapter must be auto or claude-code, not "codex"
```

Both accepted values resolve to the same adapter, and the report's adapter
field is set to `claude-code` directly in `cmd/entire-umbra/analyze.go`, so the
flag is validated and then not otherwise used. It cannot mislead, because an
unsupported value never runs, but it also cannot select anything.

## Limitations

- Read evidence is file and line-range based. A mention in the agent's text is
  attention, not reading, and is ranked as the weakest tier.
- Graph edges are not complete: reflection, dynamic dispatch and configuration
  are invisible, so umbra is a **lower bound** on what was unexamined. The
  sweep's leak reasons are how a reader sees the gap.
- Fault lines and beacons are matched by a regular expression window around the
  call-site line, not by a parser.
- Ranking weights are hand set and printed. They are a heuristic, not a
  measurement.
- Scrubbing of command prefixes and failure excerpts is pattern based.
- One transcript adapter, for Claude Code. An agent whose transcript carries no
  tool records gets the unknown state.
- A destructive `--test` runner does what it says. The echo before execution is
  the only guard.
- **Worktrees are keyed by commit in a shared directory, so two checkouts of
  one repository collide on one worktree.** They are removed on exit unless
  `--keep-worktrees` was passed, but a run that died before its cleanup, or a
  checkout that has since been deleted, leaves one behind. A leftover is now
  reused only when it is a healthy checkout sitting at the wanted commit, and
  otherwise removed and pruned first, which covers the deleted-checkout case.
  What it does not cover is a healthy worktree belonging to a different
  checkout of the same repository: that one is reused, and it carries the other
  checkout's `.git` link with it, so if that checkout sits somewhere Graph
  refuses to run the failure lands in the run that did nothing wrong. There is
  no `umbra clean` command; `SECURITY_AND_ACCESS.md` describes one and it was
  never built. Delete them by hand:

  ```
  rm -rf "$ENTIRE_PLUGIN_DATA_DIR/wt"                  # managed install
  rm -rf "${XDG_CACHE_HOME:-$HOME/.cache}/umbra/wt"    # otherwise
  ```

  Under a managed install that first path is usually
  `~/.local/share/entire/plugins/data/umbra/wt`.
- **A check that has never been observed to fail is not yet a check.** Four
  verifications in this build returned a confident pass that was wrong: local
  tests green on a repository a clone could not build, a cracked-probe count
  that named a test outside the selection, a commit check that counted `ok`
  lines instead of looking for `FAIL`, and a refs audit that reported thirty
  refs of zero bytes because `git ls-tree` is scoped to the working directory.
  The clearest case had no symptom at all: all four label-placement tests are
  built on `Box.Overlaps`, and making it return false for every pair left every
  one of them passing, across all eight scenarios. They had been proving
  determinism, not collision. Every check named in this README now has a test
  that plants what the check is for and asserts it fires, and each of those was
  proved by mutating the code it checks until it failed. That is the standard;
  it is not a guarantee that nothing else here is passing for the wrong reason.
- The examined set is built from the session's tool activity. An agent that
  reads and edits through shell commands rather than through file tools leaves
  no read or edit event, so its work looks unexamined. Umbra found this in its
  own build; the report is in `umbra/site/self/`. Every report now carries a
  coverage line saying how many of the session's events were file reads and how
  many were shell commands, so a thin report can be told from a real finding.
- **The report's script is about 1215 lines against a design target of 700,**
  and its stylesheet about 480 against 400. Three things account for almost all
  of the excess and none of them were foreseen when the target was set: the
  attention replay carries a second implementation of the classifier, because
  the map has to be reconstructed at any point in the session and that means
  the state rules exist in Go and again in the browser; the day scheme and the
  print block are a second and third set of rules for the light field, since
  removing a glow on a light ground reads as more light rather than less; and
  the node shapes are drawn per state and per penumbra tier so the four states
  survive greyscale. The duplication of the classifier is the part worth
  regretting, and it is guarded rather than trusted: a test feeds eight
  recorded scenarios through both implementations at three playhead positions
  each and compares every state, tier and announcement.

## Not built, and why

Adapters for other agents (they get the unknown state, which is correct for a
transcript with no tool records). Subagent transcripts. Live mode during a
session and a blind-spot budget gate (shadow-branch data is not documented
output). Mutability analysis of callees (nothing in the documented surface
provides it). Coverage deltas (needs coverage tooling in the target
repository). Trend dashboards. Cross-repository dependents. And Umbra never
writes code: it shows the shadow, it does not fix it.

## Corrections to the design documents

Five places where the four planning documents in `umbra/docs/` describe
something the installed tools do not do. All five were forced by real output,
none by preference. The documents are the original design and are left as
written; these are the corrections the code carries.

1. **The checkpoint trailer is not hexadecimal.** ARCHITECTURE.md section 1
   describes `Entire-Checkpoint: <12 hex>`. The installed CLI writes a 26
   character ULID, for example `01M1PVD1WBK1J1J8GWDNJJR7XZ`. A hex-only pattern
   matched nothing, so every commit fell through to the session-window fallback
   while reporting, wrongly, that it carried no trailer. The pattern now takes
   alphanumeric ids of 6 to 40 characters, which covers the ULID, the 12 hex id
   of imported history and the 40 hex id of a carry-forward entry.

2. **`pytest -q` cannot be verified.** PRD.md uses it in the command surface
   and in the demo. Quiet mode prints no per-test ids, so `graph verify`
   answers `output format not recognised` and records zero results. The default
   is now `pytest -v`, and the table says `add -v` when a baseline comes back
   exit-code only.

3. **Flags after the reference were silently dropped.** The surface in PRD.md
   is `entire umbra <ref> [flags]`, but Go's flag package stops parsing at the
   first non-flag argument, so `entire umbra HEAD --run none` ignored
   `--run none` and then failed asking for a test runner. Argv is reordered
   before parsing, with boolean flags handled so they do not swallow the
   reference.

4. **`graph checkpoint` answers about the wrong commit when the pairing was a
   guess.** ARCHITECTURE.md calls it the documented bridge between Graph and
   Checkpoints, and `LoadSources` tried it first. It returns the changes of the
   commit *that checkpoint belongs to*. When a reference is paired with a
   checkpoint by session window, that is a guess about which session produced
   the commit and says nothing about which commit the checkpoint owns. This
   never showed here, because the pairing happens to choose an imported
   checkpoint that owns no commit and the fallback runs. In a clone it chose a
   hook-written checkpoint that does own one, and the report described that
   commit's changes under the header of the one the reader asked for: the
   header said `0063443` while the sources were the transcript package from
   phase 3. The bridge is now used only when the reference resolved through the
   trailer or through the checkpoint id, where the two are the same work.

5. **A stale worktree poisoned every later run.** Worktrees are keyed by commit
   under a shared data directory, so two checkouts of one repository, or a run
   that died before its cleanup, collide on one path. The code reused anything
   with a `.git` in it. A worktree left behind by a checkout that had since
   been deleted made git unable to read its metadata and Graph refuse to run
   there, for every subsequent run against that commit from any checkout. A
   leftover is now reused only when it is a healthy checkout sitting at the
   wanted commit, and otherwise removed and pruned first. The residual case is
   in the limitations above.

Corrections 4 and 5 were both found by the end-to-end tests failing in a fresh
clone after passing here, which is the same lesson as the `.gitignore` below.

Two smaller ones, for completeness. Entire has more than one wording for an
absent summary, so any fully italicised summary block is treated as absent. And
`graph impact --format json` carries `call_site.line` and `additional_sites`
directly, so it is the primary path and the text parser the design sketched is
the fallback, still tested.

## The final review

Phase 11 was a semantic-diff review of the whole build to that point,
comparing the first commit against the last:

```
entire graph diff --base 6e0dff8e92417fc25e28f07c4950753bd1701966 --head HEAD --repo .
```

The diff itself is unremarkable: 1014 added entities, no signature changes, and
no function without an incoming edge, which is what a build that only ever added
code should look like. The interesting part was what the diff did not contain,
and it is the single most useful thing Graph did in this build.

### The .gitignore that hid the command package

Reading the diff, `umbra/cmd` was missing. Not thin, not partial: the graph had
no symbols for it at all, while it had 1014 for everything else. A directory of
Go files that the build had been compiling and testing for nine phases was, as
far as Graph could see, not there.

Graph builds its snapshot from the repository, and it honours `.gitignore`. So
the question was not what Graph had failed to parse. It was what git had been
told to ignore. The answer was the second line of a `.gitignore` I wrote myself
in the first commit, under a heading that says exactly what I meant it to do:

```
# Go build output
entire-umbra
```

I meant that to ignore the compiled binary, which is written as `entire-umbra`.
A gitignore pattern with no slash in it is not anchored to the repository root
and does not distinguish a file from a directory: it matches any path component
with that name, at any depth. The binary is called `entire-umbra`. So is the
directory the binary's source lives in. The pattern matched
`umbra/cmd/entire-umbra/` and everything under it.

`git ls-files umbra/cmd/` returned nothing. `main.go`, `analyze.go`,
`execute.go`, `options.go`, `output.go`, `record.go` and `options_test.go` had
never been committed. The command package went in during phase 2 and was
untracked through phase 10, nine phases in which every commit message reported
a rising test count that included tests in a file the repository did not have.

Nothing caught it, and nothing was going to. `go build ./...` and
`go test ./...` read the working tree, and the files were on disk, so they were
green throughout and told me nothing. `git status` says nothing about a path it
has been told to ignore. `git commit -a` does not add an ignored file. The only
symptom available to me was one I never looked at: a clone.

The fix was to anchor the patterns to the paths the binaries are actually
written to, so a directory name can never collide with them again:

```
/entire-umbra
/umbra/entire-umbra
/umbra-out/
```

The seven files were then committed, and the thing that should have been done
long before was done at last:

```
git clone . /tmp/clone && cd /tmp/clone/umbra
go build ./...   # OK
go test ./...    # every package ok
```

That clone is the verification. Before the fix it would have failed to build,
because `package main` did not exist in the repository. After it, it builds and
every test passes, which is the only evidence that means anything here.

This is the clearest case in the build of Graph driving a decision rather than
confirming one. Nothing else in the toolchain was looking at the repository.
The compiler, the test runner and my own reading were all looking at the
working tree, where the code was present and correct. Graph was looking at what
had actually been committed, and the gap between those two things was an entire
package. A green test run is not the same as a correct repository, and it takes
a tool that reads the repository to tell you which one you have.

### The self report

Then Umbra was run on the session that built it. On the commit where it changed
its own `shadow.Build`, all twelve dependents came back umbra, including every
scenario test. That reading is correct, and the reason is the most useful thing
this project learned about itself: the change was made with a shell command
rather than the editing tool, so the session produced no read and no edit event
for that file. **Umbra's examined set is built from tool activity, so an agent
that edits through the shell leaves no trace Umbra can see.** It is the same
shape of blind spot the product exists to find, pointed at the product. The
report is in `umbra/site/self/`.

Running the report on itself also found two bugs in the scrubber, both fixed:
paths from outside the repository were reaching the timeline through search
output, and the base64 heuristic was redacting git object ids and worktree
paths, which is exactly the text a reader needs in order to check a line.

## Development record

Umbra was built in a standalone repository and then grafted into this fork of
`entireio/entire-graph`. The `umbra/` layout from ARCHITECTURE.md was preserved
exactly through the build so the module could be moved without relocating a
single file, which is why the graft is two commits rather than a rewrite.

**The build's own checkpoint trail did not come across, and that is on purpose.**
The original repository's checkpoint refs hold whole session transcripts,
including material read from an unrelated project in the first minutes of the
build. None of it is pushed here. This mirror starts clean and stays clean, and
this fork's trail begins at the graft rather than at the first line of code. The
phase by phase record survives as prose in [NOTES.md](NOTES.md), which covers
all the phases and carries what each one found and where the design documents
turned out to be wrong.

The only other change outside `umbra/` is one pointer line in the repository
README and the agent files `entire graph init-agents` writes.

Seventeen phases, of which the first thirteen built the product. The last four
were not features:

- **14** replaced the landing page's scripted sample with a real run, added a
  second map from an imported session in another project, and prepared two
  scripts that were deliberately not executed: the graft, and a read-only audit
  of what the checkpoint refs on the remote contain.
- **15** made the scrubber the only exit. Five leaks had been the same bug five
  times, an output path that wrote without scrubbing, so the renderers now take
  a value that only the scrubbing path can produce. It also gave every check in
  the project a test that proves the check can fail.
- **16** restyled the landing page around a drawn eclipse. No raster asset and
  no request for one: the corona is inline SVG built deterministically from the
  report's own numbers.
- **17** is this pass: two visual checks decided from screenshots rather than
  from the CSS, [PUBLISH.md](PUBLISH.md), and the verification of
  this README against the repository.

[PUBLISH.md](PUBLISH.md) is a checklist for the sitting in which
this repository is made public. Nothing in it has been run.

**Which maps are which.** The landing page shows two, and they are not the same
kind of thing.

- The first is a **real session on a seeded fixture**. The session, its tool
  activity and the test results are real: checkpoint `b20f84567474` against
  commit `0063443`, where an agent changed a signature and five probes cracked.
  What is arranged is the fixture underneath it, `umbra/fixtures/app`, which was
  written to have callers a session would plausibly miss. The sentence above the
  map is the agent's own, taken from the stored transcript.
- The second is **not seeded at all**: a real session from another project of
  the builder's, imported with `entire import` and analysed unchanged. Every one
  of the four symbols that depend on what it changed came back umbra. Nothing
  about it was arranged, including the result.

**Nothing on the page is scripted.** There is no map drawn from an invented
report. Until phase 14 there was one: the sample was assembled by the site
generator, with a checkpoint id of `sample000class`, a commit of all zeros and a
session sentence written by hand, under a caption calling it a real report. It
was replaced by a real run, and the generator now refuses a report carrying
those markers so the caption cannot drift away from the file again.

The eight scenarios under `umbra/fixtures/recorded/` are **authored
transcripts**, written to exercise one classifier rule each. They drive tests
and are never rendered as a map on the page. `umbra/fixtures/recorded/minimal/`
is different: a full recording of a real short session, hand reviewed before it
was committed.

## Prior work and AI disclosure

No code was reused. The idea of subtracting the examined set from a blast
radius was developed in planning for this event.

All product code across all seventeen phases was written during the build with
an AI coding agent, captured in Entire checkpoints. That includes the tests, the
landing page, the corona renderer and every document in this repository except
the four planning documents in `umbra/docs/`, which were written before the
build with AI assistance and are the first commit.

What came from the human rather than the agent: the problem, the four settled
decisions in the kickoff, the scope of each phase, the decision to keep the
repository private, and the visual direction for the landing page in phase 16,
which was specified before it was built and not proposed by the agent. Every
phase was reviewed and accepted by hand before the next one started.

`umbra/NOTES.md` records what the probe found, which degradations are in
effect, the places the design documents turned out to be wrong about the
installed tools, and the mistakes: the four confident wrong answers, the five
scrubbing leaks, and the label tests that were proving nothing. Those are in
there because a build record that only lists what worked is not a record.

## Layout

```
umbra/
  cmd/entire-umbra/     flags, wiring, exit codes
  internal/runner/      the one boundary to external processes, and its replay
  internal/checkpoint/  resolve, worktrees, transcript fetch
  internal/transcript/  normalized events; the Claude Code adapter
  internal/graph/       snapshot, relations, BFS, impact, commit, verify
  internal/shadow/      examined set, classifier, ranker, selection, forensics
  internal/report/      table, json, layout, html, packet
  fixtures/app/         the seeded Python service
  fixtures/recorded/    the eight scenarios, plus one real minimal recording
  scripts/              the graft and the read-only checkpoint refs audit
  site/                 landing page, the corona renderer, the drop-in viewer
  docs/                 the four planning documents, and docs/renders/
  NOTES.md              the phase by phase build record
  PUBLISH.md            the checklist for making this repository public
```

Built during Bengaluru Tech Week on the Entire ecosystem.
