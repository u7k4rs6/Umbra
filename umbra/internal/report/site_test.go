package report

import (
	"bytes"
	"os"
	"path/filepath"
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
