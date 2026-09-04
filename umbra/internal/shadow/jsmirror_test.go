package shadow

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// The report's JavaScript re-implements the classifier so the attention
// replay can show the state at any point in the session. Two implementations
// of one rule set drift apart silently, so this test feeds the recorded
// scenarios through both and compares.
//
// It needs node on the path and nothing else: no browser, no network, no
// agent. Without node it skips rather than failing, because the Go side is
// still covered by the classifier truth table.

type jsNode struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Span [2]int `json:"span"`
}

type jsExposure struct {
	Seq   int     `json:"seq"`
	Kind  string  `json:"kind"`
	Range *[2]int `json:"range,omitempty"`
}

type jsCase struct {
	Node      jsNode       `json:"node"`
	Exposures []jsExposure `json:"exposures"`
	T0        int          `json:"t0"`
	HasAny    bool         `json:"hasAny"`
	WantState string       `json:"wantState"`
	WantTier  string       `json:"wantTier"`
}

const harness = `
const path = require('path');
globalThis.window = { matchMedia: () => ({ matches: false }) };
require(path.resolve(process.argv[2]));
const cases = JSON.parse(require('fs').readFileSync(process.argv[3], 'utf8'));
const out = cases.map(c =>
  globalThis.__umbra.stateAt(c.node, c.exposures, c.t0, c.hasAny, Infinity));
process.stdout.write(JSON.stringify(out));
`

func TestJavaScriptMirrorsTheGoClassifier(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on the path; the Go classifier is covered by its own truth table")
	}
	jsPath, err := filepath.Abs(filepath.Join("..", "report", "assets", "umbra.js"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(jsPath); err != nil {
		t.Fatalf("the report script is missing: %v", err)
	}

	scenarios := []string{
		"everything-lit", "classic", "partial-read", "pre-change",
		"no-reads", "echo", "leak", "beacon",
	}

	var cases []jsCase
	for _, name := range scenarios {
		r := runScenario(t, name, "")
		anyEvidence := false
		for _, n := range r.nodes {
			if n.State != Unknown {
				anyEvidence = true
			}
		}
		cut := 0
		for _, n := range r.nodes {
			_ = n
		}
		// The cut is whatever the examined set found for this scenario.
		session := loadScenario(t, name)
		session.ResolveMentions(Vocabulary(r.field))
		ex := BuildExamined(session, map[string]bool{"app/service.py": true})
		cut = ex.Cut

		for _, n := range r.nodes {
			c := jsCase{
				Node:      jsNode{ID: n.Symbol.ID, Name: n.Symbol.Name, Span: n.Symbol.Span},
				T0:        cut,
				HasAny:    anyEvidence,
				WantState: n.State.String(),
				WantTier:  string(n.Tier),
			}
			for _, x := range n.Exposures {
				c.Exposures = append(c.Exposures, jsExposure{
					Seq: x.Seq, Kind: x.Kind.String(), Range: x.Range,
				})
			}
			cases = append(cases, c)
		}
	}
	if len(cases) < 40 {
		t.Fatalf("expected the scenarios to produce plenty of cases, got %d", len(cases))
	}

	dir := t.TempDir()
	casePath := filepath.Join(dir, "cases.json")
	blob, err := json.Marshal(cases)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(casePath, blob, 0o644); err != nil {
		t.Fatal(err)
	}
	harnessPath := filepath.Join(dir, "harness.js")
	if err := os.WriteFile(harnessPath, []byte(harness), 0o644); err != nil {
		t.Fatal(err)
	}

	out, err := exec.Command(node, harnessPath, jsPath, casePath).Output()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			t.Fatalf("node failed: %v\n%s", err, ee.Stderr)
		}
		t.Fatalf("node failed: %v", err)
	}

	var got []struct {
		State string `json:"state"`
		Tier  string `json:"tier"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("reading the JavaScript answer: %v\n%s", err, out)
	}
	if len(got) != len(cases) {
		t.Fatalf("the JavaScript answered %d cases, Go had %d", len(got), len(cases))
	}

	mismatches := 0
	for i, c := range cases {
		if got[i].State != c.WantState || got[i].Tier != c.WantTier {
			mismatches++
			if mismatches <= 8 {
				t.Errorf("%s: Go says %s/%s, the report script says %s/%s",
					c.Node.Name, c.WantState, c.WantTier, got[i].State, got[i].Tier)
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d of %d cases disagree between the two implementations", mismatches, len(cases))
	}
}
