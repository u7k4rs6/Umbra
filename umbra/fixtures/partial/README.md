# The partial-analysis fixture

A small Python project built to contain analysis the graph cannot complete. It
is the ninth scenario's graph side, captured in
`internal/graph/testdata/partial-snapshot.ndjson`.

Everything recorded here was measured from real provider output, not assumed.

## What is in it, and what the provider says about each

| File | What it is for | What the provider reported |
|---|---|---|
| `pricing.py` | The changed module. `compute_total` gains a parameter. | `CALLS exact 0.92` from `line_total`, and `exact` to `round_money` |
| `billing.py` | The chain the reader has to check | `RateCard.apply -> compute_total` is `import_resolved 0.86`; `quote -> RateCard.apply` is `type_inferred 0.8`, reason "method call resolved via chained constructor type"; `invoice -> quote` is `exact 0.92` |
| `dispatch.py` | Dynamic dispatch three ways: a table, `getattr`, `importlib` | **Not one edge to `compute_total`.** The parser sees `handler(*args)` and `fn(*args)` and draws nothing |
| `test_dispatch.py` | The tests | Four of them reach `compute_total` only at runtime and are absent from its field entirely; `test_invoice_through_the_rate_card` reaches it through the inferred hop |
| `legacy_rates.cob` | An inventory-only language | `language_tiers` says `COBOL: inventory-only`. One file, one symbol, **zero relations** |
| `rate_table.min.json` | One line over the 5000 byte minified threshold | `partial_failures` gains `E_MINIFIED`, "file record emitted but symbol parsing skipped". JSON is also inventory-only |

Snapshot stats for the fixture: `5 of 6 files parsed, 19 symbols, 61 relations,
1 partial failure, completeness_level ok`.

## Two things measuring settled that reading would not have

**An inventory-only file does not on its own produce a partial failure.**
`legacy_rates.cob` produces a file record, one symbol and no relations, and
`partial_failures` stays empty for it. The non-empty `partial_failures` this
fixture needs comes from the minified JSON, which is a policy skip the provider
reports explicitly. Both belong in the fixture and they are not the same thing.

**The provider's healthy completeness level is `ok`, not `complete`.** The
levels it emits are `ok`, `degraded` and `unsafe`. Umbra had been written to
treat anything other than `complete` as a degradation, which would have put an
incompleteness warning on every healthy snapshot: the crying-wolf failure that
makes an honesty feature worthless. It was caught by running the provider
against this fixture, and `graph.CompletenessIsHealthy` is the fix.

## Running the fixture's own tests

```
cd umbra/fixtures/partial && python3 -m pytest -q
```

Five tests, all passing. No dependencies beyond the standard library.

## Recapturing the snapshot

The snapshot is taken from a throwaway repository holding only this directory,
so its summary record describes this fixture and nothing else:

```
R=$HOME/.cache/umbra-partial-fixture
rm -rf "$R" && mkdir -p "$R" && cp -r umbra/fixtures/partial/. "$R"/
git -C "$R" init -q . && git -C "$R" add -A && git -C "$R" commit -q -m fixture
entire graph snapshot --repo "$R" --format ndjson
```

The committed capture is that output verbatim, except that `repo_root` and
`repo_key` carry the absolute path of the throwaway repository and are replaced
with `/repo` and `gh/example/partial`. That is the scrub every artifact in this
project goes through, not an edit to what the tool reported.
