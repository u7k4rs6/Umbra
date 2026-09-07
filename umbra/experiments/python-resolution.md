# Is it click, or is it Graph's Python resolution?

Two findings came out of the `pallets/click` experiment and both were stated
about click without knowing whether they were about click at all:

- **A.** 79 percent of the calls leaving `tests/` attached to external `click.*`
  nodes rather than to the definitions in the same repository, so no test
  reached the change and probe selection selected one test out of 2,059.
- **B.** `self.ctx.get_usage()` produced no edge, which is why
  `UsageError.show` was unreachable at any depth.

**Verdict: both are Graph's Python resolution, not click.** A is caused by the
shape of the call expression, not by the repository, the layout or the package
name, and a three-file controlled probe reproduces it from nothing. B is
absolute: 27 attribute-chain call sites across three repositories produced 27
non-edges, while 31 of 36 matched `self.method()` control calls resolved.

No product code was changed, no weight was touched and nothing was fixed.

## The three repositories

Chosen to vary one thing at a time against click. Click is a library with a
`src/` layout whose own tests import it by name and call through the package
object.

| | `pallets/click` | `encode/httpx` | `pre-commit/pre-commit` |
|---|---|---|---|
| Commit | `562e458` | `b5addb6` | `a9bba55` |
| License | BSD-3-Clause | BSD-3-Clause | MIT |
| Kind | library | library | **application**, a CLI tool |
| Layout | **`src/click/`** | flat `httpx/` | flat `pre_commit/` |
| Test import style | `import click` then `click.X()` | `import httpx` then `httpx.X()` | **`from pre_commit.x import y`** then `y()` |
| Source lines / files | 12,727 / 17 | 8,827 / 23 | 7,163 / 67 |
| Test lines / files | 15,240 / 58 | 8,926 / 37 | 11,062 / 59 |
| Snapshot | 158 files, 3,310 symbols, 9,582 relations | 111 files, 2,228 symbols, 6,757 relations | 190 files, 1,519 symbols, 7,307 relations |
| Snapshot time | 0.12s warm | 1.61s cold | 1.63s cold |

**httpx varies the layout and holds the import style.** Same attribute-style
test calls as click, flat package instead of `src/`. If the layout were the
cause, httpx would come out clean.

**pre-commit varies the import style and holds the layout.** Direct symbol
imports, and it is an application rather than a library, so nobody imports it
by name the way click's own tests import click.

No Umbra analysis was run on either. No session, no checkpoint, no probes. The
snapshot answers both questions on its own.

## Finding A: where do calls leaving the tests attach?

### The query, stated so it can be rerun

```
entire graph snapshot --repo <repo> --format ndjson > snap.ndjson

symbols   = {r["id"]: r for r in snap if r["record_type"] == "symbol"}
externals = {r["id"]    for r in snap if r["record_type"] == "external"}

for r in snap where r["record_type"] == "relation" and r["type"] == "CALLS":
    origin = symbols.get(r["from_id"])        # skip file-level edges
    if origin is None: continue
    bucket = "test" if origin["file_path"].startswith("tests/") else "source"
    dest   = "same-repo" if r["to_id"] in symbols else "external"
```

Counting `CALLS` edges only, which is what the click number counted. The script
is `findingA.py` in the run log; it is twenty lines and the block above is the
whole of it.

### Raw counts

| Repo | Origin | Same-repo | External | Total | External |
|---|---|---|---|---|---|
| click | test | 400 | **1,522** | 1,922 | **79.2%** |
| click | source | 420 | 352 | 772 | 45.6% |
| httpx | test | 260 | **965** | 1,225 | **78.8%** |
| httpx | source | 266 | 109 | 375 | 29.1% |
| pre-commit | test | **1,271** | 586 | 1,857 | **31.6%** |
| pre-commit | source | 431 | 443 | 874 | 50.7% |

The click row reproduces the experiment's number exactly, 1,522 external, which
is the check that this query is the same query.

**httpx, with a flat layout, comes out at 78.8 percent against click's 79.2.**
The `src/` layout is not the cause.

**pre-commit comes out at 31.6 percent**, and it is the only one of the three
whose tests do not call through a package or module object.

### The refinement that removes the confound

A raw external ratio is confounded: a call to `os.path.join` is external and
legitimately so. What matters is how much of a repository's **own** code its
tests fail to attach to. Splitting the external destinations by whether the
first dotted segment of the external node's `value` is the repository's own
top-level package:

