package report

import (
	"math"
	"sort"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// Layout is computed here and written into the JSON so the report and the
// landing-page viewer draw identical positions from the same numbers. The
// JavaScript never computes a position; it only styles what it is given.
//
// The geometry is from FRONTEND_SPEC.md "Geometry": a 1200 by 800 viewBox,
// sources at the origin or on an inner ring of radius 56, and depth rings at
// 170, 280 and 380 with co-change at 460.
type Layout struct {
	ViewBox  [4]float64           `json:"view_box"`
	Centre   Point                `json:"centre"`
	Rings    []Ring               `json:"rings"`
	Sources  map[string]Point     `json:"sources"`
	Nodes    map[string]NodePlace `json:"nodes"`
	Collapse bool                 `json:"sources_collapsed"`
}

// Point is a position in the viewBox.
type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// Ring is one depth guide.
type Ring struct {
	Depth  int     `json:"depth"`
	Radius float64 `json:"radius"`
	Dashed bool    `json:"dashed"`
	Label  string  `json:"label"`
	// LabelAt is where the ring's label sits. The top of the ring is
	// preferred; when a node label already occupies that arc the label slides
	// around the ring to the first free one.
	LabelAt Point `json:"label_at"`
	// LabelAngle is the angle it slid to, in degrees clockwise from the top.
	LabelAngle float64 `json:"label_angle"`
	// LabelBox is the space it takes, so a reader of the JSON can check the
	// placement without re-deriving the text metrics.
	LabelBox Box `json:"label_box"`
}

// Box is a rectangle in viewBox units.
type Box struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	W float64 `json:"w"`
	H float64 `json:"h"`
}

// Overlaps reports whether two boxes intersect, with a small gap so labels
// that merely touch still read as separate.
func (b Box) Overlaps(o Box) bool {
	const gap = 2
	return !(b.X+b.W+gap < o.X || o.X+o.W+gap < b.X ||
		b.Y+b.H+gap < o.Y || o.Y+o.H+gap < b.Y)
}

// Label is where a node's name is drawn and whether it is drawn at all.
type Label struct {
	Point
	// Anchor is the SVG text-anchor: start, middle or end.
	Anchor string `json:"anchor"`
	Box    Box    `json:"box"`
	// Visible is false when a higher scoring label already occupies the space.
	// The disc is still drawn; only the name waits for interaction.
	Visible bool `json:"visible"`
}

// NodePlace is where one node sits and which way it faces.
type NodePlace struct {
	Point
	// Angle is degrees clockwise from the top, kept so the half disc of a
	// penumbra node can face its source without the JavaScript computing it.
	Angle float64 `json:"angle"`
	Ring  float64 `json:"ring"`
	Depth int     `json:"depth"`
	// Anchor is where the label goes: left, right, above or below.
	Anchor string `json:"anchor"`
	// Label is the placed name, with the space it occupies.
	Label Label `json:"label"`
	// Toward is the point the node's lit half faces, which is its source or
	// the origin.
	Toward Point `json:"toward"`
}

const (
	viewW      = 1200.0
	viewH      = 800.0
	innerRing  = 56.0
	groupGapDe = 6.0
	minPerNode = 12.0

	// Text sizes in viewBox units. The map scales to its column, so these are
	// map units and not screen pixels: a name set at 12 was legible in the
	// column and too small once the map was scaled down on a narrow screen.
	NodeLabelSize   = 14.0
	FileLabelSize   = 12.0
	SourceLabelSize = 22.0
	// SourceKindOffset is how far the change kind sits below the source name.
	SourceKindOffset = 18.0

	// monoAdvance is the width of one character as a fraction of the font
	// size in a monospace face. Labels are monospace, so a box can be
	// computed without measuring, which is what lets the placement be decided
	// here and tested.
	monoAdvance = 0.6

	// The centre carries the source name and its change kind. Nothing else
	// may sit inside this box, or a dependent lands on top of the label that
	// says what changed.
	centreKeepW = 140.0
	centreKeepH = 44.0

	// ringLabelStep is how far a ring label slides when the top of its ring
	// is already taken.
	ringLabelStep = 20.0
)

// textBox returns the space a piece of monospace text occupies, given where it
// is anchored.
func textBox(at Point, anchor string, text string, size float64) Box {
	w := float64(len([]rune(text))) * size * monoAdvance
	h := size
	x := at.X
	switch anchor {
	case "middle":
		x -= w / 2
	case "end":
		x -= w
	}
	// SVG text sits on its baseline, so the box rises above the y given.
	return Box{X: round2(x), Y: round2(at.Y - h*0.78), W: round2(w), H: round2(h)}
}

// ringRadius is the radius for a depth. Co-change sits outside depth 3.
func ringRadius(depth int, coChange bool) float64 {
	if coChange {
		return 460
	}
	switch depth {
	case 1:
		return 170
	case 2:
		return 280
	default:
		return 380
	}
}

