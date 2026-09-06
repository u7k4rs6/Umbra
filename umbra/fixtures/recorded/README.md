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
| `leak` | A test four hops from the change that the selection missed, with its reason |
| `beacon` | A caller under a `SAFETY` comment inside an `except` block, pinned to the top |

## Full Runner recordings

`entire umbra record <ref> --out <dir>` performs a real analysis and stores
every Runner call with its output, so a replay can drive the whole pipeline
including the graph and verify calls.

`minimal/recording.json` is one, from a short real session: one full read, one
partial read, one search, one edit, one test file. It is 135 KB and was hand
reviewed against SECURITY_AND_ACCESS.md before commit. See `minimal/notes.txt`.

**The recording of the session that built Umbra is deliberately not committed.**
A full recording embeds the checkpoint transcript, and for this repository that
is the whole build session: about 900 KB of conversation. `umbra record`
applies the scrub, so absolute paths, the user and host name, the author name,
the email address and token shapes are all gone. But SECURITY_AND_ACCESS.md
requires a fixture recording to be **hand reviewed** before it is committed,
and prose of that length cannot honestly be reviewed line by line. Saying it
was reviewed would be the kind of claim this project exists to catch. The
minimal recording is twelve transcript records and was actually read end to
end, which is what the rule asks for.

Recording the same short session inside the Umbra repository produces 3.8 MB,
of which 3.4 MB is a single snapshot of the entire codebase. It was therefore
recorded against a repository holding only the fixture app.

To regenerate one locally:

```
entire umbra record <ref> --out umbra/fixtures/recorded/<name> \
    --scenario <name> --notes "what it shows" --test "pytest -v"
```
