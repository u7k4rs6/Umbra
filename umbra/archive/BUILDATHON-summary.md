# Umbra — Track 2: Build with Graph Intelligence

## What it decides

Umbra turns an agent-authored change into an impact-aware code-review workflow:
it identifies changed definitions, follows their dependents through Entire
Graph, subtracts the code the checkpoint shows the agent examined, and ranks
the remaining shadow as review targets. It selects the tests that reach those
targets, runs them, and then runs a full-suite sweep to audit that selection.

The decision is actionable: review the ranked `umbra` nodes first and run the
tests attached to them before accepting the change.

## Graph evidence and verification

Each report preserves the evidence behind its recommendation:

- Changed definitions with source file and line span.
- Traversed graph relations, including callers, data-flow dependents, type
  consumers, imports, co-change files, and test reachability.
- Checkpoint-derived evidence of reads, edits, searches, and agent statements.
- Source locations, selected test identifiers, individual test results, and a
  full-suite leak check.

The HTML report is a replayable eclipse map; `umbra.packet.md` is the concise
review handoff; `umbra.json` is the inspectable machine-readable record.

## Run

Requirements: the Entire CLI with Checkpoints enabled, the Entire Graph plugin,
Go, and the test runner for the target repository.

```sh
go install github.com/u7k4rs6/Umbra/umbra/cmd/entire-umbra@main
entire umbra <checkpoint-id-or-commit> --test "pytest -v" --out ./umbra-out
open ./umbra-out/umbra.html
```

For the included seeded fixture:

```sh
cd umbra/fixtures/app && ./setup.sh && cd ../..
. umbra/fixtures/app/.venv/bin/activate
entire umbra 1c2cf29 --test "pytest -v" --out ./umbra-out
```

## Evidence artifacts

- `umbra/site/index.html` and `umbra/site/map.html`: static, inspectable demo.
- `umbra/site/sample/umbra.json`: seeded fixture report used by the demo.
- `umbra/site/self/umbra.html`: Umbra analyzing one of its own mapped commits.
- `umbra/fixtures/app`: source and tests for reproducing the seeded failure.

## Limits

Umbra is intentionally conservative: unavailable transcript evidence is shown
as `unknown`, not inferred. Graph relations that are unavailable or not
traversed are listed in the report. A selected-test pass is not treated as
proof; the full-suite sweep reports any missed failures as leaks.
