# Umbra on a repository that is not our fixture

**Umbra performed badly.** It ran without crashing, it was fast, and its
report was internally consistent, but on `pallets/click` it never traversed
the dependents of the method the session actually changed, it selected one
probe out of 2,059 tests, and when a planted mutation broke 51 tests its
full-suite sweep reported **zero leaks**. The sweep is the mechanism the
README calls "a selection that audits itself", and it failed silently in
exactly the case it exists for. Three of the five failures below are
invisible on `fixtures/app` for structural reasons: the fixture changes only
module-level functions, its blast radius is two hops from a test, and it never
breaks more than five tests at a time. The ranking, judged node by node, was
better than the plumbing: one of the top five was right, one defensible, and
three were wrong, and the one node whose behaviour actually changed was ranked
third.

Nothing in this document was fixed. This is a finding, not a patch.

---

## 1. The target

| | |
|---|---|
| Repository | `pallets/click`, https://github.com/pallets/click |
| License | BSD-3-Clause |
| Base commit | `6aabf099bfdd4c1e75fe8d0e0d4241372b988ab1` ("Stable (#3851)") |
| Analysed commit | `562e458efd9de02e0f0e398381178c794c10990f` |
| Source | 12,727 lines of Python across 17 files under `src/` |
| Tests | 15,240 lines across 58 files under `tests/` |
| Parsed by Graph | 158 files, of which 90 are Python |

### The candidates, before anything was cloned

Sizes and licenses came from the GitHub API. Line counts and suite runtimes
were estimates from prior knowledge at proposal time and were replaced with
measurements after the pick was cloned.

| Candidate | License | Est. source lines | Est. suite | Judgement |
|---|---|---|---|---|
| `pallets/click` | BSD-3-Clause | ~9.5k | ~5s, no network | Picked |
| `pallets/flask` | BSD-3-Clause | ~7k | ~8s | Right size, but much of Flask's behaviour delegates to Werkzeug, which is out of repo. Umbra only traverses in-repo edges, so the blast radius would have been thin for a reason that has nothing to do with Umbra |
| `Textualize/rich` | MIT | ~32k | ~25s | At the top of the range, and many modules are leaf renderables with few internal callers. Slower suite, shallower graph |

Click was picked because the property under test, a changed symbol having
several genuine in-repo dependents at depth 1 and 2 with tests among them, is
a property of click's own structure rather than something a fixture author
arranged. Its suite is also fast enough that probe selection has to earn its
keep. The estimate of ~9.5k source lines was low: the real figure is 12,727.

## 2. Setup, and the baseline

Cloned under `$HOME` because Graph refuses git subprocesses under paths it
considers unsafe, which the build notes already record. A venv with
`pip install -e . pytest`. `entire enable --agent claude-code`, then
`entire configure --skip-push-sessions`, and the origin push URL was set to a
non-existent value so nothing from this experiment could ever reach
`pallets/click`.

Baseline on a clean checkout, three consecutive runs:

```
2059 passed, 24 skipped, 31000 deselected, 1 xfailed in 3.36s   wall 4.02
2059 passed, 24 skipped, 31000 deselected, 1 xfailed in 2.85s   wall 3.45
2059 passed, 24 skipped, 31000 deselected, 1 xfailed in 2.82s   wall 3.42
```

The 31,000 deselected tests are click's `stress` marker, excluded by its own
`addopts`. **A 3 second suite changes what probe selection is worth.** On this
repository a reviewer can simply run everything, and the only thing selection
buys is a named list of which tests to look at first. That is worth saying
plainly before any of the numbers below: the strongest argument for probe
selection does not apply here.

### One environment problem worth recording

`pip install -e .` under flit writes a `click.pth` containing the absolute
path of the original checkout. Umbra runs tests in a **detached worktree**, so
`import click` inside that worktree resolves back to the working tree, not to
the commit under analysis. Both the baseline run and the head run would have
imported the same source, and `verify` would have found no difference no
matter what the commit did.

This was worked around, not fixed: a wrapper on `PATH` that puts `$PWD/src`
at the front of `PYTHONPATH` before invoking pytest, which wins over a `.pth`
entry. It was verified to work by planting mutations and watching them show up.

The general shape is worth naming: **Umbra's worktree isolation is real for
git and imaginary for an interpreted language installed in editable mode.**
Every Python project in the world develops this way. Nothing in the report
tells the reader whether isolation actually held.

