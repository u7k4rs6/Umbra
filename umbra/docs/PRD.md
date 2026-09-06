# Umbra: product requirements

Umbra is a plugin for the Entire CLI, run as `entire umbra <checkpoint>`, that finds the code an AI agent's change affects but the agent never looked at. It treats each changed symbol as a light source, takes the dependents Entire Graph knows about (callers, type consumers, data flows, co-change files, and the tests that reach them), and subtracts what the session actually examined according to the checkpoint's tool activity and the agent's own words. What remains in shadow is ranked, the tests that reach it are run, and the full suite is swept once afterwards so the selection audits itself. The report is an eclipse map you can replay in the order the agent looked at things, plus a one-page packet a reviewer reads in a minute. The core is deterministic and offline. There is no model anywhere in the product. Nothing in the stack costs money.

## The problem

Agents edit a function without reading its callers. The diff is clean, the description is confident, and the break is in a file the session never opened. Reviewers cannot see this from the diff, because the diff shows what changed, not what the agent failed to consider. Entire Graph knows the dependents. The checkpoint knows which files the session read, with line ranges and order, and what the agent said while doing it. Nobody puts the two together.

The everyday case: a signature change on `compute_total`, three callers, one of them a test in `tests/test_service.py` that the session never opened, committed with a summary that says "checked the callers, all tests pass" because only `tests/test_api.py` ran.

## Users

- Primary: the reviewer of an agent-authored change, who wants to know where to look first.
- Secondary: the developer who ran the session, before pushing.
- Tertiary (continuation): the agent itself, as a pre-commit gate that says "you changed X and never read its callers".

## What Umbra does

1. Resolves a checkpoint, checks out its commit and parent into temporary worktrees, and asks Graph for the changed entities and their dependents.
2. Builds the examined set from every signal the checkpoint carries: `Read` calls with their line ranges, Grep and Glob hits, edits, file content quoted in tool results, and paths or symbol names that appear in the agent's own text. Classifies every dependent as lit, penumbra, umbra or unknown.
3. Ranks the shadowed dependents with six printed factors, selects the tests whose call chain reaches them, runs exactly those with a baseline from the parent commit, then sweeps the full suite once and explains every test the selection missed.
4. Writes a terminal table, a JSON report, a self-contained HTML eclipse map with attention replay, and the Blind Spot Packet as markdown.

## States

| State | Glyph | Definition |
|---|---|---|
| Lit | ● | The session read the file with a range covering the symbol's span, at or after the change began; or edited the file |
| Penumbra | ◐ | Partial evidence, strongest first: a `Read` whose range does not cover the symbol (glance); a Grep or Glob hit on the file (glimpse); file content quoted in a tool result; a full read that happened only before the change began (afterimage); the path or symbol name mentioned in the agent's own text with no tool event at all (echo) |
| Umbra | ○ | Nothing in the session touched or mentioned the file |
| Unknown | ? | The transcript carries no tool events and no mentions; the state cannot be computed |

Modifiers shown alongside the state: `afterimage`, `glance`, `glimpse`, `echo`, `test`, `transitive`, `co-change only`, `far field` (crosses a package boundary), `fault line` (call site inside error handling), `beacon` (annotated call site), `scar` (history factor, opt-in). Every modifier carries its plain meaning next to it on the page; the JSON uses plain names.

## Ranking

Score per shadowed node, deterministic, capped at six factors, each printed in the report so a reader can recompute it:

1. relation weight: direct caller 3, data flow 2, type consumer 2, transitive caller 1.5, co-change file 1
2. multiplied by `1 + log2(1 + dependents of the node)`
3. multiplied by state weight: umbra 1.0, penumbra 0.5 (afterimage 0.6, echo 0.7, since those are weaker evidence than a glance)
4. multiplied by 1.5 when the node lives in a different top-level package or directory than its source (far field)
5. multiplied by 1.5 when the call site sits inside error handling: `except`, `catch`, `finally`, `rescue`, `defer`, `if err != nil`, within four lines (fault line)
6. plus 3 if the node is a test function, plus 2 if any test reaches the node

Override: a call site within four lines of a `SAFETY`, `CRITICAL` or `INVARIANT` comment is pinned to the top of the docket regardless of score (beacon), with the comment line shown.

Opt-in factor `--history`: multiplied by `1 + min(fixes, 5) / 5` where `fixes` is the number of commits in the last six months touching the file whose message matches `fix` or `bug` (scar). Off by default because it is noisy; printed like every other factor when on.

Ties break by file path, then symbol name, so two runs on the same checkpoint produce the same order.

## Test selection, execution, and the sweep

Candidate tests are test functions in the Graph snapshot with a CALLS path to a changed symbol within three hops, or living in a file that Graph marks as co-changing with a changed file. Selection order: tests that are themselves shadowed, then tests reaching shadowed nodes, then the rest. Execution goes through `entire graph verify` in the head worktree with a baseline recorded in the parent worktree, so pre-existing failures are not attributed to the change.

The sweep runs by default: after the selected tests, the full suite runs once, and every test that changed state but was not selected is reported as a leak with the reason the selection missed it: `no call path within 3 hops`, `path exists at depth 4 via <symbol>`, `no edge from the test file (dynamic dispatch or unparsed call)`, `test file not in the co-change set`. The count is printed even when it is zero. `--no-audit` skips the sweep and the header says so.

## Command surface (v1)

