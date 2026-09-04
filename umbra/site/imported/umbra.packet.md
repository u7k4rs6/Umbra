# Blind spot packet

Checkpoint `9b940c5e4916`, commit `0941dad`, parent `b77883b`, agent Claude Code, resolved via session window.

**0 of 4 dependents were examined (0%).** 0 are in penumbra, 4 are in full shadow.

> commit 0941dad carries no Entire-Checkpoint trailer, so it was paired with checkpoint 9b940c5 by session time

> the checkpoint was paired with this commit by session time and owns a commit of its own, so the change came from graph commit

> 21 changed entity(s) in languages the graph cannot resolve calls for were not treated as sources

## 1. What changed

| Symbol | Change | Where | Dependents |
|---|---|---|---|
| `main` | body changed | `certify/rmsnorm_artifact.py:206` | 5 |
| `block_width` | body changed | `certify/rmsnorm_trace.py:35` | 2 |
| `report/upstream-comment-51187.md` | body changed | `report/upstream-comment-51187.md:1` | 0 |

## 2. What was examined

How the session worked: 40 file reads, 122 edits, 0 searches, 1156 shell commands.

Tool activity in the session: 40 read, 122 edit, 40 quoted, 644 mention, 1156 command.

## 3. Ranked shadow

The 4 highest ranked, of 4 in shadow. Each score is the product of its printed factors.

### 1. `summarize_repeat` ○

- `certify/rmsnorm_trace.py:150`, direct caller at depth 1, score 7.8
- nothing in the session touched or mentioned certify/rmsnorm_trace.py
- factors: relation 3, dependents 2.585, state 1, farfield 1, faultline 1, test 0

### 2. `per_token_widths` ○

- `certify/sched_trace.py:73`, direct caller at depth 1, score 7.8
- nothing in the session touched or mentioned certify/sched_trace.py
- factors: relation 3, dependents 2.585, state 1, farfield 1, faultline 1, test 0

### 3. `analyse` ○

- `certify/rmsnorm_trace.py:168`, transitive caller at depth 2, score 5.0
- nothing in the session touched or mentioned certify/rmsnorm_trace.py
- transitive (reached through another symbol)
- factors: relation 1.5, dependents 3.322, state 1, farfield 1, faultline 1, test 0

### 4. `collect` ○

- `certify/rmsnorm_artifact.py:77`, transitive caller at depth 2, score 3.9
- nothing in the session touched or mentioned certify/rmsnorm_artifact.py
- transitive (reached through another symbol)
- factors: relation 1.5, dependents 2.585, state 1, farfield 1, faultline 1, test 0

## 4. Tests reaching the shadow

Tests were not run (`--run none`).

## 5. Results

## Reproduce

```
entire umbra 9b940c5e4916 --run none
```

## Limitations

- graph edges are incomplete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined
- read evidence is file and line-range based; a mention in the agent's text is attention, not reading, and is the weakest tier
- fault lines and beacons are matched by a regex window around the call site, not by a parser
- ranking weights are hand set and printed; they are a heuristic, not a measurement
- relations not traversed: CONFIGURES, CONSTRUCTS, EMITS, HANDLES_GRAPHQL, HANDLES_GRPC, HANDLES_ROUTE, HANDLES_TOOL, HANDLES_TRPC, HTTP_CALLS, LISTENS_ON, RESOURCE_DEPENDS_ON, SIMILAR_TO

<details><summary>Commands run</summary>

- `entire checkpoint list --json`
- `git -C <repo> rev-parse --verify 0941dad^{commit}`
- `git -C <repo> rev-list --parents -n 1 0941dade22e8237d27408d96d362c8d68812bfe1`
- `git -C <repo> log -1 --format=%B 0941dade22e8237d27408d96d362c8d68812bfe1`
- `git -C <repo> log -1 --format=%cI 0941dade22e8237d27408d96d362c8d68812bfe1`
- `git -C <repo> worktree add --detach <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/head 0941dade22e8237d27408d96d362c8d68812bfe1`
- `git -C <repo> worktree add --detach <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/base b77883b57483cc80d19fb79d6b5c7b7ff1d66d16`
- `git -C <repo> config --get user.name`
- `git -C <repo> config --get author.name`
- `git -C <repo> config --get committer.name`
- `entire graph capabilities --json`
- `entire graph snapshot --repo <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/head --format ndjson`
- `entire graph commit 0941dade22e8237d27408d96d362c8d68812bfe1 --repo <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/head --json`
- `entire graph impact --symbol main --repo <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/head --format json --file certify/rmsnorm_artifact.py --line 206`
- `entire graph impact --symbol block_width --repo <home>/.local/share/entire/plugins/data/umbra/wt/0941dade22e8237d27408d96d362c8d68812bfe1/head --format json --file certify/rmsnorm_trace.py --line 35`
- `entire checkpoint explain 9b940c5e4916 --raw-transcript`
- `entire checkpoint explain 9b940c5e4916 --short`

</details>
