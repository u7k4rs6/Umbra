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
// is. TestNoGeneratedArtifactCarriesAnythingPrivate is the wide check; this
// file holds the ones that need to understand the report structure rather than
// only its bytes.

// everyCommittedReport parses every umbra.json under site/. It walks rather
// than listing, because a list is a thing to forget to add to.
func everyCommittedReport(t *testing.T) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	root := filepath.Join("..", "..", "site")
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Base(path) != "umbra.json" {
			return nil
		}
		blob, readErr := os.ReadFile(path)
		if readErr != nil {
			return readErr
		}
		var m map[string]any
		if jsonErr := json.Unmarshal(blob, &m); jsonErr != nil {
			return jsonErr
		}
		out[path] = m
		return nil
	})
	if err != nil {
		t.Fatalf("walking %s: %v", root, err)
	}
	if len(out) == 0 {
		t.Fatal("no committed report was found at all")
	}
	return out
}

// The reproduce list is only worth having if a reader can check a line, and a
// git object id is forty hex characters, which matches the base64 shape.
func TestCommittedReportsKeepTheirGitObjectIDs(t *testing.T) {
	sha := regexp.MustCompile(`\b[0-9a-f]{40}\b`)
	d := everyCommittedReport(t)[filepath.Join("..", "..", "site", "self", "umbra.json")]

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
