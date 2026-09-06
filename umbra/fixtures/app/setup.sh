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
echo "  cd $here && ./.venv/bin/python -m pytest -q"
