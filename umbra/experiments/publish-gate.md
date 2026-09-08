# The pre-publish gate, run against this repository

2026-09-09. Read-only throughout. Nothing was deleted, nothing was pushed, no
ref was touched and the repository's visibility was not changed.

**Verdict: no-go.** Four things below have to be dealt with first, and the first
two are the reason.

## Which PUBLISH.md this is

Worth stating before the numbers, because it decides what "the gate" means.

This repository's `umbra/PUBLISH.md` is 33 lines: pre-push checks, a paragraph
on checkpoint privacy, and a demo smoke test. It names no options. The 354-line
version with the three options, the recommendation and the scrub gate is in the
fork, `~/entire-graph/umbra/PUBLISH.md`, and stage A did not bring it across
because both files exist and it was not a file that conflicted with nothing.

The long one was written about **this** repository even though it lives in the
other. Its opening reading is 43 refs on the remote; the fork has none. Its
Pages URL is `u7k4rs6.github.io/Umbra`. Its option 1 deletes checkpoint refs
from a remote, which only this repository has. So the gate below is the fork's
text run against this repository, which is what it was written for.

## 1. The refs audit, today

`scripts/checkpoint-refs-audit.sh`, unmodified:

```
53 refs   961,352,355 bytes   122,953 lines with a home path   1,291 with an email address
```

Against the reading recorded in PUBLISH.md when it was written:

| | Then | Now | Change |
|---|---|---|---|
| Refs on the remote | 43 | **53** | +10 |
| Bytes | 497,290,896 | **961,352,355** | +93% |
| Lines with a home path | 83,937 | **122,953** | +46% |
| Lines with an email address | 860 | **1,291** | +50% |

The remote holds 53 refs. This clone holds **76** under `refs/entire/`, so 23
exist only locally. The audit reports the remote, which is the right thing for
this decision.

**One caveat on that last column, from `b6cb02a`.** The script's email pattern
is the plain one, so its 1,291 is a lower bound. Section 4 measures the real
figure.

## A note on the redaction in this file

This report named the second project 11 times when it was written, including
twice as an `<owner>/<repo>` GitHub path, which is the same identifier it was
reporting as a leak. It is written as `<Project>` throughout now. Nothing about
the finding needs the name: the shape, the counts and the mechanism are what
make it checkable, and the person who has to act on it knows which project it
is. The 43-name list below is one shorter for the same reason, and the count of
43 in the sentence above it is the measured figure, not the length of the list
as printed.

## 2. The blocker: the sibling project's material

PUBLISH.md names this as the thing that decides the question: verbatim content
from **Impeach**, an unreleased project read in the first minutes of the build.
It is present, and it is on the remote.

### By name

| Where | Occurrences | Refs |
|---|---|---|
| `Impeach` in checkpoint refs | 3,031 | 52 |
| `impeach` in checkpoint refs | 2,046 | 52 |
| **Total** | **5,077** | **52 local refs** |

**All 52 of those refs are on the remote.** The remote carries 53 refs in total,
so 52 of 53 carry the blocker. They include `b20f84567474`, which is the
checkpoint the landing page's own sample report is built from.

Nothing in any committed file carries it. `git grep -i impeach` over the whole
tree returns nothing, so the scrubber did its job on the artifacts.

### By directory path, which found a second one

Scanning every ref for the 39 project directory names on this machine plus
`/Desktop/<name>` and `-Desktop-<name>` found **43 distinct non-Umbra project
names**: Agoda, Alibi, allocator-probe, Apex, Argmax, argmax-recompute, badge,
Blink, Cairn, Compute-Elasticity, Detect, DevOps, Dynamo, factpack-sweep, Game,
GCC, GitDate, HH, Impeach, Jaeger, jaeger-ui, LFX-Demo-Kubernetes,
local-inference-probe, Miku, MikuAI, MIRR, newest, Nvidia,
order-probe, Pinetree, PS-GIT, Reading-the-judges, Resume, Rust, Salvage,
Shadowbook, SlipStream, SPAR, Starling, threadA-note, thrice.

Most are single mentions of a path. One is not, and it is in a committed file.

### One of them, in two committed artifacts

This is the finding the audit was not looking for.

| File | Occurrences |
|---|---|
| `umbra/site/imported/umbra.html` | 317 |
| `umbra/site/imported/umbra.json` | 317 |

634 word-bounded occurrences of another project's name, in the directory
PUBLISH.md publishes wholesale. Two shapes:

```
/tmp/claude-1000/-home-<user>-Desktop-<Project>/e28b7845-.../scratchpad
https://raw.githubusercontent.com/<owner>/<Project>/main/README.md
git remote add origin <redacted>:<owner>/<Project>.git
```

Eight of them name the project as an `<owner>/<repo>` GitHub path, so it is
identified by owner and repository, not merely by a directory name.