| Repo | Origin | Same-repo | External, **own package** | External, foreign | Own-package leak |
|---|---|---|---|---|---|
| click | test | 400 | **1,313** | 209 | **68.3%** |
| click | source | 420 | 54 | 298 | 7.0% |
| httpx | test | 260 | **750** | 215 | **61.2%** |
| httpx | source | 266 | **0** | 109 | **0.0%** |
| pre-commit | test | 1,271 | 292 | 294 | 15.7% |
| pre-commit | source | 431 | 160 | 283 | 18.3% |

This is the real shape of it. Click and httpx leak 61 to 68 percent of their
own-package calls out of the repository **from the tests**, while leaking 7 and
0 percent **from the source**. The asymmetry is enormous, and it is the same in
a `src/` layout and a flat one.

pre-commit has no such asymmetry: 15.7 percent from tests against 18.3 percent
from source. Its tests are no worse than its own code.

Sample own-package externals, which are the pathology made concrete:

```
click       click.Argument, click.BadParameter, click.Choice, click.Command,
            click.Context, click.DateTime, click.File, click.FileError
httpx       httpx.ASGITransport, httpx.AsyncClient, httpx.BasicAuth,
            httpx.Client, httpx.Cookies, httpx.DigestAuth, httpx.HTTPTransport
```

Every one of those is a class defined a few directories away in the same
checkout.

### Graph's own label agrees

`CALLS` edges in the snapshot carry a `resolution` field. Its distribution says
the same thing in the provider's own words:

| Repo | `import_resolved` | `import_external` | `exact` | `type_inferred` | `name_only` |
|---|---|---|---|---|---|
| click | 296 | **1,874** | 201 | 287 | 48 |
| httpx | 40 | **1,074** | 111 | 162 | 10 |
| pre-commit | **1,059** | 1,029 | 569 | 64 | 31 |

pre-commit resolves 1,059 imports to in-repo definitions. httpx resolves 40.

### The controlled probe

Correlation across three repositories is suggestive. This is the isolation. One
git repository, one package, three test files, identical semantics, nothing
else different:

```python
# pkg/core.py
def helper(a): return a + 1
class Widget:
    def render(self): return "w"

# tests/test_bare.py
from pkg.core import helper
from pkg.core import Widget
def test_bare_name_call():
    assert helper(1) == 2
    assert Widget().render() == "w"

# tests/test_module_attr.py
from pkg import core
def test_module_attribute_call():
    assert core.helper(1) == 2
    assert core.Widget().render() == "w"

# tests/test_package_attr.py
import pkg
def test_package_attribute_call():
    assert pkg.helper(1) == 2
    assert pkg.Widget().render() == "w"
```

What the snapshot produces:

| File | Call written as | Edge |
|---|---|---|
| `test_bare.py` | `helper(1)` | **CALLS SAME-REPO** `helper` (`pkg/core.py`) |
| `test_bare.py` | `Widget()` | **CONSTRUCTS SAME-REPO** `Widget` (`pkg/core.py`) |
| `test_module_attr.py` | `core.helper(1)` | CALLS **external** `pkg.helper` |
| `test_module_attr.py` | `core.Widget()` | CALLS **external** `pkg.Widget` |
| `test_package_attr.py` | `pkg.helper(1)` | CALLS **external** `pkg.helper` |
| `test_package_attr.py` | `pkg.Widget()` | CALLS **external** `pkg.Widget` |
| all three | `Widget().render()` | **CALLS SAME-REPO** `Widget.render` |

The same function, in the same repository, imported statically and explicitly
in every case. **A call written as a bare name resolves. The identical call
written as `module.name` or `package.name` does not.** Note also that the
relation type degrades: the bare form gives `CONSTRUCTS` to the class, the
dotted form gives `CALLS` to an external node.

`Widget().render()` resolves everywhere, which shows the resolver does follow a
method call on an instance it can type. It is specifically the module or
package object it will not look through.

### It is not even a per-repository property

pre-commit's residual 15.7 percent is the same bug in a repository that mostly
avoids it. `tests/error_handler_test.py` uses both styles in one file:

```python
from pre_commit import error_handler          # module import
from pre_commit.errors import FatalError      # symbol import
...
error_handler._log_and_exit(...)              # 8 edges to external pre_commit.error_handler
```

