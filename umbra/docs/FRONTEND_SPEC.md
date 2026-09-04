# Umbra: frontend specification

Umbra's report is an eclipse map. Each changed symbol is a light source at the centre; its dependents sit on rings by graph distance; the ones the agent read are lit, the ones it half-saw are in penumbra, the ones it never opened sit in pools of shadow. The map is generated from the report JSON and lives in one self-contained HTML file with no external requests. It can be replayed: press play and the session's reads light the map in the order the agent looked, the edits ripple outward along the edges, and the map settles on the eclipse, which is the state at commit. A docket beside the map lists the shadowed nodes by score, and a detail panel shows the evidence behind any node. The landing page carries the same map live, plus a viewer that renders any `umbra.json` dropped onto it, client-side.

This document is the build spec for both surfaces, the terminal rendering, and the tokens they share. Where it is specific it is meant to be followed exactly; where it leaves a choice it says so.

## Design direction

The subject is light and shadow, so the visual language is light and shadow, and nothing else. There are no cards, no panels with drop shadows, no gradients as decoration, no icon set, no numbered markers except on the replay timeline, which is a real sequence. Colour is spent entirely on illumination and test outcomes. The type is quiet and the map is loud.

The one memorable thing on the report is the light field itself: a warm glow around each source with soft dark pools where the unexamined dependents are, so the reader sees the blind spots before reading a single label. Everything else on the page is disciplined around that.

The page is dark by default because the subject is shadow and lit things must read as lit. This is a content decision, and it is kept honest by two things: the light scheme exists and is a first-class rendering (paper with ink shadows), and the dark scheme uses a blue-black night with a warm light ramp, not a neutral near-black with one neon accent.

## Vocabulary

The page speaks one language, drawn from light and shadow, and every coined word appears next to its plain meaning the first time it shows in a panel (as a parenthetical in prose, as a `title` and visible sublabel in chips). The JSON and the terminal use the plain names. This keeps the page distinctive without making a judge guess.

| Word | Plain meaning | Where it appears |
|---|---|---|
| Source | The changed symbol; the light in the map | Map centre, header, packet section 1 |
| Field | Everything the change can reach: the dependents Graph found | Header, detail panel path, packet |
| Exposure | Any moment the session touched a file: read, grep, glob, edit, quoted content, mention | Detail panel list, replay ticks |
| Lit | Read fully after the cut, or edited | State word, docket, map |
| Penumbra | Half seen; one of the tiers below | State word |
| Umbra | Never touched or mentioned | State word |
| Unknown | The transcript cannot tell | State word, flat map |
| Afterimage | Read fully, but only before the cut; the picture the agent holds is of the old code | Penumbra tier chip |
| Glance | A partial read whose range missed the symbol | Penumbra tier chip |
| Glimpse | Only a grep or glob hit on the file | Penumbra tier chip |
| Quoted | The file's content appeared inside a tool result, for example a diff | Penumbra tier chip |
| Echo | The agent named the path or symbol in its own words and never opened it | Penumbra tier chip |
| The cut | The first edit to a source file; the moment the change begins | Replay timeline, detail sentences |
| Sweep | The attention replay: the session's gaze sweeping the field | Replay control label |
| Eclipse | The final state at commit: what is still dark when the light source is done | Map label at the end of the sweep, page title of the report |
| Pool | A dark region where several umbra nodes sit together | Map |
| Halo | The glow around a lit node | Map |
| Torch | The spotlight cursor that reveals labels in shadow | Map hint text |
| Docket | The ranked list of shadowed nodes | Column heading |
| Probe | A test selected because it reaches shadow | Docket tests column, packet section 4 |
| Crack | A probe that failed | Node motif, docket outcome word |
| Full sweep | The audit run of the entire suite after the probes | Header, packet section 5 |
| Leak | A test the full sweep found that selection missed, with its reason | Header count, docket filter, packet section 5 |
| Far field | A dependent across a package or directory boundary from its source | Modifier chip |
| Fault line | A call site inside error handling | Modifier chip |
| Beacon | An annotated call site (`SAFETY`, `CRITICAL`, `INVARIANT`) pinned to the top | Modifier chip, docket pin marker |
| Scar | The opt-in history factor: a file with many recent fixes | Modifier chip when `--history` is on |
| Session said | The agent's own summary sentence, displayed above the map | Header |

