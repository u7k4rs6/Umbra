// Command gen-site builds the landing page.
//
// It copies the report's own CSS and JavaScript into site/ so the two surfaces
// cannot drift apart, and embeds the committed sample report into the page.
//
// The sample is not built here. It is produced by a real run and committed,
// and this program reads it. A page whose caption calls the map a real session
// must not draw a report that a generator invented, which is what it used to
// do: an assembled report with a checkpoint id of "sample000class", a commit
// of all zeros and a session sentence written by hand.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	root := flag.String("root", ".", "the umbra module directory")
	flag.Parse()

	if err := run(*root); err != nil {
		fmt.Fprintf(os.Stderr, "gen-site: %v\n", err)
		os.Exit(1)
	}
}

func run(root string) error {
	site := filepath.Join(root, "site")
	if err := os.MkdirAll(filepath.Join(site, "sample"), 0o755); err != nil {
		return err
	}

	// The page uses the report's own stylesheet and script.
	for _, name := range []string{"umbra.css", "umbra.js"} {
		src := filepath.Join(root, "internal", "report", "assets", name)
		blob, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(site, name), blob, 0o644); err != nil {
			return err
		}
		fmt.Printf("copied %s\n", name)
	}

	samplePath := filepath.Join(site, "sample", "umbra.json")
	blob, err := os.ReadFile(samplePath)
	if err != nil {
		return fmt.Errorf("the sample report is missing; produce one with a real run:\n"+
			"  entire umbra <ref> --test \"pytest -v\" --out %s\n%w", filepath.Dir(samplePath), err)
	}
	if err := CheckSampleIsReal(blob); err != nil {
		return err
	}

	// The sample is embedded in the page as well as sitting beside it. A page
	// opened from file:// cannot fetch its own sibling, and the landing page
	// has to work from file://.
	if err := embedSample(filepath.Join(site, "index.html"), blob, sampleTag); err != nil {
		return err
	}
	fmt.Printf("embedded %s (%d bytes)\n", samplePath, len(blob))

	// The second map, when there is one, is a report from a session in another
	// project. It is optional: the page renders without it.
	importedPath := filepath.Join(site, "imported", "umbra.json")
	if blob, err := os.ReadFile(importedPath); err == nil {
		if err := embedSample(filepath.Join(site, "index.html"), blob, importedTag); err != nil {
			return err
		}
		fmt.Printf("embedded %s (%d bytes)\n", importedPath, len(blob))
	}
	return nil
}

// CheckSampleIsReal refuses the markers of a report that was written rather
// than run. The landing page says the map is a real session; this is what
// stops that sentence from drifting away from the file it describes.
func CheckSampleIsReal(blob []byte) error {
	var d struct {
		Checkpoint struct {
			ID     string `json:"id"`
			Commit string `json:"commit"`
		} `json:"checkpoint"`
	}
	if err := json.Unmarshal(blob, &d); err != nil {
		return fmt.Errorf("reading the sample: %w", err)
	}
	if strings.HasPrefix(d.Checkpoint.ID, "sample") {
		return fmt.Errorf("the sample carries an invented checkpoint id %q", d.Checkpoint.ID)
	}
	if d.Checkpoint.Commit == "" || strings.Trim(d.Checkpoint.Commit, "0") == "" {
		return fmt.Errorf("the sample carries a placeholder commit %q", d.Checkpoint.Commit)
	}
	return nil
}

const (
	sampleTag   = "umbra-data"
	importedTag = "umbra-imported"
	closeTag    = "</script>"
)

// embedSample replaces the contents of one data block in the page.
func embedSample(path string, blob []byte, id string) error {
	page, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	open := `<script type="application/json" id="` + id + `">`
	text := string(page)
	start := strings.Index(text, open)
	if start < 0 {
		return fmt.Errorf("%s has no %s block", path, id)
	}
	from := start + len(open)
	end := strings.Index(text[from:], closeTag)
	if end < 0 {
		return fmt.Errorf("%s has an unclosed %s block", path, id)
	}

	// The same escaping the report uses: a literal closing tag inside a string
	// would end the script element early.
	safe := strings.ReplaceAll(string(blob), "</", `<\/`)
	return os.WriteFile(path, []byte(text[:from]+safe+text[from+end:]), 0o644)
}
