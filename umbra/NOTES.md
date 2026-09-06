# Umbra: build notes

Working notes kept during the build. Written after phase 5, and extended as
later phases land. The design documents in `docs/` stay as written; anything
this file records that contradicts them is a finding from real output, and the
answered questions at the bottom of `docs/ARCHITECTURE.md` carry the detail.

## Where the build stands

All thirteen phases are done and committed. `go test ./...` is green at every
commit, and a fresh clone was checked to build and test green after phase 11
found that it would not have.

The repository stays **private**. That decision was taken after the phase 12
scrub pass, and it is what settles the one blocker that pass found: see
"Before this repository is ever made public" at the end of this file.

| Phase | What landed | Tests |
|---|---|---|
| 0 | Repository, docs, README pointer | 0 |
| 1 | Step 0 probe, eight questions answered, degradations written | 0 |
| 2 | Module, Runner, resolve, worktrees, command surface | 40 |
| 3 | Transcript adapter, mentions, timeline, session-said fallback | 67 |
| 4 | Snapshot loader, relation map, BFS, impact parser, sources | 89 |
| 5 | Classifier, call-site window, six-factor ranker, terminal table | 158 |
| 6 | Test selection, verify with baseline, the sweep and leak forensics | 187 |
| 7 | JSON, layout, packet, umbra record, the eight scenarios | 227 |
| 8 | HTML map, docket, detail panel | 240 |
| 9 | Attention replay, keyboard, spotlight, day scheme, print | 241 |
| 10 | Landing page, drop-in viewer, README | 243 |
| 11 | Semantic-diff review, the self report, scrubber fixes | 249 |
| 12 | Coverage line, runner check, minimal record, scrub pass | 274 |
| 13 | Exit-code tests end to end, the real four-hop leak chain | 286 |

Nothing in the code pretends to do what it does not: the table prints
`probes not run` when tests were skipped, `sweep skipped` when the audit was
turned off, and says the verdict is suite level when the runner gave no
per-test names.

## Setup that had to be built rather than assumed

The kickoff said to assume a fork of `entireio/entire-graph` with Entire
enabled, the graph plugin installed, and `umbra/` holding HANDOFF.md,
KICKOFF_PROMPT.md and docs/. None of that was present: the working directory
held only the four docs in a folder named `files (1)`, was not a git
repository, and had no Entire CLI on the machine. HANDOFF.md and
KICKOFF_PROMPT.md were never supplied, and the files of those names in
`~/Downloads` belong to a different project, Impeach, so they were not used.

On instruction, the repository was created fresh as `u7k4rs6/Umbra` rather
than forked, the docs were moved to `umbra/docs/`, and the kickoff message
itself stands in for KICKOFF_PROMPT.md. The Entire CLI 0.10.5 and entire-graph
v0.4.1-nightly.202609030616.ddcebd05 were installed from their GitHub releases
into `~/.local/bin`. `entire enable --agent claude-code` works with no login;
checkpoints are local git refs, so the whole build is offline.

Because Umbra lives in its own repository rather than inside the fork, the
`umbra/` directory layout from ARCHITECTURE.md is preserved exactly, so the
module can be grafted into a fork later without moving a file.

## Probe findings, in short

The full answers are at the bottom of `docs/ARCHITECTURE.md`. The short version:

Confirmed as designed:

- `Read` inputs carry `offset` and `limit` for a partial read and neither for a
  full one, so the glance tier is real.
- `graph impact` reports exact call-site lines in both text and JSON, so fault
  line and beacon are live and the ranker keeps all six factors.
- `graph commit --json` distinguishes `signature_changed` from `body_changed`
  for Python and carries dependent counts, so the source weights stand.
- `graph verify` names newly failing tests against a recorded baseline.
- `ENTIRE_PLUGIN_DATA_DIR` is set for an unmanaged plugin, and kubectl-style
  dispatch passes arguments through unchanged.
- Every entry in the installed Graph's `features_requiring_network_access` is
  false. A unit test asserts this rather than trusting the claim.

## Degradations in effect

