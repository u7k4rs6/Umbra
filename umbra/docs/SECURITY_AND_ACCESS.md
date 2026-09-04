# Umbra: security and access

Umbra reads agent transcripts, loads a full symbol graph of the repository, checks out historical commits, runs selected tests, and produces an HTML map that people will share. The sensitive surfaces are: transcript text (may hold secrets despite redaction), source structure and optional code snippets (leave the machine inside the report), test execution (runs as the user), and a landing-page viewer that accepts files. Entire runs CLI plugins as native executables with the user's permissions, filtering the environment but not sandboxing, so Umbra is held to the same standard as the user's own shell. The default path is offline, contains no model, and executes only test IDs that Umbra constructed and validated, with a runner the user named.

## Assets and trust boundaries

| Asset | Where it lives | Exposure |
|---|---|---|
| Transcript and tool activity | Checkpoints branch, read through the Entire CLI | Repository readers; public on a public fork |
| Symbol graph of the repository | Graph NDJSON in memory; never written to `--out` except as the node list the report needs | Report readers see names, files, spans and relations of nodes in the blast radius only |
| Source at head and parent | Temporary worktrees under the plugin data directory | Local user |
| Test execution | `entire graph verify` in the head and base worktrees | Local user; outcomes and short excerpts reach the report |
| Reports | `--out`; wherever the user pastes them | Whoever receives them |
| Files dropped on the landing-page viewer | The reader's browser memory only | Nobody else; nothing is uploaded |

Umbra trusts Graph and CLI output as data, never as instructions. It trusts nothing inside a transcript. It trusts flags and a committed `.umbra.json`.

## Threat model

| Threat | Impact | Control |
|---|---|---|
| Secret in a transcript passes redaction | Leaks into a report | Umbra quotes at most one transcript-derived sentence: the "session said" line, taken from Entire's stored checkpoint summary (or the last assistant sentence as fallback), scrubbed for token shapes and capped at 200 characters. Everything else in the timeline is kinds, sequence numbers, timestamps, paths, symbol names and, for commands, the first 120 characters of the command line after the same scrub. Mentions record which paths and symbols appeared in the agent's text, never the sentences. Tool result content is used to detect paths and is then discarded |
| Code snippets reveal more than the reader should see | Signature lines in shared reports | `--snippets` is off by default; when on, only the declaration line of each node (from `graph def`) and, for beacons, the single annotation comment line are included, each capped at 200 characters, with the same scrub applied. The call-site windows read for the `fault line` and `beacon` checks are matched in memory and never written out |
| Test IDs built from graph data reach a shell | Injection through a crafted symbol or path name | Test IDs must match `^[A-Za-z0-9_./:\[\]\-]+$`; anything else is dropped and listed under "not runnable"; IDs are passed as separate argv elements to `verify`; no `sh -c` in Umbra |
| The runner prefix is destructive or slow | Data loss or hang | The runner comes only from `--test` or a committed `.umbra.json`; Umbra echoes the full command before running; default timeout 10 minutes per invocation, and the default sweep (full suite) has its own 20 minute timeout after which the report says the sweep was cut short; everything runs in detached worktrees, never in the user's working tree |
| Symbol names or paths flow into HTML | Stored XSS in the map | `html/template` contextual escaping for all values; embedded JSON in a `<script type="application/json">` block with `</` and `<!--` escaped; a `Content-Security-Policy` meta forbidding external scripts, styles, images and connections; SVG text nodes built with `textContent`, never `innerHTML` |
| Drop-in viewer on the landing page parses hostile JSON | Script execution in a reader's browser | The viewer renders only known fields into DOM via `textContent` and attribute setters; unknown fields are ignored; JSON above 20 MB is refused; no `eval`, no `innerHTML`, same CSP as the report |
| Worktrees leak or fill the disk | Stale analysis, disk pressure | Keyed by sha under `ENTIRE_PLUGIN_DATA_DIR` or `$XDG_CACHE_HOME/umbra`; removed on exit unless `--keep-worktrees`; `umbra clean` removes all |
| Plugin binary swapped on `$PATH` | Arbitrary code as the user | Release checksums with each tag; README instructs installing from the reviewed release; the binary prints version and commit on every run |
| Public fork exposes session data | Prompts and transcripts public | Documented before the first push; fixture recordings scrubbed of absolute paths, user and host names; no real secrets ever in fixtures |

## Data handling rules

1. No transcript text in any output, with one named exception: the "session said" sentence, scrubbed and capped. Everything else is paths, symbol names, kinds, ranges, sequence numbers, timestamps and scrubbed command prefixes.
2. Graph data in reports is limited to nodes in the blast radius and their paths to sources; the full snapshot never leaves memory.
3. Test failure excerpts are capped at 300 characters, scrubbed, and taken from the runner's summary lines.
4. `umbra record` writes fixtures to the path the user names with the scrub applied, and is the only command that persists transcript-derived data.
5. Every report names the checkpoint, commit, parent and the exact commands run, so the reader can reproduce any line without trusting the file.

## Execution policy

- Commands Umbra runs: `git` (log, worktree, rev-parse; with `--history`, `git log --since --format=%s -- <file>`, read-only), `entire` (checkpoint, graph), and the user's runner through `entire graph verify`, twice by default: the selected tests and the sweep.
- Argv arrays through `Runner` everywhere. The runner prefix is split with a shell-words parser once, at flag parsing, and never re-joined into a shell string by Umbra.
- No test runs without `--test` or a committed `.umbra.json`; `--run none` skips execution entirely and the map still renders.
- Timeouts: 10 minutes for a `verify` invocation, 60 seconds for any single Graph call, 30 seconds for snapshot loading before a size warning is printed.

## Network policy

None in the default path. Graph is documented as local-only. `entire checkpoint explain` may fetch checkpoint metadata missing locally from the checkpoint remote; that is Entire's behaviour and the README states it. The landing page makes no requests beyond its own files; the viewer works from `file://`.

## Repository and mirror policy

- Regular branches carry code, docs, fixtures and the site. The checkpoints branch carries the session record and travels with pushes to the elected checkpoint sync remote. Shadow branches are local and never pushed.
- Before the first push: `entire status` to confirm the checkpoint remote; read one checkpoint transcript for anything that must not be public.
- Fixture recordings are scrubbed with `umbra record --scrub` and hand-reviewed before commit.
- The mirror on Entire follows the fork's collaborator permissions.

## Supply chain

- Go modules pinned in `go.mod` and `go.sum`; no cgo; `-trimpath`; static binary.
- No runtime downloads. HTML template, CSS and JS are embedded with `embed.FS`.
- Releases: tag, build for linux and darwin, publish `SHA256SUMS`. Install via `entire plugin install` from a local path, or the unmanaged path (`chmod +x`, place on `$PATH`).

## Access model

- Runs as the invoking user; asks for nothing more.
- Sees exactly what the user can see through git and the Entire CLI. If a checkpoint cannot be resolved locally, Umbra says so and stops rather than reaching for credentials.
- Managed installs receive `ENTIRE_PLUGIN_DATA_DIR`; Umbra uses it only for worktrees and baselines.
- `entire-umbra` is a CLI plugin discovered by the `$PATH` lookup; it is not an `entire-agent-*` integration and does not participate in the agent protocol.

## Residual risks, stated in the README

- Scrubbing of command prefixes and failure excerpts is pattern-based.
- A destructive `--test` runner does what it says; the echo before execution is the only guard.
- Node names, file paths and spans in a shared report reveal repository structure inside the blast radius; that is the product, and the README says to share reports with the same care as a diff.
- On a public fork, the checkpoints branch is public; a property of Entire's default storage.
