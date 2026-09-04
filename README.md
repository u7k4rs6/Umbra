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
go install github.com/u7k4rs6/Umbra/umbra/cmd/entire-umbra@main
```

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
git clone https://github.com/u7k4rs6/Umbra && cd Umbra
cd umbra/fixtures/app && ./setup.sh && cd ../../..
. umbra/fixtures/app/.venv/bin/activate
entire umbra 0063443 --test "pytest -v"
```

That reports 8 probes selected and 5 cracked, naming `test_empty_is_zero`,
`test_negative` and `test_rounding` in `tests/test_service.py` and both refund
tests in `tests/test_refunds.py`, with a full sweep and 0 leaks.

**The runner has to be on `PATH`, which is why the venv is activated.** Umbra
runs the tests in a detached worktree of the commit, not in your working tree,
so a runner given as a path relative to the repository root does not resolve
from there, and the virtualenv is not in the worktree at all because it is not
committed. A runner that cannot start produces no output, and `verify` then has
nothing to parse: the verdict degrades to a suite-level pass or fail and no
test is named. That looks exactly like the `pytest -q` failure and has a
different cause. Activating the venv, or giving an absolute path to the
interpreter, avoids both.

The fixture is seeded. It is built to produce lit, penumbra and umbra nodes and
a signature change that breaks tests in two files the session never opened. The
report says so and so does this README.

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
- The examined set is built from the session's tool activity. An agent that
  reads and edits through shell commands rather than through file tools leaves
  no read or edit event, so its work looks unexamined. Umbra found this in its
  own build; the report is in `umbra/site/self/`.

## Not built, and why

Adapters for other agents (they get the unknown state, which is correct for a
transcript with no tool records). Subagent transcripts. Live mode during a
session and a blind-spot budget gate (shadow-branch data is not documented
output). Mutability analysis of callees (nothing in the documented surface
provides it). Coverage deltas (needs coverage tooling in the target
repository). Trend dashboards. Cross-repository dependents. And Umbra never
writes code: it shows the shadow, it does not fix it.

## The final review

The last step of the build was a semantic-diff review of the whole thing,
comparing the first commit against the last:

```
entire graph diff --base 6e0dff8e92417fc25e28f07c4950753bd1701966 --head HEAD --repo .
```

The diff itself is unremarkable: 1014 added entities, no signature changes, and
no function without an incoming edge, which is what a build that only ever added
code should look like. The interesting part was what the diff did not contain.

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

## Prior work and AI disclosure

No code was reused. The idea of subtracting the examined set from a blast
radius was developed in planning for this event.

All product code was written during the build with an AI coding agent, captured
in Entire checkpoints. The four planning documents in `umbra/docs/` were written
before the build with AI assistance and are the first commit. `umbra/NOTES.md`
records what the probe found, which degradations are in effect, and the places
the design documents turned out to be wrong about the installed tools.

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
  fixtures/recorded/    the eight scenarios the replay tests drive
  site/                 landing page and the drop-in viewer
  docs/                 the four planning documents
```

Built during Bengaluru Tech Week on the Entire ecosystem.
