package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// Known-positive tests for the output-wide check.
//
// Four verifications in this project have returned a confident pass that was
// wrong, and every one of them had never been observed to fail. A check that
// has only ever passed is not a check yet; it is a check-shaped thing. These
// plant what the checker is for and assert it says so.
//
// The values planted here are invented. plantedperson is not a user on any
// machine and example.invalid is reserved by RFC 2606 precisely so it can
// never be a real address.
const (
	plantedHome   = "/home/plantedperson/work/secret-project"
	plantedEmail  = "planted.address@example.invalid"
	plantedAuthor = "Planted Person"
)

func TestScannerCatchesPlantedHomePathEmailAndAuthorName(t *testing.T) {
	artifact := []byte(`{
  "commands_run": ["entire graph snapshot --repo ` + plantedHome + `"],
  "session_said": "asked ` + plantedEmail + ` about it",
  "timeline": [{"seq": 1, "kind": "command", "cmd": "git commit --author ` + plantedAuthor + `"}]
}`)

	found := ScanArtifact(artifact, []string{plantedAuthor})
	want := map[string]string{
		"an absolute home path": "/home/plantedperson",
		"an email address":      plantedEmail,
		"an author name":        plantedAuthor,
	}
	for what, match := range want {
		hit := false
		for _, f := range found {
			if f.What == what && strings.Contains(f.Match, match) {
				hit = true
				if f.Line < 1 {
					t.Errorf("%s was found without a line number", what)
				}
			}
		}
		if !hit {
			t.Errorf("the scanner missed %s (%q); it found %v", what, match, found)
		}
	}
}

// The map escapes its angle brackets and the packet is markdown, so a leak
// must be caught in whatever shape the artifact happens to have.
func TestScannerCatchesAPlantedLeakInEveryArtifactShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		blob string
	}{
		{"json", `{"cmd": "ls ` + plantedHome + `"}`},
		{"json with an escaped slash", `{"cmd": "ls <\/b>` + strings.ReplaceAll(plantedHome, "/", `\/`) + `"}`},
		{"html", `<p>ran in <code>` + plantedHome + `</code></p>`},
		{"markdown", "Reproduce:\n\n    cd " + plantedHome + "\n"},
	} {
		if got := ScanArtifact([]byte(tc.blob), nil); len(got) == 0 {
			t.Errorf("%s: the scanner found nothing in %q", tc.name, tc.blob)
		}
	}
}

// A clean artifact must not trip it, or the check would be noise and get
// switched off.
func TestScannerPassesACleanArtifact(t *testing.T) {
	clean := []byte(`{"commands_run":["entire graph snapshot --repo <repo>"],"cmd":"git -C <home>/x log"}`)
	if got := ScanArtifact(clean, []string{plantedAuthor}); len(got) != 0 {
		t.Errorf("the scanner tripped on a scrubbed artifact: %v", got)
	}
}

// The other half of the same proof: what the checker finds, sealing removes.
// Planting into the analysis rather than into the bytes is what makes this a
// test of the choke point and not of the regular expressions.
func TestSealRemovesWhatTheScannerFinds(t *testing.T) {
	a := mkAnalysis()
	a.Commands = append(a.Commands, "entire graph snapshot --repo "+plantedHome)
	a.SessionSaid = "wrote to " + plantedEmail + " about " + plantedHome
	a.Timeline = append(a.Timeline, TimelineEvent{
		Seq: 99, Kind: "command", Cmd: "git commit --author " + plantedAuthor,
	})
	a.Nodes[0].Failure = "AssertionError in " + plantedHome + "/tests/test_x.py"

	s := NewScrubber("")
	s.Home = "/home/plantedperson"
	s.User = "plantedperson"
	s.Names = []string{plantedAuthor}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, Seal(a, s)); err != nil {
		t.Fatalf("WriteJSON: %v", err)
	}
	if got := ScanArtifact(buf.Bytes(), []string{plantedAuthor}); len(got) != 0 {
		t.Errorf("sealing left %v in the report", got)
	}
	// And it really did carry them a moment ago, so the assertion above is
	// not passing because the values never arrived. The JSON encoder escapes
	// angle brackets, so the placeholder is read back rather than grepped.
	var out struct {
		CommandsRun []string `json:"commands_run"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	placeholder := false
	for _, c := range out.CommandsRun {
		if strings.Contains(c, "<home>") {
			placeholder = true
		}
	}
	if !placeholder {
		t.Error("no placeholder in the output, so the scrub may have found nothing to do")
	}
}

// The zero value is the only Sealed a caller inside this package can make
// without going through Seal, and every renderer has to refuse it. Without
// this, a forgotten seal would write an empty report and look like it worked.
func TestRenderersRefuseAnUnsealedAnalysis(t *testing.T) {
	var zero Sealed
	var buf bytes.Buffer

	if err := WriteJSON(&buf, zero); err != ErrUnsealed {
		t.Errorf("WriteJSON accepted an unsealed analysis: %v", err)
	}
	if _, err := MarshalJSON(zero); err != ErrUnsealed {
		t.Errorf("MarshalJSON accepted an unsealed analysis: %v", err)
	}
	if err := HTML(&buf, zero); err != ErrUnsealed {
		t.Errorf("HTML accepted an unsealed analysis: %v", err)
	}
	if err := Packet(&buf, zero); err != ErrUnsealed {
		t.Errorf("Packet accepted an unsealed analysis: %v", err)
	}
	if err := Table(&buf, zero, TableOptions{}); err != ErrUnsealed {
		t.Errorf("Table accepted an unsealed analysis: %v", err)
	}
	if buf.Len() != 0 {
		t.Errorf("a refused render still wrote %d bytes", buf.Len())
	}
}

// The layout is built from scrubbed values, so a label cannot carry what the
// node name no longer does.
func TestSealScrubsTheLayoutLabels(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Symbol.File = plantedHome + "/app/service.py"

	s := NewScrubber("")
	s.Home = "/home/plantedperson"
	s.User = "plantedperson"

	sd := Seal(a, s)
	if sd.Analysis().Layout == nil {
		t.Fatal("sealing did not build a layout")
	}
	blob, err := MarshalJSON(sd)
	if err != nil {
		t.Fatal(err)
	}
	if got := ScanArtifact(blob, nil); len(got) != 0 {
		t.Errorf("the layout carried %v", got)
	}
}
