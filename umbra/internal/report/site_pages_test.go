package report

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The site is more than one page, and its header and footer are written into
// each page rather than assembled at build time. That is fine as long as
// nothing is allowed to drift, which is what these check.

func sitePages(t *testing.T) map[string]string {
	t.Helper()
	dir := filepath.Join("..", "..", "site")
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	out := map[string]string{}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".html" {
			continue
		}
		blob, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatalf("reading %s: %v", e.Name(), err)
		}
		out[e.Name()] = string(blob)
	}
	if len(out) < 2 {
		t.Fatalf("expected a site of several pages, found %d", len(out))
	}
	return out
}

// Every page carries the same navigation, so a link added to one is added to
// all of them or this fails.
func TestEveryPageCarriesTheSameNav(t *testing.T) {
	link := regexp.MustCompile(`<a href="([a-z]+\.html)"[^>]*>([^<]+)</a>`)
	var want []string
	var from string
	for name, page := range sitePages(t) {
		start := strings.Index(page, `<nav class="pill mono"`)
		end := strings.Index(page, "</nav>")
		if start < 0 || end < start {
			t.Errorf("%s has no page navigation", name)
			continue
		}
		var got []string
		for _, m := range link.FindAllStringSubmatch(page[start:end], -1) {
			got = append(got, m[1]+" "+strings.TrimSpace(m[2]))
		}
		if len(got) == 0 {
			t.Errorf("%s has an empty navigation", name)
			continue
		}
		if want == nil {
			want, from = got, name
			continue
		}
		if strings.Join(got, "|") != strings.Join(want, "|") {
			t.Errorf("%s navigation differs from %s:\n  %v\n  %v", name, from, got, want)
		}
	}
}

// Every navigation target exists, so no page links into nothing.
func TestEveryNavTargetExists(t *testing.T) {
	pages := sitePages(t)
	href := regexp.MustCompile(`href="([a-z]+\.html)"`)
	for name, page := range pages {
		for _, m := range href.FindAllStringSubmatch(page, -1) {
			if _, ok := pages[m[1]]; !ok {
				t.Errorf("%s links to %s, which is not a page", name, m[1])
			}
		}
	}
}

// Every page carries the same footer, and the stylesheets and the security
// policy every page depends on.
func TestEveryPageCarriesTheSameChrome(t *testing.T) {
	for name, page := range sitePages(t) {
		for _, want := range []string{
			`<link rel="stylesheet" href="umbra.css">`,
			`<link rel="stylesheet" href="landing.css">`,
			`<meta http-equiv="Content-Security-Policy"`,
			`class="skip mono"`,
			`<footer class="site-foot">`,
			`Built during Bengaluru Tech Week`,
			`<script src="site.js"></script>`,
			`<script src="motion.js"></script>`,
		} {
			if !strings.Contains(page, want) {
				t.Errorf("%s is missing %s", name, want)
			}
		}
		// Nothing on this site fetches anything.
		if strings.Contains(page, "fetch(") {
			t.Errorf("%s fetches something", name)
		}
		if strings.Contains(page, "https://fonts.") || strings.Contains(page, "cdn.") {
			t.Errorf("%s reaches for an external asset", name)
		}
	}
}

// Only the pages that draw a map carry the report data, so the rest stay
// small. A page that carries one block must carry both.
func TestOnlyMapPagesCarryTheReportData(t *testing.T) {
	for name, page := range sitePages(t) {
		hasSample := strings.Contains(page, `id="umbra-data"`)
		hasImported := strings.Contains(page, `id="umbra-imported"`)
		if hasSample != hasImported {
			t.Errorf("%s carries one data block and not the other", name)
		}
		drawsMap := strings.Contains(page, `id="map"`) || strings.Contains(page, `id="map-imported"`)
		if drawsMap && !hasSample {
			t.Errorf("%s draws a map but carries no report data", name)
		}
		if !drawsMap && hasSample {
			t.Errorf("%s carries the report data but draws no map", name)
		}
		if hasSample && !strings.Contains(page, `<script src="umbra.js"></script>`) {
			t.Errorf("%s draws a map without the report's script", name)
		}
	}
}
