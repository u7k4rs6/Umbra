package transcript

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Path shapes seen in tool results. Two forms matter:
//
//   - a grep style hit, "app/refunds.py:23:  refundable = ...", which the
//     Step 0 probe confirmed is how search output reaches the transcript in a
//     harness whose search runs through Bash;
//   - a bare path on its own line, which is how a file listing or a glob
//     result arrives.
//
// Anything that does not look like a path with a file extension is ignored.
// Candidates are filtered against the real file list later, so a false
// positive here cannot invent evidence for a file that does not exist.
var (
	hitRE  = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_./@+-]+\.[A-Za-z0-9_+-]+):\d+[:-]`)
	bareRE = regexp.MustCompile(`(?m)^\s*([A-Za-z0-9_./@+-]+\.[A-Za-z0-9_+-]+)\s*$`)
	// A diff header names the file whose content follows.
	diffRE = regexp.MustCompile(`(?m)^(?:\+\+\+|---)\s+[ab]?/?([A-Za-z0-9_./@+-]+\.[A-Za-z0-9_+-]+)`)
)

// extractPaths pulls candidate repository paths out of tool result text.
//
// Only the paths are kept. The surrounding text is discarded here and never
// reaches a report, which is what SECURITY_AND_ACCESS.md requires: tool result
// content is used to detect paths and is then dropped.
func extractPaths(text, root string) []string {
	if text == "" {
		return nil
	}
	seen := map[string]bool{}
	add := func(p string) {
		p = strings.TrimSpace(p)
		if p == "" {
			return
		}
		p = rel(root, p)
		if p == "" || strings.HasPrefix(p, "..") {
			return
		}
		// A single dot segment or a bare extension is noise.
		if !strings.Contains(filepath.Base(p), ".") {
			return
		}
		seen[p] = true
	}

	for _, re := range []*regexp.Regexp{hitRE, diffRE, bareRE} {
		for _, m := range re.FindAllStringSubmatch(text, -1) {
			add(m[1])
		}
	}

	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}
