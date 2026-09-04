# Not seeded

A real session from another project of the builder's, imported into Entire with
`entire import claude-code` and analysed with `entire umbra 0941dad --run none`.
Nothing about it was arranged.

The project is a Python one. In the commit under analysis the session changed
`block_width` in `certify/rmsnorm_trace.py` and `main` in
`certify/rmsnorm_artifact.py`. Four symbols depend on those and **all four came
back umbra**: `summarize_repeat` and `analyse` in the same file it was editing,
`per_token_widths` in `certify/sched_trace.py`, and `collect` in
`certify/rmsnorm_artifact.py`.

That is not a session that ignored its files. The coverage line reads 40 file
reads and 122 edits against 1156 shell commands, so it was working with the
file tools throughout. It read plenty. It did not read these four.

Tests were not run: `--run none`, because that project's test runner is not set
up on this machine, so the map shows the shadow without probes.

## How it was produced without touching that project

The other repository was cloned to a scratch directory and everything was done
in the clone. Nothing was added to the original: no Entire hooks, no settings,
no commits. The session transcript was relocated onto the clone's path the same
way the minimal recording was, so the tool events line up with the checkout;
every event, range, result and timestamp is the real one.

Scrubbed with the same scrubber as every other artifact and checked by the
tests in `internal/report/artifacts_test.go`: no absolute paths, no user, host
or author name, no email address, no token shapes.

The commands captured in the timeline are that session's own, quoted as they
ran and capped. Five of them contain an em dash, because that session was
itself searching its repository for em dashes. Editing them would falsify the
record, so they stand.
