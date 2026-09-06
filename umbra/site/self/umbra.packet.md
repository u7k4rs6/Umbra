# Blind spot packet

Checkpoint `01M1PY8KZ28HS7TG95ZVKQSZ2W`, commit `13adcdd`, parent `31fa246`, agent Claude Code, resolved via trailer.

**0 of 12 dependents were examined (0%).** 0 are in penumbra, 12 are in full shadow.

> 16 changed entity(s) in languages the graph cannot resolve calls for were not treated as sources

## 1. What changed

| Symbol | Change | Where | Dependents |
|---|---|---|---|
| `drawField` | body changed | `umbra/internal/report/assets/umbra.js:36` | 1 |
| `boot` | body changed | `umbra/internal/report/assets/umbra.js:508` | 0 |
| `tierRank` | added | `umbra/internal/report/assets/umbra.js:729` | 2 |
| `stateAt` | added | `umbra/internal/report/assets/umbra.js:736` | 1 |
| `consider` | added | `umbra/internal/report/assets/umbra.js:742` | 2 |
| `wireReplay` | added | `umbra/internal/report/assets/umbra.js:782` | 1 |
| `render` | added | `umbra/internal/report/assets/umbra.js:822` | 16 |
| `goTo` | added | `umbra/internal/report/assets/umbra.js:835` | 3 |
| `step` | added | `umbra/internal/report/assets/umbra.js:840` | 4 |
| `play` | added | `umbra/internal/report/assets/umbra.js:847` | 1 |
| `pause` | added | `umbra/internal/report/assets/umbra.js:857` | 3 |
| `tick` | added | `umbra/internal/report/assets/umbra.js:864` | 6 |
| `anyEvidence` | added | `umbra/internal/report/assets/umbra.js:898` | 2 |
| `indexOf` | added | `umbra/internal/report/assets/umbra.js:906` | 8 |
| `eventAt` | added | `umbra/internal/report/assets/umbra.js:914` | 2 |
| `describe` | added | `umbra/internal/report/assets/umbra.js:919` | 2 |
| `applyState` | added | `umbra/internal/report/assets/umbra.js:943` | 2 |
| `rewireNodes` | added | `umbra/internal/report/assets/umbra.js:975` | 1 |
| `wireSpotlight` | added | `umbra/internal/report/assets/umbra.js:1055` | 1 |
| `toViewBox` | added | `umbra/internal/report/assets/umbra.js:1082` | 1 |
| `revealNear` | added | `umbra/internal/report/assets/umbra.js:1093` | 1 |
| `wireKeyboard` | added | `umbra/internal/report/assets/umbra.js:1103` | 1 |
| `focusAt` | added | `umbra/internal/report/assets/umbra.js:1109` | 1 |
| `toggleLabels` | added | `umbra/internal/report/assets/umbra.js:1134` | 1 |
| `Build` | body changed | `umbra/internal/shadow/build.go:27` | 4 |
| `AddCoChange` | body changed | `umbra/internal/shadow/build.go:96` | 0 |
| `exposuresFor` | added | `umbra/internal/shadow/build.go:185` | 2 |
| `jsNode` | added | `umbra/internal/shadow/jsmirror_test.go:20` | 3 |
| `jsExposure` | added | `umbra/internal/shadow/jsmirror_test.go:26` | 3 |
| `jsCase` | added | `umbra/internal/shadow/jsmirror_test.go:32` | 1 |
| `TestJavaScriptMirrorsTheGoClassifier` | added | `umbra/internal/shadow/jsmirror_test.go:51` | 0 |

## 2. What was examined

How the session worked: 3 file reads, 5 edits, 0 searches, 285 shell commands.

> This session worked almost entirely through the shell: 8 file tool events against 285 shell commands. Umbra builds the examined set from file tool events, so a mostly umbra report is the expected outcome here and says little about the change.

Tool activity in the session: 3 read, 5 edit, 1 quoted, 154 mention, 285 command.

## 3. Ranked shadow

The 12 highest ranked, of 12 in shadow. Each score is the product of its printed factors.

### 1. `runScenario` ○

- `umbra/internal/shadow/scenarios_test.go:87`, direct caller at depth 1, score 15.5
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- factors: relation 3, dependents 5.17, state 1, farfield 1, faultline 1, test 0

### 2. `TestScenarioBeacon` ○

- `umbra/internal/shadow/scenarios_test.go:252`, direct caller at depth 1, score 9.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function)
- factors: relation 3, dependents 2, state 1, farfield 1, faultline 1, test 3

### 3. `hasMod` ○

- `umbra/internal/report/assets/umbra.js:210`, data flow at depth 1, score 7.2
- nothing in the session touched or mentioned umbra/internal/report/assets/umbra.js
- factors: relation 2, dependents 3.585, state 1, farfield 1, faultline 1, test 0

### 4. `nodeGroup` ○

- `umbra/internal/report/assets/umbra.js:253`, data flow at depth 2, score 7.2
- nothing in the session touched or mentioned umbra/internal/report/assets/umbra.js
- transitive (reached through another symbol)
- factors: relation 2, dependents 3.585, state 1, farfield 1, faultline 1, test 0

### 5. `styleEdge` ○