| Finding | Effect |
|---|---|
| The snapshot marks no test symbols | `IsTest` comes from file and name conventions only |
| `graph checkpoint` has no commit for imported checkpoints | Sources come from `graph commit`, and the header names the fallback |
| Imported checkpoints carry no stored summary | The session-said line falls back to the agent's own last sentence |
| The probe harness has no `Grep` or `Glob` tool | Search runs through Bash, so a file seen only in search output reaches `quoted` rather than `glimpse`; both are penumbra, so no state is lost |
| `pytest -q` is not parseable by verify | The documented runner becomes `pytest -v`; an exit-code-only baseline degrades to a suite-level verdict and the header says so |
| verify exits 0 even on a regression | The verdict is parsed from text, never from the exit status |

## Where the documents turned out to be wrong

Three corrections, all forced by real output rather than preference.

1. **The checkpoint trailer is not hexadecimal.** ARCHITECTURE.md section 1
   describes `Entire-Checkpoint: <12 hex>`. The installed CLI writes a 26
   character ULID, for example `01M1PVD1WBK1J1J8GWDNJJR7XZ`. A hex-only pattern
   matched nothing and every commit fell through to the session-window
   fallback while reporting, wrongly, that the commit carried no trailer. The
   pattern now accepts alphanumeric ids of 6 to 40 characters, which covers the
   ULID, the 12 hex id of imported history and the 40 hex id of a carry-forward
   entry, and preserves case so `git log --grep` can match the exact text.

2. **`pytest -q` cannot be verified.** PRD.md uses `pytest -q` in the command
   surface and in the 90-second demo. Quiet mode prints no per-test ids, so
   `graph verify` answers `output format not recognised` and records zero
   results. `pytest -v` parses cleanly. The default is now `pytest -v`, and the
   table says `add -v` when a baseline comes back exit-code only.

3. **Flags after the reference were silently dropped.** The surface in PRD.md
   is `entire umbra <ref> [flags]`, but Go's flag package stops parsing at the
   first non-flag argument, so `entire umbra HEAD --run none` ignored
   `--run none` and failed asking for a test runner. Argv is now reordered
   before parsing, with boolean flags handled so they do not swallow the
   reference.

Two smaller ones:

- Entire has more than one wording for an absent summary. Imported checkpoints
  say `*No summary...*` and hook-written ones say `*Not generated yet...*`.
  Any fully italicised block is now treated as absent.
- `graph impact --format json` is better than the text shape the design
  sketched, because it carries `call_site.line` and `additional_sites`
  directly. It is now the primary path; the text parser is the fallback and is
  still tested.

## The table at the end of phase 5

`entire umbra HEAD --run none --no-audit`, on the fixture app, with the
exposure taken from this build session's own checkpoint:

```
Umbra  01M1PVD1WBK1J1J8GWDNJJR7XZ  60ea65f  claude-code  depth 2
session said  "Making that demo change now, touching only the two files the session has actually opened:"
              from the last thing the agent said, because Entire stored no summary
handle_order  body changed  umbra/fixtures/app/app/api.py:14
quote  body changed  umbra/fixtures/app/app/api.py:26
compute_total  signature changed  umbra/fixtures/app/app/service.py:20
summarize  body changed  umbra/fixtures/app/app/service.py:35
light  ●●◐◐◐◐◐◐◐○○   2 lit  7 penumbra  2 umbra  0 unknown   18% examined

!◐  apply_refund             umbra/fixtures/app/app/refunds.py:17     direct caller      depth 1  echo fault line beacon       10.5  
    # SAFETY: a pricing failure must not refund the full amount paid.
 ◐  test_handle_order        umbra/fixtures/app/tests/test_api.py:11  direct caller      depth 1  echo test                     7.2  
 ◐  test_quote_single_line   umbra/fixtures/app/tests/test_api.py:19  direct caller      depth 1  echo test                     7.2  
 ○  test_apply_refund_normal ...fixtures/app/tests/test_refunds.py:18 transitive caller  depth 2  test                          6.9  
 ○  ...y_refund_caps_at_paid ...fixtures/app/tests/test_refunds.py:13 transitive caller  depth 2  test                          6.0  
 ◐  test_empty_is_zero       ...fixtures/app/tests/test_service.py:19 direct caller      depth 1  glance test                   6.0  
 ◐  test_negative            ...fixtures/app/tests/test_service.py:13 direct caller      depth 1  glance test                   6.0  
 ◐  test_rounding            .../fixtures/app/tests/test_service.py:7 direct caller      depth 1  glance test                   6.0  
 ◐  test_round_money_places  ...fixtures/app/tests/test_service.py:23 data flow          depth 2  glance test                   5.0  

 2 lit node(s) not shown; pass --all to list them

probes  not run (--run none)
sweep   skipped (--no-audit), so the selection is unaudited
```

That output shows what the product is for. `apply_refund` is pinned to the top
by the beacon override: its call site sits under a `# SAFETY:` comment inside
an `except` block, and the session never opened `app/refunds.py`; it only named
`apply_refund` in passing, which is the echo tier. The three tests in
`tests/test_service.py` are `glance`: the session read that file with a range
of lines 1 to 5 and every test function starts at line 7 or later. The two
tests in `tests/test_refunds.py` are full umbra, reached transitively through
`apply_refund`. Two lit nodes are counted rather than listed.

## Honest limits of this particular demo

The exposure comes from the checkpoint of the session that built Umbra, which
is one long session covering the whole build rather than a short session that
ends at the commit. Two consequences show in the output above:

- The session-said line falls back to the last thing the agent said, which is a
  sentence of narration rather than a commit summary. In a short session ending
  at a commit it would be the closing account of the change. The fallback is
  bounded by the last edit to a source file where it can be, and the header
  always names where the sentence came from.
- Files touched anywhere in the long session are lit for every commit in it, so
  the illumination fraction is not comparable to a short session's.

The dedicated recorded scenarios in phase 7 are the fix for both, and the
fixture is disclosed as seeded in PRD.md.

## Deviations from the letter of the plan, and why

- `fixtures/app` had four modules and four pytest files as the kickoff
  specifies. Phase 13 added a fifth of each, `app/report.py` and
  `tests/test_report.py`, because the `leak` scenario in ARCHITECTURE.md needs
  a chain four hops long through `render_lines` and the kickoff's inventory did
  not contain one. See the phase 13 section below.
- Before editing `docs/ARCHITECTURE.md`, which is a file this build did not
  create, `entire graph impact --symbol Open-questions --repo .` was run as the
  rules require. It answered `IMPACT DEGENERATE: Open-questions has no callers,
  callees or type consumers`, which is expected for a Markdown section, so the
  edit was safe.

## Phases 6 to 11

### What the later phases added

- **Phase 6.** Selection order is shadowed tests, then tests reaching shadow,
  then the rest. Ids are validated and passed as argv. Execution runs through
  `graph verify` with a baseline on the parent tree, so a failure that predates
  the change is labelled rather than blamed on it. The sweep runs by default and
  gives every missed test the first reason that applies.
- **Phase 7.** The JSON schema from ARCHITECTURE.md, the layout computed in Go
  with a determinism test, the packet, `umbra record`, and eight authored
  scenarios the replay tests drive.
- **Phase 8.** The HTML map. The light field was built and screenshotted before
  any node existed, as the plan requires.
- **Phase 9.** The attention replay, with the JavaScript classifier cross-checked
  against the Go one over all eight scenarios.
- **Phase 10.** The landing page and the drop-in viewer, both working from
  `file://`.
- **Phase 11.** The semantic-diff review and the self report.

### QA checklist results

Run against the generated report with a headless browser. Everything below was
checked by rendering the page, not by reading the code.

| Check | Result |
|---|---|
| Renders from `file://` with no network request of any kind | pass, 1 request, the file itself |
| Docket and header render with JavaScript off | pass, 11 rows and the map fallback text |
| The four states are distinguishable in greyscale | pass, by fill, hatch, half disc and dashed outline |
| Sweep from seq 0 to the end reaches the load state | pass |
| Every coined word sits next to its plain meaning | pass, checked in the docket, the packet and the page |
| Keyboard walk: enter the map, reach a node, open detail, close it | pass |
| Reduced motion: no tweens, no torch, replay still works | pass |
| Print: day scheme, replay hidden, rows not split | pass |
| A symbol named like markup renders as text | pass, in the JSON, the docket and the map |
| Narrow layout at 360px with no sideways scroll | pass, after two fixes |
| Drop a text file, a foreign report, a file over 20 MB | pass, a plain sentence for each |

### Bugs the rendering found that reading would not have

1. `html/template` treats the contents of any `<script>` element as JavaScript,
   so the embedded report was escaped into a string literal and the map drew
   nothing. It needs `template.JS`, not `template.HTML`.
