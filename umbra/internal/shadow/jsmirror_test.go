package shadow

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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
	Seq       int          `json:"seq"`
	MaxSeq    int          `json:"maxSeq"`
	WantState string       `json:"wantState"`
	WantTier  string       `json:"wantTier"`
	// Where in the sweep this case sits, for the failure message.
	At string `json:"at"`
	// WantNote is the qualifier a screen reader should hear, empty at the end.
	WantNote string `json:"wantNote"`
}

const harness = `
const path = require('path');
globalThis.window = { matchMedia: () => ({ matches: false }) };
require(path.resolve(process.argv[2]));
const cases = JSON.parse(require('fs').readFileSync(process.argv[3], 'utf8'));
const out = cases.map(c => {
  const st = globalThis.__umbra.stateAt(c.node, c.exposures, c.t0, c.hasAny, c.seq);
  const note = globalThis.__umbra.seqNoteFor(c.seq, c.maxSeq);
  const announced = globalThis.__umbra.nodeLabel(
    Object.assign({}, c.node, {state: st.state, tier: st.tier, depth: 1}), note);
  return {state: st.state, tier: st.tier, note: note, announced: announced};
});
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
		session := loadScenario(t, name)
		session.ResolveMentions(Vocabulary(r.field))
		ex := BuildExamined(session, map[string]bool{"app/service.py": true})

		maxSeq := 0
		for _, e := range session.Events {
			if e.Seq > maxSeq {
				maxSeq = e.Seq
			}
		}

		// Three positions: the start, where nothing has happened yet; the cut,
		// where the change begins and a read before it becomes an afterimage;
		// and the end, which is the eclipse. The middle one is where the two
		// implementations are most likely to disagree, because it is the only
		// one where the t0 rule is doing any work.
		mid := ex.Cut
		if !ex.HasCut || mid <= 0 {
			mid = maxSeq / 2
		}
		positions := []struct {
			at  string
			seq int
		}{
			{"the start", 0},
			{"the cut", mid},
			{"the end", maxSeq},
		}

		for _, pos := range positions {
			for _, n := range r.nodes {
				state, tier := ClassifyAt(ex, n.Symbol.File, n.Symbol.Name, n.Symbol.Span, pos.seq)
				c := jsCase{
					Node:      jsNode{ID: n.Symbol.ID, Name: n.Symbol.Name, Span: n.Symbol.Span},
					T0:        ex.Cut,
					HasAny:    anyEvidence,
					Seq:       pos.seq,
					MaxSeq:    maxSeq,
					At:        name + " at " + pos.at,
					WantState: state.String(),
					WantTier:  string(tier),
				}
				if pos.seq < maxSeq {
					c.WantNote = fmt.Sprintf("as of seq %d", pos.seq)
				}
				for _, x := range n.Exposures {
					c.Exposures = append(c.Exposures, jsExposure{
						Seq: x.Seq, Kind: x.Kind.String(), Range: x.Range,
					})
				}
				cases = append(cases, c)
			}
		}
	}
	if len(cases) < 120 {
		t.Fatalf("expected three positions across eight scenarios, got %d cases", len(cases))
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
		State     string `json:"state"`
		Tier      string `json:"tier"`
		Note      string `json:"note"`
		Announced string `json:"announced"`
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
				t.Errorf("%s, %s: Go says %s/%s, the report script says %s/%s",
					c.At, c.Node.Name, c.WantState, c.WantTier, got[i].State, got[i].Tier)
			}
			continue
		}
		// The announcement has to name the state the map is showing, and say
		// which moment it is showing when that is not the end.
		if got[i].Note != c.WantNote {
			mismatches++
			if mismatches <= 8 {
				t.Errorf("%s, %s: announced qualifier %q, want %q",
					c.At, c.Node.Name, got[i].Note, c.WantNote)
			}
			continue
		}
		if !strings.Contains(got[i].Announced, c.WantState) {
			mismatches++
			if mismatches <= 8 {
				t.Errorf("%s, %s: announcement %q does not name the state %q",
					c.At, c.Node.Name, got[i].Announced, c.WantState)
			}
		}
	}
	if mismatches > 0 {
		t.Fatalf("%d of %d cases disagree between the two implementations", mismatches, len(cases))
	}
	t.Logf("compared %d cases: eight scenarios at three playhead positions each", len(cases))
}
