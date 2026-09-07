# Umbra: build notes

Working notes kept during the build. Written after phase 5, and extended as
later phases land. The design documents in `docs/` stay as written; anything
this file records that contradicts them is a finding from real output, and the
answered questions at the bottom of `docs/ARCHITECTURE.md` carry the detail.

## Where the build stands

All seventeen phases are done and committed. `go test ./...` is green at every
commit, and a fresh clone was checked to build and test green after phase 11
found that it would not have. The reproduce block in the README was re-run from
a fresh clone in phase 17.

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
| 14 | A real sample on the landing page, the imported map, light field, labels, two prepared scripts | 298 |
| 15 | The scrubber becomes the only exit; a known-positive for every check | 316 |
| 16 | The eclipse hero: a drawn corona, the four beats, the day and night divider | 316 |
| 17 | Ground and focus checked from screenshots, PUBLISH.md, the README submission pass | 316 |

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
KICKOFF_PROMPT.md were never supplied, and similarly named files in
`~/Downloads` belonged to an unrelated project, so they were not used.

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
entire umbra 1c2cf29 --test "umbra/fixtures/app/.venv/bin/python -m pytest -v"
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
entire umbra 1c2cf29 --test "pytest -v"
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
said `1c2cf29` while the sources were the transcript package from phase 3. The
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

## Phase 14: the two prepared scripts

Both are written and neither has been run. `scripts/graft.sh --dry-run` and
`scripts/checkpoint-refs-audit.sh` were exercised; the graft was never run for
real and the audit only reads.

**`scripts/graft.sh`** forks `entireio/entire-graph`, branches from its default
branch, copies `umbra/` in unchanged, enables Entire, and makes two commits:
the four planning documents first, then everything else, so the design is in
the history before the code that implements it. It pushes nothing.

Writing it turned up a documentation problem. The instruction was to enable
Entire "with the India mirror as documented", and **there is no way to do that
from this CLI**. `entire enable`, `entire login` and `entire configure` were
all checked against 0.10.5 and none accepts a region flag; no region appears
anywhere in `entire --help`. The only mention of a mirror in `docs/` is one
line in SECURITY_AND_ACCESS.md saying it follows the fork's collaborator
permissions. The India region comes from a handoff belonging to a different
project, which was read in the first minutes of this build while establishing
that it was not Umbra's. The script therefore enables Entire the way the CLI
actually supports, says in a comment that the mirror follows the account you
are logged in as rather than the checkout, and prints who that is so it can be
checked.

The disclosure the fork's README would carry, three lines:

> Umbra was developed in a standalone repository, `u7k4rs6/Umbra`, with its own
> Entire checkpoint trail covering every phase of the build.
> That repository is the development record; this fork is the delivery.
> The checkpoint trail here begins at the graft, not at the first line of code.

**`scripts/checkpoint-refs-audit.sh`** reads the checkpoint refs on a remote
and reports, per ref, the bytes it holds and how many lines carry an absolute
home path or an email address. It deletes nothing and pushes nothing. Its
output grows with every push, which is the point of running it rather than
guessing. Before the phase 14 commits went up it read 30 refs, 288,677,347
bytes, 48,408 lines with a home path and 420 with an email address. Straight
after that push it read **39 refs, 413,032,568 bytes, 71,147 lines with a home
path and 708 with an email address**: nine commits added 124 MB and 288 more
addressed lines. The largest single ref is about 15.8 MB. The two
smallest are the ones worth noticing: `120ebb04a45d` at 27 KB with 15 path
lines and no addresses, which is the minimal session recorded in phase 12, and
it shows what a checkpoint looks like when the session that made it was short.

Three bugs were found by running it rather than by reading it, and the last one
is the reason a script like this should never be trusted unread:

1. `grep` exits non-zero when it matches nothing, which under `pipefail` ended
   the run at the first clean transcript.
2. Holding a 10 MB archive in a shell variable produced zero for every ref.
3. `git ls-tree` is scoped to the working directory. Run from `umbra/` it
   listed nothing, and the script reported thirty refs of zero bytes carrying
   no paths and no addresses. That is a confident, precise, completely wrong
   all-clear, on exactly the question the script exists to answer. It needs
   `--full-tree`.

## Phase 15: the scrubber becomes the only exit

Five scrubbing bugs so far have been the same bug five times: an output path
that wrote without going through the scrubber. Each was fixed where it was
found, which fixed that path and left the shape intact.

| # | Phase | The path that wrote unscrubbed |
|---|---|---|
| 1 | 11 | Paths from outside the repository reached the timeline through search output |
| 2 | 11 | The base64 heuristic redacted git object ids and worktree paths, so the reproduce list could not be checked |
| 3 | 12 | The recording carried an email address; the scrubber had no rule for one |
| 4 | 14.1 | `graph.Verify` appended its commands to `a.Commands` raw, bypassing the scrubber every other command went through |
| 5 | 14.2 | Only `umbra record` set `scrub.Names`, so an ordinary run left the author's display name in the timeline |

Number 2 is the odd one out: it removed too much rather than too little. The
other four are the same shape.

### The inventory, before anything changed

Every place an artifact is written, and whether it scrubbed:

| Writer | What it writes | Scrubbed before phase 15 |
|---|---|---|
| `emit` stdout, `--format table` | terminal table | only whatever the builder happened to scrub |
| `emit` stdout, `--format json` | report JSON | same |
| `emit` stdout, `--format packet` | packet markdown | same |
| `emit --out`, `umbra.json` | report JSON | same |
| `emit --out`, `umbra.packet.md` | packet markdown | same |
| `emit --out`, `umbra.html` | map, with the JSON embedded | same |
| `report.BuildLayout` via `emit` | node and ring label text inside the map | **no**, it ran before any scrub and copied node names into the layout |
| `runner.Recorder.Save` | `fixtures/recorded/*/recording.json` | yes, but through a nil-able function, and `--no-scrub` turned it off |
| `gen-site` | `site/umbra.css`, `site/umbra.js` | not applicable, they are source assets |
| `gen-site` `embedSample` | the JSON embedded in `site/index.html` | **no check of its own**, it inherited whatever the committed sample carried |
| `analyze.go` | the reproduce list | yes, since 14.1 |
| `execute.go` | the verify commands in the reproduce list | yes, since 14.1 |

"Only whatever the builder happened to scrub" is the honest description of the
old state. Values were scrubbed as they were built, one call site at a time,
and the renderers took an `*Analysis` and wrote whatever was in it. Nothing in
the type system knew the difference between an analysis that had been through
the scrubber and one that had not.

### The choke point

`report.Sealed` wraps an `*Analysis` in an unexported field. `report.Seal` is
the only way to make one, and `WriteJSON`, `MarshalJSON`, `HTML`, `Packet` and
`Table` all take a `Sealed` and refuse a zero value. `BuildJSON` is now
`buildJSON` and unexported. A new output path cannot skip the scrubber by
omission, because there is nothing for it to write from until it has sealed
something.

`Seal` scrubs every string in the analysis that could carry text from the
machine, then builds the layout from the scrubbed values. That ordering is the
fix for the row marked no above: the layout carries node names and file names
as label text, and building it first meant a scrubbed report could ship with an
unscrubbed picture on it. Nothing had leaked that way yet. It was open.

A nil scrubber means the default one for this machine, never no scrubbing.
There is no flag for less. `umbra record --no-scrub` is gone: a recording is
committed, and a flag that turns off scrubbing on a committed artifact is the
next bug in the table above waiting for someone to forget it.

### One derivation of the scrub inputs

`Options.Scrubber` derives home, repository, user, host and the git author
names once and caches them on the options. `umbra record` and the analysis
pipeline both call it and get the same value. Bug 5 was possible only because
there were two constructions and one of them was missing a line.

### The output-wide check

`report.ScanArtifact` reads finished bytes and reports absolute home paths,
addresses, token shapes and supplied display names, with line numbers. It does
not care which code path produced the bytes, which is the point: a new writer
is covered the day it exists rather than the day someone adds it to a list.

`TestNoGeneratedArtifactCarriesAnythingPrivate` walks `site/`,
`fixtures/recorded/` and `docs/renders/` rather than naming files, and takes
the author names from `git log` so it fails for whoever cloned the repository
as readily as for its author. **It scans 25 artifacts and found nothing.** The
existing corpus needed no re-scrubbing, and the self report regenerated through
the sealed pipeline is byte-identical to the committed one.

`gen-site` now runs the same scan before embedding either sample, so a report
produced by an older build or edited by hand cannot reach the landing page.

## Phase 15: a known-positive for every check

Four verifications in this project have returned a confident pass that was
wrong: the `.gitignore` case where the local tests were green but a clone would
not build, the cracked-probe count that named a test outside the selection, the
commit check that counted `ok` lines instead of looking for `FAIL`, and the
refs audit that reported thirty refs of zero bytes because `git ls-tree` is
scoped to the working directory. Every one of them had only ever passed. A
check that has never been observed to fail is not a check yet.

Each of these plants what the check is for and asserts it says so. All the
planted values are invented: `plantedperson` is not a user on any machine and
`example.invalid` is reserved by RFC 2606, so the address can never be real.

| Test | Planted | Mutation that proved it fires |
|---|---|---|
| `TestScannerCatchesPlantedHomePathEmailAndAuthorName` | a home path, an address, an author name | removed the address pattern; the scanner missed the address |
| `TestSealRemovesWhatTheScannerFinds` | the same three, into the analysis | stopped sealing `a.Commands`; the home path reached line 262 of the JSON |
| `TestRenderersRefuseAnUnsealedAnalysis` | a zero `Sealed` | none needed; it asserts an error each renderer only returns for the unsealed case |
| `TestCrackedProbesNeverCountsAFailureOutsideTheSelection` | a failure the selection never ran | made `CrackedProbes` return `NewFailures`; the leak was counted as a crack |
| `TestParseVerdictCatchesAPlantedFailure` | a run reporting `NEWLY FAILING` | broke the pattern; newly failing came back empty |
| `TestParseVerdictDoesNotMiscountTheWordOk` | passing lines with `ok` in the test names | the negative half of the same pair |
| `TestCheckSampleIsRealCatchesAPlantedFabrication` | `sample000class`, an all-zero commit, no commit | disabled the prefix check; the invented id was accepted |
| `TestCheckSampleIsCleanCatchesAPlantedAddress` | an address in a report about to be published | shares the scanner's mutation above |
| `TestAuditCatchesPlantedEmail` | a checkpoint ref whose transcript carries both | see below |
| `TestAuditReportsTheSameNumbersFromASubdirectory` | the same ref, read from two directories | removed `--full-tree`; 178 bytes from the root, 0 from a subdirectory |
| `TestNoGeneratedArtifactCarriesAnythingPrivate` | a home path added to `site/sample/umbra.json` | caught it on line 1254 |

### The check that had no known-positive and was not on the list

`Box.Overlaps`. All four label-placement tests are built on it, across all
eight scenarios. Making it return false for every pair was tried, and **every
one of those tests still passed.** They had been proving nothing about
collision the whole time; what they proved was that the layout is deterministic
and that some labels are visible. `TestBoxOverlapsCatchesAPlantedOverlap` is
the case that has to fail for the rest to mean anything, including the two-unit
gap that separates a label which merely touches another from one that hides it.

### A cache that reports a pass for a version that never ran

The audit tests run a shell script through `bash`. Go's test cache keys on the
files the **test binary** opens, and bash opening the script does not count. So
the script could be edited, `go test ./scripts/` would print `ok` from cache,
and nothing would have been run. That is the same shape as the four failures
above, arriving through the tooling rather than the code. The test now reads
the script itself before running it, which puts it in the cache key. Verified:
with `--full-tree` removed and no `-count=1`, the test fails.

### The audit script

The three bugs named in the phase 14 record were fixed when they were found, in
the commit that added the script: `--full-tree`, `grep` guarded with `|| true`
so a clean transcript does not end the run under `pipefail`, and a tree walk
with awk instead of a multi-megabyte archive in a shell variable. What phase 15
adds is the proof that each fix matters. Re-run today, from the repository
root: **40 refs, 432,226,689 bytes, 74,551 lines carrying a home path, 740
carrying an address.** The numbers keep climbing because every push writes
another checkpoint; that is the finding, not a defect in the script.

## Phase 16: the eclipse hero

The landing page is restyled around a drawn total eclipse. Nothing in the
report, the report template, the drop-in viewer's behaviour or any Go file
changed. The new files are `site/landing.css`, `site/corona.js` and a rewritten
`site/index.html` and `site/site.js`. `site/umbra.css` and `site/umbra.js` are
still the report's own, copied by `gen-site`, and still byte-identical to it.

### The corona is a renderer

There is no raster asset on the page and no request for one. `corona.js` builds
the eclipse in inline SVG: layered radial gradients warped by a `feTurbulence`
`fractalNoise` with a fixed seed through an `feDisplacementMap`, streamers laid
along the x axis and rotated to their angles, an occluding disc taken out by a
mask, a bright ring hugging the limb, and one bright limb point with an
anisotropic flare made from a rotated linear gradient and one blur.

It is deterministic. Nothing calls `Math.random`. The limb angle comes from a
hash of the checkpoint's commit and the streamer geometry from a small linear
generator seeded by the same number, so the same `umbra.json` always draws the
same corona. The filament count reads the report: twenty, plus forty six times
the lit fraction, plus three per source. More of the field in the light means a
denser corona, and the number comes from `summary`, not from a constant chosen
because it looked good.

Cost was a real constraint, not a note. Every filter has an explicit
`filterUnits="userSpaceOnUse"` region, so it rasterises once at a fixed size.
No filter primitive is ever animated. The only thing that moves after the first
paint is a clip inset and an opacity. The day layer is a second copy of the
same two SVGs, so it would rasterise the turbulence twice; it is therefore
built only when the divider is off its resting position, which means an
ordinary visit pays for one corona.

Four attempts, each judged from a screenshot rather than from the code:

1. Uniform thick filaments over a strong body read as a cartoon sunburst, and
   the brightest part of the gradient sat *inside* the moon where nothing could
   see it.
2. Peaking the gradient at the limb fixed that and produced a visible circular
   rim at the outer edge.
3. Softening the tail moved the rim but did not remove it. The cause was that
   the outer stops were written as multiples of the limb radius, so the last
   one fell past the end of the gradient and the alpha dropped from 0.012 to
   zero inside the final one percent of the radius. Writing the tail in
   absolute fractions removed it.
4. The limb flare was drawn on top of the moon and read as a scratch across the
   disc. It is now drawn behind the moon, with only the bead in front, which is
   also what actually happens.

### The hero is the live map wearing the eclipse

The map is the same `renderInto` from `umbra.js`, drawn from the layout already
in `umbra.json`. The JS still computes no positions. The occluding disc sits at
the layout centre, so the four changed symbols are inside it and their
dependents sit in the corona at ring radii the Go code chose.

