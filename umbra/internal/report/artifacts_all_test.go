package report

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The output-wide check.
//
// The earlier artifact tests named the files they knew about, which is how a
// new output path gets missed. This walks the trees instead, so a file that appears
// tomorrow is covered tomorrow.

// generatedArtifacts walks every tree Umbra writes into and returns the text
// files in them.
func generatedArtifacts(t *testing.T) []string {
	t.Helper()
	var out []string
	for _, root := range []string{
		filepath.Join("..", "..", "site"),
		filepath.Join("..", "..", "fixtures", "recorded"),
		filepath.Join("..", "..", "docs", "renders"),
	} {
		err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			switch strings.ToLower(filepath.Ext(path)) {
			case ".json", ".html", ".md", ".js", ".css", ".txt", ".ndjson":
				out = append(out, path)
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walking %s: %v", root, err)
		}
	}
	if len(out) < 5 {
		t.Fatalf("only %d artifacts found; the walk is looking in the wrong place", len(out))
	}
	return out
}

// authorNames returns the display names that appear in this repository's
// history, which are the names an artifact must not carry. Reading them from
// git rather than hard-coding one means the test works in a clone made by
// somebody else, and fails for their name as readily as for the author's.
func authorNames(t *testing.T) []string {
	t.Helper()
	seen := map[string]bool{}
	var out []string
	add := func(name string) {
		name = strings.TrimSpace(name)
		if len(name) > 2 && !seen[name] {
			seen[name] = true
			out = append(out, name)
		}
	}
	for _, args := range [][]string{
		{"log", "--format=%an%n%cn", "-n", "200"},
		{"config", "--get", "user.name"},
	} {
		cmd := exec.Command("git", args...)
		cmd.Dir = filepath.Join("..", "..")
		blob, err := cmd.Output()
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(blob), "\n") {
			add(line)
		}
	}
	return out
}

// Every artifact this repository publishes, whichever code path produced it.
func TestNoGeneratedArtifactCarriesAnythingPrivate(t *testing.T) {
	names := authorNames(t)
	if len(names) == 0 {
		t.Log("git named no authors, so the author-name half of this check did nothing")
	}
	files := generatedArtifacts(t)
	t.Logf("scanning %d generated artifacts against %d author name(s)", len(files), len(names))
	for _, path := range files {
		blob, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("reading %s: %v", path, err)
		}
		for _, f := range ScanArtifact(blob, names) {
			t.Errorf("%s carries %s", path, f)
		}
	}
}

// A path that leaves the repository says nothing about the change and gives
// away the layout of the machine the session ran on. This is the structural
// half: the scanner reads bytes, this reads the fields that are meant to hold
// repository-relative paths.
func TestNoGeneratedReportCarriesAnOutOfRepoPath(t *testing.T) {
	for name, d := range everyCommittedReport(t) {
		timeline, _ := d["timeline"].([]any)
		for _, raw := range timeline {
			e, _ := raw.(map[string]any)
			var paths []string
			if p, ok := e["path"].(string); ok && p != "" {
				paths = append(paths, p)
			}
			if list, ok := e["paths"].([]any); ok {
				for _, p := range list {
					if s, ok := p.(string); ok {
						paths = append(paths, s)
					}
				}
			}
			for _, p := range paths {
				if strings.HasPrefix(p, "/") {
					t.Errorf("%s: absolute path in the timeline: %q", name, p)
				}
				if strings.Contains(p, "..") {
					t.Errorf("%s: path escapes the repository: %q", name, p)
				}
			}
		}
		for _, key := range []string{"nodes", "sources"} {
			list, _ := d[key].([]any)
			for _, raw := range list {
				n, _ := raw.(map[string]any)
				f, _ := n["file"].(string)
				if strings.HasPrefix(f, "/") || strings.Contains(f, "..") {
					t.Errorf("%s: %s carries a file outside the repository: %q", name, key, f)
				}
			}
		}
	}
}
