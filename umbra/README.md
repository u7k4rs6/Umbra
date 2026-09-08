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
Umbra  01M1PVD1WBK1J1J8GWDNJJR7XZ  1c2cf29  claude-code  depth 2
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

## The three evidence tiers

The state says how well the *session* saw a dependent. It says nothing about
how well the *graph* knows the dependent is there. That is a separate axis with
three values, printed as a mark beside the state and in full in every other
surface.

| Tier | Mark | Meaning |
|---|---|---|
| confirmed | `=` | The graph resolved every relation on this path to a definition |
| heuristic | `~` | The graph derived this relation rather than parsing it, so it may be wrong |
| needs verification | `?` | Reached through a relation the graph could not resolve, or found under a partial analysis; check it against the source |

A relation is heuristic when the graph resolved it by something other than a
definition: `type_inferred`, `name_only`, `package`, `pattern` or
`git_history`. If such a hop is the last one, the node itself is heuristic. If
it is further up the path, the node needs verification instead, because the
reader cannot see the hop that is in doubt from the row.

A run the graph reports as incomplete puts every node in needs verification,
whatever its own edges look like, because the graph cannot say which edges the
incompleteness affected. The header then names the reason.

**The tiers are not a fifth state and the ranker does not read them.** A
heuristic edge does not lower a score. The two axes answer different questions
and mixing them would make both unreadable: a node can be in full shadow on
evidence the graph is certain of, or lit on evidence it guessed.

For a node that needs verification, every surface prints the command that
settles it, `entire graph def` with the symbol and file, plus the call site and
the test that reaches it where there is one. A healthy run has none, so that
section is absent rather than empty.

## Install

```
git clone https://github.com/u7k4rs6/Umbra
cd Umbra/umbra
go build -o "$HOME/.local/bin/entire-umbra" ./cmd/entire-umbra
```

**`go install` works, but only with `GOPRIVATE` set**, and the README has said
three different things about this. The module path is
`github.com/u7k4rs6/Umbra/umbra` and `main` does carry `umbra/`, so the obvious
command resolves. It then fails in verification, because the repository is
private and the public checksum database cannot see it:

```
$ go install github.com/u7k4rs6/Umbra/umbra/cmd/entire-umbra@main
go: ...verifying module: reading https://sum.golang.org/lookup/...: 404 Not Found
    not found: invalid version: git ls-remote ...: exit status 128
```

With `GOPRIVATE='github.com/u7k4rs6/*'` in front, the same command succeeds and
puts `entire-umbra` in `$(go env GOBIN)`. Both forms were run before this was
written.

What it installs is the last pushed commit on `main`, not your checkout. The
clone and build above is the path every run in this project has actually used,
and it is the one to use while working on Umbra itself.

While this repository stays private, `go install` is only useful to someone
whose git already has credentials for it.

Any executable named `entire-<name>` on `$PATH` is dispatched by the Entire CLI
kubectl style, so this is all the installation there is. Requires the Entire
CLI with Checkpoints enabled and the entire-graph plugin.

## Run

```
entire umbra HEAD --test "pytest -v" --out ./umbra-out
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
git clone https://github.com/u7k4rs6/Umbra
cd Umbra
(cd umbra && go install ./cmd/entire-umbra)
cd umbra/fixtures/app && ./setup.sh && cd ../../..
. umbra/fixtures/app/.venv/bin/activate
entire umbra 1c2cf29 --test "pytest -v"
```

That reports 8 probes selected and 5 cracked, naming `test_empty_is_zero`,
`test_negative` and `test_rounding` in `tests/test_service.py` and both refund
tests in `tests/test_refunds.py`, with a full sweep and 0 leaks.

`1c2cf29` is the seeded signature change, committed with an agent's own message,
`make the tax rate explicit on compute_total`. Two commits matter and they are
adjacent. Its parent, `6eb4434`, `umbra: fixture app for the probe`, is the
state with all 16 fixture tests green, which is what the probe run records as
its baseline so that a test that was already red is never reported as newly
broken. If a rebase has moved them:

```
git log --format='%H %s' | grep -m1 'make the tax rate explicit on compute_total'
```

An earlier version of this block carried `0063443` and looked the commit up by
the subject `umbra: seed the fixture signature change`. Both belong to the fork,
`u7k4rs6/entire-graph`, whose history is not this one. Neither the hash nor that
subject resolves here, so the block could not run.

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
`go: cannot find main module, but found .git/config`, because there is no Go
module at the root of this repository at all. In the fork the same line fails
differently, with `main module (github.com/entireio/entire-graph) does not
contain package ...`, because there the root module is the upstream one. The clone also
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
whichever checkpoint the reference pairs with. The probes, the cracks and the
sweep are the same everywhere; the states are not.

In this repository the checkpoints are present, so the block above reports

```
coverage  2 file reads, 1 edit, 0 searches, 2 shell commands
light  ●●◐○○○○○○○○   2 lit  1 penumbra  8 umbra  0 unknown   18% examined
graph  11 confirmed  0 heuristic  0 needs verification   profile full
probes  8 selected  5 cracked
sweep   full suite  0 leaks
```

which was measured rather than remembered. A clone that fetched no checkpoint
refs gets the same probes, cracks and sweep and different states, because there
is then no examined set to subtract.

The same run in the fork is a good illustration and an unflattering one. It
reported `0 lit  9 penumbra  4 umbra  0 unknown` and `0% examined`, above a
coverage line reading `0 file reads, 0 edits, 0 searches, 33 shell commands`.
The session that made the change worked through the shell rather than through
file tools, so it left no read events for Umbra to subtract, and almost
everything came back unexamined. That is the first limitation below. The
coverage line exists so that a thin report of this kind cannot be mistaken for
a real finding.

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
   header said `1c2cf29` while the sources were the transcript package from
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

## Layout

Everything below is relative to `umbra/`, which is the whole module.

```
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

`NOTES.md` carries the build record, the corrections above in fuller form, and
the running list of checks that passed while asserting nothing. The repository
this module sits in is described in the [repository README](../README.md).