## 3. The session

Nested `claude` invocations are blocked in this environment, so the session
could not be produced by a separate headless agent run inside the click repo.
The work was done from this session instead, and the session's own Claude Code
transcript was imported into the click repository with
`entire import claude-code --path`, unmodified. No part of the transcript was
edited or removed. Three checkpoints were created, one per user turn, and
Umbra paired the commit with the last of them, `65ddfb5e3bd3`, by session
window.

**The work.** Click issue #3362: `HelpFormatter.write_usage` breaks hyphenated
option names across a line, because `textwrap.TextWrapper` has
`break_on_hyphens=True` by default and click gives no way to change it. The
issue was chosen from click's open issue list before anything was read, on the
grounds that it was a genuine bug with a clear cause, not because of what it
would do to a graph.

The fix: `wrap_text` gained a `break_on_hyphens: bool = True` parameter passed
through to the `TextWrapper` constructor, and both `wrap_text` calls inside
`HelpFormatter.write_usage` pass `False`. Three regression tests were appended
to `tests/test_formatting.py` and a `CHANGES.md` entry added. The full suite
was run (2,062 passed) and the change committed.

**How the work was done, which turns out to matter more than what it was.**
The bug was reproduced with an inline `python -c`. `src/click/_textwrap.py`
and `src/click/formatting.py` were read in full with the file tool.
`tests/test_formatting.py` was read with an offset, lines 505 to 604. Four
edits to `formatting.py` went through the edit tool. The tests were appended
with a shell heredoc and `CHANGES.md` was patched with an inline Python
script, both through Bash. Everything else, greps, test runs, the commit, was
Bash.

Umbra's own coverage line describes this accurately:

```
coverage  3 file reads, 4 edits, 0 searches, 43 shell commands
```

Three reads and forty-three shell commands. This session is a textbook case of
the limitation the README already documents, and it is not an unusual session.

## 4. The run

```
entire umbra 562e458 --test "click-pytest -v" --out ./out
```

### Field size and timings

| | |
|---|---|
| Snapshot | 158 files parsed, **3,310 symbols, 9,582 relations**, completeness `ok` |
| Relations by kind | CALLS 2,706, DEFINES 3,310, CONTAINS 1,070, IMPORTS 607, USES_TYPE 271, CONSTRUCTS 266, DATA_FLOWS 264, SIMILAR_TO 236, INHERITS 117, PARAM_TYPE 98, OVERRIDES 86, **TESTS 18** |
| `graph snapshot`, cold | **2.92s** |
| `graph snapshot`, warm | 0.10s to 0.14s |
| `graph commit --json` | 0.17s |
| `graph impact --format json` | **3.11s and 3.14s** per call, five calls in this run |
| Whole run, wall clock | **28s** |

The impact calls are the run. Each one reports `index_latency_ms: 2911` and
`query_latency_ms: 1`: the index is rebuilt per invocation and the query
itself is free. Five sources means five rebuilds of the same index, roughly
15.5 of the 28 seconds. Everything else, the two worktrees, the snapshot, the
commit diff, the transcript parse, the probe pair and the full sweep pair,
fits in the remaining 12.5 seconds, and the sweep pair alone is about 7 of
those because the suite runs twice.

### The state breakdown and the coverage line

```
light  ●●●◐◐◐◐◐◐◐◐◐◐◐◐○   3 lit  12 penumbra  1 umbra  0 unknown   19% examined
coverage  3 file reads, 4 edits, 0 searches, 43 shell commands
```

Sixteen nodes. Thirteen in shadow. On a repository of 3,310 symbols, a change
to the help formatter produced a field of sixteen.

The three lit nodes are `HelpFormatter.write_dl`, `write_text` and
`write_usage`, all in `formatting.py`, which was read in full. Correct.

### Probes, cracks and the sweep

```
probes  1 selected  0 cracked
sweep   full suite  0 leaks
verdict NO EFFECT, the target tests behave exactly as before your edit
```

One probe, `tests/test_formatting.py::test_wrap_text_visible_width`, out of
2,059 tests. The sweep's zero leaks is correct **for this commit**: the change
genuinely alters no existing assertion. Section 6 shows what the same sweep
does when the change is not benign.

### Evidence tiers

Umbra's own tiers, on the thirteen shadowed nodes: **1 glance, 11 echo,
1 umbra, 0 unknown**, plus 3 lit.