// BuildLayout places every source and node.
//
// The placement is deterministic: groups are ordered by their highest score
// then by file name, and nodes inside a group by span then name, so the same
// JSON always draws the same map. A determinism test asserts this.
func BuildLayout(a *Analysis) *Layout {
	centre := Point{X: viewW / 2, Y: viewH / 2}
	l := &Layout{
		ViewBox: [4]float64{0, 0, viewW, viewH},
		Centre:  centre,
		Sources: map[string]Point{},
		Nodes:   map[string]NodePlace{},
	}

	// Sources: one at the origin, two to four on the inner ring, more than
	// four collapsed into a single disc.
	ids := make([]string, 0, len(a.Sources))
	for _, s := range a.Sources {
		if s.Symbol != "" {
			ids = append(ids, s.Symbol)
		}
	}
	sort.Strings(ids)

	switch {
	case len(ids) == 0:
	case len(ids) == 1:
		l.Sources[ids[0]] = centre
	case len(ids) <= 4:
		for i, id := range ids {
			ang := float64(i) * 360 / float64(len(ids))
			l.Sources[id] = polar(centre, innerRing, ang)
		}
	default:
		l.Collapse = true
		for _, id := range ids {
			l.Sources[id] = centre
		}
	}

	// Rings present in this report.
	depths := map[int]bool{}
	coChange := false
	for _, n := range a.Nodes {
		if isCoChange(n) {
			coChange = true
			continue
		}
		depths[n.Depth] = true
	}
	var ds []int
	for d := range depths {
		ds = append(ds, d)
	}
	sort.Ints(ds)
	for _, d := range ds {
		l.Rings = append(l.Rings, Ring{Depth: d, Radius: ringRadius(d, false), Label: ringLabel(d)})
	}
	if coChange {
		l.Rings = append(l.Rings, Ring{Depth: 0, Radius: ringRadius(0, true), Dashed: true, Label: "co-change"})
	}

	// Group the nodes of each ring by file, then place them.
	byRing := map[float64][]*shadow.Node{}
	for _, n := range a.Nodes {
		r := ringRadius(n.Depth, isCoChange(n))
		byRing[r] = append(byRing[r], n)
	}
	for radius, nodes := range byRing {
		place(l, centre, radius, nodes)
	}

	clearCentre(l, centre)
	placeLabels(l, a)
	placeRingLabels(l, centre)
	return l
}

// clearCentre pushes any node out of the box the source name and its change
// kind occupy, along its own angle so the ring order is not disturbed.
func clearCentre(l *Layout, centre Point) {
	keep := Box{
		X: centre.X - centreKeepW/2, Y: centre.Y - centreKeepH/2,
		W: centreKeepW, H: centreKeepH,
	}
	for id, p := range l.Nodes {
		disc := Box{X: p.X - 12, Y: p.Y - 12, W: 24, H: 24}
		if !disc.Overlaps(keep) {
			continue
		}
		// Step outward along the node's own angle until the disc is clear.
		for step := 0; step < 40 && disc.Overlaps(keep); step++ {
			p.Ring += 10
			np := polar(centre, p.Ring, p.Angle)
			p.X, p.Y = np.X, np.Y
			disc = Box{X: p.X - 12, Y: p.Y - 12, W: 24, H: 24}
		}
		l.Nodes[id] = p
	}
}

// placeLabels puts every node's name beside its disc and then hides the ones
// that would collide.
//
// The order is by score, highest first, so the label that survives a collision
// is the one the reader most needs. Ties break on the id, so the same report
// always hides the same label.
func placeLabels(l *Layout, a *Analysis) {
	type entry struct {
		id    string
		score float64
		name  string
	}
	var order []entry
	for _, n := range a.Nodes {
		if _, ok := l.Nodes[n.Symbol.ID]; !ok {
			continue
		}
		order = append(order, entry{id: n.Symbol.ID, score: n.Score, name: n.Symbol.Name})
	}
	sort.Slice(order, func(i, j int) bool {
		if order[i].score != order[j].score {
			return order[i].score > order[j].score
		}
		return order[i].id < order[j].id
	})

	var taken []Box
	for _, e := range order {
		p := l.Nodes[e.id]
		at, anchor := labelPoint(p)
		box := textBox(at, anchor, e.name, NodeLabelSize)

		visible := true
		for _, t := range taken {
			if box.Overlaps(t) {
				visible = false
				break
			}
		}
		if visible {
			taken = append(taken, box)
		}
		p.Label = Label{Point: at, Anchor: anchor, Box: box, Visible: visible}
		l.Nodes[e.id] = p
	}
}

// labelPoint is where a node's name sits, given the side of the map it is on.
func labelPoint(p NodePlace) (Point, string) {
	const pad = 13
	switch p.Anchor {
	case "right":
		return Point{X: round2(p.X + pad), Y: round2(p.Y + 4)}, "start"
	case "left":
		return Point{X: round2(p.X - pad), Y: round2(p.Y + 4)}, "end"
	case "above":
		return Point{X: p.X, Y: round2(p.Y - pad - 6)}, "middle"
	default:
		return Point{X: p.X, Y: round2(p.Y + pad + 12)}, "middle"
	}
}

