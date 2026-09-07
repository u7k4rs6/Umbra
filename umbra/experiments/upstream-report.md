# Draft bug reports for the entire-graph maintainers

Five findings, written as five separate issues because they have different
causes and could be fixed independently. Nothing here has been filed. This is a
draft for review.

Context, once, so it does not need repeating in each issue. We build a tool
that takes the dependents of a changed symbol from `entire graph` and ranks
them, so we are a consumer of `snapshot`, `commit` and `impact` rather than of
the CLI's own output. All five findings came out of running against real
repositories and were then reduced to minimal reproductions that do not involve
our code. Every reproduction below was re-run against the build named in each
issue on the day this was written.

**How we pinned the version.** `entire-graph --version` prints `dev`, and the
snapshot header reports `"provider_version":"dev"`, so we took the module
version from the binary's Go build info instead:

```
$ go version -m $(command -v entire-graph)
    mod  github.com/entireio/entire-graph
         v0.4.1-nightly.202609050613.0a448c50.0.20260905152923-3a2a715fad19
    dep  github.com/smacker/go-tree-sitter v0.0.0-20240827094217-dd81d9e9be82
```

That may be worth a one-line fix on its own, but it is not one of the five
below.

Where we think we may have the wrong end of something, we say so in the issue.

---

## Issue 1: calls through a module or package object do not resolve to same-repo definitions

**Summary.** A call written as `mod.func()` or `pkg.func()` resolves to an
external node even when the target is defined in the same repository and the
import is static and explicit, while the identical call written as a bare name
resolves correctly.

**Version.** entire-graph
`v0.4.1-nightly.202609050613.0a448c50.0.20260905152923-3a2a715fad19`, Entire
CLI 0.10.5, Linux x86_64, Python 3.14.4 sources.

### Minimal reproduction

Three test files, one package, identical semantics. No `src/` layout, no test
framework, nothing but the call expression differing.

```
pkg/__init__.py     from pkg.core import helper as helper
                    from pkg.core import Widget as Widget

pkg/core.py         def helper(a):
                        return a + 1

                    class Widget:
                        def render(self):
                            return "w"

tests/test_bare.py          from pkg.core import helper
                            from pkg.core import Widget
                            def test_bare_name_call():
                                assert helper(1) == 2
                                assert Widget().render() == "w"

tests/test_module_attr.py   from pkg import core
                            def test_module_attribute_call():
                                assert core.helper(1) == 2
                                assert core.Widget().render() == "w"

tests/test_package_attr.py  import pkg
                            def test_package_attribute_call():
                                assert pkg.helper(1) == 2
                                assert pkg.Widget().render() == "w"
```

```sh
git init && git add -A && git commit -m probe
entire graph snapshot --repo . --format ndjson
```

### Observed

| File | Call written as | Edge produced | `resolution` |
|---|---|---|---|
| `test_bare.py` | `helper(1)` | CALLS to same-repo `helper` | `import_resolved` |
| `test_bare.py` | `Widget()` | **CONSTRUCTS** to same-repo `Widget` | `import_resolved` |
| `test_module_attr.py` | `core.helper(1)` | CALLS to **external** `pkg.helper` | `import_external` |
| `test_module_attr.py` | `core.Widget()` | **CALLS** to **external** `pkg.Widget` | `import_external` |
| `test_package_attr.py` | `pkg.helper(1)` | CALLS to **external** `pkg.helper` | `import_external` |
| `test_package_attr.py` | `pkg.Widget()` | **CALLS** to **external** `pkg.Widget` | `import_external` |
| all three | `Widget().render()` | CALLS to same-repo `Widget.render` | `type_inferred` |

Two things are lost, not one. The **target** becomes an external node, and the
**relation type** degrades: constructing a class through the package object
produces a `CALLS` to an external rather than a `CONSTRUCTS` to the class.

That `Widget().render()` resolves in all three files suggests the resolver is
willing to follow a method call on a value it can type, and that it is
specifically the module or package object it does not look through.

### Expected