The eleven echoes are the finding in this section, and section 5 returns to
them.

Relation mix: **4 direct caller, 12 type consumer**. No data flow, no
transitive caller, no co-change, no import. On the fixture almost every node
is a direct caller. Here three quarters of the field arrived over
`PARAM_TYPE`, `USES_TYPE` and `RETURNS_TYPE` edges, which say "this function
takes a `HelpFormatter`", not "this function calls the thing you changed".

**On "how many dependents came back heuristic or needing verification": none,
because this Graph build does not offer that.** `entire graph impact
--format json` in `v0.4.1-nightly.202609030616.ddcebd05` returns endpoints
with `relation`, `direction` and `depth`, and carries no `evidence`,
`confidence` or `verified` field anywhere in the document. There is no tier
for Umbra to report and none is reported. Any expectation of a heuristic
versus verified split in the output was an expectation about a Graph that does
not exist yet.

## 5. Judging the top five as a reviewer

For context, what a reviewer should actually worry about: `write_usage` now
wraps every usage line differently, which reaches every `--help` screen and
every parameter error in every program built on click, and `wrap_text` gained
a keyword parameter that no existing caller passes.

### 1. `CustomFormatter`, `tests/test_custom_classes.py:52`, umbra, 7.755

**Defensible.** It is the only subclass of `HelpFormatter` in the repository
and the session never opened that file, so "who extends the class you changed"
is a question a reviewer should ask and Umbra answered it correctly. But the
subclass overrides `write_heading` and nothing else, and the test around it
asserts on a styled heading. Nothing in this change can reach it. It takes the
top slot on a far-field multiplier of 1.5 awarded for sitting in `tests/`
rather than `src/`, which is a fact about directory layout, not about risk,
and that 1.5 is what puts it above the one node whose behaviour actually
changed. A reviewer loses fifteen seconds here. Right question, wrong answer,
top billing.

### 2. `test_wrap_text_visible_width`, `tests/test_formatting.py:487`, glance, 7.5

**Right, and the best thing in the report.** It is the only test in click that
calls `wrap_text` directly, and `wrap_text` is the function whose signature
and `TextWrapper` construction changed. If the `break_on_hyphens` plumbing had
been wrong, this is the test that would have said so. The `glance` tier is
also exactly right and non-obvious: the file was read, but with a range of
lines 505 to 604, and this test starts at 487, so it was in a file the session
opened and outside the part it saw. Selecting it as the sole probe was the
correct single choice.

### 3. `Command.format_usage`, `src/click/core.py:1163`, echo, 5.33

**Right node, wrong rank. This should have been first.** Its body is
`formatter.write_usage(ctx.command_path, " ".join(pieces))`. It is the only
call site in click of the method that changed, and every usage line click has
ever printed goes through it. It is third for two reasons, and both are
defects rather than judgements:

- It was demoted from umbra to **echo**, a 0.7 multiplier, because the string
  `src/click/core.py` appears in the session's own prose. It appears in a
  table comparing candidate repositories, written **before the repository was
  cloned**, in the sentence "dense intra-package call graph (`core.py` and
  `types.py` and `parser.py` and `decorators.py` and `termui.py`)". A
  filename typed while choosing between three projects downgraded eleven
  unexamined symbols in the winner. The packet states the reason plainly, "the
  session named `src/click/core.py` in its own words and never opened it",
  which is honest and still the wrong conclusion.
- It is labelled a **type consumer** rather than a direct caller, weight 2
  instead of 3, because Umbra never asked Graph about the method that changed.
  Section 6 has the mechanism. Asked directly, Graph answers
  `Command.format_usage CALLS depth 1`.

### 4. `Command.format_options`, `src/click/core.py:1300`, echo, 5.019

**Wrong.** It calls `formatter.write_dl(opts)` and never touches
`write_usage`. `write_dl` calls `wrap_text` with the default
`break_on_hyphens=True`, which this change deliberately left alone. There is
no input for which its output differs before and after the commit. It is in
the field for one reason only: its signature is
`(self, ctx: Context, formatter: HelpFormatter)`, a `PARAM_TYPE` edge from the
changed class. Ranking a node that cannot be affected above the public API
that renders the changed line is the ranker asserting something false.

### 5. `Command.format_arguments`, `src/click/core.py:1312`, echo, 4.651

