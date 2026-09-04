# Recorded scenarios

Each directory is one scenario: a transcript the replay tests drive the
classifier, ranker and selector from, and a note saying what it shows. The
graph side of every scenario comes from the snapshot captured in
`internal/graph/testdata/snapshot.ndjson`, which is real output from the
fixture app with absolute paths and the repository name scrubbed out.

No test here starts a process, reaches the network or needs an agent.

| Scenario | What it shows |
|---|---|
| `everything-lit` | Every caller read in full after the change began, so nothing is in shadow |
| `classic` | The everyday case: one caller read, one only searched, one never opened |
| `partial-read` | A read whose offset and limit stop before the caller's span, which is the glance tier |
| `pre-change` | A caller read in full, but only before the cut, which is the afterimage tier |
| `no-reads` | No tool activity at all, so every node is unknown rather than judged |
| `echo` | The agent names a symbol it never opened, which is the weakest tier |
| `leak` | A test the sweep found that the selection missed, with its reason |
| `beacon` | A caller under a `SAFETY` comment inside an `except` block, pinned to the top |

## Full Runner recordings

`entire umbra record <ref> --out <dir>` performs a real analysis and stores
every Runner call with its output, so a replay can drive the whole pipeline
including the graph and verify calls. Those recordings are not committed here.

A full recording embeds the checkpoint transcript, which for this repository is
the session that built Umbra: about 900 KB of conversation. `umbra record`
applies the scrub, so absolute paths, the user name and token shapes are gone,
but SECURITY_AND_ACCESS.md requires a fixture recording to be hand reviewed
before it is committed, and prose of that length cannot honestly be reviewed
line by line. The scenario transcripts above are authored instead: they are
small, they were written to exercise one rule each, and they contain no real
conversation.

To regenerate one locally:

```
entire umbra record <ref> --out umbra/fixtures/recorded/<name> \
    --scenario <name> --notes "what it shows" --test "pytest -v"
```
