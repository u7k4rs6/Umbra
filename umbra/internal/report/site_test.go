package report

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// The landing page uses the report's own stylesheet and script, copied by
// gen-site. If they drift, the page and the report disagree about what a state
// looks like, so this checks the copies are current.
func TestSiteAssetsMatchTheReportAssets(t *testing.T) {
	for _, name := range []string{"umbra.css", "umbra.js"} {
		want, err := assets.ReadFile("assets/" + name)
		if err != nil {
			t.Fatalf("reading the embedded %s: %v", name, err)
		}
		got, err := os.ReadFile(filepath.Join("..", "..", "site", name))
		if err != nil {
			t.Fatalf("the site copy of %s is missing; run go run ./internal/report/gen-site: %v", name, err)
		}
		if !bytes.Equal(want, got) {
			t.Errorf("site/%s is out of date; run go run ./internal/report/gen-site -root .", name)
		}
	}
}

// The sample the page draws must be a real report with a layout, or the map
// silently renders nothing.
func TestSiteSampleIsAUsableReport(t *testing.T) {
	blob, err := os.ReadFile(filepath.Join("..", "..", "site", "sample", "umbra.json"))
	if err != nil {
		t.Fatalf("the sample is missing: %v", err)
	}
	for _, key := range []string{`"layout"`, `"nodes"`, `"timeline"`, `"summary"`} {
		if !bytes.Contains(blob, []byte(key)) {
			t.Errorf("the sample has no %s", key)
		}
	}
	page, err := os.ReadFile(filepath.Join("..", "..", "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(page, []byte(`"layout"`)) {
		t.Error("the sample is not embedded in index.html, so the page cannot draw from file://")
	}
	// A page opened from file:// cannot fetch, so it must not try.
	if bytes.Contains(page, []byte("fetch(")) {
		t.Error("the landing page must not fetch anything")
	}
}

// The sample the landing page draws has to be a report that was run, not one
// that was written. It used to be assembled by the generator: checkpoint id
// "sample000class", a commit of all zeros, session id "scenario-session", and
// a session sentence composed by hand, under a caption calling it a real
// report. These are the markers of that, and they must never come back.
func TestSiteSampleIsNotFabricated(t *testing.T) {
	for _, rel := range sampleReports(t) {
		blob, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		var d struct {
			Checkpoint struct {
				ID         string   `json:"id"`
				Commit     string   `json:"commit"`
				SessionIDs []string `json:"session_ids"`
			} `json:"checkpoint"`
		}
		if err := json.Unmarshal(blob, &d); err != nil {
			t.Fatalf("parsing %s: %v", rel, err)
		}

		if strings.HasPrefix(d.Checkpoint.ID, "sample") {
			t.Errorf("%s: checkpoint id %q was invented", rel, d.Checkpoint.ID)
		}
		if d.Checkpoint.Commit == "" || strings.Trim(d.Checkpoint.Commit, "0") == "" {
			t.Errorf("%s: commit %q is a placeholder", rel, d.Checkpoint.Commit)
		}
		for _, sid := range d.Checkpoint.SessionIDs {
			if sid == "scenario-session" {
				t.Errorf("%s: session id %q belongs to an authored scenario", rel, sid)
			}
		}
	}
}

// sampleReports lists the reports the landing page draws.
func sampleReports(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, rel := range []string{
		filepath.Join("..", "..", "site", "sample", "umbra.json"),
		filepath.Join("..", "..", "site", "imported", "umbra.json"),
	} {
		if _, err := os.Stat(rel); err == nil {
			out = append(out, rel)
		}
	}
	if len(out) == 0 {
		t.Fatal("the landing page has no sample report at all")
	}
	return out
}

// The caption must not call a map real unless the report behind it is.
func TestSiteCaptionDoesNotOverclaim(t *testing.T) {
	page, err := os.ReadFile(filepath.Join("..", "..", "site", "index.html"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(page)
	if strings.Contains(text, "This is a real report from the fixture app in this repository.") {
		t.Error("the old caption is back; it called an assembled report a real one")
	}
	// The page has to name what is arranged about the sample.
	if !strings.Contains(text, "seeded fixture") {
		t.Error("the caption should say the fixture is seeded")
	}
	// And it has to name the checkpoint the reader can go and check.
	if !strings.Contains(text, "b20f84567474") {
		t.Error("the caption should name the checkpoint the map came from")
	}
}
