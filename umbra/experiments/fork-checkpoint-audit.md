# The fork's checkpoint branch, audited

2026-09-07. Read-only. Nothing was deleted, rewritten or pushed.

`repo-inventory.md` ended with an open question: the fork's
`entire/checkpoints/v1` branch carries 6.27 GB of session transcripts, NOTES
records that the private repository's checkpoint refs hold home paths, an
address and material from an unrelated project, and the same question had never
been asked of the fork. This asks it.

## The answer first

**It is already public.** `refs/heads/entire/checkpoints/v1` is on
github.com at commit `207775b11b8e22ea2e921fda4afeead07f0e3bef`, and
`u7k4rs6/entire-graph` is a public fork of `entireio/entire-graph`. The local
branch is at the same commit and zero ahead. This is not a decision about
whether to publish. It was published.

**Ours is a twentieth of a percent of it, and it contains no credential and
nothing from any other project.** What it does contain is the owner's identity:
117 occurrences of the real email address, 10,136 absolute home paths, 9,840
lines naming the operating-system user.

That is a narrower exposure than the private repository's, and the one blocker
NOTES names for the private repository does not apply here.

## What is on the branch

2,957 files at the tip, 6,273,788,844 bytes. The store is
`<2 hex>/<10 hex>/<n>/` per checkpoint, holding `full.jsonl`, a metadata
`.json` and a `prompt.txt`. 976 transcripts account for 6.15 GB of the 6.27.

988 checkpoints have metadata. Partitioned by the branch they were taken on and
by whether they touched `umbra/`:

| | checkpoints | files | bytes | share |
|---|---|---|---|---|
| **Ours** | 30, from 3 sessions, all 2026-09-06 | 78 | 35,301,438 | **0.56%** |
| Upstream's | 958, from 2026-08 onward | 2,879 | 6,238,487,406 | 99.44% |

Ours are the 24 on `umbra-buildathon` plus 6 on `main` that touch `umbra/`.
Upstream's span about forty branches and two agents, 345 Codex and 194 Claude
Code, from thirteen other contributors.

**The 6.27 GB is not ours and was inherited by forking.** It was in
`entireio/entire-graph` before this fork existed. I did not scan it and it is
not ours to act on. I did not verify that every byte of it is identical to
upstream's copy, only that none of it is attributable to our sessions.

Three files have ever been deleted from the branch, all upstream `prompt.txt`
files from August and early September. None of ours. So for our thirty
checkpoints the tip tree is the whole store, and there is nothing hiding in
history that is not also at the tip.

## What our 35 MB contains

Scanned with the patterns `scripts/checkpoint-refs-audit.sh` already uses, so
the numbers are comparable to the ones NOTES records, plus token shapes.

| Pattern | Lines | Note |
|---|---|---|
| absolute home path | 9,465 | 10,136 occurrences, all `/home/utkuputku` except 36 |
| operating-system user name | 9,840 | |
| email address | 174 | see below |
| GitHub token shape | 0 | |
| OpenAI token shape | 0 | |
| Slack token shape | 24 | not a token, see below |
| AWS access key shape | 24 | not a key, see below |
| private key header | 24 | not a key, see below |

### Addresses

| Address | Occurrences | What it is |
|---|---|---|
| `fixture@example.invalid` | 120 | invented, from the project's own fixtures |
| **`utkarshbahuguna10@gmail.com`** | **78** | **real** |
| `u003cutkarshbahuguna10@gmail.com` | 39 | **the same real address**, JSON-escaped `<` |
| `someone@example.com` | 36 | invented, the scrubber's test value |

**117 occurrences of the real address**, in two encodings. A search for the
plain form finds 78 of them and misses 39, which is worth knowing before anyone
tries to measure this with one grep.

### The token shapes are quotations, not credentials

All 24 of each appear in one passage, and it is NOTES.md quoting itself:

```
scrubber's own test inputs are token-shaped by design: `REDACTED`,
`sk-0123...`, `xoxb-1234567890-abcdefghij`, `AKIAIOSFODNN7EXAMPLE`
(Amazon's published example key), `BEGIN RSA PRIVATE KEY`, `/home/someone`,
`someone@example.com`
```