2. Four sources each drew a 500px gradient and the overlap washed the middle of
   the map out, taking the labels with it. The share is scaled by the count.
3. Labels collided. The lower-scored one is hidden until interaction, and its
   disc is still drawn, so the map never claims a node does not exist.
4. In the day scheme the shadow mask made pools **brighter** than their
   surroundings, inverting the meaning on a light ground. Pools are painted with
   a token that is transparent at night and a soft dark by day.
5. At 360px the page scrolled sideways: the last replay tick sat on the edge and
   the reproduce command would not wrap.
6. Re-rendering appended a second set of replay ticks instead of replacing them.
7. `.replay button` outranked `.tick`, so every tick drew as a 40px button.

### What phase 11 found

The semantic diff reported **no symbols at all** for `umbra/cmd`. The cause was
a `.gitignore` pattern I wrote in phase 0: an unanchored `entire-umbra`, meant
for the built binary, also matches the source directory `cmd/entire-umbra/`.
The whole command package was untracked for nine phases. Local tests passed the
entire time because the files were on disk. A fresh clone would not have built.

This is the single most useful thing the build produced: a green test run is
not the same as a correct repository, and only a tool that looks at the
repository rather than the working tree could tell the difference.

Running Umbra on its own build then found a limitation worth stating rather
than tuning away. On the commit where Umbra changed `shadow.Build`, all twelve
dependents came back umbra, including every scenario test. The reading is
correct: that change was made with a shell command rather than the editing
tool, so the session produced no read and no edit event for the file. **Umbra's
examined set is built from tool activity, so an agent that works through the
shell leaves no trace it can see.** It is in the README's limitations and in
`site/self/`.

That self run also found two scrubber bugs:

- Paths from outside the repository reached the timeline through search output.
  A path outside the repository can never match a symbol in the field, so it is
  noise, and carrying it out leaks the layout of the machine. Such paths are now
  dropped at the adapter.
- The base64 heuristic redacted git object ids and worktree paths, because a
  forty character hex sha matches the base64 shape and the character class
  included the slash. The reproduce list is only useful if a reader can check a
  line, so hex runs are exempt and the slash is out of the class.

### Phase 12: the runner check

Grepped every committed file for `pytest -q`. What it found, and what was done:

- **No recorded scenario mentions pytest at all**, so none had to be
  re-recorded. The scenario transcripts exercise reads, searches, edits and
  mentions; the test command is not part of what they model.
- **`fixtures/app/setup.sh` told the reader to run `pytest -q`.** That is the
  demo path, since the README's reproduce section runs `setup.sh`. Changed to
  `-v`, and the script now also prints the activate line and says why.
- **Everything else that names `pytest -q` documents the limitation on
  purpose** and was left alone: the README paragraph that warns against it, the
  NOTES tables, the comment on `DefaultTestRunner`, the landing-page caption,
  and `verify_test.go`, which tests the degraded path and needs the quiet form
  to do it. The four documents in `docs/` are the original design and stay as
  written.
- **`internal/transcript/testdata/classic.jsonl` contains a Bash command
  `pytest tests/test_api.py`.** That is a faithful record of what the probe
  session ran, inside a fixture the adapter parses. It is never passed to
  `verify` and it is not a documented reproduce path, so it stays accurate to
  the session it models.

**The check found a second cause of the same failure, and it was in the
README's own reproduce command.** The documented line was:

```
entire umbra 0063443 --test "umbra/fixtures/app/.venv/bin/python -m pytest -v"
```

That degrades to an exit-code-only verdict even though it says `-v`, and it
names no cracked test. The runner path is relative to the repository root, but
Umbra runs the tests in a **detached worktree** of the commit, starting in the
project directory inside it. The path does not resolve from there, and the
virtualenv is not in the worktree at all because it is not committed. A runner
that cannot start produces no output, so `verify` has nothing to parse and
falls back to a suite verdict. The symptom is identical to the `pytest -q` one
and the cause is different.

The documented path now activates the venv so the runner is on `PATH`:

```
. umbra/fixtures/app/.venv/bin/activate
entire umbra 0063443 --test "pytest -v"
```

Confirmed against the fixture: 8 probes selected, **5 cracked, all named**,
`tests/test_service.py::{test_empty_is_zero,test_negative,test_rounding}` and
both tests in `tests/test_refunds.py`, with a full sweep and 0 leaks. The
reasoning is written up in `fixtures/app/runner_test_note.md` and summarised in
the README and on the landing page.

### Phase 12: the fresh minimal record

`fixtures/recorded/minimal/recording.json` is now committed: a full Runner
recording of a short real session on the fixture app. The session read
`app/models.py` in full, read `app/refunds.py` with an offset and limit that
stop before `apply_refund`, searched for `round_money`, changed that one
function so negative amounts round away from zero, and ran
`tests/test_models.py`. Nothing else.

**The 900 KB recording of the build session is deliberately not committed, and
this is why.** A full recording embeds the checkpoint transcript, and for this
repository that transcript is the entire build session. `umbra record` scrubs
it, so absolute paths, the user and host name, the author name, the email
address and token shapes are all gone. But SECURITY_AND_ACCESS.md requires a
fixture recording to be **hand reviewed** before it is committed, and 900 KB of
conversation cannot honestly be read line by line. Committing it and saying it
was reviewed would be exactly the kind of unchecked claim this project exists
to catch. The minimal recording is twelve transcript records; it was actually
read end to end, record by record, and that reading is what the rule asks for.

Two things had to be worked out to get a recording worth committing:

1. **Where it was recorded from.** Taken inside the Umbra repository the same
   short session produces **3.8 MB**, of which **3.4 MB is one `graph snapshot`
   of the whole codebase**. None of that bulk says anything about the session.
   Recorded against a repository holding only the fixture app it is **135 KB**.
   The session's paths were relocated to that root; every tool call, range,
   result and timestamp is the real one, and the scrub replaces the prefix with
   `<repo>` in any case.

2. **Graph refuses a worktree under a restricted path.** With the fixture
   repository in the session scratch directory, `graph commit` answered
   `refuse Git subprocesses for unsafe or unreadable repository metadata`,
   because the worktree metadata pointed into a directory it could not treat as
   safe. Moving the repository under `$HOME` fixed it. Worth knowing before
   anyone tries to record from a temporary directory.

**The hand review found a real scrubber gap.** The first recording contained
the repository owner's real name and email address, from the author line that
`checkpoint explain --short` prints and Umbra reads for the session-said
sentence. The scrubber had no rule for an email address and no way to know a
display name. Both are fixed: an email pattern is always redacted, and
`umbra record` now asks git for the author names configured in the repository
and removes them by value, since a name cannot be recognised by shape. The
author line in the committed recording reads `author   <author> <<redacted>>`.

Checks that now run as tests: the recording replays every call it holds, it
contains the calls a real analysis makes, and it carries no absolute home path,
email address or token shape while still showing the `<repo>` and `<home>`
placeholders that prove the scrub ran rather than found nothing.

### Phase 12: the pre-public scrub pass

Swept every committed file, re-ran the scrubber over `site/self/` and the
recordings, read a checkpoint transcript, and confirmed the two earlier
scrubber fixes.

**Four things were found in committed artifacts and fixed.**

1. **`site/self/umbra.html` and `umbra.json` carried the repository owner's
   real email address.** They were generated before the email rule existed.
   Regenerated.
2. **An email address was being extracted as a file path.** The path character
   class has to allow `@`, and an address then matches the bare-path shape
   exactly: dots and a trailing extension. `github.com` got in the same way.
   Addresses are now rejected at the adapter, and the timeline is filtered
   against the repository's real file list at the renderer, so nothing that is
   not a file in this repository can reach a report.
3. **The session-said line was quoting an arbitrary sentence.** When a session
   leaves no edit event to bound the search, the code fell back to the last
   sentence of the whole transcript. For the build session that is a line of
   narration written hours after the commit, and putting it in a shared report
   publishes conversation that has nothing to do with the change. The
   unbounded fallback is gone: the line is omitted and the header says why.
4. **A test fixture and a note carried a real person's name.** Both anonymised.

**What is left, and why it is left.**

- The scrubber's own test inputs are token-shaped by design:
  `ghp_0123456789abcdefghijABCDEF`, `sk-0123...`, `xoxb-1234567890-abcdefghij`,
  `AKIAIOSFODNN7EXAMPLE` (Amazon's published example key),
  `BEGIN RSA PRIVATE KEY`, `/home/someone`, `someone@example.com`. All are
  invented and all exist so a test can prove the scrubber removes them.
- `fixtures/recorded/minimal/recording.json` contains two em dashes, both
  inside verbatim `entire graph verify` output: the punctuation Graph itself
  puts between `VERDICT: NO EFFECT` and the sentence explaining it. Editing
  recorded tool output would make the fixture a lie about what the tool
  printed, so they stay.

**The checkpoint transcripts are the real exposure, and they are not Umbra's to
scrub.** Every hook-written checkpoint stores the whole session transcript:
8.9 MB, 1836 records. Sixteen of those refs are already on `origin` under
`refs/entire/checkpoints/`, and `entire status` reports twenty more waiting for
the next push. Umbra scrubs its own output; it does not and cannot rewrite
Entire's storage. A scan of one transcript found roughly 1790 absolute home
paths, 2196 occurrences of the operating-system user name, the owner's email
address, paths naming three unrelated projects on the machine, and verbatim
content from one of them. No real credential: the three token-shaped matches
are the synthetic values from the scrubber's own tests.

This is written up as a go/no-go item rather than acted on, because publishing
a repository is the owner's decision, not the build's.

### Phase 13: the real four-hop leak chain

**Path taken: the preferred one. The fixture gained the chain, and
ARCHITECTURE.md was left describing what it always described.**

The design's `leak` scenario is a test four hops from the changed symbol,
reported as `path exists at depth 4 via render_lines`. The fixture had no such
chain, so the forensics had been tested at depth four against a graph built
inside the test. That proved the code path and proved nothing about the
fixture.

`app/report.py` now holds `render_lines`, which calls `line_text`, which calls
`line_total`, which calls `compute_total`. `tests/test_report.py` reaches
`render_lines`. The hop count was measured before anything was asserted, using
the same path search the forensics use, against the re-captured snapshot:

```
line_total           depth 1 via compute_total
line_text            depth 2 via line_total
render_lines         depth 3 via line_text
test_receipt_total   depth 4 via render_lines
```

`entire graph impact` cannot show this on its own: it caps at `--depth 2`, so
it reports `line_total` and `line_text` and stops. The measurement had to come
from the field.

Confirmed end to end rather than only in the scenario test. A throwaway commit
adding a fourth parameter to `compute_total`, with `app/report.py` left
unchanged, produces:

```
probes  8 selected  0 cracked
        1 further test(s) failed that no probe covered; they are listed as leaks below
sweep   full suite  1 leak
        tests/test_report.py::test_receipt_total  path exists at depth 4 via render_lines
```

That is the string in the JSON example in ARCHITECTURE.md section 7, produced
by a real run rather than written into a fixture.

**What it perturbed, which was little.** The scenarios read a captured snapshot
rather than the live fixture, so re-capturing it was the only way the new chain
could reach them. Exactly one scenario broke: `everything-lit` asserts nothing
is in shadow, and the two new symbols in `app/report.py` were unread. Its
transcript gained a read of that module and of the new test file, which is what
the scenario means. No other scenario's assertions moved. One unit test
asserted `compute_total` starts at line 20; the fixture had grown by then, so
that assertion is now structural rather than a line number, which is what it
was really testing.

**A reporting bug the new scenario exposed.** With a leak that actually fails,
the terminal read `probes 8 selected 1 cracked` and then named a test that was
not among the eight. The sweep's failures were being folded into the same list
as the probes' failures. A test the sweep turns up is a leak by definition, so
cracked now counts only probes that were selected and failed, and a failure no
probe covered is called out separately and listed as a leak with its reason.
`--fail-on failure` still fires on any new failure, which is correct: a new
failure is a new failure whoever found it.

### Phase 13: what the end to end tests found by failing in a clone

The exit-code tests passed here and failed the moment they ran in a fresh
clone. That is the phase 11 lesson again, and this time it surfaced two product
bugs and one piece of fragility in the test itself.

**1. `graph checkpoint` was answering about the wrong commit.** `LoadSources`
tries `graph checkpoint <id>` first, because ARCHITECTURE.md calls it the
documented bridge between Graph and Checkpoints. But it returns the changes of
the commit *that checkpoint belongs to*. When a reference is paired with a
checkpoint by session window, that is a guess about which session produced the
commit and says nothing about which commit the checkpoint owns. In this working
copy the pairing happens to choose an imported checkpoint, which has no commit,
so the fallback runs and the report is correct. In a clone it chose a
hook-written checkpoint that does own a commit, and the report described that
commit's changes under the header of the one the reader asked for: the header
said `0063443` while the sources were the transcript package from phase 3. The
bridge is now used only when the reference resolved through the trailer or
through the checkpoint id, where the two are the same piece of work.

**2. A stale worktree poisoned every later run.** Worktrees are keyed by commit
under a shared data directory, so two checkouts of one repository, or a run
that died before its cleanup, collide on the same path. The code reused
anything with a `.git` in it. A worktree left behind by a checkout that had
since been deleted made git unable to read its metadata and Graph refuse to run
there, for every subsequent run against that commit from any checkout. A
leftover is now reused only when it is a healthy checkout sitting at the wanted
commit, and otherwise removed and pruned first.

**3. The tests were asserting on states that are not repository state.** They
checked that a given commit has umbra nodes. Node states come from the examined
set, which comes from whichever checkpoint the reference pairs with, and the
set of checkpoints present differs between a working copy and a clone: imported
history is local and is not pushed. The same commit reports six umbra nodes
here and none in a clone, both correct. The tests now run the binary once,
read the summary from its JSON, and derive a condition that must fire from what
the report actually says. They also skip when the toolchain cannot analyse the
checkout at all, which happens where Graph refuses to run git, the same
restriction phase 12 hit when recording from a temporary directory.

Verified in three places before the commit: this working copy, a clone under
`$HOME`, and a clone under the restricted scratch directory, where the suite
skips cleanly rather than failing.

### Where the build deviates from the letter of the plan

- **The README is at the repository root, not a one-line pointer.** That rule
  protects a host project's README in a fork. This repository is Umbra itself,
  so there is no host project to leave alone and the root README is Umbra's.
- **`--test-root` was added to the command surface.** PRD.md does not list it.
  A runner has to start in the project directory, and the fixture app is nested
  at `umbra/fixtures/app`. It is found automatically from the project files near
  the tests; the flag only exists for when that search picks wrong.
- **The recording of the build session is deliberately not committed; a small
  one is.** See the phase 12 section below.
- **The docket merges the file path into the symbol cell** rather than giving it
  its own column. The column list in FRONTEND_SPEC.md has six; the wireframe in
  the same document shows the path under the symbol, and at 38 percent of the
  page six columns broke symbol names mid-word. The information is the same.
- **`umbra.js` is about 1200 lines and `umbra.css` about 450**, against targets
  of 700 and 400. The targets were not treated as hard limits. The excess is the
  replay's state reconstruction and the day-scheme and print blocks.
- **Resolved in phase 13.** The `leak` scenario now uses the real four-hop
  chain through `app/report.py` that ARCHITECTURE.md describes.

## Before this repository is ever made public

It is private, and the phase 12 scrub pass found one reason it should stay that
way until this is dealt with. Recording it here so the decision is not lost.

**The checkpoint refs on `origin` carry another project's material.** Every
hook-written checkpoint stores the whole session transcript: 8.9 MB, 1836
records. Those refs live under `refs/entire/checkpoints/` and are pushed with
every `git push`. One of them contains verbatim content from **Impeach**, an
unreleased project that happens to live on the same machine, captured in the
first few minutes of the build when its handoff files were read to establish
that they were not Umbra's. The same transcripts carry roughly 1790 absolute
home paths, 2196 occurrences of the operating-system user name, the owner's
email address, and paths naming two other projects.

No real credential is in them. The three token-shaped matches are the synthetic
values from the scrubber's own tests.

Umbra scrubs what Umbra writes. It does not rewrite Entire's storage, so this
is not something the tool can fix for you.

To publish, one of these has to happen first:

1. Delete the checkpoint refs from the remote and stop syncing them, or
2. Read them and accept what they contain, or
3. Rebuild the history in a fresh repository with checkpoint sync off.

The committed working tree itself is clean and stays clean: three tests in
`internal/report/artifacts_test.go` check the published artifacts directly for
absolute paths, email addresses and token shapes, so a regenerated report
cannot quietly reintroduce a leak.
