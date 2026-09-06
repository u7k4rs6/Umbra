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

## Phase 16: the eclipse hero

The landing page, restyled around a drawn eclipse. The corona is inline SVG
built from the report's own numbers, not a photograph: there is no raster asset
on the page and no request for one. Every one of these was taken with a
headless browser and looked at before the next step was built.

| File | Scheme | State |
|---|---|---|
| `corona-alone-night.png` | night | the corona by itself, before the map was put on it |
| `landing-night-01-claim.png` | night | beat 01, the field before the session had opened anything |
| `landing-night-02-field.png` | night | beat 02, the field as the session had it just before the first edit |
| `landing-night-03-cut.png` | night | beat 03, the cut |
| `landing-night-04-sweep.png` | night | beat 04, the eclipse at commit; the page loads here |
| `landing-day-04-sweep.png` | day | the same state, divider all the way left |
| `landing-split-night-day.png` | both | the divider mid drag, the eclipse cut in half |
| `landing-night-legend-umbra-only.png` | night | the umbra orb pressed; lit and penumbra nodes drop back |
| `landing-mid-900.png` | night | 900px |
| `landing-narrow-360.png` | night | 360px |
| `landing-print-day.png` | day | print media, day scheme forced |
| `landing-night-reduced-motion.png` | night | prefers-reduced-motion: reduce |
| `landing-night-greyscale.png` | night | greyscale, to check the states read without colour |
| `landing-night-no-javascript.png` | night | script disabled |

What to look for. In `corona-alone-night.png` the light is brightest just
outside the limb and reaches nothing by the frame's edge, with no rim: three
earlier attempts had one, and each rim came from a gradient stop that fell off
a cliff rather than a tail. In `landing-split-night-day.png` the day half is a
second drawing under day tokens rather than a filter over the first, which is
why the shadow pools there are darker than the paper rather than lighter. In
`landing-night-greyscale.png` the three states are still separable: a lit node
is a filled white disc, a penumbra node a half disc, an umbra node an outlined
hatched disc, and the legend orbs differ in radius as well as in value.
