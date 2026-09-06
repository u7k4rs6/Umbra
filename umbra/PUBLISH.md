# Publishing Umbra

A checklist for the sitting in which this repository becomes public. Nothing in
this document has been executed. It is written so that the person doing it can
work top to bottom without stopping to think about what a command will do.

Read `NOTES.md` first, under "Before this repository is ever made public". This
file is the executable form of that section.

## 1. What is on the remote right now

`scripts/checkpoint-refs-audit.sh` reads the checkpoint refs on a remote and
reports nothing else. It deletes nothing and pushes nothing.

```sh
cd umbra
./scripts/checkpoint-refs-audit.sh              # defaults to origin
./scripts/checkpoint-refs-audit.sh --remote origin
```

The reading at the time of writing:

```
43 refs   497,290,896 bytes   83,937 lines with a home path   860 with an email address
```

Three refs are worth knowing by name:

| Ref | Bytes | Home paths | Addresses | What it is |
|---|---|---|---|---|
| `01M1RT1RC60VQNX0PCYDMJ8JE5` | 31,464,311 | 3,381 | 43 | the largest single checkpoint |
| `b20f84567474` | 871,388 | 274 | 2 | the checkpoint the landing page's sample report is built from |
| `120ebb04a45d` | 27,520 | 15 | 0 | the minimal session recorded in phase 12, and the shape a short session leaves |

The number climbs with every push, because every push writes another
checkpoint. Re-run the audit immediately before acting, not from this table.

What those refs contain, from the phase 12 read: whole session transcripts,
absolute home paths, the operating-system user name, the owner's email address,
paths naming two other projects, and verbatim content from **Impeach**, an
unreleased project that was read in the first minutes of this build while
establishing that its handoff was not Umbra's. No real credential is in them;
the token-shaped matches are the synthetic values from the scrubber's own tests.

Umbra scrubs what Umbra writes. It does not rewrite Entire's storage, so none
of this is something the tool can fix for you.

## 2. The three ways to handle them

These are the three recorded in `NOTES.md`, each written out as the commands
that carry it out.

### Option 1. Delete the refs from the remote and stop syncing

```sh
cd umbra

# See exactly what will go, and keep the list.
./scripts/checkpoint-refs-audit.sh > /tmp/umbra-refs-before.txt
git ls-remote origin 'refs/entire/checkpoints/*' | awk '{print $2}' > /tmp/umbra-refs.txt
wc -l /tmp/umbra-refs.txt

# Stop new ones being pushed, before deleting the old ones. Without this the
# next git push puts them straight back.
entire configure --skip-push-sessions
entire status                                   # confirm sessions no longer sync

# Delete them from the remote, one ref per push argument.
xargs -a /tmp/umbra-refs.txt git push origin --delete

# Confirm.
git ls-remote origin 'refs/entire/checkpoints/*' | wc -l    # expect 0
./scripts/checkpoint-refs-audit.sh                          # expect "none. nothing to decide."
```

**Kept:** every commit, every file, the whole code history, all sixteen phases
of commit messages, the tests, the site, the reports. The local checkpoint refs
stay in the local clone, so `entire umbra <ref>` still works on this machine.

**Lost:** the checkpoint trail on the remote. Nobody who clones the public
repository can run `entire umbra` against a historical checkpoint of this
build, and `NOTES.md`'s claim that the build has a checkpoint trail becomes a
claim about a local machine rather than something a reader can verify. The
session record stops being part of what is published.

**Not fixed:** anything already fetched by somebody else, and anything GitHub
has cached. Deleting a ref is not a recall.

### Option 2. Read them and accept what they contain

```sh
cd umbra
./scripts/checkpoint-refs-audit.sh

# Read the ones that carry the most, largest first.
for ref in $(git ls-remote origin 'refs/entire/checkpoints/*' | awk '{print $2}'); do
    echo "== $ref"
    git ls-tree -r --long --full-tree "$ref" | sort -k4 -rn | head -5
done

# Read one in full. These are large; page them rather than cat them.
git ls-tree -r --full-tree <ref> | awk '$4 ~ /\.jsonl$/ { print $3 }' \
  | xargs -I{} git cat-file blob {} | less

# The specific thing to look for, from the phase 12 read.
git ls-tree -r --full-tree <ref> | awk '{print $3}' \
  | xargs -I{} git cat-file blob {} | grep -in "impeach" | head
```

**Kept:** everything, including the checkpoint trail, which is the strongest
evidence that the build happened the way `NOTES.md` says it did.

**Lost:** nothing technical. What is given up is privacy: the home paths, the
user name, the email address, the names of two other projects and the Impeach
content all become public and permanently so.

**Precondition:** this option is only available if the owner of Impeach agrees.
That is not a technical question and this checklist cannot answer it.

### Option 3. Rebuild in a fresh repository with checkpoint sync off

`scripts/graft.sh` already does the mechanical part. It has a dry run and it
never pushes.

```sh
cd umbra
./scripts/graft.sh --dry-run                    # prints every command, runs none
./scripts/graft.sh                              # do it for real

# What the script leaves you: a clone under ~/.cache/umbra-graft on a branch
# named umbra, with umbra/ copied in and two commits, documents first.

cd ~/.cache/umbra-graft
entire configure --skip-push-sessions           # before the first push, not after
entire status
git push -u origin umbra
git ls-remote origin 'refs/entire/checkpoints/*' | wc -l    # expect 0
```

**Kept:** the code, the documents, the tests, the site, and a clean remote with
no transcripts on it at all.

**Lost:** the per-phase commit history and the checkpoint trail both start at
the graft. Sixteen phases of commit messages collapse into two commits. The
development record survives only as prose in `NOTES.md`, which is exactly the
thing the phase 14 disclosure was written to say out loud.

### A fourth option the CLI supports, which is not in NOTES.md

`entire configure` accepts a checkpoint remote that is different from the code
remote. This was found while writing this checklist and is not one of the three
recorded options.

```sh
# Create a second, private repository first, for example u7k4rs6/umbra-checkpoints.
cd umbra
entire configure --checkpoint-remote github:u7k4rs6/umbra-checkpoints
entire status                                   # confirm where checkpoints now go

# The refs already on the public remote are still there. Delete them as in
# option 1, then let new ones go to the private repository.
git ls-remote origin 'refs/entire/checkpoints/*' | awk '{print $2}' \
  | xargs git push origin --delete
```

**Kept:** the full commit history, and a live checkpoint trail that continues
to be written, just somewhere private.

**Lost:** a reader of the public repository still cannot verify the trail. The
difference from option 1 is for the author, not for the reader.

**Untested.** Unlike the other three, nothing in this build has exercised it.
Dry-run it against a throwaway repository before trusting it.

## 3. The recommendation

Take **option 1**: turn session pushing off, then delete the checkpoint refs
from the remote, and keep every commit. It is the only one that removes another
person's unreleased project from a public remote without also throwing away the
sixteen phases of commit history that are the actual evidence of how this was
built. Option 2 needs a consent this checklist cannot obtain, option 3 pays for
a clean remote with the history that makes the repository worth reading, and
the fourth option solves the author's problem rather than the reader's.

## 4. GitHub Pages

The landing page is `umbra/site/`: `index.html` plus five files it loads and
three report directories. It is static, makes no request beyond its own files,
and works from `file://`, so it needs no build step and no workflow.

Pages can serve from a branch root or from `/docs` on a branch. It cannot serve
from `umbra/site`, so one of these two.

### Option A. Same repository, a gh-pages branch

```sh
cd /path/to/Umbra          # the repository root, not umbra/

# Do this after checkpoint sync is off. git subtree push calls git push, which
# fires the Entire pre-push hook, which is what puts checkpoint refs on the
# remote in the first place.
entire status              # confirm sessions do not sync

git subtree push --prefix=umbra/site origin gh-pages
```

Then in the repository's **Settings, Pages**: Source **Deploy from a branch**,
Branch **gh-pages**, folder **/ (root)**. The page appears at
`https://u7k4rs6.github.io/Umbra/`.

Re-publishing after a change to the site:

```sh
cd umbra && go run ./internal/report/gen-site -root .   # keep site/ and the report in step
cd .. && git add -A && git commit -m "site: ..." && git push
git subtree push --prefix=umbra/site origin gh-pages
```

Add `umbra/site/.nojekyll` once. Nothing in the site starts with an underscore
today, so Jekyll would not currently eat anything, but the file costs nothing
and removes the failure mode entirely.

### Option B. Split repository, only the site is public

Use this if the code repository stays private. A second public repository holds
the site and nothing else, so no transcript, no checkpoint ref and no source
can leak from it.

```sh
# Create u7k4rs6/umbra-site as a public repository, empty.

cd /tmp && rm -rf umbra-site && mkdir umbra-site && cd umbra-site
git init -b main
rsync -a --delete /path/to/Umbra/umbra/site/ ./
touch .nojekyll

# Entire is not enabled here and must not be. No hook, no checkpoint refs.
git add -A
git commit -m "umbra: the landing page"
git remote add origin https://github.com/u7k4rs6/umbra-site.git
git push -u origin main
```

Then **Settings, Pages**: Source **Deploy from a branch**, Branch **main**,
folder **/ (root)**. The page appears at
`https://u7k4rs6.github.io/umbra-site/`.

