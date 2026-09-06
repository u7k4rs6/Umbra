package report

import (
	"encoding/json"
	"math"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

func layoutAnalysis() *Analysis {
	nodes := []*shadow.Node{
		mkNode("a1", "app/api.py", 10, shadow.Umbra, shadow.TierNone, 9),
		mkNode("a2", "app/api.py", 20, shadow.Umbra, shadow.TierNone, 8),
		mkNode("r1", "app/refunds.py", 17, shadow.Penumbra, shadow.TierEcho, 7),
		mkNode("t1", "tests/test_a.py", 5, shadow.Umbra, shadow.TierNone, 6),
	}
	nodes[3].Depth = 2
	for _, n := range nodes {
		n.Sources = []string{"src1"}
	}
	a := &Analysis{
		Version: "0.1.0",
		Sources: []graph.Source{{Symbol: "src1", Name: "compute_total", File: "app/service.py"}},
		Nodes:   nodes,
	}
	a.Summary = shadow.Summarize(nodes)
	return a
}

func TestLayoutViewBoxAndCentre(t *testing.T) {
	l := BuildLayout(layoutAnalysis())
	if l.ViewBox != [4]float64{0, 0, 1200, 800} {
		t.Fatalf("viewBox = %v, want the 1200 by 800 box from the spec", l.ViewBox)
	}
	if l.Centre.X != 600 || l.Centre.Y != 400 {
		t.Fatalf("centre = %v", l.Centre)
	}
}

// One source sits at the origin.
func TestLayoutSingleSourceAtOrigin(t *testing.T) {
	l := BuildLayout(layoutAnalysis())
	p, ok := l.Sources["src1"]
	if !ok {
		t.Fatal("the source was not placed")
	}
	if p != l.Centre {
		t.Fatalf("a single source sits at the origin, got %v", p)
	}
	if l.Collapse {
		t.Fatal("one source does not collapse")
	}
}

// Two to four sources sit on an inner ring of radius 56.
func TestLayoutFewSourcesOnInnerRing(t *testing.T) {
	a := layoutAnalysis()
	a.Sources = []graph.Source{
		{Symbol: "s1", Name: "a"}, {Symbol: "s2", Name: "b"}, {Symbol: "s3", Name: "c"},
	}
	l := BuildLayout(a)
	for id, p := range l.Sources {
		d := math.Hypot(p.X-l.Centre.X, p.Y-l.Centre.Y)
		if math.Abs(d-56) > 0.5 {
			t.Fatalf("source %s is %v from the centre, want 56", id, d)
		}
	}
	if l.Collapse {
		t.Fatal("three sources do not collapse")
	}
}

// More than four collapse into one disc at the origin.
func TestLayoutManySourcesCollapse(t *testing.T) {
	a := layoutAnalysis()
	a.Sources = nil
	for _, n := range []string{"s1", "s2", "s3", "s4", "s5"} {
		a.Sources = append(a.Sources, graph.Source{Symbol: n, Name: n})
	}
	l := BuildLayout(a)
	if !l.Collapse {
		t.Fatal("more than four sources collapse")
	}
	for _, p := range l.Sources {
		if p != l.Centre {
			t.Fatalf("a collapsed source sits at the origin, got %v", p)
		}
	}
}

// Depth 1 at 170, depth 2 at 280, depth 3 at 380, co-change at 460 and dashed.
func TestLayoutRingRadii(t *testing.T) {
	cases := []struct {
		depth    int
		coChange bool
		want     float64
	}{{1, false, 170}, {2, false, 280}, {3, false, 380}, {1, true, 460}}
	for _, c := range cases {
		if got := ringRadius(c.depth, c.coChange); got != c.want {
			t.Errorf("ringRadius(%d, %v) = %v, want %v", c.depth, c.coChange, got, c.want)
		}
	}

	l := BuildLayout(layoutAnalysis())
	for _, n := range l.Nodes {
		d := math.Hypot(n.X-l.Centre.X, n.Y-l.Centre.Y)
		if math.Abs(d-n.Ring) > 0.5 {
			t.Fatalf("a node is %v from the centre but claims ring %v", d, n.Ring)
		}
	}
}

func TestLayoutCoChangeRingIsDashed(t *testing.T) {
	a := layoutAnalysis()
	a.Nodes[2].Modifiers = append(a.Nodes[2].Modifiers, "co-change only")
	l := BuildLayout(a)
	found := false
	for _, r := range l.Rings {
		if r.Radius == 460 {
			found = true
			if !r.Dashed {
				t.Fatal("the co-change ring is drawn dashed")
			}
		}
	}
	if !found {
		t.Fatal("expected a co-change ring")
	}
}

// The same input must place every node identically, every time, or two runs on
// one checkpoint would draw different maps.
func TestLayoutIsDeterministic(t *testing.T) {
	first, err := json.Marshal(BuildLayout(layoutAnalysis()))
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 10; i++ {
		again, err := json.Marshal(BuildLayout(layoutAnalysis()))
		if err != nil {
			t.Fatal(err)
		}
		if string(again) != string(first) {
			t.Fatalf("run %d produced a different layout", i)
		}
	}
}

// Nodes in the same file are grouped and ordered by span.
func TestLayoutGroupsByFileAndOrdersBySpan(t *testing.T) {
	l := BuildLayout(layoutAnalysis())
	a1 := l.Nodes["app/api.py:a1"]
	a2 := l.Nodes["app/api.py:a2"]
	if a1.Angle >= a2.Angle {
		t.Fatalf("within a file, the earlier span comes first: a1 %v, a2 %v", a1.Angle, a2.Angle)
	}
	if a1.Ring != a2.Ring {
		t.Fatal("both sit at depth 1")
	}
}

func TestLayoutLabelAnchors(t *testing.T) {
	cases := map[float64]string{
		0: "above", 10: "above", 90: "right", 180: "below", 270: "left", 350: "above",
	}
	for angle, want := range cases {
		if got := anchorFor(angle); got != want {
			t.Errorf("anchorFor(%v) = %q, want %q", angle, got, want)
		}
	}
}

// Every node carries the point its lit half should face, so the JavaScript
// never computes an angle.
func TestLayoutNodesCarryTheirSourcePoint(t *testing.T) {
	l := BuildLayout(layoutAnalysis())
	for id, n := range l.Nodes {
		if n.Toward != l.Centre {
			t.Fatalf("node %s faces %v, want the single source at the centre", id, n.Toward)
		}
	}
}

func TestLayoutEmptyAnalysis(t *testing.T) {
	l := BuildLayout(&Analysis{})
	if len(l.Nodes) != 0 || len(l.Sources) != 0 {
		t.Fatal("an empty analysis places nothing")
	}
	if l.ViewBox[2] != 1200 {
		t.Fatal("the viewBox is fixed even when empty")
	}
}
