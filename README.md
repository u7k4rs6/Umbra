# Umbra

**Umbra finds the code an AI agent's change affects but the agent never looked
at.** It is a plugin for the [Entire CLI](https://github.com/entireio/cli), run
as `entire umbra <checkpoint>`, and it is deterministic and offline: there is
no model anywhere in it.

**The product manual is [umbra/README.md](umbra/README.md)**: install, run, the
four states, the three evidence tiers, the ranking, the limitations and the
corrections. This file is about the repository that holds it.

## What is in this repository

This repository is `u7k4rs6/Umbra`. Umbra lives entirely in `umbra/`, which is
its own Go module, `github.com/u7k4rs6/Umbra/umbra`. Nothing outside `umbra/`
is product code.

```
umbra/          the module; see umbra/README.md
Concepts/       working notes that predate the build
.entire/        Entire CLI settings
```

**This repository is not itself a fork of `entireio/entire-graph`.** It has no
upstream tree and no Go module at its root, and `go install` from the root
fails with `cannot find main module` for that reason. Where the two are easy to
confuse: the layout under `umbra/` is the one `docs/ARCHITECTURE.md` specifies,
kept exactly so the module can be grafted into such a fork without moving a
file, and `umbra/scripts/graft.sh` does that graft, with a dry run and without
pushing.

That graft was run once, into `u7k4rs6/entire-graph`, which **is** a fork of
`entireio/entire-graph` with this `umbra/` tree inside it. The two then
diverged, and the fork's work was merged back into this repository in six
stages recorded in [umbra/NOTES.md](umbra/NOTES.md). A statement that is true
of one of these repositories is often false of the other, which is how the two
READMEs came to disagree with each other about install commands, commit hashes
and whether the checkpoint history exists. It does exist here: 74 refs under
`refs/entire/checkpoints/`.

The repository is private. That is not incidental to using it: `go install`
against the module path needs `GOPRIVATE` set, because the public checksum
database cannot see the repository. The manual says so at the install step.
[umbra/PUBLISH.md](umbra/PUBLISH.md) is the checklist for the sitting in which
this repository is made public. Nothing in it has been run.

The submission document for the event Umbra was built at is kept at
[umbra/archive/BUILDATHON.md](umbra/archive/BUILDATHON.md). Its durable content
was moved into [umbra/NOTES.md](umbra/NOTES.md) as phases 29 to 31, and its
header says what left it and what was dropped.

## The final review

Phase 11 was a semantic-diff review of the whole build to that point,
comparing the first commit against the last:

```
entire graph diff --base 2495309553ee8c01a61a3c4922e45861d3d4de6d --head HEAD --repo .
```

The diff itself is unremarkable: 1014 added entities, no signature changes, and
no function without an incoming edge, which is what a build that only ever added
code should look like. The interesting part was what the diff did not contain,
and it is the single most useful thing Graph did in this build.

### The .gitignore that hid the command package

Reading the diff, `umbra/cmd` was missing. Not thin, not partial: the graph had
no symbols for it at all, while it had 1014 for everything else. A directory of
Go files that the build had been compiling and testing for nine phases was, as
far as Graph could see, not there.

Graph builds its snapshot from the repository, and it honours `.gitignore`. So
the question was not what Graph had failed to parse. It was what git had been
told to ignore. The answer was the second line of a `.gitignore` I wrote myself
in the first commit, under a heading that says exactly what I meant it to do:

```
# Go build output
entire-umbra
```

I meant that to ignore the compiled binary, which is written as `entire-umbra`.
A gitignore pattern with no slash in it is not anchored to the repository root
and does not distinguish a file from a directory: it matches any path component
with that name, at any depth. The binary is called `entire-umbra`. So is the
directory the binary's source lives in. The pattern matched
`umbra/cmd/entire-umbra/` and everything under it.

`git ls-files umbra/cmd/` returned nothing. `main.go`, `analyze.go`,
`execute.go`, `options.go`, `output.go`, `record.go` and `options_test.go` had
never been committed. The command package went in during phase 2 and was
untracked through phase 10, nine phases in which every commit message reported
a rising test count that included tests in a file the repository did not have.

Nothing caught it, and nothing was going to. `go build ./...` and
`go test ./...` read the working tree, and the files were on disk, so they were
green throughout and told me nothing. `git status` says nothing about a path it
has been told to ignore. `git commit -a` does not add an ignored file. The only
symptom available to me was one I never looked at: a clone.

The fix was to anchor the patterns to the paths the binaries are actually
written to, so a directory name can never collide with them again:

```
/entire-umbra
/umbra/entire-umbra
/umbra-out/
```

The seven files were then committed, and the thing that should have been done
long before was done at last:

```
git clone . /tmp/clone && cd /tmp/clone/umbra
go build ./...   # OK
go test ./...    # every package ok
```

That clone is the verification. Before the fix it would have failed to build,
because `package main` did not exist in the repository. After it, it builds and
every test passes, which is the only evidence that means anything here.

This is the clearest case in the build of Graph driving a decision rather than
confirming one. Nothing else in the toolchain was looking at the repository.
The compiler, the test runner and my own reading were all looking at the
working tree, where the code was present and correct. Graph was looking at what
had actually been committed, and the gap between those two things was an entire
package. A green test run is not the same as a correct repository, and it takes
a tool that reads the repository to tell you which one you have.

### The self report

Then Umbra was run on the session that built it. On the commit where it changed
its own `shadow.Build`, all twelve dependents came back umbra, including every
scenario test. That reading is correct, and the reason is the most useful thing
this project learned about itself: the change was made with a shell command
rather than the editing tool, so the session produced no read and no edit event
for that file. **Umbra's examined set is built from tool activity, so an agent
that edits through the shell leaves no trace Umbra can see.** It is the same
shape of blind spot the product exists to find, pointed at the product. The
report is in `umbra/site/self/`.

Running the report on itself also found two bugs in the scrubber, both fixed:
paths from outside the repository were reaching the timeline through search
output, and the base64 heuristic was redacting git object ids and worktree
paths, which is exactly the text a reader needs in order to check a line.

## Development record

Umbra was built here, in this repository, and the build's own checkpoint trail
is here with it: 74 refs under `refs/entire/checkpoints/`, beginning at the
first line of code. The `umbra/` layout from `docs/ARCHITECTURE.md` was
preserved exactly through the build so the module could be moved into a fork of
`entireio/entire-graph` without relocating a single file, which is why that
graft is two commits rather than a rewrite.

**The trail did not come across to the fork, and that is on purpose.** These
checkpoint refs hold whole session transcripts, including material read from an
unrelated project in the first minutes of the build. None of it is pushed
there. The fork's trail begins at the graft, and its own README says so; the
phase by phase record survives there as prose. Both READMEs used to carry that
paragraph, and it was only ever true of the fork.

The phase by phase record is [umbra/NOTES.md](umbra/NOTES.md), which carries
what each phase found and where the design documents turned out to be wrong.

Seventeen phases built and shipped the product, of which the first thirteen
were features. The last four were not:

- **14** replaced the landing page's scripted sample with a real run, added a
  second map from an imported session in another project, and prepared two
  scripts that were deliberately not executed: the graft, and a read-only audit
  of what the checkpoint refs on the remote contain.
- **15** made the scrubber the only exit. Five leaks had been the same bug five
  times, an output path that wrote without scrubbing, so the renderers now take
  a value that only the scrubbing path can produce. It also gave every check in
  the project a test that proves the check can fail.
- **16** restyled the landing page around a drawn eclipse. No raster asset and
  no request for one: the corona is inline SVG built deterministically from the
  report's own numbers.
- **17** was the submission pass: two visual checks decided from screenshots
  rather than from the CSS, [umbra/PUBLISH.md](umbra/PUBLISH.md), and the
  verification of this README against the repository.

The phases after 17 are not part of that build. They are recorded in
[umbra/NOTES.md](umbra/NOTES.md) and run to 28: running Umbra against a real
outside repository and fixing what that found, two upstream bug reports, and
the six stage merge that brought the fork's divergent work back here.

**Which map is which.** The landing page shows one, a **real session on a
seeded fixture**. The session, its tool activity and the test results are real:
checkpoint `b20f84567474` against commit `1c2cf29`, where an agent changed a
signature and five probes cracked. What is arranged is the fixture underneath
it, `umbra/fixtures/app`, which was written to have callers a session would
plausibly miss. The sentence above the map is the agent's own, taken from the
stored transcript.

It used to show two. The second was a real session from another project of the
builder's, imported with `entire import` and analysed unchanged, and every one
of the four symbols that depended on what it changed came back umbra. The page
stopped drawing it in `7661eb2` and its report files are now deleted, because
that report named the other project: its directory, its GitHub owner and
repository, and the source files and symbols the session changed. Publishing
someone else's code structure to illustrate a point about ours was not a trade
worth making, and the finding it carried is recorded here rather than shown.

**Nothing on the page is scripted.** There is no map drawn from an invented
report. Until phase 14 there was one: the sample was assembled by the site
generator, with a checkpoint id of `sample000class`, a commit of all zeros and a
session sentence written by hand, under a caption calling it a real report. It
was replaced by a real run, and the generator now refuses a report carrying
those markers so the caption cannot drift away from the file again.

The eight scenarios under `umbra/fixtures/recorded/` are **authored
transcripts**, written to exercise one classifier rule each. They drive tests
and are never rendered as a map on the page. `umbra/fixtures/recorded/minimal/`
is different: a full recording of a real short session, hand reviewed before it
was committed.

## Prior work and AI disclosure

No code was reused. The idea of subtracting the examined set from a blast
radius was developed in planning for this event.

All product code was written with an AI coding agent, captured in Entire
checkpoints, across the seventeen phases of the build and the phases after it. That includes the tests, the
landing page, the corona renderer and every document in this repository except
the four planning documents in `umbra/docs/`, which were written before the
build with AI assistance and are the first commit.

What came from the human rather than the agent: the problem, the four settled
decisions in the kickoff, the scope of each phase, the decision to keep the
repository private, and the visual direction for the landing page in phase 16,
which was specified before it was built and not proposed by the agent. Every
phase was reviewed and accepted by hand before the next one started.

[umbra/NOTES.md](umbra/NOTES.md) records what the probe found, which
degradations are in effect, the places the design documents turned out to be
wrong about the installed tools, and the mistakes: the confident wrong answers,
the five scrubbing leaks, the label tests that were proving nothing, and the
running list of checks that passed while asserting nothing. Those are in
there because a build record that only lists what worked is not a record.

Built during Bengaluru Tech Week on the Entire ecosystem.
