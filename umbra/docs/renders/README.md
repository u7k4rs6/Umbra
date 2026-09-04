# Render checks

Screenshots of the map, taken with a headless browser rather than judged by
reading the code. Six of them: both schemes at three points of the sweep.

| File | Scheme | Playhead |
|---|---|---|
| `lightfield-dark-seq0.png` | night | seq 0, before the session touched anything |
| `lightfield-dark-cut.png` | night | the cut, the first edit to a source file |
| `lightfield-dark-end.png` | night | the end, the eclipse |
| `lightfield-light-seq0.png` | day | seq 0 |
| `lightfield-light-cut.png` | day | the cut |
| `lightfield-light-end.png` | day | the end |

They come from a real run: checkpoint `b20f84567474` against commit `0063443`,
the same report the landing page draws.

`lightfield-first-pass.png` is the very first render of the light field, taken
before any node, edge or label existed, which is the order the plan requires
the map to be built in. It was sitting at the repository root; it belongs here.

What to look for at seq 0 against the end: at seq 0 every dependent is in
shadow and every pool is at full radius, because the session had not yet looked
at anything. By the end two nodes are lit and their pools have gone, three are
penumbra and their pools have shrunk, and six are still in full shadow. The
pools are part of the replay, not a static decoration.