Chip rendering: the coined word in 12px, the plain meaning in 10.5px `--ink-2` directly under it, both inside one chip with a hairline border, no fill. Chips are separated by whitespace, never by dot characters.

## Tokens

Colour, night scheme (default):

- `--night: #10141C` page and map background
- `--ink: #D9DBE1` primary text
- `--ink-2: #8B909C` secondary text, ring guides
- `--rule: #262B35` hairlines, table rules
- `--lit: #FFF1C9` lit node fill
- `--lit-glow: #F3C46B` light field, lit edges
- `--penumbra: #8E7B57` penumbra fill (half disc)
- `--umbra: #2B303A` umbra node fill
- `--umbra-edge: #4A5060` hatch strokes on umbra nodes and unlit edge traces
- `--unknown: #5A6070` dashed outlines when the examined set is unavailable
- `--fail: #C8553D` failing test crack strokes and outcome text
- `--pass: #8FBF9F` passing test ring and outcome text
- `--focus: #9CC4FF` focus rings and links

Colour, day scheme (`prefers-color-scheme: light`, and forced for print):

- `--night: #FBFBFA` (the name is kept so the CSS does not fork)
- `--ink: #1C1F26`, `--ink-2: #5E6470`, `--rule: #DAD9D4`
- `--lit: #FFFFFF` with halo `--lit-glow: #E8B85A`
- `--penumbra: #B9AE93`
- `--umbra: #1C1F26` with hatch `--umbra-edge: #6B7080`
- `--unknown: #9AA0AC`
- `--fail: #B3452F`, `--pass: #3E8A5C`, `--focus: #1A4F9C`

Type (system stacks only; the report must not fetch fonts):

- UI text: `ui-sans-serif, system-ui, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`
- Symbol names, paths, commands, test IDs: `ui-monospace, "SF Mono", Menlo, Consolas, "Liberation Mono", monospace`
- Scale: body 14px at 1.5; docket rows 13px; detail panel 14px; section headings 18px weight 600; the source name at the map centre 22px monospace; map labels 12px monospace; file paths under labels 10.5px.
- Weight 400 everywhere except headings and state words at 600. No all-caps labels anywhere. No letter-spacing tricks.

Layout:

- Report: two columns above 1100px, map 62% and docket 38%, with a 1px `--rule` between. Below 1100px, map on top at 16:10, docket below. Detail panel overlays the docket column (desktop) or slides up from the bottom (narrow).
- Page padding 20px; no max width on the map column; docket text measure capped at 72 characters.
- Landing page: single column, 960px max, left-aligned, the map full-bleed inside its section.

Motion:

- Nothing moves on load. The page renders in its final state (the eclipse).
- Motion happens only in response to a user action: play, step, scrub, select, hover.
- Every tween is 180 to 400 milliseconds with an ease-out curve; nothing loops.
- `prefers-reduced-motion: reduce` turns tweens into instant state changes and replaces the edit ripple with a one-step edge highlight. The spotlight cursor is disabled.

## Page anatomy