Two things to fix in the copied site before the first push, because they point
at a repository a reader may not be able to open:

- the four links to `github.com/u7k4rs6/Umbra` in `index.html`, in the nav, the
  footer, and the two code blocks under Install and Reproduce;
- the `go install github.com/u7k4rs6/Umbra/umbra/cmd/entire-umbra@main` line,
  which cannot resolve against a private module path.

Keeping the site in step after a code change is a copy and a push:

```sh
cd /path/to/Umbra/umbra && go run ./internal/report/gen-site -root .
rsync -a --delete /path/to/Umbra/umbra/site/ /tmp/umbra-site/
cd /tmp/umbra-site && git add -A && git commit -m "umbra: site refresh" && git push
```

## 5. The pre-publish scrub gate

Run all of this in one sitting, immediately before making anything public. It
is the last point at which a leak is cheap to fix.

```sh
cd umbra

# 1. The working tree is what the tests will judge.
git status --short                       # expect no output

# 2. The site matches the report it claims to be.
go run ./internal/report/gen-site -root .
git status --short                       # expect no output again

# 3. Everything green, from a cold cache.
go test -count=1 ./...

# 4. The refs on the remote.
./scripts/checkpoint-refs-audit.sh
```

### The tests that are the gate

All of `go test ./...` must be green, but these are the ones that exist for
this moment:

| Test | What it would catch |
|---|---|
| `TestNoGeneratedArtifactCarriesAnythingPrivate` | any home path, address, token shape or git author name in any file under `site/`, `fixtures/recorded/` or `docs/renders/`, whichever code path wrote it |
| `TestNoGeneratedReportCarriesAnOutOfRepoPath` | a timeline or node path that leaves the repository |
| `TestCommittedReportsKeepTheirGitObjectIDs` | over-scrubbing: a reproduce command a reader cannot check |
| `TestSiteSampleIsNotFabricated` | a landing-page map that was written rather than run |
| `TestSiteAssetsMatchTheReportAssets` | `site/umbra.css` or `site/umbra.js` drifting from the report's own |
| `TestScannerCatchesPlantedHomePathEmailAndAuthorName` | the scanner itself having stopped working |
| `TestAuditCatchesPlantedEmail` and `TestAuditReportsTheSameNumbersFromASubdirectory` | the refs audit itself having stopped working |

The last three matter as much as the first four. Every one of the four wrong
answers this project has produced came from a check that had never been
observed to fail; those tests are the ones that prove the gate is a gate.

### What a clean result looks like

```
$ git status --short
$ go run ./internal/report/gen-site -root .
copied umbra.css
copied umbra.js
embedded site/sample/umbra.json (40655 bytes)
embedded site/imported/umbra.json (443228 bytes)
$ git status --short
$ go test -count=1 ./...
ok      github.com/u7k4rs6/Umbra/umbra/cmd/entire-umbra      36.3s
ok      github.com/u7k4rs6/Umbra/umbra/internal/checkpoint   0.004s
ok      github.com/u7k4rs6/Umbra/umbra/internal/graph        0.023s
ok      github.com/u7k4rs6/Umbra/umbra/internal/report       0.314s
ok      github.com/u7k4rs6/Umbra/umbra/internal/report/gen-site 0.003s
ok      github.com/u7k4rs6/Umbra/umbra/internal/runner       0.017s
ok      github.com/u7k4rs6/Umbra/umbra/internal/shadow       0.080s
ok      github.com/u7k4rs6/Umbra/umbra/internal/transcript   0.016s
ok      github.com/u7k4rs6/Umbra/umbra/scripts               0.111s
$ ./scripts/checkpoint-refs-audit.sh
checkpoint refs on origin
-------------------------------------------------------------------
none. nothing to decide.
```

Two lines of that are the whole gate. `git status --short` printing nothing
after `gen-site` means the published site is the one the tests judged. The
audit printing `none. nothing to decide.` means the remote carries no
transcripts. Anything else on either line, stop and read it before publishing.

A failure looks like this, and it is the real output from phase 14.1, when
`graph.Verify` appended its commands without going through the scrubber:

```
--- FAIL: TestNoGeneratedArtifactCarriesAnythingPrivate
    artifacts_all_test.go:97: ../../site/sample/umbra.json carries an absolute
    home path on line 1254: /home/<user>
FAIL
```

## 6. After

- Re-run `./scripts/checkpoint-refs-audit.sh` once more, after the visibility
  change, against the now public remote.
- Open the Pages URL in a private window and confirm the map draws, the divider
  drags, and the network panel shows only the page's own files.
- If option 3 was taken, put the three-line disclosure from `NOTES.md` at the
  top of the new repository's README, so the missing history is stated rather
  than left to be noticed.