One thing had to give. The map paints its own background rectangle in
`--night`, as the spec requires, and that rectangle covered the corona behind
it. The rectangle is made transparent by a CSS rule on the landing page and the
stage carries the ground colour instead. `umbra.js` is untouched; a CSS `fill`
outranks the presentation attribute it sets.

### The four beats

`01 CLAIM`, `02 FIELD`, `03 THE CUT`, `04 SWEEP`, in the project's own words.
Each drives the map to a real point in the timeline: seq 0, the last event
before the cut, `t0` itself, and the end. The page loads at `04`, which is the
eclipse, so nothing moves before first paint.

The beats drive the map **through the replay strip's own controls**: they click
a tick or send the key the strip already listens for. The landing page does not
reach inside `umbra.js` and does not duplicate the state rules.

### Day and night

The divider is a real inversion, not a filter. The day side is a second drawing
of the same two SVGs under day tokens, clipped by `clip-path`, so its gradients
and pools are computed from day values and the shadow pools stay darker than
the paper around them. Cloning needed one thing that is easy to miss: duplicate
ids in one document resolve to whichever came first, so the day copy would have
pointed at the night layer's gradients and inherited the night palette. Every
id in the copy is suffixed.

It is a `role="slider"`, operable with the arrow keys, and it settles to
whichever scheme it was left nearest. `Home`, `End` and `Space` are stopped at
the divider rather than passed on, because `umbra.js` binds those on the
document for the replay and a focused divider should not also scrub the sweep.
Verified: the sweep counter is unchanged after driving the divider by keyboard.

### The font decision

**A system serif stack, no font files.** The task offered a choice between that
and a subsetted woff2 committed at `site/assets/`. There is no such file in
this repository, and FRONTEND_SPEC.md says the surfaces must not fetch fonts,
so the stack is the option that can actually be carried out today without
inventing an asset. The headline is `ui-serif, Georgia, "Iowan Old Style",
"Palatino Linotype", "Times New Roman", serif`.

### Where this direction conflicts with FRONTEND_SPEC.md

Every one of these is confined to `site/`. The report obeys the spec exactly,
and the two files they share are untouched.

| Spec says | This page does | How it is resolved |
|---|---|---|
| "No all-caps labels anywhere. No letter-spacing tricks." | Every label, nav item, legend word and marker is monospace, uppercase, tracked at 0.18em | Landing page only. The report's own labels are unchanged, and the test that keeps `site/umbra.css` byte-identical to the report's stylesheet still passes |
| "no numbered markers except on the replay timeline, which is a real sequence" | Numbered section markers, `01` to `11` | The four in the rail are a real sequence: they are positions in the session's timeline and clicking one moves the playhead. The section markers are navigation and are the one place this page takes the liberty knowingly |
| Night is `#10141C`, "a blue-black night, not a neutral near-black" | Pure black `#000000` | Landing page only. The reason the spec gives for the blue-black is that a neutral near-black with one neon accent reads as a template; this page answers that with a full warm ramp rather than with a single accent |
| `--focus: #9CC4FF`, a blue | `--focus` is the warm accent | "No third hue anywhere" was the instruction. Focus rings stay 2px with a 2px offset and were checked at every tab stop |
| `--pass: #8FBF9F` green, `--fail: #C8553D` red | `--pass` is the warm off-white ink, `--fail` is the accent | The landing page has no docket, so pass and fail appear only as node motifs, and those are already distinguished by shape: a cracked probe carries two crack strokes whatever colour it is drawn in. The report keeps the green and the red |
| "Landing page: single column, 960px max" | Two columns to 1340px, with the map as a full-height stage beside the copy | The direction asked for the hero to be the map with the copy beside it. Below 1000px it collapses to the single column the spec describes |
| "There are no cards" | The legend and the beats sit in hairline-bordered cells | They are controls, not decoration: each one filters or moves the map. No fills, no shadows, no radii |

### The QA checklist, re-run

Every line of the checklist in FRONTEND_SPEC.md, against the landing page.

| Check | Result |
|---|---|
| Renders from `file://`, night and day, no network requests | Six requests, all `file://`: the page, two stylesheets, three scripts. Nothing else |
| Greyscale: the states are distinguishable by shape and hatch alone | Yes. `landing-night-greyscale.png`. Lit is a filled disc, penumbra a half disc, umbra an outlined hatched disc; the legend orbs differ in radius as well as in value. `unknown` does not occur in this report |
| Sweep from seq 0 to the end produces the same final state as the page load | Yes, checked by comparing every node's aria-label after a full 1x then 4x play against a fresh load: identical |
| The afterimage transition at the cut is visible; a mention lights a node only to echo | Yes. Beat 03 is the cut, and the node aria-labels carry the tier |
| Every coined word has its plain meaning beside it | Yes: the three legend orbs, the five tier chips, the state table |
| Keyboard-only walk | Sixteen tab stops in reading order: four nav links, the install command, copy, the map as one stop, the divider, three legend orbs, four beats, then the replay. Every stop has an accessible name |
| Screen reader announces node labels and the current replay event | Node groups are `role="button"` with labels like `apply_refund, penumbra, echo, depth 1`; the map is `role="group"` with a one-sentence label; the replay's current event is in an `aria-live="polite"` region |
| Reduced motion: no tweens, replay still works | Yes. Transitions are off and the divider settles instantly instead of tweening |
| No JavaScript: the page reads | Yes. The stage, legend, beats and replay are hidden rather than left as empty boxes, and the noscript line explains why. Everything else is text and reads identically |
| Drop a report from another repository | Renders. The second map on the page is exactly that case, committed |
| Drop a text file | "That file was not an Umbra report: it is not valid JSON." |
| Drop a 25 MB file | "That file is 25 MB. The viewer refuses anything over 20 MB." |
| A symbol named `<img src=x onerror=alert(1)>` renders as text | No `img` element is created and the raw tag does not appear in the map's `innerHTML`; the name reaches the node's aria-label as text. Its visible label happens to be hidden by the existing overflow rule for that node, which is placement, not escaping |
| Narrow layout at 360px | No horizontal overflow at 360, 390, 560, 900, 1280 or 1600. `landing-narrow-360.png` |
| Nothing moves before first paint | The page loads at beat 04 with the divider at rest. No transition runs on load |

One checklist line does not apply. "Enter the map, reach the top umbra node,
open detail, reach a docket row, filter to tests only" is the report's walk;
the landing page has no docket and no detail panel, which is what the spec's
own landing-page section describes. The map is still one tab stop and its nodes
still carry their labels.

## Phase 17: two visual checks

### Ground colour, measured rather than argued

The hero rendered twice at the same viewport and the same playhead, once on the
current pure black and once on FRONTEND_SPEC.md's `#10141C`:
`docs/renders/ground-black.png` and `docs/renders/ground-nearblack.png`.

Sampling the luminance along the horizontal from the eclipse centre outwards,
the corona's geometry is identical in both, reaching the ground at the same
+372px. What differs is whether the tail can be seen getting there.

| Distance from the limb | On black | On `#10141C` |
|---|---|---|
| ground level | 0.00 | 19.73 |
| +60px | 14.05 above ground, rgb 20,13,7 | 11.98 above ground, rgb 34,31,32 |
| +120px | 32.17 above ground | 28.31 above ground, rgb 60,46,33 |
| +200px | 2.49 above ground | 1.49 above ground, rgb 20,21,27 |
| +280px and beyond | 0.00 | 0.00 |