```
┌─────────────────────────────────────────────────────────────────────────────┐
│ Umbra  checkpoint a1b2c3d4e5f6  commit 3f9c2e1  parent 8b1d0a7  claude-code  │
│ Session said  "Checked the callers of compute_total; all tests pass."       │
│ ◑ 3 of 7 dependents lit     2 probes cracked     full sweep, 0 leaks        │
├─────────────────────────────────────────┬───────────────────────────────────┤
│                                         │ Docket   sort: score  filter: all │
│            ○ test_rounding              │ ○ test_rounding    tests/test_s… │
│        ╌╌╌╌╌╌╌╌╌╌╌╌                     │   direct caller · depth 1 · 6.0   │
│    ◐ apply_refund     ● handle_order    │   fail  assert total == 10.5      │
│        ╲                 ╱              │ ○ format_receipt   app/report.py  │
│          ╲   compute_total   ╱          │ ○ RefundLine       app/models.py  │
│            ╲   (signature)  ╱           │ ◐ apply_refund     app/refunds.py │
│              ●            ●             │   grep hit only                   │
│     ○ RefundLine    ● test_order_total  │ ● handle_order  ● test_order_…    │
│                                         │ ● Invoice.total                   │
│  ○ format_receipt                       │                                   │
├─────────────────────────────────────────┴───────────────────────────────────┤
│ Sweep    ▶  ◀◀ ▶▶   1x 4x   ├──●──●──◌─◆──●──────▮──────────┤  seq 14 / 31  │
│          read service.py   grep compute_total   the cut   pytest…           │
├─────────────────────────────────────────────────────────────────────────────┤
│ Limitations. Reproduce: entire umbra a1b2c3d4e5f6 --test "pytest -q"         │
└─────────────────────────────────────────────────────────────────────────────┘
```

Narrow (under 1100px): header, map (full width), replay strip, docket, footer. The detail panel becomes a bottom sheet.

The row separator in the docket wireframe above uses a middle dot only in this diagram; in the real page, chips are separated by whitespace and a hairline, never by dot characters.

## The map

### Geometry

Layout is computed in Go (`internal/report/layout.go`) and written into `umbra.json` under `layout`, so the report and the landing-page viewer render identical positions from the same numbers. The viewBox is 1200 by 800; the map scales to its column and preserves aspect ratio.

- Sources. One source sits at the origin. Two to four sources sit on an inner ring of radius 56 at evenly spaced angles starting at 90 degrees (top). More than four collapse into one disc at the origin labelled with the count, with the individual sources selectable from the header; selecting one re-lays the map around it.
- Rings. Depth 1 at radius 170, depth 2 at 280, depth 3 at 380. The co-change ring is at 460 and drawn dashed. Ring guides are 1px `--rule` circles at 0.5 opacity; the depth number sits at the ring's top in 10.5px `--ink-2`.
- Angular placement. Nodes on a ring are grouped by file; groups are ordered by the group's highest score, descending, starting at 90 degrees and proceeding clockwise; inside a group, nodes are ordered by span. Angular budget per ring is 360 degrees minus a 6 degree gap between groups; nodes within a group are spaced evenly in the group's share, where each group's share is proportional to its node count with a minimum of 12 degrees per node. The result is deterministic; the same JSON always draws the same map.
- Overflow. A ring holding more than 28 nodes shows labels for the top 10 by score and hides the rest until hover, focus or the labels toggle. Above 150 rendered nodes in total, depth 3 and co-change nodes are collapsed into per-file clusters drawn as a larger hatched disc with the count, expanding in place on click.
- Edges. Each node draws one edge to the previous node on its shortest path to a source (depth 1 nodes draw to the source). Edges are straight. Co-change nodes draw a dashed edge to the source.

### The light field

This is the part that must look right. Draw it in this order, all inside one `<svg>`:

1. Background rectangle in `--night`.
2. Ring guides.
3. Light field. For each source, a circle of radius 500 filled with a radial gradient from `--lit-glow` at 0.32 opacity at the centre to 0 at the edge. The whole light group is masked.
4. Shadow mask. The mask is white (light shows) with black blobs subtracting light: for each umbra node a circle of radius 42 with a Gaussian blur of 14; for each penumbra node a circle of radius 28 at 55 percent black with the same blur; for a cluster, radius 60. Where several umbra nodes sit together the pools merge into one dark region, which is exactly the point. Unknown state draws no mask and no light field: the map goes flat.
5. Edges, drawn under the nodes. Edges to lit nodes are `--lit-glow` at 0.55 opacity, 1.25px. Edges to penumbra nodes are the same colour at 0.3. Edges to umbra nodes are `--umbra-edge` dotted (2 on, 3 off) at 0.45. Hovering or selecting a node raises its whole path to sources to 1.0 opacity and 1.75px.
6. Nodes.
7. Labels.