`core.helper(1)` and `pkg.helper(1)` to produce the same edge as `helper(1)`,
since `from pkg import core` and `import pkg` are both statically resolvable to
a module in the repository, and `pkg/__init__.py` re-exports `helper`
explicitly.

### Supporting evidence at scale

The same measurement over three real repositories, counting `CALLS` edges only:

```
symbols   = {r["id"]: r for r in snapshot if r["record_type"] == "symbol"}
externals = {r["id"]    for r in snapshot if r["record_type"] == "external"}

for r in snapshot where record_type == "relation" and type == "CALLS":
    origin = symbols.get(r["from_id"])          # skip file-level edges
    bucket = "test" if origin["file_path"].startswith("tests/") else "source"
    dest   = "same-repo" if r["to_id"] in symbols else "external"
```

| Repo | Origin | Same-repo | External | External % |
|---|---|---|---|---|
| `pallets/click` `562e458` | test | 400 | 1,522 | 79.2% |
| | source | 420 | 352 | 45.6% |
| `encode/httpx` `b5addb6` | test | 260 | 965 | 78.8% |
| | source | 266 | 109 | 29.1% |
| `pre-commit/pre-commit` `a9bba55` | test | 1,271 | 586 | 31.6% |
| | source | 431 | 443 | 50.7% |

Splitting the external destinations by whether the first dotted segment names
the repository's own top-level package, which removes the stdlib and
third-party confound:

| Repo | Origin | Same-repo | External, own package | External, foreign |
|---|---|---|---|---|
| click | test | 400 | **1,313** | 209 |
| click | source | 420 | 54 | 298 |
| httpx | test | 260 | **750** | 215 |
| httpx | source | 266 | **0** | 109 |
| pre-commit | test | 1,271 | 292 | 294 |
| pre-commit | source | 431 | 160 | 283 |

The **source and test split rules out a test-specific cause**. click and httpx
lose 61 to 68 percent of their own-package calls from tests and 7 and 0 percent
from source, and the difference between the two halves is only the call style:
their tests exercise the public surface through the package object, their
internals use relative imports and bare names. pre-commit, whose tests import
symbols directly, shows no such asymmetry, 15.7 percent against 18.3.

Two of these are `src/` layouts and one is not, so layout is not involved.
Sample own-package externals: `click.Command`, `click.Context`,
`httpx.AsyncClient`, `httpx.BasicAuth`.

**Impact.** For any library whose tests drive its public API through the
package object, which is the ordinary style, the test suite is largely
disconnected from the source in the graph, so test-reachability queries and
anything built on them see almost nothing.

**See also Issue 5.** The counts above took a workaround to produce. Because a
call you could not resolve is reported as `external`, the same as a call to the
standard library, we had to split the external destinations by string-matching
the repository's own package name against the external node's `value`. Without
Issue 5 a consumer cannot size this bug at all.

---

## Issue 2: attribute chains such as `self.x.y()` produce no edge

**Summary.** A call on an attribute of `self` produces no relation at all, even
when the attribute's type is inferable from an edge the same snapshot already
contains.

**Version.** As Issue 1.

### Minimal reproduction

Two classes, one file, no imports involved.

```python
# pkg/car.py
class Engine:
    def start(self):
        return "vroom"


class Car:
    def __init__(self):
        self.engine = Engine()

    def honk(self):
        return "beep"

    def drive(self):
        a = self.engine.start()   # attribute chain
        b = self.honk()           # control: direct method on self
        return a + b
```

```sh
git init && git add -A && git commit -m probe
entire graph snapshot --repo . --format ndjson
```

### Observed

`Car.drive` has exactly one outgoing edge:

```
CALLS  ->  Car.honk    resolution=type_inferred  confidence=0.9
```

There is no edge for `self.engine.start()`. `Engine.start` has **no incoming
caller anywhere in the graph**.

The type is available: `Car.__init__` carries a `CONSTRUCTS` edge to `Engine`
in the same snapshot, so the assignment `self.engine = Engine()` was understood
well enough to record what was constructed.

