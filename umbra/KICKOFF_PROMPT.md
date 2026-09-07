# Kickoff prompt

This is the brief Umbra was built from, recorded here because the build never
kept it as a file. It is reconstructed from the session record rather than
copied from an existing document, and it is placed in the first commit so the
intent is in the repository before any code is.

## The problem

Find the code an AI agent's change affects but the agent never looked at.

A clean diff and a confident description tell a reviewer what changed. They do
not tell the reviewer what the agent never opened. Umbra takes the dependents of
every changed symbol from Entire Graph, subtracts what the session actually
examined according to the checkpoint's tool activity and the agent's own words,
ranks what is left in shadow, runs the tests that reach it, and then sweeps the
whole suite once so the selection audits itself.

## Settled decisions, not open questions

- Go, one static binary named `entire-umbra`, dispatched by the Entire CLI
  kubectl style. Its own Go module under `umbra/`.
- Inputs come only through documented Entire and Graph output. No scraping of
  internal state, no undocumented files.
- Four boundaries behind interfaces, and exactly one Runner for external
  processes, so tests replay recorded fixtures.
- **No model anywhere in Umbra.** The core is deterministic and offline.
- Four states: lit, penumbra, umbra, unknown, with five penumbra tiers.
- Missing data is a state, never an error. A transcript with no tool records
  gives every node `unknown`, which is the correct answer rather than a guess.
- **Umbra never executes commands from a transcript.** Test ids are validated
  against a strict pattern and passed as separate argv elements. There is no
  `sh -c` anywhere in it.
- Layout is computed in Go and written into the report JSON. The browser never
  computes positions, only styles and states.
- Vanilla JavaScript and CSS for the report and the site. No libraries, no
  fonts fetched, no network requests, no motion on load in the report.
- No em dashes in any file, code comments included.
- Tests replay recorded fixtures. No test may need a live agent or a network.
- `go test ./...` green at every commit. Commit after every phase, never squash.
- Touch nothing outside `umbra/` except a one line pointer in the repository
  README.
- Run `entire graph impact` before changing a file the build did not create.
- Keep the machine light: one agent session, no parallel builds, no background
  processes left running.

## What the reviewer is meant to get

An eclipse map that can be replayed in the order the agent looked at things,
plus a one page packet a reviewer reads in a minute, plus an exit code a CI job
can act on.
