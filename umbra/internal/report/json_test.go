package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

func jsonOf(t *testing.T, a *Analysis) map[string]any {
	t.Helper()
	var buf bytes.Buffer
	if err := WriteJSON(&buf, a); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("the report must be valid JSON: %v", err)
	}
	return out
}

// The schema in ARCHITECTURE.md is the contract the HTML and the landing-page
// viewer read, so the top-level keys are pinned.
func TestJSONHasTheDocumentedKeys(t *testing.T) {
	got := jsonOf(t, mkAnalysis())
	for _, key := range []string{
		"umbra_version", "checkpoint", "inputs", "session_said", "sources",
		"nodes", "timeline", "t0", "execution", "summary", "limitations", "commands_run",
	} {
		if _, ok := got[key]; !ok {
			t.Errorf("the report is missing the key %q", key)
		}
	}
}

// A consumer must never have to distinguish null from an empty list.
func TestJSONUsesEmptyListsNotNull(t *testing.T) {
	a := &Analysis{Version: "0.1.0", Channels: map[string]bool{}}
	got := jsonOf(t, a)
	for _, key := range []string{"sources", "nodes", "timeline", "limitations", "commands_run"} {
		v, ok := got[key]
		if !ok {
			t.Fatalf("missing %q", key)
		}
		if v == nil {
			t.Errorf("%q is null, want an empty list", key)
		}
	}
}

func TestJSONUsesPlainStateNamesNotCoinedWords(t *testing.T) {
	a := mkAnalysis()
	got := jsonOf(t, a)
	nodes := got["nodes"].([]any)
	states := map[string]bool{}
	for _, n := range nodes {
		states[n.(map[string]any)["state"].(string)] = true
	}
	for s := range states {
		switch s {
		case "lit", "penumbra", "umbra", "unknown":
		default:
			t.Errorf("state %q is not one of the four plain names", s)
		}
	}
}

func TestJSONNodeCarriesItsFactorsAndSentence(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Factors = map[string]float64{"relation": 3, "state": 1}
	got := jsonOf(t, a)
	n := got["nodes"].([]any)[0].(map[string]any)
	if _, ok := n["factors"]; !ok {
		t.Fatal("every node must carry its factors so a reader can recompute the score")
	}
	sentence, _ := n["sentence"].(string)
	if sentence == "" {
		t.Fatal("every node must carry its plain state sentence")
	}
}

// Illumination is null rather than zero when nothing could be judged.
func TestJSONIlluminationIsNullWhenAllUnknown(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("x", "app/a.py", 1, shadow.Unknown, shadow.TierNone, 1)}
	a.Summary = shadow.Summarize(a.Nodes)
	got := jsonOf(t, a)
	if v := got["summary"].(map[string]any)["illumination"]; v != nil {
		t.Fatalf("illumination = %v, want null", v)
	}
}

// Signature lines are a snippet, so they appear only when --snippets is on.
func TestJSONWithholdsSignaturesUnlessSnippetsIsOn(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Symbol.Signature = "def test_rounding()"
	a.Sources[0].NewSignature = "def compute_total(items, tax_rate)"

	off := jsonOf(t, a)
	if s := off["nodes"].([]any)[0].(map[string]any)["signature"]; s != nil && s != "" {
		t.Fatalf("signature = %v, want it withheld by default", s)
	}

	a.Snippets = true
	on := jsonOf(t, a)
	if s := on["nodes"].([]any)[0].(map[string]any)["signature"]; s == nil || s == "" {
		t.Fatal("--snippets should include the declaration line")
	}
}

// The timeline carries kinds, paths and sequence numbers, never prose.
func TestJSONTimelineCarriesNoProse(t *testing.T) {
	a := mkAnalysis()
	a.Timeline = []TimelineEvent{
		{Seq: 3, Kind: "read", Path: "app/service.py"},
		{Seq: 9, Kind: "mention", Paths: []string{"app/refunds.py"}, Symbols: []string{"apply_refund"}},
		{Seq: 12, Kind: "command", Cmd: "pytest -v"},
	}
	got := jsonOf(t, a)
	for _, raw := range got["timeline"].([]any) {
		e := raw.(map[string]any)
		for k := range e {
			switch k {
			case "seq", "ts", "kind", "path", "range", "paths", "symbols", "cmd":
			default:
				t.Errorf("unexpected timeline field %q; the timeline must not carry prose", k)
			}
		}
	}
}

// A symbol name that looks like markup must survive as text, not as HTML.
func TestJSONEscapesHostileSymbolNames(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Symbol.Name = `<img src=x onerror=alert(1)>`
	var buf bytes.Buffer
	if err := WriteJSON(&buf, a); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if strings.Contains(buf.String(), "<img") {
		t.Fatal("angle brackets must be escaped in the embedded JSON")
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("still valid JSON: %v", err)
	}
	name := out["nodes"].([]any)[0].(map[string]any)["name"].(string)
	if name != `<img src=x onerror=alert(1)>` {
		t.Fatalf("the name must round trip as text, got %q", name)
	}
}

func TestJSONExposuresUsePlainKindNames(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Exposures = []shadow.Exposure{
		{Seq: 2, Kind: transcript.Read, Range: &[2]int{1, 12}},
		{Seq: 5, Kind: transcript.Mention},
	}
	got := jsonOf(t, a)
	ex := got["nodes"].([]any)[0].(map[string]any)["exposures"].([]any)
	if len(ex) != 2 {
		t.Fatalf("exposures = %d, want 2", len(ex))
	}
	if ex[0].(map[string]any)["kind"] != "read" {
		t.Fatalf("kind = %v, want read", ex[0].(map[string]any)["kind"])
	}
	if ex[0].(map[string]any)["range"] == nil {
		t.Fatal("a partial read must carry its range")
	}
}

func TestJSONSourceCarriesChangeAndWeight(t *testing.T) {
	a := mkAnalysis()
	a.Sources = []graph.Source{{
		Symbol: "s1", Name: "compute_total", File: "app/service.py",
		Span: [2]int{20, 30}, Change: "signature", Weight: 1.0, Dependents: 7,
	}}
	got := jsonOf(t, a)
	s := got["sources"].([]any)[0].(map[string]any)
	if s["change"] != "signature" || s["weight"].(float64) != 1.0 || s["dependents"].(float64) != 7 {
		t.Fatalf("source = %v", s)
	}
}