So the sessions that read or wrote that paragraph carried it into their
transcripts. `/home/someone` (36) and `someone@example.com` (36) are from the
same passage. **No real credential of any kind was found.**

### Nothing from any other project

This is the difference that matters, because it is the blocker NOTES names.

| Probe | Result |
|---|---|
| Names of the 15 other project directories on this machine | **0** |
| `/Desktop/` in any form | **0** |
| `/home/utkuputku/Desktop/` | **0** |
| `files (1)` | 42, and all of them are quotations |

The 42 `files (1)` hits are the sessions reading NOTES.md's own setup
paragraph, the one that begins "the working directory held only the four docs
in a folder named `files (1)`". That paragraph is already a committed file in
the public repository. It names no other project, and neither do the
transcripts.

NOTES records that one of the **private** repository's transcripts carries
verbatim content from an unrelated project, captured in the first minutes of
the build. **Nothing equivalent is in the fork.** The fork's Umbra sessions
began after that, on a machine directory that held only this work.

## Side by side with the private repository

Same patterns, same method, run today against both.

| | fork, ours only | private repository |
|---|---|---|
| Store | one branch, `entire/checkpoints/v1` | 65 refs under `refs/entire/checkpoints/` |
| Checkpoints | 30, from 3 sessions | 65 refs |
| Scannable text | 35,301,438 bytes | 576,685,015 bytes |
| Home path lines | 9,465 | 79,763 |
| User name lines | 9,840 | 81,541 |
| Email lines | 174 | 939 |
| Real address, occurrences | 117 | 591, across three encodings |
| Other-project material | **none found** | **present**, per NOTES |
| Published | **yes, on github.com** | no, repository is private |

The private repository holds sixteen times the text and five times the
identifying material, and it is the one that is not public. The fork holds less
and is.

## What this changes for the decision

The inventory listed "the fork is public and was never re-checked" as something
to decide rather than default. It is now checked, and the shape of the decision
is different from what NOTES assumes.

NOTES offers three options for the private repository's refs: delete them from
the remote and stop syncing, read them and accept what they contain, or rebuild
in a fresh repository with sync off. For the fork, **the third has already
expired**. The data has been on a public GitHub repository since 2026-09-06.
Rebuilding elsewhere does not unpublish it, and deleting the branch now removes
it from the default clone path but not from anything that already fetched it.

What is still worth deciding:

- Whether to stop the fork syncing further checkpoints, which is
  `entire configure --skip-push-sessions` and is a decision about the future
  rather than the past.
- Whether the 117 address occurrences and 10,136 home paths matter to you.
  They identify the author and the machine layout. They are not credentials
  and they are the same class of thing a commit author line publishes anyway,
  which the address already is.
- Whether deleting `entire/checkpoints/v1` from the fork is worth doing for the
  6.24 GB rather than for the privacy, since a clone currently pays for
  upstream's transcripts as well as ours.

None of that is acted on here.

## Method and its limits

Read-only throughout. Metadata for 970 checkpoints read with
`git cat-file --batch`, partitioned by the `branch` field and by whether
`files_touched` contains a path under `umbra/`. Sizes from
`git ls-tree -r --long --full-tree`. Content scanned with the same regular
expressions as `scripts/checkpoint-refs-audit.sh`, which counts lines rather
than matches, plus token shapes. Publication confirmed with
`git ls-remote https://github.com/u7k4rs6/entire-graph 'refs/heads/entire/*'`
and the GitHub API.

Four limits worth stating.

1. **I scanned our thirty checkpoints, not upstream's 958.** Upstream's 6.24 GB
   was inherited by forking and is upstream's to answer for. If the question is
   "what does a clone of this fork expose", that 6.24 GB is part of the answer
   and has not been characterised here.
2. **Pattern-based.** It finds what the patterns describe. The JSON-escaped
   address form was found only because the first pass reported a suspicious
   `u003c` prefix; there may be other encodings no pattern here matches.
3. **The tip tree, plus a deletion check.** Verified that none of our files
   were ever deleted from the branch, so the tip is complete for ours. Not
   verified for upstream's.
4. **`entire status` reports the fork syncs checkpoints to origin**, so this
   audit describes a moving target. It is accurate as of commit
   `207775b11b8e22ea2e921fda4afeead07f0e3bef`.
