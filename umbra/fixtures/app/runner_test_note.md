# Why the demo activates the virtualenv

Umbra runs the selected tests in a detached git worktree of the commit under
analysis, not in your working tree. Two consequences shape the runner you pass
to `--test`:

1. **The worktree has no `.venv`.** The virtualenv is not committed, so a
   runner given as `umbra/fixtures/app/.venv/bin/python` cannot be found from
   inside the worktree.
2. **A relative path is relative to the worktree.** Umbra starts the runner in
   the project directory inside the worktree, so a path written relative to the
   repository root does not resolve either.

A runner that cannot start produces no output, and `entire graph verify` has
nothing to parse. The verdict degrades to a suite-level pass or fail and no
test is named, which looks exactly like the `pytest -q` failure and has a
different cause. Umbra reports the degradation either way and says to add `-v`.

Either of these works:

```
. umbra/fixtures/app/.venv/bin/activate
entire umbra 0063443 --test "pytest -v"
```

```
entire umbra 0063443 --test "$PWD/umbra/fixtures/app/.venv/bin/python -m pytest -v"
```
