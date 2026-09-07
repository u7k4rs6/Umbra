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

---

# Second pass, after the two fixes

Everything above is the first pass and is left exactly as written. This section
is a re-run of the same experiment on the same input after `44cecd0` (the
sweep's parser) and `e18fd58` (binding a changed method). It is measurement,
not fixing: no ranking weight was touched, nothing was tuned, and the defects
found here are recorded rather than repaired.

**The input is byte-identical.** Same clone of `pallets/click`, same commit
`562e458`, same three imported checkpoints, same pairing with `65ddfb5e3bd3`,
same runner. The snapshot confirms it: 158 files, 3,310 symbols, 9,582
relations, and the same count for every relation kind. Any difference below is
attributable to the two fixes.

## The two number sets, side by side

| | First pass | Second pass |
|---|---|---|
| Snapshot | 158 files, 3,310 symbols, 9,582 relations | **identical, every relation count too** |
| `graph snapshot`, cold | 2.92s | 3.17s |
| `graph snapshot`, warm | 0.10s to 0.14s | 0.10s to 0.13s |
| `graph commit --json` | 0.17s | 0.14s |
| `graph impact`, per call | 3.11s, 3.14s | 3.00s, 3.03s, 3.08s |
| `graph impact`, calls made | **5** | **6** |
| Whole run, wall clock | **28s** | **29s** |
| Field size | 16 nodes | **15 nodes** |
| Nodes in shadow | 13 | 13 |
| Lit | 3 | **2** |
| Penumbra | 12 | 12 |
| Umbra | 1 | 1 |
| Unknown | 0 | 0 |
| Examined | 19% | **13%** |
| Coverage line | 3 reads, 4 edits, 0 searches, 43 shell commands | identical |
| Evidence tiers | 1 glance, 11 echo, 1 umbra, 3 lit | 1 glance, 11 echo, 1 umbra, **2 lit** |
| Relation mix | 4 direct caller, 12 type consumer | **4 direct caller, 10 type consumer, 1 transitive caller** |
| Probes selected | 1 | 1 |
| Probes cracked | 0 | 0 |
| Sweep | full suite, 0 leaks | full suite, 0 leaks |
| Verdict | NO EFFECT | NO EFFECT |

Three of those differences are the fix and one is noise.

**The sixth impact call is the fix, visible in the command list.** The first
pass queried `wrap_text`, `HelpFormatter` and the three added tests. The second
pass queries those five and `HelpFormatter.write_usage`, which is the method
that actually changed and which the first pass never asked about.

**The field lost a node and the lost node is `HelpFormatter.write_usage`
itself.** In the first pass it did not bind, so it was not recognised as a
changed symbol and instead appeared as a *dependent* of `wrap_text`, lit,
because `formatting.py` was read in full. It is now a source, and a source is
not its own dependent. That is why lit fell from 3 to 2 and examined from 19%
to 13%: the numerator and the denominator both lost the same node. **Nothing
was examined less.** The illumination fraction went down because the field got
more correct, which is worth knowing about that metric.

**Two relation labels changed and both are truer.** `Command.format_usage`
moved from `type consumer` to `direct caller`, and `Command.get_usage` from
`type consumer` to `transitive caller`. Both now sit on the `CALLS` edges Graph
always had; before, they arrived only over `PARAM_TYPE` and `USES_TYPE` from
the enclosing class.

**Wall clock is a wash.** One more three-second impact call, one second more
total. The impact calls are still the run: six index rebuilds of the same index
at about 3s each is roughly 18 of the 29 seconds, and each still reports
`query_latency_ms: 1`.

## The docket, before and after

| Rank | First pass | Second pass |
|---|---|---|
| 1 | `CustomFormatter` 7.755, type consumer, umbra | **`Command.format_usage` 7.995, direct caller, echo** |
| 2 | `test_wrap_text_visible_width` 7.5 | `CustomFormatter` 7.755, type consumer, umbra |
| 3 | `Command.format_usage` 5.33, type consumer | `test_wrap_text_visible_width` 7.5 |
| 4 | `Command.format_options` 5.019 | `Command.format_options` 5.019 |
| 5 | `Command.format_arguments` 4.651 | `Command.format_arguments` 4.651 |
| 6 to 11 | `format_commands`, `format_epilog`, `format_help`, `format_help_text`, **`get_usage` 4.651**, `make_formatter` | `format_commands`, `format_epilog`, `format_help`, `format_help_text`, `make_formatter`, `get_help` |
| 12 | `get_help` 4.2 | `Group.format_options` 3.619 |
| 13 | `Group.format_options` 3.619 | **`Command.get_usage` 3.488, transitive caller** |

Exactly two nodes in this field can actually reach the changed method.
`format_usage` went from tenth-of-thirteen scoring band to first. **`get_usage`
went from tenth to last.** Its edge became more accurate and its rank got
worse, because a transitive caller is weighted 1.5 and a type consumer 2. A
truer description of the same relationship demoted it below six nodes the
change cannot touch. That is not a tuning complaint; it is a measurement, and
it says the weight table and the relation taxonomy disagree with each other.

## Step 3: the two questions the bind defect was hiding

### Does `UsageError.show` appear? No, and depth is not the reason.

`src/click/exceptions.py:106` reads `echo(f"{self.ctx.get_usage()}\n{hint}", ...)`,
so every parameter error in every click program prints a usage line built by
the method that changed. It is absent from the field.

The first pass guessed this was the `--depth 2` limit. **It is not.** The
snapshot carries five outgoing edges from `UsageError.show` and not one of them
reaches `get_usage`:

```
UsageError.show  CALLS     -> gettext.gettext                (external)
UsageError.show  CALLS     -> echo                           src/click/utils.py
UsageError.show  CALLS     -> get_text_stderr                src/click/_compat.py
UsageError.show  CALLS     -> ClickException.format_message   src/click/exceptions.py
UsageError.show  OVERRIDES -> ClickException.show             src/click/exceptions.py
```

Graph resolved the module-level functions and the `self.` method call on the
same line, and produced nothing for `self.ctx.get_usage()`. The pattern is
specific and nameable: **a method call on an attribute whose type is not
annotated produces no edge**, while a call on `self` and a call on an imported
function both do.

Replaying Umbra's own traversal outside the tool confirms it is not depth. From
these three sources the closure is 17 dependents at depth 2, 19 at depth 3, and
**19 at depth 4 and at depth 6**. It saturates and never contains
`UsageError.show`. Raising `--depth` cannot reach a node with no path to it.

### Do the 46 `Usage:` assertions appear? No, and none is selected.

Forty-six assertions across six test files, and not one of their test functions
enters the field at any depth. Only five symbols under `tests/` appear anywhere
in the closure at any depth: `test_wrap_text_visible_width`,
`test_wrap_text_break_on_hyphens` (itself a source, being one of the added
tests), `CustomFormatter`, and at depth 3 the two `OptParseCommand` overrides
in `test_commands.py`, which are class methods rather than tests. None of the
46 is among them.

The reason is visible in their edges. A representative one:

```
test_basic_functionality (tests/test_formatting.py)
    CONTAINS -> cli            tests/test_formatting.py
    CALLS    -> click.command  (external)
```

Its whole graph presence is its own nested `cli` function and a call to an
**external** node. The call that actually runs the code under test,
`runner.invoke(cli, [...])`, produces no edge at all: `runner` is a pytest
fixture parameter with no inferable type, which is the same shape of miss as
`self.ctx` above.

Measured across the whole suite, edges leaving symbols in `tests/`:

| Destination | CALLS edges |
|---|---|
| external `click.*` | **1,522** |
| in-repo `src/` | 326 |
| within `tests/` | 74 |

**Seventy-nine percent of the calls click's test suite makes are attributed to
external nodes rather than to the definitions in the same repository.** The
cause is the import style against a `src/` layout: inside the package,
`from .utils import echo` resolves to `src/click/utils.py`, which is why
`UsageError.show` has an in-repo `echo` edge. From the tests, `import click`
then `click.command(...)` resolves to an external `click.command`, and 495
external nodes in this snapshot are exactly that. The tests are not far from
the source in the graph; for four calls in five they are attached to a
different node entirely.

The `TESTS` relation does not rescue it either. There are 18 in the whole
repository, all name matches of the `test_echo -> echo` shape, and Graph's own
`capabilities` lists `TESTS` under `heuristic_relation_types`. None touches the
formatter.

So probe selection selects one test out of 2,059, and that one arrives through
`wrap_text`, a module-level function, over the only kind of edge that survives
in this codebase.

## Step 4: the top five, re-judged against source

Judged fresh against the current edges, not carried over.

### 1. `Command.format_usage`, `src/click/core.py:1163`, direct caller, echo, 7.995. **Right.**

Its body is two lines and the second is
`formatter.write_usage(ctx.command_path, " ".join(pieces))`. It is the only
call site in click of the method that changed, and every usage line the library
prints goes through it. It is first, it is labelled by the `CALLS` edge, and no
weight was touched to put it there.

One thing to record. It leads by 0.24, about three percent, and it does that
while carrying a 0.7 echo multiplier. Without that multiplier it would score
11.42 and lead by nearly half. It is first on the correct answer by a margin
thin enough that a node with one more dependent would have taken it.

### 2. `CustomFormatter`, `tests/test_custom_classes.py:52`, type consumer, umbra, 7.755. **Defensible, and now visibly overweighted.**

Unchanged in score between the passes; it is second only because something
correct overtook it. It is the repository's only `HelpFormatter` subclass and
was never opened, so surfacing it answers a question a reviewer should ask. But
it overrides `write_heading` alone and its test asserts on a styled heading, so
this change cannot reach it, and it still outranks nine `core.py` methods on a
far-field multiplier of 1.5 awarded for sitting in `tests/` rather than `src/`,
which is a fact about directory layout rather than about risk.

There is also a labelling error underneath it. The snapshot edge is `EXTENDS`.
`internal/graph/relations.go` puts `EXTENDS`, `INHERITS`, `IMPLEMENTS` and
`OVERRIDES` in the type-use family, so a subclass is reported as a "type
consumer" and weighted 2, the same as a function that merely takes a
`HelpFormatter` parameter. Inheriting a class and accepting one as an argument
are not the same dependency, and the report cannot currently tell a reader
which it has.

### 3. `test_wrap_text_visible_width`, `tests/test_formatting.py:487`, direct caller, glance, 7.5, passed. **Right.**

The only test in click that calls `wrap_text` directly, and `wrap_text` is the
function whose signature and `TextWrapper` construction changed. The `glance`
tier is right and non-obvious: the file was read at lines 505 to 604 and this
test begins at 487, so it is inside a file the session opened and outside the
part it saw. Selected as the sole probe, ran, passed. Unchanged and still the
soundest row in the table.

### 4. `Command.format_options`, `src/click/core.py:1300`, type consumer, echo, 5.019. **Wrong.**

It calls `formatter.write_dl(opts)` and never touches `write_usage`. `write_dl`
calls `wrap_text` with the default `break_on_hyphens=True`, which this change
deliberately left alone, so there is no input for which its output differs
before and after the commit. It is in the field over a `PARAM_TYPE` edge from
its `formatter: HelpFormatter` parameter, and nothing more.

The bind fix could not help it, and the second pass makes the reason plain.
`graph commit` reports **both** `class HelpFormatter` and
`method HelpFormatter.write_usage` as changed, because the class body contains
the method. Umbra treats the class as an independent source, and at depth 1
every method in `core.py` that takes a `HelpFormatter` parameter becomes its
dependent. **Ten of the thirteen shadowed nodes are manufactured by that
roll-up**, and none of them can be affected by the edit that actually happened.

### 5. `Command.format_arguments`, `src/click/core.py:1312`, type consumer, echo, 4.651. **Wrong, same cause.**

Also `write_dl`. It is tied at 4.651 with `format_commands`, `format_epilog`,
`format_help`, `format_help_text` and `make_formatter`, none of which the
change can reach, and that whole tie sits above `Command.get_usage` at 3.488,
which renders the usage line the change alters. The ordering inside the block
is broken by path and name, so within the `core.py` group the ranking still
carries no information about which nodes the change can reach.

**Scorecard: two right, one defensible, two wrong**, against one right, one
defensible and three wrong in the first pass. The improvement is real and it is
entirely at the top: the node that matters most is now first and correctly
described.

## Step 5: the echo demotion, measured

**It still fires, on eleven of the thirteen shadowed nodes, and it fires on a
node carrying a direct-caller edge.**

The evidence is the same event as in the first pass, and the report still
states it plainly: `the session named src/click/core.py in its own words and
never opened it`. It is timeline sequence 15, a mention of six paths in a
sentence comparing three candidate repositories, written before this repository
was cloned. `src/click/core.py` was never read and never edited: the timeline
carries no read or edit event for it at all. The cut is at sequence 54, so the
mention precedes the change by thirty-nine events.

What it costs, exactly, from the report's own factors:

| Node | Relation | Factors | Score with echo | Score without |
|---|---|---|---|---|
| `Command.format_usage` | **direct caller** | relation 3, dependents 3.807, state 0.7 | **7.995** | **11.42** |
| `Command.get_usage` | transitive caller | relation 1.5, dependents 3.322, state 0.7 | 3.488 | 4.98 |
| nine `core.py` methods | type consumer | relation 2, state 0.7 | 3.619 to 5.019 | 5.17 to 7.17 |

So the answer to the question as posed: **yes, a node with a direct structural
edge is being demoted by a prose mention alone**, and the mention in question
was made before the repository existed on this machine. `Command.format_usage`
survives it and is still first, but by 0.24 rather than by 3.67.

No weight was changed, and no recommendation is made here. The data is: the
demotion is real, it applies uniformly to every node in a mentioned file
regardless of the edge that put the node in the field, and in this run it very
nearly cost the correct answer its place.

## The corrected conclusion

The first pass concluded: **"the fixture validated the pipeline and could not
validate the analysis"**, and summarised it as the machinery generalising while
the analysis did not.

**That sentence was wrong in the way it apportioned blame, and it named the
wrong half.** Corrected:

- **The machinery generalises.** Unchanged, and the second pass confirms it:
  resolution, worktrees, the transcript adapter, the classifier, tiers, the
  scrubber, the renderers, exit codes and the notes all ran clean on an
  unfamiliar 28,000-line repository, twice, with byte-identical input producing
  a byte-identical snapshot.
- **The ranking generalises better than the first pass could measure.** Most of
  what looked like bad ranking was one binding defect. Fixing it moved the one
  node that matters from third to first, with a true `CALLS` label, without
  touching a single weight. The first pass judged the ranking model on a field
  built from the wrong edges and blamed the model for it.
- **The field construction does not generalise, and that is the real finding.**
  Which symbols enter the blast radius is still wrong on real code, for three
  reasons this pass measured rather than guessed: a class-level source rolls up
  a method change and manufactures ten dependents that cannot be affected; the
  most important behavioural dependent is absent at any depth because the edge
  does not exist; and 79 percent of the calls the test suite makes are
  attributed to external nodes, so no test reaches the change and probe
  selection selects one of 2,059.

So it is neither "the analysis did not generalise" nor "the binding did not".
It is: **the binding defect was hiding how good the ranking already was, and
fixing it exposed that the field it ranks is the part that does not
generalise.** Ranking a set is only as good as the set, and on real code the
set is currently built from a class roll-up plus whatever edges a
tree-sitter-based resolver can see through attribute access, which in an
idiomatic Python project with a `src/` layout and a fixture-driven test suite
is not much.

The two fixes were worth making and neither is sufficient. The next question is
not a weight; it is whether a changed class should be a source at all when the
change is entirely inside one of its methods.

## Second pass, reproducing

```sh
cd ~/umbra-experiment/click            # the pass-one clone, untouched
entire umbra 562e458 --test "click-pytest -v" --out ./out2
```

Umbra at `e18fd58`. Everything else as in the first-pass reproduce section
above.