- `umbra/internal/report/assets/umbra.js:189`, data flow at depth 2, score 6.6
- nothing in the session touched or mentioned umbra/internal/report/assets/umbra.js
- transitive (reached through another symbol)
- factors: relation 2, dependents 3.322, state 1, farfield 1, faultline 1, test 0

### 6. `TestScenarioClassic` ○

- `umbra/internal/shadow/scenarios_test.go:158`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 7. `TestScenarioEcho` ○

- `umbra/internal/shadow/scenarios_test.go:208`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 8. `TestScenarioEverythingLit` ○

- `umbra/internal/shadow/scenarios_test.go:142`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 9. `TestScenarioLeak` ○

- `umbra/internal/shadow/scenarios_test.go:219`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 10. `TestScenarioNoReads` ○

- `umbra/internal/shadow/scenarios_test.go:192`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 11. `TestScenarioPartialRead` ○

- `umbra/internal/shadow/scenarios_test.go:172`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

### 12. `TestScenarioPreChange` ○

- `umbra/internal/shadow/scenarios_test.go:181`, transitive caller at depth 2, score 6.0
- nothing in the session touched or mentioned umbra/internal/shadow/scenarios_test.go
- test (a test function); transitive (reached through another symbol)
- factors: relation 1.5, dependents 2, state 1, farfield 1, faultline 1, test 3

## 4. Tests reaching the shadow

Tests were not run (`--run none`).

## 5. Results

## Reproduce

```
entire umbra 01M1PY8KZ28HS7TG95ZVKQSZ2W --run none
```

## Limitations

- graph edges are incomplete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined
- read evidence is file and line-range based; a mention in the agent's text is attention, not reading, and is the weakest tier
- fault lines and beacons are matched by a regex window around the call site, not by a parser
- ranking weights are hand set and printed; they are a heuristic, not a measurement
- relations not traversed: CONFIGURES, CONSTRUCTS, EMITS, HANDLES_GRAPHQL, HANDLES_GRPC, HANDLES_ROUTE, HANDLES_TOOL, HANDLES_TRPC, HTTP_CALLS, LISTENS_ON, RESOURCE_DEPENDS_ON, SIMILAR_TO

<details><summary>Commands run</summary>

- `entire checkpoint list --json`
- `git -C <repo> rev-parse --verify 13adcdd^{commit}`
- `git -C <repo> rev-list --parents -n 1 13adcddaa88b36e7b6df56bb0d5f183be3957a30`
- `git -C <repo> log -1 --format=%B 13adcddaa88b36e7b6df56bb0d5f183be3957a30`
- `git -C <repo> worktree add --detach <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head 13adcddaa88b36e7b6df56bb0d5f183be3957a30`
- `git -C <repo> worktree add --detach <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/base 31fa246c254c447e3672286112684632bd7a657b`
- `git -C <repo> config --get user.name`
- `git -C <repo> config --get author.name`
- `git -C <repo> config --get committer.name`
- `entire graph capabilities --json`
- `entire graph snapshot --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format ndjson`
- `entire graph checkpoint 01M1PY8KZ28HS7TG95ZVKQSZ2W --json`
- `entire graph impact --symbol drawField --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 36`
- `entire graph impact --symbol boot --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 508`
- `entire graph impact --symbol tierRank --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 729`
- `entire graph impact --symbol stateAt --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 736`
- `entire graph impact --symbol consider --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 742`
- `entire graph impact --symbol wireReplay --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 782`
- `entire graph impact --symbol render --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 822`
- `entire graph impact --symbol goTo --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 835`
- `entire graph impact --symbol step --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 840`
- `entire graph impact --symbol play --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 847`
- `entire graph impact --symbol pause --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 857`
- `entire graph impact --symbol tick --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 864`
- `entire graph impact --symbol anyEvidence --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 898`
- `entire graph impact --symbol indexOf --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 906`
- `entire graph impact --symbol eventAt --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 914`
- `entire graph impact --symbol describe --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 919`
- `entire graph impact --symbol applyState --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 943`
- `entire graph impact --symbol rewireNodes --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 975`
- `entire graph impact --symbol wireSpotlight --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1055`
- `entire graph impact --symbol toViewBox --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1082`
- `entire graph impact --symbol revealNear --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1093`
- `entire graph impact --symbol wireKeyboard --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1103`
- `entire graph impact --symbol focusAt --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1109`
- `entire graph impact --symbol toggleLabels --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/report/assets/umbra.js --line 1134`
- `entire graph impact --symbol Build --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/build.go --line 27`
- `entire graph impact --symbol AddCoChange --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/build.go --line 96`
- `entire graph impact --symbol exposuresFor --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/build.go --line 185`
- `entire graph impact --symbol jsNode --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/jsmirror_test.go --line 20`
- `entire graph impact --symbol jsExposure --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/jsmirror_test.go --line 26`
- `entire graph impact --symbol jsCase --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/jsmirror_test.go --line 32`
- `entire graph impact --symbol TestJavaScriptMirrorsTheGoClassifier --repo <home>/.local/share/entire/plugins/data/umbra/wt/13adcddaa88b36e7b6df56bb0d5f183be3957a30/head --format json --file umbra/internal/shadow/jsmirror_test.go --line 51`
- `entire checkpoint explain 01M1PY8KZ28HS7TG95ZVKQSZ2W --raw-transcript`
- `entire checkpoint explain 01M1PY8KZ28HS7TG95ZVKQSZ2W --short`

</details>