Two things go wrong on the lifted ground. The tail is clamped: from +200px out
it sits within 1.5 luminance units of the ground, so the last 170px of corona
is in the pixels but not in the eye, and the light reads as stopping at +200px
rather than fading to nothing. And the warm tail desaturates: at +60px it
renders rgb 34,31,32, which is neutral grey, because a low-alpha warm layer
summed with a blue-black ground lands off the warm axis. At +200px the blue
channel is actually **lower** than the ground's own, so the faintest corona is
a cool smudge rather than warm light.

**Pure black holds the corona better**, and the reason is specific to this
page: the corona is the light source here, and a light field is only as long as
the floor it falls to. The spec's argument for the blue-black is that a neutral
near-black with one neon accent reads as a template, and that argument is
answered by the full warm ramp rather than by lifting the floor. Not changed.

### Focus visibility, and one thing put back

Three focus positions captured: `docs/renders/focus-nav.png`,
`focus-copy.png`, `focus-divider.png`.

On the nav link and the copy button the amber ring was findable, roughly 7 to 1
against black. On the divider it was not, and the screenshot says why in one
look: the ring came out `rgb(255, 163, 60)`, which is the exact colour of the
divider line it surrounds, so focus read as a second decorative line. The same
collision was measured on a pressed beat, whose selected marker is that colour,
and on a pressed legend orb, whose border is too.

**Fixed by restoring a distinct focus hue**, not by thickening the ring:
`--focus-ring: #9CC4FF` for night and `#1A4F9C` for day and print, which are
FRONTEND_SPEC.md's own focus colours. The ring stays 2px with a 2px offset,
4px on the divider.

This is a deviation from the phase 16 direction, which asked for one warm hue
and no third hue anywhere. It is recorded as a deviation in both directions: it
breaks the one-hue rule on purpose, and it puts the spec's value back. A focus
ring the same colour as the control it marks is not a focus ring, and this page
is the only place where the two rules could not both hold.

## Phase 18: the motion layer

The landing page was asked to feel like designcode.io: dynamic, interactive,
alive. The site was opened in Playwright and read rather than recalled, and
what was taken from it is the interaction vocabulary, not the surface.

What it is actually made of, from its own computed styles: Inter and Geist
Sans, a near-black `#020202` ground, a blue and violet gradient beam, rounded
glass cards, and one easing everywhere,
`cubic-bezier(0.16, 1, 0.3, 1)`, on transitions like
`opacity 1s, transform 1s, filter 0.9s`. That last line is the whole trick: a
blur-and-rise reveal on an expo-out curve.

**Taken:** the blur reveal and its easing, a persistent hairline grid, a sticky
bar that condenses on scroll, a section rail that tracks where you are, a
pointer-tracked highlight inside panels, and the pattern where one item in a
list is bright and its siblings sit back.

**Refused:** its fonts and its colours, which is what the brief said was wrong
with it. Inter and Geist are the default of every developer landing page
shipped this year, and the blue-violet gradient is the same. The page keeps its
own single warm hue and gains a stronger type position rather than a borrowed
one.

### Type: two families, no sans

The body was a UI sans inherited from the report's stylesheet. It is now the
serif, the same one the display uses. Two families carry the whole page: the
serif for anything a person reads in sentences, the monospace for anything the
machine named. There is no third, and there is no sans anywhere. That is the
answer to "generic fonts" that does not need a font file, which the spec
forbids fetching: distinctiveness from the pairing and the setting rather than
from the download. The display is tracked to -0.032em and set at 0.95 leading,
which is tight enough to read as a masthead rather than as a heading.

### The reveal is the vocabulary

Content arrives blurred, dim and low, and resolves to sharp and lit. That is
not a borrowed effect on this page: it is penumbra to lit, the same transition
the map performs, applied to the prose. The hero is staged rather than
revealed, five pieces a beat apart in reading order, so it assembles rather
than appears.

Only opacity, transform and filter are animated, so nothing here forces a
layout. The scroll handler is rAF-throttled and writes two custom properties.

### What the screenshots caught that reading would not have

**The sticky bar was not sticky.** `umbra.css` sets `overflow-x: hidden` on
`html, body`, which is right for the report and fatal here: an overflow on
either makes it a scrolling box, and `position: sticky` then sticks to that box
rather than to the viewport. The bar scrolled away with a `getBoundingClientRect
().top` of -2984. The overflow is now cleared on this page and horizontal
overflow is checked at six widths instead, which is the honest way to not have
any.

**Clearing it exposed an 8px overflow** at every width below 900, previously
swallowed by that same `overflow-x: hidden`. The bar bleeds through the page
gutter with a negative margin of 24px while the gutter at narrow widths is
16px. Both now read one `--pad`.

**The second map was a black box in a frame.** The plate became a panel with a
surface, and the map's own background rectangle is opaque, so the panel framed
it. Same fix as the hero: the rectangle is transparent on this page.

**Print caught a section mid-fade.** The print block set the reveals to
`opacity: 1` but left the transition running, so a print taken in the first
second put a half faded paragraph on paper. Transitions are off in print.

**The section rail put seven links before the content.** Eleven tab stops
before the install command. There is now a skip link as the first stop.

### Accessibility and the fallbacks

- `prefers-reduced-motion: reduce`: nothing arrives, nothing recedes, the
  torch is off, and every reveal resolves to its finished state with no
  transition rather than a faster one. Verified: zero elements left below
  opacity 1 after load, `transform: none` on the stage.
- Script off: the reveal rules are gated on a `js` class set in the head before
  first paint, so the page is simply visible. The grid, the progress line and
  the section rail are hidden rather than left as dead furniture.
- Print: day scheme, no grid, no rail, no progress, static bar.
- Six widths from 360 to 1600 with no horizontal overflow. The section rail is
  dropped below 760px, where the bar would otherwise be three rows tall and eat
  a third of a phone screen.
- Seven requests on load, all `file://`. No console errors.

### Where this goes further from FRONTEND_SPEC.md

Two additions to the table in phase 16, both landing page only.

| Spec says | This page does | How it is resolved |
|---|---|---|
| "Nothing moves on load. The page renders in its final state." | The hero stages itself in over about a second | The report still does not move on load and that rule is untouched there. This is the landing page, which phase 16 already established may use scroll and hover motion. It is off entirely under reduced motion, and the page is fully readable with script off |
| "There are no cards, no panels with drop shadows, no gradients as decoration" | Panels with a hairline, a 2 percent surface and a 2px radius | Still no fills, no shadows and no radius worth the name. The one gradient that is decoration is the drafting grid, and it is doing a job: it gives the columns a rhythm and makes the black read as a surface rather than as nothing |

## Phase 19: a site, and then a depth pass

### From one page to six

The landing page was one long document. It is now `index`, `map`, `how`,
`install`, `build` and `limits`, with shared chrome. `gen-site` learned about
more than one page: it embeds the report data into every page that carries the
block and skips the ones that do not, so only `index` and `map` pay the 480 KB
and the other four are a few kilobytes each. `embedSample` returns whether the
page carried the block, which turns a page without a map into an ordinary case
rather than an error.

The header and footer are written into each page rather than assembled at build
time, so four tests guard against drift: every page carries the same
navigation, every navigation target is a page that exists, every page carries
the same stylesheets, policy, skip link and footer and fetches nothing, and a
page carries the report data if and only if it draws a map.

**One layout bug worth naming.** The hero rows had `margin: 0 auto` while being
flex items of a flex column. `margin: auto` on a flex item turns off stretch,
so each row shrank to fit and centred itself: the copy column came out 386px
wide inside a 1440px hero, which is why the headline was wrapping one word per
line. Found from a screenshot, then confirmed from the box model rather than
guessed at.

