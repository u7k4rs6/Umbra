# Two repositories, side by side

2026-09-07. Written so the choice can be made from facts rather than from
memory. Nothing was migrated, merged, deleted or reorganised to produce it.
Both repositories were already on disk, so nothing was cloned either.

| | private | fork |
|---|---|---|
| Path | `~/Desktop/Umbra` | `~/entire-graph` |
| Remote | `github.com/u7k4rs6/Umbra` | `entire://aws-ap-south-1.entire.io/gh/u7k4rs6/entire-graph` |
| Branch inspected | `experiment/first-real-repo` | `umbra-buildathon` |

## Two things in the brief are the wrong way round

**The 981 line BUILDATHON.md is in the fork, not in the private repository.**
The private repository's is the 59 line one. The brief has these swapped, and
it matters, because everything task 2 wants mined lives in the fork already.

**The weakest-hop claim is true, and phase 26 was wrong to generalise.**
Commit `e41af1db` exists, in the fork. Phase 26 checked the private repository
exhaustively, found nothing, and concluded the behaviour never shipped. It
never shipped *there*. Section "The claim, settled" below has the detail. The
phase 26 entry in NOTES needs a correction; this file does not make it, because
correcting it is a change and this task is an inventory.

## Commits and dates

| | private | fork |
|---|---|---|
| Commits, all local heads | 179 | 3,049 |
| Commits on the branch inspected | 63 | 1,369 |
| Commits touching `umbra/` | 63 | **24** |
| First `umbra/` commit | `2495309` 2026-09-06 phase 0 | `d25e5b51` 2026-09-06 initial understanding |
| Last `umbra/` commit | `021de41` 2026-09-07 | `796d2eb7` 2026-09-06 |
| Dates of `umbra/` work | 51 commits 09-06, **12 commits 09-07** | 24 commits 09-06, none on 09-07 |
| Authors on the branch | 1 | 14, of which 13 are upstream `entire-graph` contributors |

The fork's 1,369 branch commits are mostly upstream `entire-graph` history
going back to 2026-05-28. Only 24 touch `umbra/`, and the second of those,
`42f215f8`, is titled "import the plugin as one commit". So the fork holds the
private repository's phase 0 to 21 work as **one squashed commit**, plus 22
commits of real work done afterwards in the fork itself.

**The private repository has a day of work the fork does not have**: the twelve
commits of 2026-09-07, which are the click experiment and everything that came
out of it.

## Checkpoints

The two use different backends, which is the first thing that would have to be
decided on a merge.

| | private | fork |
|---|---|---|
| Backend | `git-refs`, set in `.entire/settings.json` | legacy shared branch, `entire/checkpoints/v1` |
| Where the data sits | 65 refs under `refs/entire/checkpoints/` | a branch, 1,000 commits, also on `origin` |
| Checkpoints on the branch | 16 | 12 |
| `umbra/` commits carrying a trailer | **58 of 63** | **15 of 24** |
| All commits on the branch carrying a trailer | 58 | 293, mostly upstream's |
| Checkpoint payload, logical bytes | 65 refs, **968 MB** | 2,957 files at the tip, **5.9 GB** |
| `.git` on disk | 766 MB | 315 MB |
| Working tree on disk | 187 MB | 270 MB |

The logical byte figures are `git ls-tree --long` sums and are much larger than
the packed `.git`, so they say what the checkpoint data contains rather than
what it costs. The fork's 5.9 GB is mostly upstream's own accumulated
checkpoint branch, not ours.

Checkpoints resolve in both. `entire checkpoint explain 9615cecf8d52 --short`
works in the fork and returns a real session, so the fork's trailers are not
dangling.

## What exists in one and not the other

Tracked files under `umbra/`: private 215, fork 239.

### Only in the private repository, 12 files

```
umbra/experiments/first-real-repo.md          the three-pass click experiment
umbra/experiments/python-resolution.md        click vs httpx vs pre-commit
umbra/experiments/upstream-report.md          the five drafted issues
umbra/fixtures/recorded/qualified-names/      the method and nested-function scenario
umbra/internal/graph/testdata/verify/*.txt    six recorded graph verify outputs
umbra/cmd/entire-umbra/unnamed_test.go        the sweep count wiring tests
```