`_log_and_exit` is defined in-repo at `pre_commit/error_handler.py`, and the
call still attaches to `external:symbol:pre_commit._log_and_exit`. One file, two
import styles, two outcomes.

### Verdict on A

**Graph's Python resolution.** The trigger is the call expression shape:
a call through a module or package object does not resolve to the definition in
the same repository, however the import was written. Click is affected badly
because its tests are written that way throughout, which is the ordinary style
for a library testing its own public surface. Any project whose tests exercise
the public API through the package object gets the same result, and httpx
proves that includes projects with no `src/` directory.

The consequence for Umbra is unchanged and now generalised: on a project of
this shape, the test suite is largely invisible to the graph, so probe
selection has almost nothing to select from and the sweep is doing all the
work.

## Finding B: does `self.x.y()` produce an edge?

### Method, and one correction to it

The first attempt indexed `CALLS` edges by the `start_line` of their
`call_site` evidence and looked up each call site by its own line. That
returned "no edge" for everything including the control, which is wrong, and
`internal/graph/snapshot.go` already says why: snapshot evidence spans the
**calling function**, not the call expression, and only `graph impact` refines
it to the exact line.

The corrected method:

1. Build spans for every snapshot symbol.
2. For a call site at `file:line`, the enclosing symbol is the smallest span
   containing that line.
3. The call resolved if that enclosing symbol has an outgoing `CALLS` or
   `CONSTRUCTS` edge whose target's bare name equals the called method name.

Sound in the direction that matters: if no out-edge of the enclosing function
carries the called name, Graph produced no edge for that call. Two call sites
of one name inside one function collapse into one, which is why one duplicate
shows as "no edge" in the click control below.

Samples are twelve per repository, drawn with a fixed seed from every match of
`self\.(\w+)\.(\w+)\s*\(` in the non-test source, with a matched control drawn
from `self\.(\w+)\s*\(` on lines carrying no chain.

### The chain sample: 27 sites, 27 non-edges

| Repo | Site | Expression | Edge |
|---|---|---|---|
| click | `src/click/exceptions.py:106` | `self.ctx.get_usage()` | **none** |
| click | `src/click/core.py:961` | `self._parameter_source.get()` | none |
| click | `src/click/core.py:3572` | `self.name.upper()` | none |
| click | `src/click/_compat.py:138` | `self._stream.write()` | none |
| click | `src/click/_compat.py:148` | `self._stream.tell()` | none |
| click | `src/click/_compat.py:469` | `self._f.close()` | none |
| click | `src/click/_termui_impl.py:154` | `self.file.flush()` | none |
| click | `src/click/_termui_impl.py:405` | `self._stream.write()` | none |
| click | `src/click/_winconsole.py:216` | `self.buffer.isatty()` | none |
| click | `src/click/parser.py:149` | `self.prefixes.add()` | none |
| click | `src/click/testing.py:103` | `self._tmpfile.fileno()` | none |
| click | `src/click/testing.py:131` | `self.copy_to.flush()` | none |
| httpx | `httpx/_client.py:1022` | `self.cookies.extract_cookies()` | none |
| httpx | `httpx/_client.py:1301` | `self._transport.__exit__()` | none |
| httpx | `httpx/_client.py:765` | `self._mounts.items()` | none |
| httpx | `httpx/_client.py:1271` | `self._mounts.values()` | none |
| httpx | `httpx/_client.py:1986` | `self._mounts.values()` | none |
| httpx | `httpx/_decoders.py:253` | `self._buffer.seek()` | none |
| httpx | `httpx/_decoders.py:263` | `self._buffer.truncate()` | none |
| httpx | `httpx/_decoders.py:288` | `self._buffer.seek()` | none |
| httpx | `httpx/_models.py:576` | `self.headers.setdefault()` | none |
| httpx | `httpx/_models.py:972` | `self.stream.close()` | none |
| httpx | `httpx/_models.py:1208` | `self.jar.set_cookie()` | none |
| httpx | `httpx/_transports/default.py:372` | `self._pool.__aexit__()` | none |
| pre-commit | `tests/conftest.py:211` | `self._stream.getvalue()` | none |
| pre-commit | `tests/conftest.py:212` | `self._stream.seek()` | none |
| pre-commit | `tests/conftest.py:213` | `self._stream.truncate()` | none |