### Node anatomy

- Function or method: disc radius 7. Class or type: radius 9. Test: radius 7 with a second ring at radius 10.5, 1px, in the node's state colour.
- Lit: filled `--lit`, with a soft outer glow (a blurred copy of the disc in `--lit-glow` at 0.6 opacity, radius plus 6).
- Penumbra: a half disc. The lit half faces the source; the shadow half is `--penumbra`. Implemented as two path halves with the rotation set from the node's angle to the nearest source. The tier is drawn on the disc so it survives greyscale: afterimage adds a thin dotted ring at radius plus 3; glance keeps the plain half disc; glimpse reduces the lit half to a quarter; quoted draws the lit half with a 1px inset line; echo draws the whole disc in `--penumbra` at 0.5 opacity with a solid outline, since nothing was read at all.
- Modifiers from the call site: far field draws a short 6px tick on the edge just outside the node, pointing away from the source; fault line draws the edge's last 12px as a zigzag; a beacon node gets a small 4px filled triangle above it in `--lit` and is always labelled, whatever the overflow rule says.
- Umbra: filled `--umbra` with a 45 degree hatch pattern in `--umbra-edge` (pattern tile 4px, stroke 1px) and a solid 1px `--umbra-edge` outline.
- Unknown: no fill, 1px dashed outline in `--unknown`.
- Test outcome. Fail: two short crack strokes across the disc in `--fail`, 1.5px, drawn as a fixed polyline pair so every cracked node looks the same. Pass: the test ring turns `--pass`. Not run: no change.
- Selected: a 2px `--focus` ring at radius plus 5. Focused via keyboard: the same ring, dashed.

### Labels

Symbol name in 12px monospace beside the node, placed outward from the ring: nodes on the right half get left-aligned labels to their right; nodes on the left half get right-aligned labels to their left; nodes near the top or bottom get centred labels above or below. Lit labels use `--ink`; umbra and penumbra labels use `--ink-2`; on a dark pool, labels get a 1px `--night` paint-order stroke so they stay legible. File path appears under the label at 10.5px on hover, focus or selection. Labels never overlap: if two labels would collide, the lower-scored one is hidden until interaction. The source label is 22px monospace at the centre with the change kind under it in 11px `--ink-2` (`signature changed`, `body changed`, `removed`).

### Spotlight

On pointer devices without reduced motion, the pointer carries a torch: a radial gradient disc of radius 110 in `--lit-glow` at 0.18 opacity, blended with `mix-blend-mode: screen`, following the cursor over the map with no lag. Umbra labels inside the torch radius show even when hidden by the overflow rule, and the pool under the torch lightens slightly. The torch is decoration in service of exploration: it reveals what is in shadow without changing any data. On touch devices it is off; the keyboard equivalent is that a focused node reveals the labels of its ring neighbours.

## Attention replay

The strip below the map, labelled `Sweep` with the sublabel `replay the session's attention`, is a timeline of the session's events from the `timeline` array in the JSON: reads, greps and globs, quoted content, mentions, edits, commands. Loading the page shows the final state with the playhead at the end. Nothing plays until the reader presses play, step, or scrubs.

Controls, left to right: play/pause (Space), step back and step forward (`[` and `]`), speed 1x and 4x, the timeline, and a counter `seq 14 of 31`. The timeline is a horizontal rule with a tick per event: reads as small filled circles, greps and globs as hollow circles, mentions as dotted circles, quoted content as half circles, edits as diamonds, commands as short vertical bars. The `t0` edit is a larger diamond with the label `the cut` and the sublabel `change begins`. The playhead is a vertical line; drag it, click a tick, or use the arrow keys when the timeline is focused. Under the timeline, one line of text names the current event: `read app/service.py  lines 1 to 80`, `grep compute_total  hit app/api.py, app/refunds.py`, `mentioned apply_refund`, `the cut  edit app/service.py`, `pytest tests/test_api.py`.