**Why the scrubber missed it, exactly.** The agent's scratchpad directory is a
flattened absolute path, `/tmp/claude-1000/-home-utkuputku-Desktop-<Project>/`.
The scrubber replaced the user name, giving `-home-<user>-Desktop-<Project>`.
The raw form `home-utkuputku` appears zero times, so the user-name half worked.
The project name is what survived, because nothing removes it: it is no longer
inside a `/home/` path once flattened, and there is no rule for a project name.

**These files are still published even though the page no longer draws them.**
`gen-site` writes `{}` into every page's `umbra-imported` block, with the
comment "Earlier site builds embedded a second report from an unrelated project.
Clear that legacy data from any existing pages instead of republishing it." That
landed in `7661eb2`. But `site/imported/umbra.html` and `site/imported/umbra.json`
are still committed under `site/`, and PUBLISH.md publishes that directory whole,
by `git subtree push --prefix=umbra/site` or by `rsync -a umbra/site/`. They
would be served at `.../imported/umbra.html`.

**A stale claim this turned up.** The repository README says "the landing page
shows two, and they are not the same kind of thing" and describes the imported
map. Since `7661eb2` the page shows one. I moved that paragraph to the root
README in stage G and did not check that particular sentence.

## 3. Credentials, keys and absolute paths

### No real credential, anywhere in the tree

Every credential-shaped match is the scrubber's own test input, separated the
way the fork audit did it:

| Shape | Value found | Where | Real? |
|---|---|---|---|
| github token | `ghp_0123456789abcdefghijABCDEF` | `scrub_test.go`, quoted in NOTES | synthetic |
| api key | `sk-0123456789abcdefghijklmnop` | `scrub_test.go`, quoted in NOTES | synthetic |
| slack token | `xoxb-1234567890-abcdefghij` | `scrub_test.go`, quoted twice | synthetic |
| aws key | `AKIAIOSFODNN7EXAMPLE` | `scrub_test.go`, quoted twice | synthetic, Amazon's published example |
| private key | `BEGIN RSA PRIVATE KEY` | `scrub_test.go`, quoted twice | synthetic |
| jwt, google key, gitlab pat | none | | |

Three files hold all of them: `umbra/internal/report/scrub_test.go`, which is
the scrubber's fixture, and `umbra/NOTES.md` and
`umbra/experiments/fork-checkpoint-audit.md`, which quote it while explaining
that the token-shaped matches are synthetic.

### Absolute paths, tracked files

| Value | Count | Verdict |
|---|---|---|
| `/home/someone` | 10 | synthetic, the scrubber's fixture |
| `/home/plantedperson` | 6 | synthetic, the scanner's known-positive |
| `/home/user` | 2 | synthetic |
| `/home/<user>` | 1 | already scrubbed |
| **`/home/utkuputku`** | **2** | **real** |

The two real ones are both in `umbra/experiments/fork-checkpoint-audit.md`,
lines 62 and 107, where the write-up quotes the value it was auditing. That is a
document I wrote naming the machine's user in a file that would be published. It
is the only place the OS user name appears in the tree.

## 4. Email addresses, both encodings

### In tracked files

| Value | Count | Verdict |
|---|---|---|
| `someone@example.com` | 10 | synthetic |
| `planted.address@example.invalid` | 3 | synthetic, the known-positive |
| `committer@example.invalid`, `fixture@example.invalid` | 2 | synthetic |
| **`utkarshbahuguna10@gmail.com`, plain** | **3** | **real, quoted in prose** |
| **`<utkarshbahuguna10@gmail.com`** | **2** | **real, quoted in prose** |
| **`u003cutkarshbahuguna10@gmail.com`** | **1** | **real, quoted in prose** |

All six real ones are in `NOTES.md` and `experiments/fork-checkpoint-audit.md`,
in the passages that record the encoding finding itself. The address is the same
one every commit in the repository carries in its author line, so publishing the
tree does not disclose it for the first time.

One defect while I was there: `fork-checkpoint-audit.md` line 77 writes
`u003cutkarshbahuguna10@gmail.com` without its backslash, the same
heredoc-eaten-backslash that phase 28 repaired in NOTES. Not fixed here, because
this task is read-only.

### In the checkpoint refs, which is where the volume is

Scanning with up to twelve characters of leading context rather than a bare
address pattern, so the encodings show as themselves:

| Encoding | Occurrences |
|---|---|
| `utkarshbahuguna10@gmail.com` | 698 |
| `Bahuguna\nutkarshbahuguna10@gmail.com` | 156 |
| `<utkarshbahuguna10@gmail.com` | 133 |
| `nutkarshbahuguna10@gmail.com` | 123 |
| `\\\\u003cutkarshbahuguna10@gmail.com` | 96 |
| `` `u003cutkarshbahuguna10@gmail.com `` | 62 |
| `u003cutkarshbahuguna10@gmail.com` | 60 |
| `written\n`u003c...`, `printed\n`u003c...` | 84 |
| `` `utkarshbahuguna10@gmail.com `` | 42 |
| ``**`utkarshbahuguna10@gmail.com`` | 30 |
| `` `\\u003c... ``, `` `\\\\u003c... ``, `written\nu003c...` | 50 |
| **14 encodings, total** | **1,534** |

In **65** local refs, **52** of them on the remote.

**A plain-form-only scan would have reported 698.** The true figure is 1,534.
That is the `b6cb02a` lesson reproducing on a second dataset, at more than
double the under-report: a total measures the encoding as much as the content,
and printing the distinct values is what shows you the near misses.

Other addresses in the refs, for completeness: `redacted@example.com` 452, which
is the scrubber's own replacement value, `someone@example.com` 672 and the
`example.invalid` fixtures, and `contact@palletsprojects.com` 69, which came in
with the click repository during the real-repo experiment.

## 5. The tests that are the gate

`go test ./...` is green, all nine packages, 398 tests. The eight tests
PUBLISH.md names as the gate all pass:

```
--- PASS: TestNoGeneratedArtifactCarriesAnythingPrivate (0.14s)
--- PASS: TestNoGeneratedReportCarriesAnOutOfRepoPath (0.01s)
--- PASS: TestCommittedReportsKeepTheirGitObjectIDs (0.01s)
--- PASS: TestSiteSampleIsNotFabricated (0.00s)
--- PASS: TestSiteAssetsMatchTheReportAssets (0.00s)
--- PASS: TestScannerCatchesPlantedHomePathEmailAndAuthorName (0.00s)
--- PASS: TestAuditCatchesPlantedEmail (0.05s)
--- PASS: TestAuditReportsTheSameNumbersFromASubdirectory (0.07s)
```

`go vet ./...` clean. `git diff --check` clean.

**The output-wide artifact test passes over that leak, and that is the
more important result on this page than the eight PASS lines.** It walks
`site/`, `fixtures/recorded/` and `docs/renders/` and it does read
`site/imported/umbra.json`, the file carrying the project name 317 times. It passes
because `bannedInArtifacts` in `internal/report/leaks.go` holds ten patterns,
for home paths, `/Users`, `/root`, an address, four token shapes, a private key
and a jwt, plus the git author names, and **none of them is a project name**.
The one pattern that could have caught this, `/home/[A-Za-z0-9._-]+`, does not
match because the path was flattened to `-home-<user>-Desktop-<Project>` before
it ever reached an artifact.

This goes on the running list of checks that passed while asserting nothing. It
is a check that was pointed at the question and answered it correctly for the
ten things it knows about, and was never pointed at an eleventh.

## 6. The working tree

PUBLISH.md's first gate step is `git status --short`, expecting no output. It is
not clean:

```
 M umbra/site/index.html
 M umbra/site/landing.css
?? umbra/site/guide.js
?? Umbra-ui-redesign.zip
?? files (1)/
```

The first three are a site redesign in progress. `Umbra-ui-redesign.zip` is
84 MB and is a snapshot of this repository, 2,873 files. `files (1)/` holds the
eight SVGs and `gen_svgs.py`. **None of the five is covered by `.gitignore`**,
checked with `git check-ignore`, so the `git add -A` in PUBLISH.md's re-publish
flow would stage all of them including the 84 MB archive.

I did not run gate step 2, `go run ./internal/report/gen-site -root .`, because
it writes into `site/` and the working tree has uncommitted work there. Running
it would have risked the redesign. That step has to be run from a clean tree,
which is what the gate says.

## Verdict: no-go

Four things, in the order they matter.

1. **52 of the 53 checkpoint refs on the remote carry the Impeach material**,
   5,077 occurrences. This is the blocker PUBLISH.md names, it is on the remote
   rather than only local, and it is another person's unreleased project. It is
   also the one this checklist cannot decide for you: option 2 needs their
   consent, and options 1 and 3 do not.
2. **`site/imported/umbra.html` and `site/imported/umbra.json` carry a second
   project's name 634 times**, including its GitHub owner and repository. They
   are committed, they sit inside the directory the publish step copies whole,
   and the page no longer draws them, so deleting them costs nothing that is
   still used. This one is fixable by hand today and is not covered by any of
   PUBLISH.md's three options, because all three are about refs.
3. **The working tree is not clean**, so the gate cannot be completed as
   written and the site cannot be proved to match the report the tests judged.
4. **The artifact scanner has no rule for a project name**, so it cannot catch
   item 2 or a recurrence of it. Until it does, a green suite is not evidence on
   this question.

The email addresses and the two `/home/utkuputku` lines are real but are not
blockers: the address is already in every commit's author line, and both are in
documents that discuss the finding. They are listed so the decision is made with
them in view rather than in ignorance of them.

Nothing here was acted on. No ref was deleted, nothing was pushed, and the
repository is still private.
