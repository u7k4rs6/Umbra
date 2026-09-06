package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// scenarioNames are the eight recorded scenarios. Label placement is checked
// against all of them, because a rule that holds for one arrangement of nodes
// says very little.
var scenarioNames = []string{
	"everything-lit", "classic", "partial-read", "pre-change",
	"no-reads", "echo", "leak", "beacon",
}

// layoutForScenario builds a real layout from a recorded scenario: the
// captured snapshot for the graph, the recorded transcript for the states.
func layoutForScenario(t *testing.T, name string) *Layout {
	t.Helper()

	snap, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "snapshot.ndjson"))
	if err != nil {
		t.Fatalf("reading the snapshot: %v", err)
	}
	field, err := graph.LoadSnapshot(snap)
	if err != nil {
		t.Fatalf("loading the snapshot: %v", err)
	}
	capsBlob, err := os.ReadFile(filepath.Join("..", "graph", "testdata", "capabilities.json"))
	if err != nil {
		t.Fatalf("reading capabilities: %v", err)
	}
	caps, err := graph.ParseCapabilities(capsBlob)
	if err != nil {
		t.Fatalf("parsing capabilities: %v", err)
	}
	relMap := graph.NewRelationMap(caps)

	var src *graph.Symbol
	for _, id := range field.ByFile["app/service.py"] {
		if field.Symbols[id].Name == "compute_total" {
			src = field.Symbols[id]
		}
	}
	if src == nil {
		t.Fatal("compute_total is missing from the snapshot")
	}
	sources := []graph.Source{{
		Symbol: src.ID, Name: src.Name, File: src.File, Span: src.Span,
		Change: "signature", Weight: graph.Weight("signature"), Dependents: 7,
	}}

	jsonl, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "recorded", name, "transcript.jsonl"))
	if err != nil {
		t.Fatalf("reading scenario %s: %v", name, err)
	}
	session, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(jsonl)
	if err != nil {
		t.Fatalf("parsing scenario %s: %v", name, err)
	}
	const prefix = "umbra/fixtures/app/"
	for i := range session.Events {
		session.Events[i].Path = trimPrefix(session.Events[i].Path, prefix)
		for j := range session.Events[i].Paths {
			session.Events[i].Paths[j] = trimPrefix(session.Events[i].Paths[j], prefix)
		}
	}
	session.ResolveMentions(shadow.Vocabulary(field))
	examined := shadow.BuildExamined(session, map[string]bool{"app/service.py": true})

	nodes := shadow.Build(shadow.BuildInput{
		Field: field, RelMap: relMap, Sources: sources,
		Reach:    field.Dependents([]string{src.ID}, 2, relMap),
		Examined: examined, Impacts: map[string]*graph.Impact{},
	})
	shadow.SortDocket(nodes)

	a := &Analysis{Version: "test", Sources: sources, Nodes: nodes}
	a.Summary = shadow.Summarize(nodes)
	return BuildLayout(a)
}