State reconstruction at seq k: every node's state is recomputed from exposures with `Seq <= k` using the same rules as the classifier, which the JS re-implements in about forty lines. The light field, masks, node fills and edge styles update to match. Because the rules include `T0`, a node read before the cut shows as lit until the playhead passes the cut, then dims to an afterimage with its dotted ring; that transition is the most instructive moment of the sweep and should be visible at 1x. A mention lights a node only to the echo tier, visibly weaker than a read.

Event animations, all user-triggered by play or step:

- Read covering a node: the node crossfades to lit over 240 milliseconds and its pool in the mask fades out over the same time.
- Partial read, grep, glob, quoted content, mention: the node becomes the matching penumbra tier over 240 milliseconds; the pool shrinks to the penumbra size (echo shrinks it least).
- Edit on a source file: the source disc pulses once (radius plus 4 and back, 320 milliseconds) and a stroke travels outward along every edge from that source over 400 milliseconds. On the cut the label `the cut` appears under the source for the rest of the sweep.
- Command: the command text appears under the timeline; no map change.
- At the end: the playhead stops, the counter reads the last seq, and the label `eclipse` with the sublabel `what was still dark at commit` appears at the top of the map for two seconds and then stays at 0.4 opacity.

Reduced motion: no crossfades or pulses; states switch at once; the travelling stroke becomes a one-step highlight of the affected edges for the duration of that step.

Speed: at 1x each event occupies 600 milliseconds regardless of real elapsed time; at 4x, 150. Real timestamps are shown in the counter when present.

## Docket

The docket is a real HTML table rendered by the Go template, present without JavaScript, and enhanced by it.

Columns: state glyph and word, symbol, file and line, relation and depth, modifiers, score, probes and outcome. Beacons are pinned above the sorted rows with a small triangle marker and the annotation line in `--ink-2`. Rows are sorted by score by default; the header offers sort by file and by depth. A filter row offers: shadowed only (default), all, probes only, cracks only, leaks, and a file filter that lists files with counts. Lit nodes are hidden under the default filter and summarised in a single line at the bottom: `3 lit dependents hidden`. Leaks appear as their own rows with the state cell reading `leak` and the modifiers cell carrying the reason sentence.

Row content:

- State: the glyph (● ◐ ○ ?) followed by the word; the crack glyph `⨯` in `--fail` after the state when a probe failed.
- Modifiers: chips as defined in Vocabulary (afterimage, glance, glimpse, quoted, echo, far field, fault line, beacon, scar), each with its plain meaning under the word.
- Symbol: monospace, with the kind in `--ink-2` after it for classes and types.
- File: monospace, path relative to the repository, then `:line`.
- Relation: plain words, `direct caller`, `caller via handle_order`, `type consumer`, `data flow`, `co-change`; depth as `depth 2`.
- Score: the number to one decimal; on hover or focus a small popover lists all six factors (`relation 3 × dependents 1.0 × state 1.0 × far field 1.5 × fault line 1.0 + test 3`, plus `scar` when enabled), so the number can be recomputed by hand.
- Probes: count reaching this node and the worst outcome; for test nodes, the outcome and a 60 character failure excerpt.

Hovering a row highlights its node and path on the map; selecting a row (click or Enter) opens the detail panel and selects the node. Selecting a node on the map scrolls the docket to its row and highlights it. Filters and sorts apply to the map's labels too: filtered-out nodes keep their discs but drop their labels, so the map never lies about what exists.

## Detail panel

Opens on selection; closes with Escape or the close control; only one is open at a time. Content, top to bottom:

1. Symbol name in 18px monospace, kind and span in `--ink-2` beneath: `function · app/report.py lines 88 to 104` (in the real page the separator is two spaces, not a dot).
2. State sentence, in plain words, one of:
   - `Lit. Read fully at seq 21 (10:47), after the cut at seq 14.`
   - `Penumbra, glance. Read lines 1 to 80 at seq 6; the symbol spans 88 to 104. No later read.`
   - `Penumbra, glimpse. Appeared only as a grep hit at seq 9.`
   - `Penumbra, quoted. Its content appeared in a tool result at seq 17 (git diff); it was never opened.`
   - `Penumbra, afterimage. Read fully at seq 6, before the cut at seq 14; not read again.`
   - `Penumbra, echo. Named in the agent's text at seq 11; never opened.`
   - `Umbra. Nothing in the session touched or mentioned app/report.py.`
   - `Unknown. This transcript carries no tool events and no mentions.`
   Followed, when present, by the call-site facts: `Far field: app/report is a different package from app/service.` `Fault line: the call sits inside an except block at line 92.` `Beacon: line 90 reads "# SAFETY: totals must round half up".`
3. Path to the source as a chain: `compute_total  ←  handle_order  ←  format_receipt`, each link selectable.
4. Exposures list when any exist: seq, time, kind, range.
5. Probes: each test reaching the node with its outcome and, for cracks, the 300 character excerpt in a `<pre>`. For a leak row, the reason sentence and the path the sweep found, as a selectable chain.
6. Score factors as a stacked horizontal bar drawn in CSS (no chart library), one segment per factor, labelled inline with its value; for a beacon, the bar is replaced by the text `pinned to the top by a beacon`.
7. Reproduce: the commands a reader can run, monospace, each with a copy control:
   - `entire graph impact --symbol compute_total --repo <worktree>`
   - `entire graph def format_receipt --repo <worktree>`
   - `pytest -q tests/test_service.py::test_rounding`
8. If `--snippets` was used, the declaration line of the symbol, 200 characters max.

## Header

Line one, plain text: `Umbra 0.1.0  checkpoint a1b2c3d4e5f6  commit 3f9c2e1  parent 8b1d0a7  agent claude-code  depth 2`. Line two is the session's own sentence, labelled `Session said` and quoted in 16px monospace; when no stored summary exists the line is omitted rather than invented. Line three carries three facts:

- The eclipse glyph: a 22px disc in `--lit` overlapped by a `--umbra` disc offset so the covered fraction equals `1 - illumination`, followed by `3 of 7 dependents lit`. The glyph has `role="img"` and `aria-label` with the same words. When the state is unknown the glyph is a dashed outline and the text reads `examined set unavailable for this transcript`.
- Probes: `4 probes, 2 cracked` or `probes not run`.
- Full sweep: `full sweep, 0 leaks` or `full sweep, 2 leaks` (listed in the docket under the `leaks` filter, each with its reason) or `no sweep` when `--no-audit` was passed, or `sweep cut short` on timeout.

Degradation notices from the JSON `inputs` block appear as a fourth line in `--ink-2`, one sentence each: `Read ranges unavailable; partial reads counted as full.` `Call-site lines unavailable; fault lines and beacons not detected.` `Examined set built from mentions only.` `Relations ignored: IMPLEMENTS.`

## Footer

Three short paragraphs in `--ink-2`: limitations (from the JSON), reproduce (the exact `entire umbra` command and the `commands_run` list in monospace), and one line: `Layout and states are computed from the checkpoint and the graph; no model was involved.`

## States

- No dependents: the map shows the source alone with the text `No dependents in the graph at depth 2. Try --depth 3.` under it; the docket says the same.
- Everything lit: the light field has no pools; the header reads `7 of 7 dependents lit`; the docket's default filter shows the summary line and offers `all`.
- Unknown (no tool events, no mentions): flat map, dashed nodes, docket ranked by impact with the score factor `state` shown as `unknown`; header and a notice explain; the sweep strip shows only edits and commands; the full sweep and its leaks become the main event and the header says `selection by field only; the sweep found N leaks`.
- Mentions only: the map has no lit nodes and a ring of echo nodes at half opacity; the header says `examined set built from the agent's words only`.
- Clusters: expanded clusters remember their state for the page session.
- No JavaScript: the header, docket table, detail content as expandable `<details>` per row, and footer render fully; in place of the map, a paragraph: `The map needs JavaScript. The docket below lists every shadowed dependent.`
- Print: day scheme forced; map static at final state with all labels shown; docket expanded with all filters off; replay strip and spotlight hidden; page breaks avoided inside rows.

