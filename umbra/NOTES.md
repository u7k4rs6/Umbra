# Umbra: build notes

Working notes kept during the build. Written after phase 5, and extended as
later phases land. The design documents in `docs/` stay as written; anything
this file records that contradicts them is a finding from real output, and the
answered questions at the bottom of `docs/ARCHITECTURE.md` carry the detail.

## Where the build stands

All eleven phases are done and committed. `go test ./...` is green at every
commit, and a fresh clone was checked to build and test green after phase 11
found that it would not have.

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

- `fixtures/app` has four modules and four pytest files as the kickoff
  specifies. The `leak` scenario in ARCHITECTURE.md needs a call chain four
  hops long through `render_lines` in `app/report.py`, which does not exist
  yet; it is added in phase 7 when that scenario is recorded.
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

### Where the build deviates from the letter of the plan

- **The README is at the repository root, not a one-line pointer.** That rule
  protects a host project's README in a fork. This repository is Umbra itself,
  so there is no host project to leave alone and the root README is Umbra's.
- **`--test-root` was added to the command surface.** PRD.md does not list it.
  A runner has to start in the project directory, and the fixture app is nested
  at `umbra/fixtures/app`. It is found automatically from the project files near
  the tests; the flag only exists for when that search picks wrong.
- **Full Runner recordings are not committed.** `umbra record` works and was run
  for real, but a full recording embeds the checkpoint transcript, which for
  this repository is about 900 KB of the session that built Umbra.
  SECURITY_AND_ACCESS.md requires a fixture recording to be hand reviewed before
  commit, and prose of that length cannot honestly be reviewed line by line. The
  eight scenario transcripts are authored instead. The reason is in
  `fixtures/recorded/README.md`.
- **The docket merges the file path into the symbol cell** rather than giving it
  its own column. The column list in FRONTEND_SPEC.md has six; the wireframe in
  the same document shows the path under the symbol, and at 38 percent of the
  page six columns broke symbol names mid-word. The information is the same.
- **`umbra.js` is about 1200 lines and `umbra.css` about 450**, against targets
  of 700 and 400. The targets were not treated as hard limits. The excess is the
  replay's state reconstruction and the day-scheme and print blocks.
- **The `leak` scenario does not use a four-hop chain through `app/report.py`.**
  The kickoff specifies four app modules and four test files, and the fixture has
  exactly those. The leak forensics are tested at depth four with a synthetic
  chain in `sweep_test.go` instead, which exercises the same code path.