**Wrong, for the same reason.** Also `write_dl`. And it shares its score of
4.651 with `format_commands`, `format_epilog`, `format_help`,
`format_help_text`, `make_formatter` and `Command.get_usage`, so the
ordering inside that block is a tie broken by path and name. `get_usage` is
the public entry point to the behaviour that changed and it sits at the
bottom of that tie; `Command.get_help` is below all of them at 4.2. **Within
`core.py` the ranking carries no information about which nodes the change can
reach.** It is ordering by dependent count, which measures how important a
symbol is in general, not how exposed it is to this diff.

### What the top five missed

`UsageError.show` at `src/click/exceptions.py:106` calls `self.ctx.get_usage()`
and prints the usage line on every parameter error in every click program. It
is a real behavioural dependent of this change and **it is not in the field at
all**, at any rank. `--depth 3` was tried and produced a byte-identical field:
the limit is not depth, it is that `self.ctx.get_usage()` is a method call on
an attribute and Graph resolves no edge for it. The README's "graph edges are
incomplete" is a one-line limitation on the fixture and a structural one on an
object-oriented codebase.

Also absent: all 46 assertions on `Usage:` across six test files. Not one of
them entered the field.

## 6. What broke

Five product failures and three environment or setup notes. None of them
crashed anything: `entire umbra` returned exit 0 on all five runs and never
panicked.

### F1. A changed method never binds to a symbol, so it is never traversed

`graph commit --json` named the changed method `HelpFormatter.write_usage`.
Umbra's `graph.Bind` looks it up with `f.Symbols[id].Name == name`
(`internal/graph/sources.go:287`). The snapshot record is:

```json
{"name": "write_usage", "qualified_name": "HelpFormatter.write_usage",
 "kind": "method", "file_path": "src/click/formatting.py", "start_line": 167}
```

`Name` is bare, the commit diff supplies the qualified form, the comparison
fails, `Source.Symbol` stays empty, and `analyze.go:117` skips the impact call
for any source with an empty symbol. The report still lists
`HelpFormatter.write_usage  body changed  src/click/formatting.py:167` with
`dependents 8`, which reads as though those eight were followed. They were
not. The word `qualified_name` appears nowhere in Umbra's Go source outside
test fixture data.

The eight were available for the asking:

```
entire graph impact --symbol write_usage --file src/click/formatting.py --line 167
  callers total 3
    Command.format_usage   src/click/core.py  CALLS  depth 1
    Command.get_usage      src/click/core.py  CALLS  depth 2
    Command.format_help    src/click/core.py  CALLS  depth 2
```

Real `CALLS` edges, correctly ranked by any reading. Instead the field was
built from the enclosing **class**, whose impact answer is
`callers: total 0` and eighteen type consumers.

**This affects every changed method in every object-oriented codebase.** The
fixture never showed it because `compute_total`, `handle_order`, `quote` and
`summarize` are all module-level functions, whose bare and qualified names are
identical.

### F2. The sweep reports zero leaks once more than sixteen tests fail

This is the serious one.

A mutation was planted in `HelpFormatter.write_usage` that upper-cases the
usage prefix. It breaks **55 tests** by pytest's count and **51** by
`graph verify`'s. Umbra's report:

```
probes  no test reaches the shadow
sweep   full suite  0 leaks
```

A clean, confident all-clear on a change that breaks fifty-five tests.

The cause was pinned rather than guessed. `graph verify` prints both a
`NEWLY FAILING (n): ...` line and a `VERDICT: REGRESSION in n tests: ...`
line when `n` is small, and **only the `VERDICT:` line when `n` is large**.
Measured on this repository with the same runner and worktrees, varying only
the number of failing ids:

| Newly failing | `NEWLY FAILING` line present |
|---|---|
| 3, 5, 8, 12, 16 | yes |
| 17, 18, 19, 47, 51 | **no** |

Umbra's `ParseVerdict` reads ids from `newlyFailingRE` and `newlyPassingRE`
only (`internal/graph/verify.go:38`). `execute.go:99` builds the changed set
from `v.NewlyFailing` and `v.NewlyPassing`. With neither line present both are
empty, so the leak list is empty and the count printed is zero. The `VERDICT:`
line, which carries the same ids, is captured as a display string and never
parsed. Nothing anywhere says the sweep could not read its own output.

**A known-positive confirms the mechanism.** A second mutation in
`write_heading` breaks 15 tests, one under the threshold. Same repository,
same runner, same code path:

```
probes  no test reaches the shadow
sweep   full suite  15 leaks
        tests/test_formatting.py::test_basic_functionality   test file not in the co-change set
        tests/test_custom_classes.py::test_context_formatter_class   reaches only through structure
        ... 13 more, each named with a reason
```

Fifteen leaks, all named, all with forensics. The machinery works perfectly.
It stops working, without saying so, at seventeen. The fixture's worst case is
five cracked probes, so this could not have appeared there.

This is the failure mode the README's limitations section names as the
project's own recurring sin, "a check that has never been observed to fail is
not yet a check", arriving in the check that audits all the others.

### F3. A module-granularity diff is reported as "no dependents"

A different one-character mutation, adding a trailing space inside an
f-string in `write_usage`, made `graph commit --json` answer at module
granularity:

```json
{"type": "body_changed", "kind": "module",
 "name": "src/click/formatting.py", "dependents_count": 0}
```

Umbra's report for that commit:

```
src/click/formatting.py  body changed  src/click/formatting.py:1
light     0 lit  0 penumbra  0 umbra  0 unknown   examined set unavailable
 no dependents were found for the changed symbols
probes  no test reaches the shadow
sweep   full suite  0 leaks
```

That commit breaks 51 tests. "No dependents were found for the changed
symbols" is a sentence about Graph's answer being unusable, printed in the
voice of a finding about the code.

The degradation is narrow and reproducible. Three control commits were made
against the same base: a one-line change in a top-level function body resolved
to `function wrap_text` with 6 dependents; a one-character change inside
another method resolved to the class and the method; a different line inside
`write_usage` itself resolved to `method HelpFormatter.write_usage` with 8
dependents. Only the trailing-whitespace-inside-a-string-literal case
degrades. Whatever the cause inside Graph, Umbra has no notion that a `module`
kind is a degraded answer and should be reported as one.

### F4. Selection reached no test on a real change

On both planted mutations, Umbra printed `probes  no test reaches the shadow`
and selected nothing. On the real commit it selected one test out of 2,059,
and that one arrived through `wrap_text`, a module-level function, not through
the method that changed.

The `TESTS` relation count for the whole repository is **18**, against 2,059
collected tests. Graph barely knows which tests exercise what in click, and
the paths that would substitute for that knowledge, `test -> CliRunner.invoke
-> Command.main -> ... -> format_usage -> write_usage`, run through dynamic
dispatch that produces no edges.

**Probe selection did not work on this repository.** It is worth being precise
about the consequence: on a 3 second suite it does not matter, because you run
everything. It would matter a great deal on a suite where you cannot.

### F5. A filename in prose demotes unexamined code

Covered in section 5. Eleven nodes moved from umbra to echo, a 0.7 multiplier,
because a filename was typed in a sentence written before the repository was
cloned. The report is honest about why, and the effect is still wrong: naming
a file while choosing a project is not attention paid to the code in it.

An agent in an ordinary session that says "I will need to check `core.py`" and
then does not check it produces exactly the same demotion, so this is not an
artifact of the import workaround.

### F6. Worktree isolation does not survive an editable install

Section 2. Worked around with a `PYTHONPATH` wrapper; the report says nothing
about whether isolation held.

### F7. `--depth 3` changed nothing

Identical field, identical thirteen shadowed nodes, identical scores. Umbra's
own BFS does use the flag, so the finding is that there were no further edges
to follow, not that the flag is ignored. A reviewer who reaches for `--depth 3`
because the field looks thin gets no signal that depth was not the problem.

### F8. `verify` and pytest disagree on the baseline count

`BASELINE RECORDED: ... (pytest; 1668 passing, 0 failing, exit 0)` against
pytest's own `2059 passed`. Consistent across runs. Not investigated, and it
does not affect the delta, but a baseline that undercounts by 391 tests is
worth knowing before anyone trusts the number.

### F9. Cross-repository path collision, an artifact of the setup

The imported transcript covers a session that also worked in the Umbra
repository. Umbra filters timeline paths against the target repository's file
list, and `README.md` exists in both, so `wc -l README.md` run against Umbra
marked click's `README.md` as `quoted`. This is a consequence of importing a
transcript from another repository, which is not normal usage. Recorded so it
is not mistaken for a real result.

## 7. What surprised me

