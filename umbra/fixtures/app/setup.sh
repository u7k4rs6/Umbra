#!/usr/bin/env sh
# Create the fixture virtualenv and install pytest into it.
# Run from umbra/fixtures/app. Offline after the first run.
set -eu

here=$(cd "$(dirname "$0")" && pwd)
cd "$here"

if [ ! -d .venv ]; then
    python3 -m venv .venv
fi

./.venv/bin/python -m pip install --quiet --upgrade pip
./.venv/bin/python -m pip install --quiet -r requirements.txt

echo "fixture ready. run tests with:"
echo "  cd $here && ./.venv/bin/python -m pytest -v"
echo
echo "to run Umbra over this fixture, put the runner on PATH first:"
echo "  . $here/.venv/bin/activate"
echo "  entire umbra 0063443 --test \"pytest -v\""
echo
echo "use -v, not -q: quiet mode prints no per-test names, so entire graph"
echo "verify cannot name which test broke and falls back to a suite-level"
echo "pass or fail. a runner that is not on PATH fails the same way, because"
echo "the tests run in a detached worktree that has no .venv in it."
