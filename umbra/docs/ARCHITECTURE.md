# Umbra: technical architecture

Umbra is a single static Go binary named `entire-umbra`, living in the fork of `entireio/entire-graph` as its own module under `umbra/`, and dispatched by the Entire CLI as `entire umbra` because any executable named `entire-<name>` on `$PATH` runs kubectl-style. It consumes documented output only: Graph's NDJSON snapshot and edges, `graph commit`, `graph impact`, `graph verify`, `graph def`, and the checkpoint transcript and stored summary through `entire checkpoint explain`. It never imports Graph internals and never reads the checkpoints branch directly. Four boundaries carry the design: a checkpoint reader (transcript adapter), a graph reader, a classifier plus ranker plus sweep forensics, and renderers. All external processes go through one `Runner` interface so tests replay recorded fixtures without Entire, git or an agent.

## Components

```
umbra/
  go.mod                         separate module; no import of entire-graph packages
  cmd/entire-umbra/main.go       flags, wiring, exit codes
  internal/runner/               Runner interface and replaying fake
  internal/checkpoint/           resolve id <-> commit; fetch transcript via entire CLI
  internal/transcript/           normalized Event model; claudecode adapter
  internal/graph/                snapshot loader, adjacency, BFS, impact and commit parsers, verify wrapper
  internal/shadow/               examined-set builder, classifier, ranker, test selector
  internal/report/               table, json, html (map + replay), markdown
  fixtures/app/                  Python service with four pytest files
  fixtures/recorded/<scenario>/  Runner recordings for replay tests
  site/                          landing page and drop-in viewer
  testdata/
```

## Data flow

```
entire umbra <ref> --test "pytest -q"
   │
   ├─ 1. Resolve      ref -> checkpoint id, commit sha, parent sha; worktrees head/base
   ├─ 2. Sources      entire graph checkpoint <id>  (fallback: graph commit <sha>)  -> changed entities
   ├─ 3. Field        entire graph snapshot --repo <head> --format ndjson -> symbols, files, relations
   │                  BFS from each source over CALLS(in), TYPE usage, DATA flow to --depth
   │                  entire graph impact --symbol S -> co-change files, cross-check counts
   ├─ 4. Exposure     entire checkpoint explain <id> --raw-transcript -> adapter -> []Event
   │                  examined set: per file, list of (seq, kind, range), incl. mentions in agent text
   │                  entire checkpoint explain <id> --short -> "session said" line
   ├─ 5. Classify     node state lit / penumbra / umbra / unknown, modifiers, six factors, beacon
   ├─ 6. Tests        candidate tests from snapshot; select; graph verify with baseline; parse
   │                  sweep (default): full suite once; leaks = changed-state tests not selected, with reasons
   └─ 7. Render       table; json; html (map, docket, replay); packet (markdown)
```

### 1. Resolve

Same contract as any checkpoint-aware plugin. From a commit-ish, parse the `Entire-Checkpoint: <12 hex>` trailer with `git log -1 --format=%B`. From an ID or prefix, `git log --all --grep="Entire-Checkpoint: <id>" --format=%H`. Prefer `entire checkpoint list --json` if it exposes the commit. Parent is `<sha>^`; merge commits use the first parent and say so in the header. Worktrees:

```
git worktree add --detach <data>/wt/<sha>/head <sha>
git worktree add --detach <data>/wt/<sha>/base <sha>^
```

`<data>` is `ENTIRE_PLUGIN_DATA_DIR` when set, else `$XDG_CACHE_HOME/umbra`. Removed on exit unless `--keep-worktrees`.

### 2. Sources

`entire graph checkpoint <id>` analyzes the commit behind an Entire checkpoint and is the preferred source, because it is the documented bridge between Graph and Checkpoints. Fallback: `entire graph commit <sha> --repo <head>`. Both list added, removed, renamed, signature-changed and body-changed entities with dependent counts. Every changed entity is a light source; kinds carry a weight for the header: signature-changed 1.0, removed 1.0, renamed 0.8, body-changed 0.6, added 0.3 (added symbols have no callers yet unless the change also wired them in).

```go
type Source struct {
    Symbol   string  // stable identity from Graph
    Name     string
    File     string
    Span     [2]int  // start, end line at head
    Change   string  // added, removed, renamed, signature, body
    Weight   float64
}
```

### 3. Field

Load `entire graph snapshot --repo <head> --format ndjson` once. The snapshot is documented to include files, external endpoints, symbols, relations and a summary. Build:

```go
type Symbol struct { ID, Name, Kind, File string; Span [2]int; IsTest bool }
type Edge   struct { From, To, Relation string }
type Field  struct {
    Symbols map[string]*Symbol
    In      map[string][]Edge   // edges arriving at a symbol
    Out     map[string][]Edge
    ByFile  map[string][]string // symbol IDs per file, sorted by span
}
```

Relation families used, by name as reported by `entire graph capabilities --json` on the installed version: calls, type usage (references or implements), data flow, imports for file-level neighbours. The exact relation names are read from `capabilities` at startup and mapped through a small table in `internal/graph/relations.go`; unknown relations are ignored and listed in the header.

BFS from each source over incoming call edges to `--depth` (default 2), plus one hop of type usage and data flow, produces the dependents. Each dependent records its shortest path to a source. Co-change files come from `entire graph impact --symbol S --repo <head>`; symbols in those files are added at depth "co-change" with the lowest weight. `impact` counts are also kept as a cross-check and printed if they disagree with the BFS by more than the depth explains.

`IsTest` is true when Graph marks the symbol as a test, or when the file matches the runner's convention (`test_*.py`, `*_test.py`, `*_test.go`, `*.test.*`, `tests/` directory).

### 4. Exposure

The transcript adapter is the same shape as in any Entire plugin that reads Claude Code JSONL: records with `type`, `timestamp`, `message.content[]` blocks of `text`, `tool_use` (`id`, `name`, `input`) and `tool_result` (`tool_use_id`, `content`). Normalized events:

```go
type Kind int // Prompt, AssistantText, Read, Grep, Glob, Edit, Command, ResultFile, Mention

type Event struct {
    Seq    int
    TS     time.Time
    Kind   Kind
    Path   string
    Range  *[2]int   // Read with offset/limit -> [offset, offset+limit); nil means whole file
    Paths  []string  // Grep/Glob hits from the result
    Cmd    string
}
```

Mapping: `Read` input `file_path`, `offset`, `limit` -> Read with Range; `Grep` and `Glob` -> Grep/Glob with `Paths` parsed from the result content (file paths at line starts); `Edit`, `Write`, `MultiEdit`, `NotebookEdit` -> Edit; `Bash` -> Command (used for the timeline only). Paths are made repository-relative using the `cwd` field when present. `ResultFile` covers tool results that quote a path with content (for example `git diff` output), counted as partial evidence. `Mention` covers repository paths and symbol names that appear in `AssistantText`, matched against the field's file list and symbol names (exact token match, case-sensitive for symbols); it is the weakest evidence and only ever produces penumbra with the `echo` modifier.

Examined set: `map[file][]Exposure{Seq, Kind, Range}`, with symbol-level mentions kept in `map[symbolID][]Exposure`. The change start is `T0 = min(Seq of Edit events whose Path is a source file)`, called the cut in the UI.

Session said: `entire checkpoint explain <id> --short` returns Entire's stored summary of the checkpoint. The first sentence, scrubbed and capped at 200 characters, is carried into the report header as the agent's own account. It is displayed, never verified; verifying it is a different product.

### 5. Classify

For each dependent node `n` in file `f` with span `[a, b]`:

- lit: exists a Read on `f` with Range nil or covering `[a, b]`, and `Seq >= T0`; or an Edit on `f` (the agent changed the file, so it saw it)
- penumbra, strongest tier wins and names the modifier: a Read on `f` not covering the span (`glance`); a Grep or Glob hit on `f` (`glimpse`); a ResultFile on `f` (`quoted`); a covering Read that only happened before `T0` (`afterimage`); a Mention of `f` or of the symbol with no tool event on `f` (`echo`)
- umbra: no exposure and no mention on `f`
- unknown: the transcript yielded zero Read, Grep, Glob, Edit, ResultFile and Mention events; then every node is unknown and the header says "examined set unavailable"

Two more signals come from the call site, not from the session. `graph impact` reports each caller with its call-site line (the documented text output shows `Router.ServeHTTP (mux.go:203, def :188)`, where 203 is the call site). Umbra reads a window of four lines on either side of that line from the head worktree and matches, per language, error-handling openers (`except`, `catch`, `finally`, `rescue`, `defer`, `if err != nil`, `try`) for the `fault line` modifier, and annotation markers (`SAFETY`, `CRITICAL`, `INVARIANT`, case-insensitive, inside a comment) for `beacon`. The window is matched, never copied into the report; with `--snippets` the single comment line is shown. `far field` is set when the node's top-level package or directory differs from its source's.

Score as defined in PRD.md (six factors, beacon override, opt-in scar), computed with the factors stored on the node so the report can print them:

```go
type Node struct {
    Symbol     *Symbol
    Sources    []string      // source IDs this node depends on
    Relation   string        // strongest relation to any source
    Depth      int
    Path       []string      // shortest path to the nearest source
    State      State         // Lit, Penumbra, Umbra, Unknown
    Modifiers  []string
    Exposures  []Exposure    // the evidence for the state
    Dependents int
    CallSite   int           // line of the call into the source, 0 when unknown
    Score      float64
    Factors    map[string]float64 // relation, dependents, state, farfield, faultline, test; scar when enabled
    Beacon     string        // the annotation comment line, when pinned
    Tests      []TestRef     // tests reaching this node
    Result     TestOutcome   // for test nodes: pass, fail, notrun
}
```

Scar (`--history`): `git log --since=6.months --format=%s -- <file>` in the head worktree, counting messages matching `(?i)\bfix|\bbug`; factor `1 + min(n, 5) / 5`. Read-only, opt-in, printed.

Illumination fraction for the header: `lit / (lit + penumbra + umbra)`, with unknown reported separately and the fraction omitted when everything is unknown.

### 6. Tests

Candidates: symbols with `IsTest` that have a path to any source over outgoing call edges within three hops, plus tests in co-change files. Selection order: shadowed tests, then tests reaching shadowed nodes, then the rest. `--run shadow` (default) runs the first two groups; `--run all` runs all candidates; `--run none` skips.

Test IDs: pytest as `path::function` (class-qualified when the symbol has a parent), go test as `-run '^Name$' ./pkg`. IDs are validated against `^[A-Za-z0-9_./:\[\]\-]+$` before use and passed as separate argv elements, never through a shell.

Execution:

```
entire graph verify --repo <base> --test "<runner> <ids>" --record-baseline <data>/baseline.json
entire graph verify --repo <head> --test "<runner> <ids>" --pre-edit-baseline <data>/baseline.json
```

`verify` reports which tests changed state. New failures attach to the test node and to every shadowed node on the test's path to a source.

The sweep runs by default (`--no-audit` skips it): the same pair with the bare runner (full suite), then `leaks = changed-state tests not in the selected set`. For each leak, forensics computes a reason from the field already in memory, checked in this order:

1. `path exists at depth N via <symbol>`: a BFS without the depth cap finds a call path from the test to a source; N and the first hop are named.
2. `no edge from the test file`: the test symbol has no outgoing CALLS edges at all, or none that leave the file (dynamic dispatch, fixtures, or a call Graph did not resolve).
3. `reaches only through <relation>`: a path exists over a relation family Umbra does not traverse (listed from `capabilities`).
4. `test file not in the co-change set`: no path and no co-change link; the test is simply outside everything Umbra can see.

The leak count and reasons go to the header, the docket (filter `leaks`), the packet and the JSON.

### 7. Render

- Table: header line with sources and the illumination fraction as text, then one row per shadowed node sorted by score: glyph, state, symbol, file:line, relation and depth, score, tests and outcome. Lit nodes are summarised as a count unless `--all`. Then the test execution block and the audit line.
- JSON: schema below, written to `--out/umbra.json`. The HTML embeds it; the landing page viewer reads it.
- HTML: single self-contained file rendered from `html/template` with inline CSS and JS; the map, docket, replay and detail panel are specified in FRONTEND_SPEC.md.
- Packet: `--out/umbra.packet.md`, the Blind Spot Packet, five sections a reviewer reads in a minute: (1) what changed (sources with change kind and span), (2) what was examined (exposure counts per file, the session's own sentence), (3) ranked shadow (top twenty nodes with glyph, state sentence, relation, factors), (4) tests reaching it (selected IDs), (5) results (new failures with excerpts, leaks with reasons). The header line and the reproduce command top and tail it. This is also the PR comment.

```json
{
  "umbra_version": "0.1.0",
  "checkpoint": {"id": "a1b2c3d4e5f6", "commit": "…", "parent": "…", "agent": "claude-code", "session_ids": ["…"]},
  "inputs": {"adapter": "claude-code", "depth": 2, "test_runner": "pytest -q", "run": "shadow", "audit": true, "history": false,
             "relations_used": ["CALLS", "REFERENCES_TYPE", "DATA_FLOW", "CO_CHANGE"], "relations_ignored": ["…"],
             "channels": {"reads": true, "mentions": true, "callsites": true, "graph": true}},
  "session_said": "Checked the callers of compute_total; all tests pass.",
  "sources": [{"id": "…", "name": "compute_total", "file": "app/service.py", "span": [40, 61], "change": "signature", "weight": 1.0}],
  "nodes": [
    {"id": "…", "name": "test_rounding", "file": "tests/test_service.py", "span": [12, 20], "is_test": true,
     "sources": ["…"], "relation": "CALLS", "depth": 1, "path": ["…", "…"],
     "state": "umbra", "modifiers": ["test", "far field"], "exposures": [], "call_site": 14,
     "dependents": 0, "score": 9.0, "factors": {"relation": 3, "dependents": 1.0, "state": 1.0, "farfield": 1.5, "faultline": 1.0, "test": 3},
     "beacon": "",
     "tests": [], "result": "fail", "failure_excerpt": "assert total == 10.5"}
  ],
  "timeline": [
    {"seq": 3, "ts": "…", "kind": "read", "path": "app/service.py", "range": null},
    {"seq": 9, "ts": "…", "kind": "grep", "paths": ["app/api.py"]},
    {"seq": 11, "ts": "…", "kind": "mention", "paths": ["app/refunds.py"], "symbols": ["apply_refund"]},
    {"seq": 14, "ts": "…", "kind": "edit", "path": "app/service.py"},
    {"seq": 22, "ts": "…", "kind": "command", "cmd": "pytest tests/test_api.py"}
  ],
  "t0": 14,
  "execution": {"selected": ["tests/test_service.py::test_rounding", "…"], "new_failures": ["…"], "fixed": [],
                "sweep": true, "leaks": [{"test": "tests/test_report.py::test_receipt_total", "reason": "path exists at depth 4 via render_lines"}]},
  "summary": {"lit": 3, "penumbra": 1, "umbra": 3, "unknown": 0, "illumination": 0.43},
  "limitations": ["…"],
  "commands_run": ["…"]
}
```

Exit codes: 0 completed; 2 `--fail-on` met; 1 runtime error.

## The Runner boundary and recorded fixtures

```go
type Runner interface {
    Run(ctx context.Context, name string, args []string, stdin []byte) (stdout, stderr []byte, exit int, err error)
}
```

`umbra record <ref> --out fixtures/recorded/<scenario>/` runs a real analysis and stores every Runner call. The replaying fake serves by name and args. Scenarios from `fixtures/app` (Python: `app/service.py`, `app/api.py`, `app/refunds.py`, `app/models.py`, tests `test_api.py`, `test_service.py`, `test_refunds.py`, `test_models.py`):

1. `everything-lit`: the session read every caller after the change; all lit; nothing to run beyond the shadow set, which is empty.
2. `classic`: signature change on `compute_total`, callers in `api.py` (read), `refunds.py` (grep hit only), `test_service.py` (never opened); two shadowed tests fail.
3. `partial-read`: `Read` with offset and limit that misses the caller's span; penumbra.
4. `pre-change`: caller read before T0, then signature changed; penumbra with `afterimage`.
5. `no-reads`: transcript with no tool events and no mentions; every node unknown; selection by impact only; the sweep reports leaks.
6. `echo`: the agent names `apply_refund` in its text and never opens `refunds.py`; penumbra with `echo`.
7. `leak`: a test four hops away changes state; not selected; the sweep reports it with `path exists at depth 4 via render_lines`.
8. `beacon`: a caller sits under a `# SAFETY:` comment inside an `except` block; pinned to rank 1 with `fault line`.

Unit tests: adapter parsing including offset, limit and mentions; snapshot loading and BFS; classifier truth table (one test per row of the state definition, one per penumbra tier); call-site window matching per language; ranker determinism with all six factors and the beacon override; test ID construction and validation; verify output parsing; sweep forensics reasons in order. End-to-end test runs only when `entire` and `entire-graph` are on `$PATH`.

## Step 0: the go/no-go probe

1. In the fork with Entire enabled and Graph installed, run a short agent session on `fixtures/app`: read `app/service.py` fully, read `app/api.py` with an offset and limit, grep for `compute_total`, edit `app/service.py`, run `pytest tests/test_api.py`, commit.
2. `entire checkpoint explain <id> --raw-transcript > /tmp/t.jsonl`. Confirm `tool_use` blocks for Read carry `file_path`, `offset`, `limit`; Grep results list paths; Edit carries `file_path`; timestamps exist.
3. `entire graph capabilities --json`: note relation names. `entire graph snapshot --repo . --format ndjson | head`: note record shapes for symbols and relations, and whether test symbols are marked.
4. `entire graph checkpoint <id>` and `entire graph commit HEAD --repo .`: note whether signature changes are distinguished from body changes for Python.
5. `entire graph verify --repo . --test "pytest -q tests/test_api.py::test_health"`: note the output shape.
6. `entire graph impact --symbol compute_total --repo .`: confirm callers carry a call-site line, not only a definition line.
7. `entire checkpoint explain <id> --short`: confirm a stored summary exists without `--generate`; if it does not, the "session said" line falls back to the last assistant sentence before the commit, or is omitted.

Degradations:

| Finding | Effect | Where absorbed |
|---|---|---|
| Read has no offset/limit in inputs | Range always nil; penumbra collapses into lit for full reads; partial-read scenario dropped | Adapter |
| Grep results do not list paths | Grep counts as no evidence; penumbra from grep disappears | Adapter |
| No read events but mentions exist | Nodes with mentions are penumbra (`echo`), the rest umbra; header notes the evidence is text only | Classifier, header |
| No read events and no mentions | All nodes unknown; selection by impact; the sweep is the demo | Classifier, header |
| `impact` has no call-site lines | `fault line` and `beacon` disabled; header notes it; five factors instead of six | Call-site window |
| No stored summary | "session said" omitted or taken from the last assistant sentence | Header |
| Snapshot lacks test marking | `IsTest` from file conventions only | Field loader |
| Relation names differ | Mapping table updated; unknown relations listed | `relations.go` |
| No signature/body distinction | All changes weighted as body (0.6); header says so | Sources |
| `graph checkpoint` unavailable | `graph commit <sha>` | Sources |
| `verify` output not parseable | Fall back to running the runner directly through Runner and parsing pytest or go test summaries | Tests |

## Constraint-change playbook

- Another agent's transcript: new adapter; if it carries no tool records, unknown state and impact-only selection, stated in the header.
- Whole session instead of one checkpoint: union of exposures across the session's checkpoints; sources from each commit; one map per commit, one docket for the session.
- Machine consumer: JSON and exit codes exist from the first commit; the packet is the PR surface.
- No test execution allowed: `--run none`; the map and docket stand alone.
- Another language: any language Graph parses; test execution supports pytest and go test, others get selection without execution and say so.
- Depth or scale constraint (large repo): `--depth 1`, node cap in the HTML with clustering, snapshot loaded once.
- Change spans several commits: `--session` loop.
- Must connect to Checkpoint intent (prompts): the docket gains a column "asked for in a prompt" by name matching, no model. Kept out of the core so this entry and Impeach stay distinct.

## Build order

Phases, each ending in a commit that is a valid checkpoint, message `umbra: phase N <summary> (tests: <count>)`.

1. Docs and probe; findings written at the bottom of this file.
2. Skeleton: module, flags, Runner, resolve by trailer, worktrees.
3. Adapter with offset and limit, grep paths, edits, mentions, timeline; the "session said" line.
4. Field: snapshot loader, relations mapping, BFS, impact parser with call-site lines, sources from `graph checkpoint` with `commit` fallback.
5. Classifier with all penumbra tiers, call-site window, six-factor ranker with beacon; table output. Scenarios `classic`, `partial-read`, `echo`, `beacon` pass. Stable checkpoint.
6. Test selection and execution through `verify` with baseline; the sweep with forensics; `--history`. Scenarios `no-reads` and `leak` pass.
7. JSON, the packet, `umbra record`, replay tests for all eight scenarios.
8. HTML: map, docket, detail panel. Stable checkpoint.
9. HTML: attention replay, keyboard navigation, spotlight, light and dark, print.
10. Landing page with the drop-in viewer and the embedded sample; README with install, reproduce, limitations, disclosure.
11. Final semantic-diff review: `entire graph diff --base <first commit> --head HEAD`, verified against source and `go test ./...`, written into the README. Run Umbra on the checkpoint of the session that built it and ship that report.

## Reconstruction checklist

A fresh session, given only `entire checkpoint explain <latest>`, `entire graph commit <latest>` and this `docs/` directory, must be able to state: the user and problem; the four boundaries and which owns the current phase; which scenarios pass; what is unbuilt; the probe findings and degradations in effect. Commit messages carry the phase and test count so `entire checkpoint list` reads as a build log.

## Open questions

Fill in after Step 0.

- Do Read tool inputs in the stored transcript include `offset` and `limit`?
- Do Grep and Glob results list file paths in a parseable form?
- What relation names does the installed Graph report in `capabilities`, and does the snapshot mark test symbols?
- Does `graph checkpoint` exist in the installed plugin, and does it distinguish signature from body changes for Python?
- Does `graph verify` accept a test command with pytest node IDs and report per-test state changes?
- Does `graph impact` report call-site lines for callers in the installed version?
- Does `checkpoint explain --short` return a stored summary without `--generate`?
- Is `ENTIRE_PLUGIN_DATA_DIR` set for unmanaged plugins?