### The depth pass

Flat black on a good screen reads as a hole rather than a surface. What was
added, all of it opacity, transform and filter only:

- **Film grain.** One inline SVG turbulence, tiled, at 4.5 percent, stepping
  through six positions. It is the cheapest thing on the page and it does more
  for the black than anything else.
- **An edge light on every card**, a masked gradient border that brightens
  under the pointer. This is the detail that makes a rectangle read as a
  surface with an edge rather than as a box.
- **A ghost section index**, the marker's own number set at 300px behind the
  heading at 3 percent.
- **A ticker** of the project's real counts: four states, five tiers, six
  factors, 320 tests, 19 phases, zero dependencies, zero network requests, zero
  models. Nothing on it is a figure somebody chose.
- **The wordmark once at the foot**, at 340px and 6 percent.
- **A slow drift on the corona**, and one bug from it worth recording.

**The drift desynced the map from the eclipse.** Animating the whole corona SVG
scales the occluding disc with it, while the map drawn on top does not move, so
the moon grew over the four source nodes that are meant to sit on it. The
screenshot showed the centre of the disc empty where the labels had been. Only
`.corona-field` may breathe; the moon and the map stay fixed to each other.

### On the library that was suggested

The brief pointed at a Three.js component library. It was looked at and not
used, for a reason worth writing down rather than leaving implied: the kickoff
rule is vanilla JS and CSS, no libraries, no external requests, works from
`file://`, and the test suite rests on it. Three.js is several hundred
kilobytes of dependency and the good templates there are paid, so they are not
ours to copy either. The idea was taken instead. Their heroes run a stock WebGL
shader; this one draws a corona from the report's own numbers with no
dependency at all, which is the better version of the same move because it
means something.

### Checked

Six pages at 390, 900 and 1440: no horizontal overflow, no external requests,
no console errors, on any of the eighteen combinations. Reduced motion: zero
animations running anywhere on the page and nothing left hidden. Script off:
readable, with the grain, ticker, grid and pill hidden rather than left as dead
furniture. Print: white ground, no grain, no ticker, no ghost numbers. First
tab stop is the skip link.

## Before this repository is ever made public

It is private, and the phase 12 scrub pass found one reason it should stay that
way until this is dealt with. Recording it here so the decision is not lost.

**The checkpoint refs on `origin` carry another project's material.** Every
hook-written checkpoint stores the whole session transcript: 8.9 MB, 1836
records. Those refs live under `refs/entire/checkpoints/` and are pushed with
every `git push`. One of them contains verbatim content from an unrelated
project that happens to live on the same machine, captured in the
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

## Phase 22: the sweep that could not read its own verdict

`umbra/experiments/first-real-repo.md` defect 2. On `pallets/click`, a change
that broke 55 tests produced `sweep full suite 0 leaks`. The audit that exists
to cover a thin selection returned a clean answer, and the report said nothing
about being unable to read anything.

### What `graph verify` actually prints, recorded rather than guessed

There is **no machine-readable form of `graph verify`**. Checked before
parsing anything: `entire graph verify --help` lists `--test`, `--repo`,
`--setup`, `--record-baseline`, `--pre-edit-baseline` and `--max-bytes`, and
no `--format`. `entire graph capabilities --json` describes languages and
extensions and says nothing about verify's output. Human text is the only
surface there is.

Two sentences from that help text explain everything below:

> ids are, text is not, and id lists cap at 20 with a count
>
> `--max-bytes <n>` Cap the rendered verdict; the verdict clause always
> survives (default: 2048)

Five shapes were captured from real runs against a clone of `pallets/click`,
base `562e458`, and are committed verbatim under
`internal/graph/testdata/verify/`. Every one is the unedited stdout of a real
`entire graph verify` invocation.

**15 newly failing, 1865 bytes, two lines** (`15-both-lines.txt`):

```
NEWLY FAILING (15): tests/test_arguments.py::test_argument_help, ... , tests/test_formatting.py::test_wrapping_long_options_strings
VERDICT: REGRESSION in 15 tests: tests/test_arguments.py::test_argument_help, ... , tests/test_formatting.py::test_wrapping_long_options_strings
```

**17 newly failing, 1040 bytes, one line** (`17-verdict-only.txt`):

```
VERDICT: REGRESSION in 17 tests: tests/test_arguments.py::test_nested_subcommand_help, ...
```

**51 newly failing, default budget, 1443 bytes, one line**
(`51-verdict-only-truncated.txt`):

```
VERDICT: REGRESSION in 51 tests: tests/test_arguments.py::test_argument_metavar_marks_optional[kwargs0-FOO], ... , tests/test_formatting.py::test_basic_functionality, … and 31 more
```

**51 newly failing, `--max-bytes 65536`, 2873 bytes, two lines**
(`51-both-lines-truncated.txt`):

```
NEWLY FAILING (51): ... , … and 31 more
VERDICT: REGRESSION in 51 tests: ... , … and 31 more
```

**51 newly failing, `--max-bytes 200`, 188 bytes, one line**
(`51-verdict-only-bare-ellipsis.txt`):

```
VERDICT: REGRESSION in 51 tests: tests/test_arguments.py::test_argument_metavar_marks_optional[kwargs0-FOO], tests/test_arguments.py::test_argument_metavar_marks_optional[kwargs1-FOO] …
```

**38 newly passing, 1541 bytes, two lines**
(`38-newly-passing-no-count-in-verdict.txt`), which is the same pair of
worktrees adjudicated the other way round:

```
NEWLY PASSING (38): tests/test_arguments.py::test_argument_metavar_marks_optional[kwargs0-FOO], ... , … and 18 more
VERDICT: PASS ... the change fixes the target behavior and introduces no regressions.
```

That last one is why the `(n)` header has to be read as well as the verdict
clause. **A passing verdict carries no number at all**, so `NEWLY PASSING (38)`
is the only place the count exists, and its list is capped at twenty like every
other. It is also the shape that proves the verdict clause must not be parsed
as though every verdict were a regression. The recorded file carries an em dash
in Graph's own verdict wording; it is unedited tool output, for the same reason
`fixtures/recorded/minimal/recording.json` keeps the two it holds.

### The rule is bytes, not a count

The experiment writeup describes the threshold as "present at 16 or fewer,
absent at 17 or more". That is what was measured, and it is the wrong shape of
explanation. **`--max-bytes` caps the whole rendered verdict, the `VERDICT:`
clause is the one guaranteed to survive, and `NEWLY FAILING` is dropped to fit
the budget.** At 15 click ids both lines total 1865 bytes and both survive. At
17 they would total roughly 2080, over the 2048 default, so the output is the
1040 byte verdict alone. At 51, raising the budget to 65536 brings the line
back. The count where it happens therefore depends on how long the test ids
are, and there is no number to hard-code. `first-real-repo.md` still carries
the count-shaped description and should be read with this correction.

### Three things the old parser got wrong

1. **The count was never read.** `NEWLY FAILING (15):` carries an authoritative
   count in its header and `ParseVerdict` discarded it, taking the length of
   the id list as the truth. Above the cap of twenty those two numbers are
   different even when the line is present.
2. **The guaranteed line was never parsed.** `VERDICT: REGRESSION in 51 tests:`
   carries the same count and up to twenty of the same ids, and it is the one
   clause verify promises to print. It was captured as a display string and
   read for nothing.