### Only in the fork, 35 files

```
umbra/internal/graph/evidence.go              resolution vocabulary and ranking
umbra/internal/graph/evidence_test.go
umbra/internal/shadow/evidence.go             the three evidence tiers
umbra/internal/report/evidence_render_test.go
umbra/internal/report/partial_analysis_test.go
umbra/internal/report/partial_scenario_test.go
umbra/internal/report/sweep_inconclusive_test.go
umbra/internal/report/verify_path.go
umbra/internal/shadow/partial_scenario_test.go
umbra/internal/graph/testdata/partial-snapshot.ndjson
umbra/fixtures/partial/                       7 files, the incomplete-analysis fixture
umbra/fixtures/recorded/dynamic-dispatch/     a ninth scenario
umbra/internal/transcript/failed_read_test.go and its testdata
umbra/docs/img/                               8 SVGs, plus docs/gen_svgs.py
umbra/site/imported/                          4 files, the imported-session sample
umbra/KICKOFF_PROMPT.md
umbra/README.md                               636 lines
```

The private repository's README is at the repository root instead, 533 lines,
which NOTES records as a deliberate deviation.

## Where the code has actually diverged, which the file list understates

Neither repository is a superset of the other. Checked by grep against the
current source of each.

| Feature | private | fork |
|---|---|---|
| Snapshot keeps `resolution` and `confidence` | yes, as `EdgeQuality`, phase 25 | **yes, as `Edge.Resolution` and `Edge.Confidence`, `e41af1db`** |
| Walk carries the weakest hop | yes, phase 25 stage 2 | **yes, `e41af1db`** |
| Three evidence tiers, confirmed and heuristic and needs verification | **no** | **yes**, `internal/shadow/evidence.go` |
| Reads `Capabilities.HeuristicRelations` | no | **yes** |
| Snapshot completeness header and summary record | no | **yes** |
| Malformed records counted rather than skipped | no | **yes** |
| Partial-analysis fixture and tests | **no** | **yes** |
| Sweep says inconclusive when the runner printed no ids | **no** | **yes**, `1eaad486` |
| Sweep reads the count when the id list is truncated | **yes**, phase 22 | **no** |
| Bind a changed method by qualified name | **yes**, phase 23 | **no** |
| Report names symbols the graph was never asked about | **yes**, phase 23 | no |
| Reach line, calls that leave the repository | **yes**, phase 25 stage 3 | no |
| A failed read no longer marks a node lit | no | **yes**, `b4134ca4` |
| Test functions | 358 | 354 |
| `go test ./...` | green | green |

**The two halves of the same bug are in different repositories.** The fork's
`1eaad486` fixes the sweep printing "0 leaks" when verify produced no per-test
ids at all. The private repository's phase 22 fixes the sweep printing "0
leaks" when verify produced a count but dropped the id list for exceeding its
byte budget. The fork's `verify.go` still reads only `newlyFailingRE`, so the
fork still has the phase 22 bug. The private repository has no
`SweepInconclusive`, so it still has the other one.

**Phase 25 duplicated work that already existed.** `e41af1db` did what phase 25
stages 1 and 2 did, earlier and more thoroughly. The phase 25 task named
`internal/graph/evidence.go` and `internal/shadow/evidence.go` as files to
read; I reported that neither existed, which was true of the private
repository and false of the project. Both exist in the fork and are the better
implementation: eight named resolution constants, a rank order, a `Structural`
predicate, and the three tiers built on top.

## The claim, settled

Phase 26 asked whether `e41af1db` and the weakest-hop rule were real. They are.

`e41af1db`, in the fork, 2026-09-06, "umbra: keep the resolution and confidence
Graph puts on every relation". It adds `umbra/internal/graph/evidence.go` at
208 lines and `evidence_test.go` at 280, and touches ten files for 1,019 added
lines. Its own message says:

> The walk carries the weakest hop along each path rather than only the first
> hop's relation, because a chain is only as resolved as its least resolved
> link, and names the hop that made a path heuristic.

BUILDATHON.md line 395 says the same in its table of which files consume what:

> `umbra/internal/graph/bfs.go` | The relation graph. Carries the **weakest**
> resolution along each path, because a chain is only as resolved as its least
> resolved link

So the document is accurate about the fork. Two details of the phase 26
write-up need correcting when someone gets to it:

1. The rule is on **resolution**, ranked by a table in `evidence.go`, with
   confidence carried alongside. Phase 25 stage 2 in the private repository
   ranks by **confidence**. Same idea, different key.
2. There is no fixture called `partial-analysis`. The fork has
   `umbra/fixtures/partial/` and three test files with `partial` in the name.
   The nearest assertion is in `internal/report/partial_analysis_test.go`.
   `partial-read`, which phase 26 checked, is a transcript tier fixture and was
   the wrong thing to look at.

## What the curveball section actually contains

Since task 2 will want it. `## Noon Curveball` runs from line 317 to line 616,
about 300 lines, and is the only prose record of the evidence-tier design. Its
constraint sentence is:

> **Graph is evidence, not an oracle.** Umbra must not present an incomplete
> Graph relationship as certain, must say when its own analysis may be partial,
> must give a way to check anything it is unsure of, must keep working
> unchanged for code Graph fully resolved, and must ship a fixture representing
> incomplete analysis.

and the tier table it defines is:

| Word | What it means |
|---|---|
| `confirmed` | the graph resolved every relation on this path to a definition |
| `heuristic` | the graph derived this relation rather than parsing it, so it may be wrong |
| `needs verification` | reached through a relation the graph could not resolve, or found under a partial analysis; check it against the source |

It also states that this is a fourth axis rather than a fifth state, that the
four states and the six ranking factors are untouched, and that `Score` reads
none of the new fields. That is design content and none of it depends on the
event.

## What would be lost by picking each

**Picking the fork loses**, unless ported: the twelve commits of 2026-09-07,
the three experiments documents, the six recorded `graph verify` outputs, the
qualified-names scenario, and four fixes, the truncated-verdict sweep count,
the qualified-name bind, the unresolved-source reporting and the reach line.
It also loses the per-commit checkpoint trail for phases 0 to 21, which the
fork holds as one squashed commit, and it loses the `git-refs` checkpoint
backend in favour of the legacy branch.

**Picking the private repository loses**, unless ported: `evidence.go` in both
packages and the three evidence tiers, the partial-analysis fixture and its
four test files, the inconclusive sweep, the failed-read classifier fix, the
dynamic-dispatch scenario, the heuristic-relations reading, the snapshot
completeness handling, the 981 line BUILDATHON.md, the eight SVGs and their
generator, `KICKOFF_PROMPT.md`, `umbra/README.md` and the imported-session site
sample. It also loses the position inside a fork of `entireio/entire-graph`,
which is where `scripts/graft.sh` was always meant to put this.

**Neither loses history outright.** The fork's squashed import means the
private repository is the only place phases 0 to 21 exist as separate commits
with their own checkpoints. The private repository has no copy of the fork's 22
follow-on commits at all.

## Things worth deciding rather than defaulting

- The checkpoint backends differ. A merge has to pick one, and the fork's is
  the legacy branch form that `entire configure --checkpoint-backend refs`
  exists to migrate away from.
- The fork's `entire/checkpoints/v1` branch is on `origin` and carries 5.9 GB
  of upstream and local session transcripts. NOTES already records that the
  private repository's checkpoint refs carry home paths, an address and
  material from an unrelated project. The same question applies to the fork,
  and has not been asked of it.
- The private repository is private and the fork is not. Everything in
  "Before this repository is ever made public" in NOTES applies to the fork
  today, and was never re-checked against it.
- Two independent implementations of edge quality now exist. Whichever
  repository wins, one of them should be deleted rather than merged, and the
  fork's is the more complete.

## Method

Read-only. Commit counts from `git rev-list --count`, trailers from
`git log --format=%B | grep -c '^Entire-Checkpoint:'`, file lists from
`git ls-files umbra/` compared with `comm`, checkpoint payloads from
`git ls-tree -r --long --full-tree` summed per ref, feature presence from grep
against the working tree of each. `go test ./...` was run in both. Nothing was
written to either repository except this file.