```
entire umbra <checkpoint-id | commit-ish> [flags]

  --test CMD          test runner prefix, e.g. "pytest -q"; test IDs are appended
  --run none|shadow|all   which selected tests to run; default shadow
  --no-audit          skip the full-suite sweep (on by default)
  --history           add the scar factor from git history (off by default)
  --depth 1|2|3       dependency hops; default 2
  --format table|json|html|packet
  --out PATH          write json, html and the packet
  --fail-on umbra|penumbra|failure|leak   non-zero exit when the condition holds
  --adapter auto|claude-code
  --snippets          include signature lines for nodes in the HTML (off by default)
```

Exit codes: 0 completed; 2 the `--fail-on` condition was met; 1 runtime error.

## The 90-second demo

1. `entire umbra a1b2c3d4e5f6 --test "pytest -q"` on the fixture app.
2. Terminal: the header quotes the session's own summary, `checked the callers, all tests pass`, then `compute_total` (signature changed) lights 7 dependents: 3 lit, 1 penumbra, 3 umbra. Top of the docket: `tests/test_service.py::test_rounding`, umbra, direct caller, test.
3. Umbra runs the four tests reaching shadowed nodes. Two fail, both in the never-opened test file. The sweep adds zero leaks.
4. Open the HTML. The session's sentence sits above the map; below it the eclipse: the source at the centre, lit callers glowing, three umbra nodes in a pool of shadow, two of them cracked. Press play: reads light nodes in the order they happened; at the cut the afterimage dims; the last edit ripples along the edges; the map settles on the eclipse.
5. Click a shadowed node: the relation path, the sentence `nothing in the session touched or mentioned tests/test_service.py`, the test IDs and the failing assertion, the six factors.
6. Closing beats: a real, unseeded example from one of the builder's past sessions imported with `entire import`, then Umbra run on the checkpoint of the session that built Umbra.

## Scope

In v1:

- Claude Code transcripts (tool calls with inputs, results and order; `Read` offset and limit honoured; mentions from assistant text).
- Relation families: CALLS in (direct and transitive), type usage, data flow, co-change; whatever the installed Graph exposes as edges.
- Six-factor ranking with the beacon override and the opt-in scar factor.
- Selection, execution through `verify`, the default sweep with leak reasons.
- Terminal, JSON, HTML with replay, the Blind Spot Packet (what changed, what was examined, ranked shadow with state sentences, tests reaching it, results and leaks).
- Python and Go fixtures; any language Graph parses semantically works for the map; test execution works for pytest and go test.
- Recorded fixtures and Go unit tests for classification, ranking, selection, sweep forensics and parsers; one end-to-end test that runs only when Entire and Graph are present.
- Landing page that renders any `umbra.json` dropped onto it, client-side, with real examples from imported past sessions.

Deliberately out of v1, with the reason:

- Adapters for other agents' transcripts (they get the unknown state, which is the correct answer for Cursor, whose transcripts carry no tool records).
- Subagent transcripts.
- Live mode during a session and a blind-spot budget gate (shadow-branch data is not documented output; the hook route is a second project).
- Mutability analysis of callees (nothing in the documented surface provides it).
- Coverage deltas between the selected run and the sweep (needs coverage tooling in the target repository).
- Trend dashboards across sessions (a third surface with no data on day one).
- Any fix, edit or suggestion. Umbra shows the shadow; it does not write code.
- Cross-repository dependents.

## Success criteria, mapped to judging

| Judging criterion | What Umbra shows |
|---|---|
| Defined user | The reviewer of agent changes, first line of the README |
| Real problem | Unexamined dependents; the fixture reproduces the everyday case; imported past sessions show it unseeded |
| Scope that finishes | One adapter, one algorithm, one map, one packet; unbuilt list stated with reasons |
| Inspectable core | Every state prints the exposure or its absence; every score prints six factors; every leak prints why; the replay is the raw event sequence |
| Disclosure | Prior-work and AI sections in README and this document |
| Continuation path | Pre-commit gate for agents, other adapters, cross-repo via Graph's cross-service links, coverage deltas |
| Entire is essential | Dependents come only from Graph; the examined set comes only from the checkpoint; the product is the set difference and exists nowhere else |

## Limitations to disclose

- Read evidence is file and line-range based. Mentions in the agent's text are attention, not reading, and are ranked as the weakest penumbra tier.
- Graph edges are not complete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined. The sweep's leak reasons are how a reader sees the gap.
- Fault lines and beacons are detected by a regex window around the call-site line, not by a parser.
- Ranking weights are hand-set and printed; they are a heuristic, not a measurement.
- The fixture is seeded to produce lit, penumbra and umbra nodes and two failing shadowed tests; the report says so. Imported past sessions are not seeded.
- One transcript adapter.

## Prior work and AI disclosure

- No code reused. The idea of subtracting examined files from a blast radius was developed in planning for this event.
- All product code is written during the build with an AI coding agent inside the fork, captured in Entire checkpoints.
- The four planning documents were written before the build with AI assistance and are the first commit.

## Continuation path

1. `entire umbra --gate` as a pre-commit check the agent runs on itself, with a blind-spot budget (top shadowed callers must be examined or dismissed with a reason), once live session data is documented output.
2. Adapters for Codex and Gemini transcripts.
3. Coverage deltas between the selected run and the sweep.
4. Cross-repository shadows through Graph's cross-service links.
5. Feeding the packet into `entire review --prompt` so reviewer agents start in the shadow.
6. Intent-aware review: pair the sentence "I checked the callers" with Umbra's evidence that only 2 of 4 were opened, without a model.

## Cost

Entire CLI and Graph are open source and local. Go, Python, pytest and GitHub Pages are free. Umbra makes no network calls and no model calls.