func trimPrefix(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

// A ring label that lands on a node label makes both unreadable. The ring
// label slides around its ring until it finds a free arc.
func TestRingLabelsClearOfNodeLabels(t *testing.T) {
	for _, name := range scenarioNames {
		t.Run(name, func(t *testing.T) {
			l := layoutForScenario(t, name)
			if len(l.Rings) == 0 {
				t.Skip("this scenario has no rings")
			}
			for _, r := range l.Rings {
				if r.Label == "" {
					continue
				}
				for id, p := range l.Nodes {
					if !p.Label.Visible {
						continue
					}
					if r.LabelBox.Overlaps(p.Label.Box) {
						t.Errorf("ring label %q at %.0f degrees overlaps the label of %s",
							r.Label, r.LabelAngle, id)
					}
				}
			}
		})
	}
}

// Two visible labels never occupy the same space. The one that gives way is
// the lower scoring one, and its disc is still drawn.
func TestVisibleNodeLabelsDoNotCollide(t *testing.T) {
	for _, name := range scenarioNames {
		t.Run(name, func(t *testing.T) {
			l := layoutForScenario(t, name)
			var boxes []Box
			var ids []string
			for id, p := range l.Nodes {
				if p.Label.Visible {
					boxes = append(boxes, p.Label.Box)
					ids = append(ids, id)
				}
			}
			for i := 0; i < len(boxes); i++ {
				for j := i + 1; j < len(boxes); j++ {
					if boxes[i].Overlaps(boxes[j]) {
						t.Errorf("visible labels overlap: %s and %s", ids[i], ids[j])
					}
				}
			}
			if len(boxes) == 0 && len(l.Nodes) > 0 {
				t.Error("every label was hidden, which cannot be right")
			}
		})
	}
}

// Nothing may sit on the source name or the change kind under it.
func TestNothingSitsOnTheSourceLabel(t *testing.T) {
	for _, name := range scenarioNames {
		t.Run(name, func(t *testing.T) {
			l := layoutForScenario(t, name)
			keep := Box{
				X: l.Centre.X - centreKeepW/2, Y: l.Centre.Y - centreKeepH/2,
				W: centreKeepW, H: centreKeepH,
			}
			for id, p := range l.Nodes {
				disc := Box{X: p.X - 12, Y: p.Y - 12, W: 24, H: 24}
				if disc.Overlaps(keep) {
					t.Errorf("%s sits inside the centre keep-out at (%.0f, %.0f)", id, p.X, p.Y)
				}
			}
		})
	}
}

// The sizes the spec asks for, so a later edit cannot quietly shrink them.
func TestLabelSizes(t *testing.T) {
	if NodeLabelSize != 14 {
		t.Errorf("node label size = %v, want 14", NodeLabelSize)
	}
	if FileLabelSize != 12 {
		t.Errorf("file label size = %v, want 12", FileLabelSize)
	}
	if SourceKindOffset != 18 {
		t.Errorf("source kind offset = %v, want 18", SourceKindOffset)
	}
}

// Placement is part of the layout, so it has to be as deterministic as the
// positions are.
func TestLabelPlacementIsDeterministic(t *testing.T) {
	for _, name := range scenarioNames {
		first, err := json.Marshal(layoutForScenario(t, name))
		if err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			again, err := json.Marshal(layoutForScenario(t, name))
			if err != nil {
				t.Fatal(err)
			}
			if string(again) != string(first) {
				t.Fatalf("%s: run %d placed labels differently", name, i)
			}
		}
	}
}

// Known-positive for the collision tests above.
//
// All four of them are built on Box.Overlaps. If it answered false for every
// pair, every one of them would pass on all eight scenarios and prove nothing.
// This is the case that has to fail for the rest to mean anything.
func TestBoxOverlapsCatchesAPlantedOverlap(t *testing.T) {
	a := Box{X: 100, Y: 100, W: 80, H: 20}
	for _, tc := range []struct {
		name string
		b    Box
		want bool
	}{
		{"the same box", a, true},
		{"a box inside it", Box{X: 110, Y: 105, W: 10, H: 5}, true},
		{"a box straddling its left edge", Box{X: 60, Y: 100, W: 60, H: 20}, true},
		{"a box straddling its top edge", Box{X: 100, Y: 90, W: 80, H: 20}, true},
		{"a box inside the two unit gap", Box{X: 181, Y: 100, W: 80, H: 20}, true},
		{"a box just clear of the gap", Box{X: 183, Y: 100, W: 80, H: 20}, false},
		{"a box just clear below", Box{X: 100, Y: 123, W: 80, H: 20}, false},
		{"a box in another quadrant", Box{X: -400, Y: -400, W: 80, H: 20}, false},
	} {
		if got := a.Overlaps(tc.b); got != tc.want {
			t.Errorf("%s: Overlaps = %v, want %v", tc.name, got, tc.want)
		}
		if got := tc.b.Overlaps(a); got != tc.want {
			t.Errorf("%s, the other way round: Overlaps = %v, want %v", tc.name, got, tc.want)
		}
	}
}