3. **The truncation marker was not recognised.** `splitIDs` skipped a part
   beginning `and `. The real marker is `… and 31 more`, beginning with a
   Unicode ellipsis, and at a small budget it is a bare ` …` glued to the last
   id with a space rather than a comma. Both would have been taken for test
   ids.

### What changed

`Verdict` gains `FailingCount` and `PassingCount`, read from the `(n)` header
and from `REGRESSION in n tests`, whichever is larger, and
`Verdict.UnnamedFailing()` returns the difference between the count and the
ids actually recovered. When the `NEWLY FAILING` line is absent, ids and count
both come from the verdict clause.

`Execution.UnnamedFailures` carries that number into the report, and
`Execution.SweepNamedEverything()` is the single predicate the table, the
packet and the HTML header all ask. It is false when verify counted failures
it did not name and false when the runner printed no per-test ids at all,
because those are the same rule: **a sweep whose ids could not be read must
never render as a clean audit.** That is why this reuses `Degraded` rather
than standing beside it.

`--max-bytes` was deliberately **not** passed. Raising it restores the
`NEWLY FAILING` line but the id list still caps at twenty, so it yields
exactly the same count and the same twenty ids the verdict clause already
carries. It would buy nothing and would make Umbra depend on a flag an older
Graph may not accept.

### Known-positives

`internal/graph/verify_test.go` drives the five recorded files. Each was proved
to fail before the fix:

| Test | Recorded input | Mutation that proved it fires | What it caught |
|---|---|---|---|
| `TestParseVerdictReadsTheCountAtFifteen` | 15, both lines | none needed; it is the negative half | 15 ids still named, count 15, nothing unnamed |
| `TestParseVerdictReadsTheCountAtSeventeen` | 17, verdict only | removed the verdict-clause fallback | count 0, no ids at all |
| `TestParseVerdictReadsTheCountAtFiftyOne` | 51, verdict only | the same, and separately the old marker handling | count 0, no ids at all |
| `TestParseVerdictDropsTheTruncationMarker` | 51, three recordings | restored the old `and ` prefix skip | `… and 31 more` parsed as a test id |
| `TestParseVerdictReadsACountWithNoUsableIDs` | 51, `--max-bytes 200` | removed the verdict-clause fallback | count 0 on a run with 51 regressions |
| `TestParseVerdictReadsTheHeaderCountWhenTheVerdictCarriesNone` | 38 newly passing | discarded the `(n)` header count | passing count 0, because a PASS verdict has no number |
| `TestParseVerdictNoEffectCountsNothing` | synthetic NO EFFECT | none needed | the fix must not invent failures on a clean run |
| `TestSweepNeverReportsACleanAuditItCouldNotRead` | 31 unnamed | reverted the table sweep line | `sweep full suite 0 leaks` |
| `TestSweepStillNamesEveryLeakWhenNothingWasTruncated` | 2 named leaks | none needed; it is the negative half | plain count and full forensics preserved |
| `TestPacketAndMapHeaderAlsoRefuseACleanAudit` | 31 unnamed | reverted packet.go and html.go | packet and map header still claimed a clean sweep |
| `TestNoteUnnamedCarriesTheCountOntoTheReport` | 51 counted, 20 named | made `noteUnnamed` a no-op | the count never reached the report |
| `TestNoteUnnamedKeepsTheLargerCount` | probe then sweep | the same | a small probe verdict erased what the sweep found |
| `TestFailOnFiresOnFailuresVerifyCouldNotName` | 31 unnamed | gated on list lengths again | `--fail-on failure` and `--fail-on leak` exited 0 |
| `TestFailOnDoesNotFireOnADegradedRunWithNoFailures` | degraded, no failures | none needed; it is the negative half | a suite-level pass must not exit 2 |

Seven mutations in all, each applied on its own and reverted. Every one of them
made at least one test fail, and the three marked "negative half" are the ones
that must keep passing under all of them.

The 51 case is the same commit `first-real-repo.md` describes as breaking 55
tests. Fifty-five is pytest's count and fifty-one is verify's; the recorded
file is verify's own output, so the tests use its number.

### What is still wrong here and was left alone

`splitIDs` splits on commas, and a pytest id can contain one:
`tests/test_arguments.py::test_deprecated_usage_help_record[use, deprecated]`
arrives in the recorded 15-failure output as two parts, `...[use` and
`deprecated]`. That has always been true, it inflates the id list, and it is a
separate defect. It is not fixed in this pass because the count is now read
from the header rather than from the length of the list, so it no longer
affects whether the sweep reports a clean audit. Recorded so it is not
rediscovered as new.

## Phase 23: the changed method that was never traversed

`umbra/experiments/first-real-repo.md` defects 1 and 3. `graph commit` names a
changed method `HelpFormatter.write_usage`. The snapshot's `Name` is the bare
`write_usage`. `Bind` compared on `Name`, the match failed, `Source.Symbol`
stayed empty, and `analyze.go` skipped the impact query for any source without
one. **A changed method's dependents were never traversed, in any repository,
ever.** Every changed symbol in `fixtures/app` is a module-level function,
where the two forms are identical, which is why nothing caught it.

`Bind` is in `internal/graph/sources.go`, not in a `bind.go`; there is no such
file.

### The shapes, collected before any code was written

A probe repository with every definition shape Graph might name differently, in
the three semantic languages most likely to matter, committed once as a
baseline and then with one line changed inside every shape. What the snapshot
publishes and what `graph commit` calls each:

| Language | Shape | snapshot `name` | snapshot `qualified_name` | snapshot `container_id` | `graph commit` `name` |
|---|---|---|---|---|---|
| Python | module function | `module_function` | `module_function` | none | `module_function` |
| Python | nested function | `nested_function` | `nested_function` | the enclosing function | `nested_function` |
| Python | class | `TopClass` | `TopClass` | none | `TopClass` |
| Python | method | `method` | `TopClass.method` | `TopClass` | `TopClass.method` |
| Python | staticmethod | `static_method` | `TopClass.static_method` | `TopClass` | `TopClass.static_method` |
| Python | classmethod | `class_method` | `TopClass.class_method` | `TopClass` | `TopClass.class_method` |
| Python | property | `prop` | `TopClass.prop` | `TopClass` | `TopClass.prop` |
| Python | class nested in a function | `NestedClass` | `NestedClass` | the enclosing function | `NestedClass` |
| Python | method on a function-nested class | `nested_method` | `NestedClass.nested_method` | `NestedClass` | `NestedClass.nested_method` |
| Python | class nested in a class | `Inner` | `Inner` | `TopClass` | `Inner` |
| Python | method on a class-nested class | `inner_method` | `Inner.inner_method` | `Inner` | `Inner.inner_method` |
| Python | async function | `async_function` | `async_function` | none | `async_function` |
| Go | function | `ModuleFunction` | `ModuleFunction` | none | `ModuleFunction` |
| Go | value-receiver method | `Method` | `TopStruct.Method` | `TopStruct` | `TopStruct.Method` |
| Go | pointer-receiver method | `PointerMethod` | `TopStruct.PointerMethod` | `TopStruct` | `TopStruct.PointerMethod` |
| Go | struct field | `Field` | `TopStruct.Field` | `TopStruct` | (unchanged in the probe) |
| Go | interface method | `InterfaceMethod` | `Named.InterfaceMethod` | `Named` | (unchanged in the probe) |
| TypeScript | function | `moduleFunction` | `moduleFunction` | none | `moduleFunction` |
| TypeScript | method | `method` | `TopClass.method` | `TopClass` | `TopClass.method` |
| TypeScript | static method | `staticMethod` | `TopClass.staticMethod` | `TopClass` | `TopClass.staticMethod` |
| TypeScript | getter | `prop` | `TopClass.prop` | `TopClass` | `TopClass.prop` |
| TypeScript | nested function | `nestedFunction` | `nestedFunction` | the enclosing function | `nestedFunction` |
| TypeScript | method on a nested class | `nestedMethod` | `NestedClass.nestedMethod` | `NestedClass` | `NestedClass.nestedMethod` |

