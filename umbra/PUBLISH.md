# Publishing Umbra

## Pre-push checks

Run these checks from the repository root before publishing:

```sh
cd umbra
go test ./...
go vet ./...
go run ./internal/report/gen-site --root .
git diff --check
```

## Checkpoint privacy

Checkpoint transcripts can contain developer prompts, local paths, and tool
output. Keep checkpoint storage private unless every transcript has been
reviewed for publication. The public repository contains only the scrubbed
reports required to inspect Umbra's decisions.

## Demo smoke test

```sh
cd umbra/fixtures/app
./setup.sh
cd ../..
. umbra/fixtures/app/.venv/bin/activate
entire umbra 1c2cf29 --test "pytest -v" --out ./umbra-out
```

Open `umbra-out/umbra.html` to inspect the graph evidence, source spans,
ranked review targets, selected tests, and verification result.