## Keyboard map and focus order

Focus order: header controls (source selector if any), map (one tab stop), replay controls, docket filters, docket rows, footer links.

Inside the map: `Tab` enters the map and focuses the top-scored shadowed node; `Up` and `Down` move through docket order; `Left` and `Right` move around the current ring; `Enter` opens the detail panel; `Escape` closes it or leaves the map; `L` toggles all labels; `Space` plays or pauses the replay from anywhere on the page unless a text field is focused; `[` and `]` step the replay; `Home` and `End` jump the playhead.

Every focused element shows a 2px `--focus` ring with 2px offset. Nothing is reachable by pointer only.

## Accessibility

- The `<svg>` has `role="group"` and an `aria-label` describing the map in one sentence; each node is a `<g role="button" tabindex="-1">` with `aria-label` `test_rounding, test, umbra, depth 1, failed` and `aria-pressed` when selected.
- The docket table has a caption, `scope="col"` headers, and is the screen-reader path to the same information; the map is never the only path.
- State is never carried by colour alone: glyphs, hatch, half discs and dashed outlines distinguish the four states in greyscale, and the state word appears in the docket and the detail panel.
- The replay's current event is announced through an `aria-live="polite"` region (the text under the timeline).
- Contrast: all text on `--night` meets 4.5:1; labels on pools use the paint-order stroke; the day scheme is checked separately.
- Reduced motion honoured as specified. Touch targets on the docket and replay are at least 40px tall.

## Landing page

`site/index.html` with `site/umbra.css` and `site/umbra.js` (the same CSS and JS as the report, copied at build time by `go run ./internal/report/gen-site` so the two cannot drift), and `site/sample/umbra.json` from the `classic` scenario. No fonts, no analytics, no requests beyond its own files.

Sections and copy:

1. Title block. `Umbra` at 40px, then: `Umbra shows the code an AI agent's change affects but the agent never looked at. It runs as a plugin for the Entire CLI, takes the dependents from Entire Graph, subtracts what the session actually read according to the checkpoint, and runs the tests that reach whatever is left in shadow.`
2. The map, live, full section width, with the session's sentence above it and the sweep strip under it, and one line: `This is a real report from the fixture app in this repository. The agent said it checked the callers. Press play to watch where it actually looked. The two cracked nodes are tests it never opened.`
   Below the fixture map, a second map from one of the builder's own past sessions imported with `entire import`, introduced with: `This one is not seeded. It is a real session from another project, imported into Entire.`
3. Drop zone. A dashed-outline region, `Drop an umbra.json here to view it. Nothing is uploaded; the map is drawn in your browser.` Also a file input for keyboard users. On drop, the page replaces the sample map with the reader's report and shows a `back to the sample` control.
4. What the states mean. The state table from the PRD with the glyphs drawn in SVG at 16px, so they match the map, followed by the penumbra tiers as chips with their plain meanings.
5. How ranking works. The six factors in plain words, the beacon override, the opt-in scar, and one worked example from the sample recomputed by hand. Then one paragraph on the full sweep and what a leak's reason tells you.
6. Install and run:
   ```
   go install github.com/<owner>/entire-graph/umbra/cmd/entire-umbra@main
   entire umbra HEAD --test "pytest -q" --out ./umbra-out
   open ./umbra-out/umbra.html
   ```
   Then: `Requires the Entire CLI with Checkpoints enabled and the entire-graph plugin. Works offline. There is no model anywhere in Umbra.`
7. Reproduce the demo. Clone, `cd umbra/fixtures/app`, the demo checkpoint id, the command. State that the fixture is seeded to produce lit, penumbra and umbra nodes and two failing shadowed tests.
8. Limitations, verbatim from the PRD.
9. Footer: `Built during Bengaluru Tech Week on the Entire ecosystem. Source, checkpoints and the session record are in the repository.` with the repository link. No badges.

The map is the hero and the only large element. Everything else is body text and one table.

## Terminal rendering

```
Umbra  a1b2c3d4e5f6  3f9c2e1  claude-code  depth 2
session said  "Checked the callers of compute_total; all tests pass."
compute_total  signature changed  app/service.py:40
light  ●●●◐○○○   3 lit  1 penumbra  3 umbra  0 unknown

 ○  test_rounding      tests/test_service.py:12   direct caller  depth 1   far field        9.0   fail
 ○  format_receipt     app/report.py:88           via handle_order  depth 2  fault line    3.9
 ○  RefundLine         app/models.py:21           type consumer  depth 1                   2.0
 ◐  apply_refund       app/refunds.py:33          direct caller  depth 1   glimpse          1.5

probes  4 selected  2 cracked  tests/test_service.py::test_rounding  tests/test_service.py::test_negative
sweep   full suite  0 leaks
packet  ./umbra-out/umbra.packet.md   report ./umbra-out/umbra.html
```

Glyphs fall back to `L`, `P`, `U`, `?` when the terminal is not UTF-8. Colour only when stdout is a TTY, and only on the glyph and the outcome word.

## Implementation notes

- Files: `internal/report/umbra.html.tmpl`, `umbra.css`, `umbra.js`, `layout.go`, embedded with `embed.FS`. JS is vanilla, no libraries, target under 700 lines; CSS under 400 lines.
- The JSON is embedded as `<script type="application/json" id="umbra-data">` with `</` and `<!--` escaped. The JS reads it, draws the SVG with `document.createElementNS`, and sets text with `textContent` only.
- Layout positions come from `layout` in the JSON; the JS never computes positions, only styles and states. `layout.go` has a determinism test: the same input twice yields byte-identical output.
- The state rules in JS mirror `internal/shadow/classify.go`, including all penumbra tiers; a test feeds the eight recorded scenarios through both and compares the final states.
- The light field uses `<radialGradient>`, `<mask>` and `<filter><feGaussianBlur>`; the hatch is a `<pattern>`; the spotlight is one `<circle>` with a gradient fill and `mix-blend-mode: screen`, moved with `transform` on `pointermove`.
- Replay state is a single integer (the current seq). Rendering at a seq is a pure function of the JSON and that integer; tweens are CSS transitions on fill, opacity and `r`.
- Content-Security-Policy meta: `default-src 'none'; style-src 'unsafe-inline'; script-src 'unsafe-inline'; img-src data:`.
- Performance: 150 nodes, 150 edges, 150 mask blobs render under 50 milliseconds on a mid laptop; blur radius is fixed, not per-node, so the filter is cached by the browser.

## QA checklist

- The map renders from `file://` in Chromium and Firefox, night and day schemes, with no network requests.
- Greyscale screenshot: the four states are distinguishable by shape and hatch alone.
- Sweep from seq 0 to end at 1x produces the same final state as the page load; the afterimage transition at the cut is visible; a mention lights a node only to echo.
- Every coined word on the page has its plain meaning visible next to it or in a chip sublabel; a reader who has never seen Umbra can name the four states from the page alone.
- Keyboard-only walk: enter the map, reach the top umbra node, open detail, read the state sentence, close, play the replay, step it, reach a docket row, filter to tests only.
- Screen reader announces node labels and the current replay event.
- Reduced motion: no tweens, no spotlight, replay still works.
- No JavaScript: docket and header render, the map placeholder text shows.
- Drop a scrubbed `umbra.json` from a different repository onto the landing page: it renders; drop a text file: a plain message explains the file was not an Umbra report; drop a 25 MB file: refused with a message.
- A symbol named `<img src=x onerror=alert(1)>` renders as text on the map, the docket and the detail panel.
- Narrow layout at 360px: map above, replay usable with touch, docket rows 40px tall, detail as a bottom sheet.
- Print preview: day scheme, static map with all labels, docket expanded, no replay strip.