pre-commit has **three** chain sites in the entire repository and all three are
in one test helper, so its whole sample is listed. click has 81 in its source
and httpx 108.

### The control: 36 sites, 31 resolved

| Repo | Resolved | No edge | Example |
|---|---|---|---|
| click | **10 / 12** | 2 | `self.format_message()` to `ClickException.format_message`, `type_inferred/0.82` |
| httpx | **11 / 12** | 1 | `self._send_handling_auth()` to `AsyncClient._send_handling_auth`, `type_inferred/0.9` |
| pre-commit | **10 / 12** | 2 | `self.exclusive_lock()` to `Store.exclusive_lock`, `type_inferred/0.9` |

The five control misses are explainable and none is an attribute chain:
`self.formatter_class()` and `self.check_fn()` are attributes holding a
callable rather than methods of the class, `self.app()` likewise, and the
`self.make_metavar()` and `self._replace()` misses are the duplicate-name
collapse described in the method.

Every resolved control edge carries `resolution: type_inferred` at confidence
0.82 to 0.9. **Graph infers the type of `self` and resolves a method on it. It
does not infer the type of `self.<attribute>`, so the second hop of a chain has
nothing to resolve against.**

### Verdict on B

**Graph's Python resolution, and it is absolute rather than statistical.** Not
one attribute-chain call in three repositories produced an edge, while the
control resolved 86 percent of the time in the same files, in the same
functions, with the same `self`. It is not click, not the layout, not the
import style, and not a matter of confidence thresholds: the edge is not
produced at all.

`UsageError.show` being unreachable at any depth in the click experiment was
therefore never a depth problem and never a click problem. It is what happens
to any Python class that reaches a collaborator through an attribute, which is
most of them.

## What predicts each finding

The two findings have different predictors and neither is the repository.

- **A is predicted by the call expression shape in the tests.** A test suite
  that calls `pkg.Thing()` loses its edges into the source; one that calls
  `Thing()` after `from pkg.mod import Thing` keeps them. Layout, package name,
  library versus application and repository size are all irrelevant, and the
  three-file probe reproduces it with none of them present.
- **B is predicted by nothing.** It fires on every attribute chain in every
  repository measured. The only way a codebase avoids it is by not writing
  attribute chains, which is what pre-commit does by being written mostly as
  free functions, and that is a property of its style rather than a mitigation
  anyone chose.

For Umbra this means the field will be thin on object-oriented Python
regardless of the project, and thin in the tests specifically for any library
whose tests drive its public surface through the package object. Neither is
something Umbra can fix in its own code; both are the provider's resolver.

## One thing found on the way, recorded not acted on

Pass 1 of the click experiment reported that "this Graph build carries no
`evidence`, `confidence` or `verified` field anywhere in the document", and
therefore that no dependent could come back marked heuristic or needing
verification. That statement was about `entire graph impact --format json` and
it is still true there.

**The snapshot is a different matter.** Its `CALLS` records carry `confidence`,
`reason`, `relation_scope`, `resolution` and `target_kind`, for example:

```json
{"record_type":"relation","type":"CALLS","confidence":0.92,
 "reason":"direct call expression resolved to same-file symbol",
 "relation_scope":"file","resolution":"exact","target_kind":"symbol",
 "evidence":[{"kind":"call_site", ...}]}
```

`internal/graph/snapshot.go` parses `record_type`, ids, kind, name, file path,
line span, signature, language and the evidence's call-site line, and discards
all five of those fields. So the evidence tier the first pass looked for and
did not find exists, on the edges rather than on the impact query, and Umbra
throws it away at load. That is a finding about Umbra, not about Graph, and it
is recorded here rather than acted on because this pass changes no product
code.

## Reproducing

```sh
git clone https://github.com/encode/httpx ~/umbra-experiment/httpx          # b5addb6
git clone https://github.com/pre-commit/pre-commit ~/umbra-experiment/pre-commit  # a9bba55

for d in click httpx pre-commit; do
  entire graph snapshot --repo ~/umbra-experiment/$d --format ndjson > $d.ndjson
done
```

Then the query in the finding A section over each `.ndjson`, and the enclosing
span method in the finding B section. The controlled probe is the three test
files quoted above in a fresh git repository, committed once.

Tooling: Entire CLI 0.10.5, entire-graph
`v0.4.1-nightly.202609030616.ddcebd05`, Umbra at `de4dc60`, Linux.
