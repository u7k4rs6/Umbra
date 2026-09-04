package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Every report committed to this repository is published when the repository
// is. These guard the two scrubber fixes and the categories
// SECURITY_AND_ACCESS.md names, against the real files rather than against a
// fixture, so a regenerated artifact cannot quietly reintroduce a leak.

func committedReports(t *testing.T) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, rel := range []string{
		filepath.Join("..", "..", "site", "self", "umbra.json"),
		filepath.Join("..", "..", "site", "sample", "umbra.json"),
		filepath.Join("..", "..", "site", "imported", "umbra.json"),
	} {
		blob, err := os.ReadFile(rel)
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		var d map[string]any
		if err := json.Unmarshal(blob, &d); err != nil {
			t.Fatalf("parsing %s: %v", rel, err)
		}
		out[rel] = d
	}
	return out
}

// A path outside the repository can say nothing about the change, and carrying
// one out leaks the layout of the machine the session ran on. An address and a
// domain both match the path shape, so both used to get through.
func TestCommittedReportsCarryOnlyRepositoryPaths(t *testing.T) {
	for name, d := range committedReports(t) {
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
				if strings.Contains(p, "@") {
					t.Errorf("%s: an address is not a path: %q", name, p)
				}
				if strings.Contains(p, "..") {
					t.Errorf("%s: path escapes the repository: %q", name, p)
				}
			}
		}
	}
}

// The reproduce list is only worth having if a reader can check a line, and a
// git object id is forty hex characters, which matches the base64 shape.
func TestCommittedReportsKeepTheirGitObjectIDs(t *testing.T) {
	sha := regexp.MustCompile(`\b[0-9a-f]{40}\b`)
	d := committedReports(t)[filepath.Join("..", "..", "site", "self", "umbra.json")]

	cmds, _ := d["commands_run"].([]any)
	if len(cmds) == 0 {
		t.Fatal("the report should record the commands it ran")
	}
	found := 0
	for _, raw := range cmds {
		c, _ := raw.(string)
		if strings.Contains(c, "<redacted>") {
			t.Errorf("a reproduce command was redacted and cannot be checked: %q", c)
		}
		if sha.MatchString(c) {
			found++
		}
	}
	if found == 0 {
		t.Error("no reproduce command carries a git object id, so none can be checked")
	}
}

// The categories SECURITY_AND_ACCESS.md says must never reach a report.
func TestCommittedArtifactsCarryNothingPrivate(t *testing.T) {
	files := []string{
		filepath.Join("..", "..", "site", "self", "umbra.json"),
		filepath.Join("..", "..", "site", "self", "umbra.html"),
		filepath.Join("..", "..", "site", "self", "umbra.packet.md"),
		filepath.Join("..", "..", "site", "sample", "umbra.json"),
		filepath.Join("..", "..", "site", "index.html"),
		filepath.Join("..", "..", "site", "imported", "umbra.json"),
		filepath.Join("..", "..", "site", "imported", "umbra.html"),
		filepath.Join("..", "..", "site", "imported", "umbra.packet.md"),
		filepath.Join("..", "..", "fixtures", "recorded", "minimal", "recording.json"),
	}
	banned := []struct {
		what string
		re   *regexp.Regexp
	}{
		{"an absolute home path", regexp.MustCompile(`/home/[A-Za-z0-9._-]+`)},
		{"an email address", regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},
		{"a github token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{16,}`)},
		{"an api key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`)},
		{"a slack token", regexp.MustCompile(`xox[abposr]-[A-Za-z0-9-]{10,}`)},
		{"an aws key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
		{"a private key", regexp.MustCompile(`BEGIN [A-Z ]*PRIVATE KEY`)},
	}
	unescape := strings.NewReplacer(`<`, "<", `>`, ">")
	for _, f := range files {
		blob, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		text := unescape.Replace(string(blob))
		for _, b := range banned {
			if m := b.re.FindString(text); m != "" {
				t.Errorf("%s contains %s: %q", f, b.what, m)
			}
		}
	}
}