### Expected

An edge from `Car.drive` to `Engine.start`, or if the inference is out of
scope, some lower-confidence or `name_only` edge rather than silence.

### At scale, with a control

Sampled from three repositories, twelve chain sites each where they exist, with
a matched control of `self.method()` calls drawn from the same files:

| Repo | `self.x.y()` resolved | `self.y()` control resolved |
|---|---|---|
| click `562e458` | 0 / 12 | 10 / 12 |
| httpx `b5addb6` | 0 / 12 | 11 / 12 |
| pre-commit `a9bba55` | 0 / 3 (all it has) | 10 / 12 |
| **total** | **0 / 27** | **31 / 36** |

Every resolved control edge carries `resolution: type_inferred` at confidence
0.82 to 0.9, in the same files and often the same functions as the chains that
produced nothing. The five control misses are not chains: they are attributes
holding a callable, called as a factory.

The plain inference: **`self` gets a type and `self.<attribute>` does not**, so
the second hop of a chain has nothing to resolve against.

**A correction to our own method, so the control can be trusted.** Our first
measurement indexed `CALLS` edges by the `start_line` of their `call_site`
evidence and looked each call site up by its own line. That reported "no edge"
for everything including the control, which was wrong: the evidence span covers
the **calling function**, not the call expression. We re-did it by finding the
enclosing symbol from snapshot spans and checking its outgoing edges for the
called name. The numbers above are from the corrected method. If the evidence
span is intended to be the call site rather than the enclosing function, that
would be a separate small bug, but we read it as intentional.

**Impact.** Any Python class that reaches a collaborator through an attribute,
which is most of them, is invisible to callers-of queries, so a dependency
graph over object-oriented Python is missing the majority of its call edges.

---

## Issue 3: `graph commit` cannot distinguish a method-body change from a class-level change

**Summary.** When a method inside a class changes, `graph commit` reports the
enclosing class as `body_changed` as well, and the output is byte-identical
whether or not anything at class level actually changed.

**Version.** As Issue 1.

### Minimal reproduction

```python
# widget.py
class Widget:
    limit = 10

    def render(self, text):
        return text.rjust(self.limit)
```

Commit that, then make two separate commits from it:

- **A**, method body only: `text.rjust(self.limit)` becomes
  `text.rjust(self.limit + 1)`.
- **C**, method body **and** class attribute: the same edit, plus
  `limit = 10` becomes `limit = 20`.

```sh
entire graph commit <sha> --repo . --json
```

### Observed

Both commits produce, sorted and serialised:

```json
[{"after_start_line":1,"before_start_line":1,"dependents_count":0,
  "kind":"class","name":"Widget","type":"body_changed"},
 {"after_start_line":4,"before_start_line":4,"dependents_count":1,
  "kind":"method","name":"Widget.render","type":"body_changed"}]
```

Identical. A is a pure roll-up of the method change. C contains a real
class-level edit. Nothing in the output separates them.

### What we checked and ruled out, so you do not repeat it

- **`before_start_line` and `after_start_line` are start lines only.** There is
  no end line and no changed-line range, and those six keys are the whole
  change record.
- **Line shifts are not a signal.** Adding or removing a class attribute
  alongside the method change shifts the method's `after_start_line` by one,
  but so does any edit that changes the line count anywhere above it, including
  inside a different method. And case C changes an attribute in place, so it
  shifts nothing at all.
- **`graph diff --base X --head Y --json`** returns the same schema and is
  byte-identical for A and C.
- **The snapshot's `body_hash` for a class is member-inclusive.** On this probe
  the `Widget` hash is `86cb9610341013b5` at baseline, `f325c8958e76e89d` in A
  and `9b316f200f9dad24` in C. It changes in A, where nothing at class level
  changed, so "the class hash moved" does not mean "the class itself changed".

### The wider behaviour, for context

Eleven distinct kinds of change against the same probe class, over twelve
commits:

| Change | Reported |
|---|---|
| method body | class `body_changed` + method `body_changed` |
| class attribute value | class `body_changed` alone |
| **method body and class attribute** | **identical to method body alone** |
| base class list | class `signature_changed`, with both signatures |
| method added | class `body_changed` + method `added` |
| method removed | class `body_changed` + method `removed` |
| method signature | class `body_changed` + method `signature_changed` |
| class docstring | class `body_changed` alone |
| method body + attribute added | as method body alone, method start line +1 |
| method body + attribute removed | as method body alone, method start line -1 |
| class decorator | see Issue 4 |

The decidable cases are the ones where no member change is reported alongside,
and the base class list, which arrives as `signature_changed`.

### Expected

Some way to tell a container change that is purely the sum of its members from
one that is not.

**Consequence for a consumer.** We wanted a rule of the form "a changed
container is not a source when every change inside it falls within the span of
a member that is itself already a source", so that changing one method does not
pull in everything that merely mentions the class. **Any rule that drops the
class in A also drops it in C, and dropping it in C loses every dependent of a
real class-level edit.** We chose not to implement it rather than approximate,
because trading a visible over-report for a silent under-report is the wrong
direction for our use case.

### Options, in your hands rather than ours

Any one of these would be enough for us, and you may well prefer something
else:

1. An end line, or a changed-line range, on each change record.
2. A member-exclusive body hash for containers, so a class hash moves only when
   the class's own lines move.
3. A `field` change record for a modified class attribute. Your output already
   carries `field` records for some kinds, which is why we expected one here.
4. An explicit flag on the container record, something like
   `"rolled_up": true`, which would be the smallest change to the schema.

**Impact.** A consumer cannot tell whether a class was reported because it
changed or because something inside it did, so a one-method edit expands the
apparent blast radius to everything that touches the class.

---

## Issue 4: adding a decorator to a class produces no class entity in the change list

**Summary.** Adding a decorator to a class reports `body_changed` on the module
and nothing about the class, so the change is invisible rather than merely
coarse.

**Version.** As Issue 1.

### Minimal reproduction

```python
# widget.py
class Widget:
    limit = 10

    def render(self, text):
        return text.rjust(self.limit)
```

Commit, then add a decorator and commit again:

```python
from dataclasses import dataclass


@dataclass
class Widget:
    ...
```

```sh
entire graph commit <sha> --repo . --json
```

### Observed

```
body_changed     module     widget.py
```

That is the entire change list. No `class Widget` entry, no member entry, and
`dependents_count` 0.

Confirmed with two decorator forms, `@dataclass` and
`@functools.total_ordering`, so it is not specific to `dataclasses`.

The class is still parsed correctly afterwards: the snapshot for the decorated
version contains `class Widget` at line 5 and `method Widget.render` at line 8.
So the symbol survives; it is the change list that loses it.

### Expected

A change record naming `Widget`, most naturally `signature_changed` with the
decorator visible in the signature, in the same way a base class list change is
reported.

### Why this one matters more than its size suggests

A module-level record carries no dependents, so a consumer asking "what depends
on what changed" gets nothing back and cannot tell that from a genuine answer
of "nothing depends on it". Adding `@dataclass`, `@functools.cache`,
`@runtime_checkable` or a framework's registration decorator is an ordinary
edit, and for `@dataclass` in particular it changes the class's constructor and
comparison behaviour, so the blast radius is real even though the change list
is empty.

We may have the wrong expectation here: if decorators are deliberately treated
as module-level because they are executable statements rather than part of the
class definition, we would still ask for a class-level record, but the
reasoning would be worth stating in the docs.

**Impact.** A common edit that changes a class's behaviour produces a change
list with no class in it, so any analysis downstream of `graph commit` sees the
edit as having no dependents at all.

---

## Issue 5: an unresolved call is reported as `external`, identical to a real third-party call

**Summary.** `target_kind` and `relation_scope` use the value `external` both
for a call that genuinely leaves the repository and for a call the resolver
could not resolve, so no consumer can tell a real external dependency from a
resolution failure.

