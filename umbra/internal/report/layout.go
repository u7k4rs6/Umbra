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
)

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
	return l
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