// placeRingLabels puts each ring's label at the top of its ring, or slides it
// around the ring to the first arc where it does not land on a node label.
func placeRingLabels(l *Layout, centre Point) {
	var taken []Box
	for _, p := range l.Nodes {
		if p.Label.Visible {
			taken = append(taken, p.Label.Box)
		}
	}

	for i := range l.Rings {
		r := &l.Rings[i]
		if r.Label == "" {
			continue
		}
		for step := 0; step < int(360/ringLabelStep); step++ {
			angle := math.Mod(float64(step)*ringLabelStep, 360)
			at := polar(centre, r.Radius+12, angle)
			box := textBox(at, "middle", r.Label, FileLabelSize)

			clash := false
			for _, t := range taken {
				if box.Overlaps(t) {
					clash = true
					break
				}
			}
			// Ring labels must not land on each other either.
			if !clash {
				for j := 0; j < i; j++ {
					if l.Rings[j].Label != "" && box.Overlaps(l.Rings[j].LabelBox) {
						clash = true
						break
					}
				}
			}
			if !clash || step == int(360/ringLabelStep)-1 {
				r.LabelAt = at
				r.LabelAngle = angle
				r.LabelBox = box
				break
			}
		}
	}
}

func ringLabel(d int) string {
	switch d {
	case 1:
		return "1 hop"
	case 2:
		return "2 hops"
	case 3:
		return "3 hops"
	}
	return ""
}

func isCoChange(n *shadow.Node) bool {
	for _, m := range n.Modifiers {
		if m == "co-change only" {
			return true
		}
	}
	return false
}

// place lays one ring out: groups by file, ordered by the group's highest
// score descending, starting at the top and going clockwise, with each group
// taking a share proportional to its size and a 6 degree gap between groups.
func place(l *Layout, centre Point, radius float64, nodes []*shadow.Node) {
	groups := map[string][]*shadow.Node{}
	for _, n := range nodes {
		groups[n.Symbol.File] = append(groups[n.Symbol.File], n)
	}

	type group struct {
		file  string
		nodes []*shadow.Node
		top   float64
	}
	var gs []group
	for file, ns := range groups {
		sort.Slice(ns, func(i, j int) bool {
			if ns[i].Symbol.Span[0] != ns[j].Symbol.Span[0] {
				return ns[i].Symbol.Span[0] < ns[j].Symbol.Span[0]
			}
			return ns[i].Symbol.Name < ns[j].Symbol.Name
		})
		top := 0.0
		for _, n := range ns {
			if n.Score > top {
				top = n.Score
			}
		}
		gs = append(gs, group{file: file, nodes: ns, top: top})
	}
	sort.Slice(gs, func(i, j int) bool {
		if gs[i].top != gs[j].top {
			return gs[i].top > gs[j].top
		}
		return gs[i].file < gs[j].file
	})

	total := 0
	for _, g := range gs {
		total += len(g.nodes)
	}
	if total == 0 {
		return
	}
	budget := 360.0 - groupGapDe*float64(len(gs))
	if budget < 60 {
		budget = 60
	}

	angle := 0.0
	for _, g := range gs {
		share := budget * float64(len(g.nodes)) / float64(total)
		if min := minPerNode * float64(len(g.nodes)); share < min {
			share = min
		}
		step := share / float64(len(g.nodes))
		for i, n := range g.nodes {
			a := math.Mod(angle+step*(float64(i)+0.5), 360)
			pt := polar(centre, radius, a)
			toward := centre
			if src, ok := l.Sources[firstSource(n)]; ok {
				toward = src
			}
			l.Nodes[n.Symbol.ID] = NodePlace{
				Point: pt, Angle: a, Ring: radius, Depth: n.Depth,
				Anchor: anchorFor(a), Toward: toward,
			}
		}
		angle = math.Mod(angle+share+groupGapDe, 360)
	}
}

func firstSource(n *shadow.Node) string {
	if len(n.Sources) == 0 {
		return ""
	}
	return n.Sources[0]
}

// polar converts an angle in degrees clockwise from the top into a point.
func polar(centre Point, radius, angleDeg float64) Point {
	rad := (angleDeg - 90) * math.Pi / 180
	return Point{
		X: round2(centre.X + radius*math.Cos(rad)),
		Y: round2(centre.Y + radius*math.Sin(rad)),
	}
}

// anchorFor decides which side the label sits on: right for the right half,
// left for the left half, and above or below near the top and bottom.
func anchorFor(angle float64) string {
	switch {
	case angle < 20 || angle > 340:
		return "above"
	case angle > 160 && angle < 200:
		return "below"
	case angle <= 180:
		return "right"
	}
	return "left"
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }
