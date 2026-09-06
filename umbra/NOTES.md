# Umbra: build notes

Working notes kept during the build. Written after phase 5, and extended as
later phases land. The design documents in `docs/` stay as written; anything
this file records that contradicts them is a finding from real output, and the
answered questions at the bottom of `docs/ARCHITECTURE.md` carry the detail.

## Where the build stands

Phases 0 through 5 are done and committed. `go test ./...` is green at every
commit. 158 top-level tests, 41 of them table driven subtests.

| Phase | What landed | Tests |
|---|---|---|
| 0 | Repository, docs, README pointer | 0 |
| 1 | Step 0 probe, eight questions answered, degradations written | 0 |
| 2 | Module, Runner, resolve, worktrees, command surface | 40 |
| 3 | Transcript adapter, mentions, timeline, session-said fallback | 67 |
| 4 | Snapshot loader, relation map, BFS, impact parser, sources | 89 |
| 5 | Classifier, call-site window, six-factor ranker, terminal table | 158 |

Phases 6 through 11 are not built. Nothing in the code pretends otherwise: the
table prints `probes  not run` and `sweep skipped` rather than inventing
results.

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