**The shell blind spot went from a footnote to the dominant effect.** It is
already in the README, and I still did not expect 3 reads against 43 shell
commands, or 19 percent illumination on a change I understood completely.
Appending tests with a heredoc and patching a changelog with an inline Python
script are not unusual things for an agent to do; they are the obvious things
to do. The stated limitation is not an edge case, it is the common case.

**The safety net fails precisely when the selection is worst.** The sweep is
sold as the reason a thin selection is acceptable. A thin selection and a
large blast radius are the same event, and a large blast radius is exactly
what silences the sweep. The two failures are correlated in the worst possible
direction, and nothing in the design anticipated that because the fixture's
blast radius is five tests.

**Graph was better than Umbra's use of it.** Asked directly about
`write_usage`, Graph returned three correct `CALLS` callers, correctly ranked,
in three seconds. Umbra never asked. The best evidence in the system was one
string comparison away and the report instead described a `PARAM_TYPE` edge as
though it were the finding.

**The ranking was more defensible than I expected and its reasons were worse
than I expected.** Two of five defensible or right, and none of the five got
there by a chain of reasoning I would endorse. The top node won on a
directory-layout multiplier. The third node lost on a filename typed before
the repository existed. When a heuristic reaches a reasonable answer through
factors that are individually wrong, more test repositories will not tell you
that, only opening the source will.

**Nothing crashed.** Five runs, exit 0 every time, no panic, no hang, and
every degradation printed a note. The failure mode of this product is not
falling over. It is answering.

## 8. What this says about generalising

The parts that generalised: resolving a commit to a checkpoint, worktrees, the
transcript adapter, the classifier, the tier assignment, the scrubber, the
report renderers, the packet, exit codes, and the honesty of the notes. All of
that ran on an unfamiliar 28,000-line repository first time, and the packet's
prose explanations of why a node has the state it has were accurate every time
I checked one against the transcript. Performance is fine: 28 seconds, of
which 15 is a Graph index rebuilt five times, which is a caching problem and
not an architectural one.

The parts that did not generalise are the parts that touch the shape of real
code:

1. Changed methods do not bind, so a class-based codebase loses the strongest
   edges in the graph. **Fixture-invisible: everything in `fixtures/app` is a
   module-level function.**
2. The sweep goes blind above sixteen failures. **Fixture-invisible: the
   fixture's worst case is five.**
3. A module-granularity diff is reported as an absence of dependents rather
   than an absence of information.
4. Tests are two or more unresolvable hops from most production symbols, so
   probe selection selects almost nothing.
5. Ranking within a group of type consumers carries no information about
   whether the change can reach them.

Three of those five are not degradations under a real repository, they are
defects that a fixture built by the same author cannot express. That is the
honest summary of this experiment: **the fixture validated the pipeline and
could not validate the analysis.**

Whether the product generalises is not yet answered by this run, because F1
and F2 have to be dealt with before a second run means anything. F1 means the
central case, an agent changing a method, has never actually been exercised
end to end. F2 means the audit that would have told us so was reporting zero.
What can be said is narrower and still worth something: the machinery
generalises, the numbers on this run were mostly measuring the wrong edges,
and the ranking, given the wrong edges, was still right about one node in
five and defensible about a second.

## 9. Reproducing this

```sh
git clone https://github.com/pallets/click ~/umbra-experiment/click
cd ~/umbra-experiment/click && git checkout 6aabf09
python3 -m venv .venv && .venv/bin/pip install -e . pytest
entire enable --agent claude-code && entire configure --skip-push-sessions
# make a change, commit it, then:
entire umbra <commit> --test "<runner> -v" --out ./out
```

The runner must be on `PATH` and must import the source from the worktree it
is standing in, not from an editable install. The wrapper used here:

```sh
#!/bin/sh
PYTHONPATH="$PWD/src${PYTHONPATH:+:$PYTHONPATH}"
export PYTHONPATH
exec /path/to/.venv/bin/python -m pytest "$@"
```

To reproduce F2 without any of the rest: record a baseline with
`entire graph verify --record-baseline`, then break seventeen or more tests
and adjudicate. `graph verify` reports the regression on its `VERDICT:` line
and prints no `NEWLY FAILING` line, and `entire umbra` reports zero leaks.

Tooling: Entire CLI 0.10.5, entire-graph
`v0.4.1-nightly.202609030616.ddcebd05`, Umbra at `7661eb2`, Python 3.14.4,
pytest 9.1.1, Linux.