**Version.** As Issue 1.

### Minimal reproduction

```
pkg/__init__.py     from pkg.core import helper as helper
pkg/core.py         def helper(a):
                        return a + 1

tests/test_both.py  import os.path
                    import pkg

                    def test_two_calls_that_are_not_the_same_thing():
                        a = os.path.join("x", "y")   # genuinely third party
                        b = pkg.helper(1)            # this repo, one dir away
                        return a, b
```

```sh
git init && git add -A && git commit -m probe
entire graph snapshot --repo . --format ndjson
```

### Observed

The two CALLS edges are identical on every field:

```
target=os.path.join   target_kind=external  relation_scope=external  resolution=import_external  confidence=0.78
target=pkg.helper     target_kind=external  relation_scope=external  resolution=import_external  confidence=0.78
```

One of those is a dependency on the standard library. The other is a function
defined in the same repository, re-exported explicitly by `pkg/__init__.py`,
which Issue 1 shows resolves correctly when it is called by a bare name. There
is nothing in the record to separate them.

### Expected

A distinct value, so the two cases can be told apart. One added enum value on
`target_kind`, say `unresolved`, would be enough, or a boolean alongside it. We
are not asking for a `resolution` change and not asking for the call to
resolve; that is Issue 1. This is the case where you already know you failed
and the output does not say so.

### The measured distributions

`target_kind` across all relation types, three repositories pooled:

| value | count |
|---|---|
| `symbol` | 15,899 |
| `external` | **5,616** |
| `file` | 1,624 |
| `config` | 348 |
| `route` | 159 |

`relation_scope` on `CALLS`, 7,058 edges pooled:

| value | count | destination |
|---|---|---|
| `external` | **3,977** | external record, 3,977 of 3,977 |
| `module` | 1,647 | a symbol, 1,647 of 1,647 |
| `file` | 1,188 | a symbol, 1,188 of 1,188 |
| `workspace` | 246 | a symbol, 246 of 246 |

Both fields separate "is the target a symbol here" perfectly and neither
separates "did resolution fail". `external` is doing two jobs.

### Why this blocks sizing Issue 1

Splitting the external `CALLS` leaving `tests/` by whether the external node's
`value` starts with the repository's own top-level package name:

| Repo | own package | genuinely foreign | own-package share |
|---|---|---|---|
| `pallets/click` `562e458` | 1,313 | 209 | **86.3%** |
| `encode/httpx` `b5addb6` | 750 | 215 | **77.7%** |
| `pre-commit/pre-commit` `a9bba55` | 292 | 294 | 49.8% |

In click's test suite, 86 percent of what the graph calls an external call is
the project's own code. That number is the size of Issue 1, and producing it
required guessing from a string. The guess is fragile in both directions: a
project whose package name differs from its import name is undercounted, and a
vendored or namespace package is overcounted. A consumer should not have to do
this, and a maintainer reading a bug report should not have to trust it.

**Impact.** Every consumer that wants to distinguish "this call leaves the
repository" from "this call could not be resolved" has to reimplement the same
string heuristic against the external node's name, and none of them can be
right about it.

## Reproduction summary

All five probes are a handful of files in a fresh git repository and need
nothing from us. In order: the three-file import probe, the two-class chain
probe, the `Widget` class with two commits, the same class with a decorator
added, and the two-call probe in Issue 5. Every one was re-run against
`v0.4.1-nightly.202609050613.0a448c50.0.20260905152923-3a2a715fad19` before
this was written, and all five still reproduce.

We have not filed these yet and are happy to split, merge or drop any of them.

**On splitting Issue 5 from Issue 1.** They are the same area and it would be
reasonable to fold them together. We kept them apart because the fixes are not
the same size: Issue 1 is a resolver change and Issue 5 is a single added value
in an existing enum. Bundled, the cheap one is likely to wait for the expensive
one, and the cheap one is the one that lets everybody else measure the
expensive one. If you would rather have them as one issue, say so and we will
merge them.