**One rule covers every row, in all three languages: `graph commit`'s `name` is
the snapshot's `qualified_name`.** So this is a lookup and not a string
problem, and the field to look up is one the provider already publishes. No
name is reconstructed here and no separator is assumed.

Two details worth recording because they constrain the implementation:

- **`qualified_name` is one level deep, never a full path.** A method on a
  class nested inside another class is `Inner.inner_method`, not
  `TopClass.Inner.inner_method`. Anything that tried to rebuild the qualified
  name from a chain of containers would produce a string Graph never emits.
- **`container_id` is a symbol id, and it is exact.** It is not needed for
  binding, because `qualified_name` already carries the container's name, but
  it is what proves the qualified name is the provider's own and not ours.

The existing `FoldFields` and `parentOf` in the same file already split a
qualified field name on its last dot, so the shape was known for fields in
phase 7 and was never generalised to methods.

### Qualified names are not unique within a file

Two functions in one file, each defining a class `Helper` with a method `run`:

```
collide.py  method  name=run  qualified=Helper.run  line=3   container=...67a82ea9
collide.py  method  name=run  qualified=Helper.run  line=10  container=...67a82ea9#2
```

`graph commit` reports both, by the same name, distinguished only by
`after_start_line` 3 and 10. Graph itself suffixes the duplicate id with `#2`.
So the file path narrows but does not decide, and a name plus a file is not an
identity.

**The rule implemented is therefore: file, then qualified name, then span
containment, and it must land on exactly one symbol.** The reported line has to
fall inside exactly one candidate's span. The old code broke a tie by picking
the nearest start line, which always returns something and is a guess. Nearest
is gone; if the line does not settle it, the source is unresolved and the
report says so by name.

### Two rules on failure

Neither of these is new behaviour bolted on; both are the same sentence problem
defect 3 describes. "The graph found no dependents" is a finding about the
code. "The graph was never asked" is a gap in the analysis. They read
identically and mean opposite things.

`report.Analysis.Unresolved` carries every changed symbol that could not be
looked up, with a reason, and `Analysis.EmptyDocketReason()` is the single
sentence the table, the packet and the map header all ask for when the docket
is empty. There are three ways to have no rows and they are now three
sentences:

| Situation | What the report says |
|---|---|
| everything was examined | nothing is in shadow: every dependent was examined |
| asked, and there is nothing | the graph was asked about every changed symbol and found no dependents |
| every symbol unresolved | no dependents could be looked up: every changed symbol is listed as unresolved below |
| some unresolved | no dependents were found for the symbols that resolved, and the unresolved ones below were never asked about |

**No skip is silent.** `analyze.go` used to `continue` past a source with no
symbol and past an impact call that errored. Both now append to `Unresolved`
with the reason, the table and the packet list them by name and location, the
map carries them in its header block, and `umbra.json` gains an `unresolved`
array. A whole-file change also gets its own note, because a changed entity
Graph could only name as a module has no symbol-level dependents to be asked
about and that is not a statement about the code.

### Fixtures

`fixtures/app/app/report.py` gains `format_footer`, which holds a nested
function `pad`, and `tests/test_report.py` gains a test for it. That is the one
definition shape in the fixture whose qualified name is bare while its
container is not. The method shape was already present and unexercised:
`app/models.py` holds `LineItem.subtotal` and `RefundLine.subtotal`, two
symbols of one bare name in one file, which is the case a bare-name lookup
cannot tell apart.

`fixtures/recorded/qualified-names/` is a ninth scenario whose changed symbols
are that method and that nested function, and whose test builds its sources
through `graph.Bind` from the names `graph commit` reports rather than by
looking a symbol up by its bare name the way the other eight do.

**The captured snapshot was spliced, not re-captured.** A clean re-capture from
a standalone copy of the fixture produced the three new symbols and also
changed eighteen unrelated edges: imports resolved to in-repo files instead of
`external:import:app.models` because the copy has `app/` at its root, and the
`FILE_CHANGES_WITH` edge between `app/api.py` and `app/service.py` disappeared
because the copy has one commit and no co-change history. Both would have moved
scenario results for reasons that have nothing to do with this change. Only the
three symbol records and the nine relations touching them were added, and the
two file records whose contents changed had their blob and size refreshed.

### Known-positives

| Test | Mutation that proved it fires | What it caught |
|---|---|---|
| `TestBindResolvesAMethodByItsQualifiedName` | matched the bare `Name` again | a changed method bound to nothing at all |
| `TestBindTellsTwoMethodsOfOneNameApart` | the same | `LineItem.subtotal` and `RefundLine.subtotal` collapsed together |
| `TestBindResolvesANestedFunctionByItsBareName` | none needed; it is the negative half | also asserts a reconstructed `format_footer.pad` resolves to nothing |
| `TestBindStillResolvesAModuleLevelFunction` | none needed; it is the negative half | what worked before still works |
| `TestBindRecordsWhyASourceDidNotResolve` | dropped the recorded reason | an unresolved source with nothing to say about itself |
| `TestBindRefusesAnAmbiguousMatchAndTakesTheLineWhenItDecides` | restored the nearest-line tiebreak, and separately the bare-name match | a guess that always answers |
| `TestBindFallsBackToTheBareNameWithoutAQualifiedName` | none needed | a snapshot without the field still binds as before |
| `TestScenarioQualifiedNames` | the bare-name match | the ninth scenario's method did not bind |
| `TestEmptyDocketDistinguishesUnresolvedFromNoDependents` | restored the old two-branch wording and dropped the table's list | "no dependents were found" printed for a commit nothing was looked up for |
| `TestPacketNamesTheSymbolsTheGraphWasNeverAskedAbout` | dropped the packet section | the packet claimed a clean analysis |
| `TestAResolvedRunSaysNothingAboutUnresolvedSymbols` | none needed; it is the negative half | a clean run stays quiet |
| `TestUnresolvedReasonAlwaysSaysSomething` | none needed | a source with no recorded reason is still named |
| `TestModuleGranularityCountsWholeFileChanges` | made the predicate always false | a whole-file change reported as a finding of no dependents |

Six mutations, each applied alone and reverted, every one making at least one
test fail. Four tests are the negative half and passed under all six.

### What it did to the real report

`entire umbra 562e458` on the `pallets/click` clone, the commit the experiment
was written from. No ranking weight was touched in this pass.

```
before   3.  format_usage  src/click/core.py:1163  type consumer  depth 1  echo  5.3
after    1.  format_usage  src/click/core.py:1163  direct caller  depth 1  echo  8.0
```

The node the writeup judged "right node, wrong rank" is now first, and it is
described by the `CALLS` edge Graph always had rather than by the `PARAM_TYPE`
edge from its enclosing class. `Command.get_usage` likewise moves from
`type consumer` to `transitive caller`. The order changed because the edges are
now the real ones, not because anything was tuned.

On the whole-file commit, where nothing could be resolved:

```
before   no dependents were found for the changed symbols
after    note   1 changed entity(s) were reported only at whole-file granularity, so no symbol-level dependents could be asked for
         note   1 changed symbol(s) could not be looked up in the graph, so nothing was traversed for them; they are named below
         no dependents could be looked up: every changed symbol is listed as unresolved below
         1 changed symbol(s) were never looked up in the graph:
            src/click/formatting.py  src/click/formatting.py:1  no symbol in the snapshot of this file carries that name
```

### Deliberately not done in this pass

The echo demotion of the twelve `core.py` nodes and the absent
`UsageError.show` are both downstream of this bug and are now standing on
different edges than when they were judged. They are left exactly as they were,
to be re-judged against the corrected field rather than tuned around here.

## Open question, not investigated: does span containment drift?

Recorded from the second real-repo pass so it is not lost. **Do not treat this
as a finding; nothing here was measured.**

Phase 23 made source resolution depend on file, then qualified name, then span
containment, and containment is what decides between two symbols of one
qualified name in one file. The line it contains comes from `graph commit`,
which reports `after_start_line` and `before_start_line` for the change. The
spans it is contained by come from `graph snapshot`, which Umbra takes at the
head worktree.

Those two are consistent as long as the head snapshot and the head side of the
diff describe the same tree, and in every run so far they have. The case worth
checking eventually is the one where they might not: a changed entity for which
Graph reports only `before_start_line`, because the symbol was removed or
because the diff attributed it to the parent, while the snapshot is at head.
`ParseCommitJSON` already prefers `after_start_line` and falls back to
`before_start_line`, so a parent line number can reach a head span. With one
symbol of a name the fallback is harmless, since a single match binds without
consulting the line at all. With two symbols of one name in one file it decides
which one, and a parent-side line number in a file that moved could put the
change inside the wrong span, or inside neither, which now reports the source
as ambiguous rather than binding it.

What would settle it: a commit that both moves a symbol and leaves two symbols
of one qualified name in the file, run against a head snapshot, checking
whether the bound symbol is the intended one. `fixtures/app` has no such case
and neither did click.

## Phase 24: the class roll-up, and why it was not fixed

Pass 2 of the real-repo experiment ended on a question: should a changed class
be a source at all when the change is entirely inside one of its methods? On
`pallets/click`, ten of thirteen shadowed nodes were methods in `core.py` that
merely take a `HelpFormatter` parameter, pulled in because `graph commit`
reported the enclosing class as changed alongside the method that actually
changed.

The proposed rule was: **a changed container is not a source when every change
inside it falls within the span of a member that is itself already a source.**

**That rule cannot be evaluated from anything the installed tools publish, so
it was not implemented.** No code changed in this phase. What follows is the
measurement that settles it, so the next person does not have to repeat it.

### The probe

A repository with one class carrying two attributes, two methods and a base
class, committed once as a baseline, then twelve commits each making one kind
of change. It lives at `~/umbra-experiment/container-probe` and is not
committed here; the recipe is a `Widget(Base)` class with `limit`, `label`,
`render` and `measure`, plus a free function, which is enough to produce every
row below.

Everything is `entire graph commit <sha> --repo . --json`, unedited:

| Case | What changed | What `graph commit` reports |
|---|---|---|
| A | one method body | `body_changed class Widget` **and** `body_changed method Widget.render` |
| B | one class attribute value | `body_changed class Widget` alone |
| C | **a method body and a class attribute** | `body_changed class Widget` **and** `body_changed method Widget.render` |
| D | the base class list | `signature_changed class Widget`, with `old_signature` and `new_signature` |
| E | a method added | `body_changed class Widget` and `added method Widget.extra` |
| F | a method removed | `body_changed class Widget` and `removed method Widget.measure` |
| G | a class decorator added | `body_changed module base.py`, and **no class or member change at all** |
| H | a method signature | `body_changed class Widget` and `signature_changed method Widget.render` |
| I | the class docstring | `body_changed class Widget` alone |
| J | a method body and a **new** attribute | as A, with the method's `after_start_line` shifted by one |
| K | a method body and a **removed** attribute | as A, with the method's `after_start_line` shifted by minus one |

### The line that decides it

**A and C are byte-identical.** Sorted and serialised, the change list for "a
method body changed" and for "a method body changed and so did a class
attribute" is the same document:

```
[{"after_start_line":12,"before_start_line":12,"dependents_count":0,
  "kind":"class","name":"Widget","type":"body_changed"},
 {"after_start_line":18,"before_start_line":18,"dependents_count":1,
  "kind":"method","name":"Widget.render","type":"body_changed"}]
```

A is a pure roll-up and the class should be dropped. C carries a real
class-level change and the class must be kept. The output does not distinguish
them, so any rule that drops the class in A also drops it in C, and dropping it
in C loses every dependent of a genuine class-level edit. That is a silent
false negative in the direction this product exists to prevent.

### Why the line numbers do not rescue it

`before_start_line` and `after_start_line` are **start** lines only. There is no
end line, no changed-line range and no per-hunk detail: the six keys above are
the entire change record, and Umbra's `commitJSON` already parses all six.

The shifts in J and K look like a signal and are not. A shift means lines were
added or removed somewhere above the member in the file, which includes above
the class entirely and inside a different method, so it does not localise
anything. And the case that matters, C, changes a class attribute in place and
produces **no shift at all**, identical to A.

### Two other surfaces, checked rather than assumed

- **`entire graph diff --base X --head Y --json`** returns the same schema and
  is byte-identical for A and C as well. It is not a finer instrument; the
  help text calls it an alias for the same analysis.
- **The snapshot's `body_hash`** is the obvious remaining candidate, since a
  class-level hash that excluded member bodies would settle this immediately.
  It does not exclude them. Snapshotting the baseline and all three cases:

  | Symbol | A, method only | B, attribute only | C, both |
  |---|---|---|---|
  | `Widget` | **changed** | **changed** | **changed** |
  | `Widget.render` | changed | same | changed |
  | `Widget.measure` | same | same | same |

  The class hash moves whenever anything inside it moves, so it separates
  "something in this class changed" from "nothing did", and never separates a
  roll-up from a class-level edit.

### What would settle it, none of which is available today

One of: an end line or a changed-line range on each change record; a
member-exclusive body hash for a container; a `field` change record for a
changed class attribute, which `FoldFields` shows the provider does emit for
some kinds and did not emit for any case here; or Umbra reading and diffing the
two worktrees' source text itself, which means reconstructing what the provider
should publish and is the mistake phase 23 recorded as worth avoiding.

### The cases that must keep the class, for whenever this is revisited

Collected while probing, and useful regardless:

- **B and I keep it and are unambiguous.** A class-level change with no member
  change reported is decidable today: there is no member source to attribute
  the class change to, so the class stays.
- **D keeps it and is unambiguous.** A base class list change arrives as
  `signature_changed` on the class with both signatures, which no member
  roll-up produces.
- **E and F are roll-ups by construction** but adding or removing a member also
  changes the class's own shape, so a rule would have to decide deliberately
  rather than fall through.
- **G cannot be reached at all.** A class decorator produces a module-level
  change and no class entity, which is the module-granularity degradation
  already recorded as defect 3 in the experiment. Two separate decorator forms
  were tried and both behave this way.

### Where this leaves the field

The ten manufactured nodes on click are still there and are still wrong. What
this phase establishes is that they cannot be removed correctly with the
information the installed Graph provides, and that removing them incorrectly
would trade a visible over-report for an invisible under-report. Ranking
weights and the echo rule remain untouched and remain waiting on a correct
field, which is now blocked on the provider rather than on Umbra.
