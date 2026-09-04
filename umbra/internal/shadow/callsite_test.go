package shadow

import (
	"os"
	"path/filepath"
	"testing"
)

// The fault line and beacon checks are a regex window, not a parser, which
// PRD.md discloses. These tests pin what the window does and does not match.

func TestFaultLinePerLanguage(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"python except", []string{"    try:", "        x = f()", "    except TypeError:"}, true},
		{"python finally", []string{"    finally:", "        close()"}, true},
		{"ruby rescue", []string{"  rescue StandardError => e", "    log(e)"}, true},
		{"go defer", []string{"\tdefer file.Close()"}, true},
		{"go error check", []string{"\tif err != nil {", "\t\treturn err"}, true},
		{"javascript catch", []string{"} catch (e) {", "  report(e)"}, true},
		{"plain code", []string{"    total = compute_total(items)", "    return total"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := ScanWindow(c.lines).FaultLine; got != c.want {
				t.Fatalf("FaultLine = %v, want %v for %v", got, c.want, c.lines)
			}
		})
	}
}

// A word that only appears inside a comment is not error handling.
func TestFaultLineIgnoresComments(t *testing.T) {
	lines := []string{"    # we could catch this later", "    total = compute_total(items)"}
	if ScanWindow(lines).FaultLine {
		t.Fatal("a mention of catch in a comment is not a fault line")
	}
}

func TestBeaconMarkers(t *testing.T) {
	cases := []struct {
		name  string
		lines []string
		want  bool
	}{
		{"python safety", []string{"        # SAFETY: must not refund the full amount"}, true},
		{"go critical", []string{"\t// CRITICAL: ordering matters here"}, true},
		{"invariant lowercase", []string{"# invariant: total is never negative"}, true},
		{"sql comment", []string{"-- CRITICAL: this view is load bearing"}, true},
		{"not a comment", []string{"CRITICAL = 3"}, false},
		{"plain comment", []string{"# just an ordinary note"}, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := ScanWindow(c.lines).Beacon != ""
			if got != c.want {
				t.Fatalf("Beacon = %v, want %v for %v", got, c.want, c.lines)
			}
		})
	}
}

func TestBeaconKeepsOnlyTheCommentLine(t *testing.T) {
	lines := []string{
		"    except TypeError:",
		"        # SAFETY: a pricing failure must not refund the full amount paid.",
		"        refundable = compute_total([])",
	}
	w := ScanWindow(lines)
	if !w.FaultLine {
		t.Fatal("expected a fault line")
	}
	if w.Beacon != "# SAFETY: a pricing failure must not refund the full amount paid." {
		t.Fatalf("beacon = %q", w.Beacon)
	}
	// The surrounding code must not be retained.
	if len(w.Beacon) > 200 {
		t.Fatal("the beacon line must be capped")
	}
}

func TestBeaconIsCapped(t *testing.T) {
	long := "# SAFETY: "
	for i := 0; i < 100; i++ {
		long += "long "
	}
	w := ScanWindow([]string{long})
	if len(w.Beacon) > 203 {
		t.Fatalf("beacon length = %d, want it capped near 200", len(w.Beacon))
	}
}

// The window is read from the head worktree around the call-site line.
func TestReadWindowFromDisk(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "app"), 0o755); err != nil {
		t.Fatal(err)
	}
	body := "line1\nline2\nline3\ntry:\n    # CRITICAL: keep this ordering\n    x = f()\nexcept E:\n    pass\nline9\nline10\nline11\nline12\n"
	if err := os.WriteFile(filepath.Join(root, "app", "x.py"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	w := ReadWindow(root, "app/x.py", 6)
	if !w.FaultLine {
		t.Fatal("expected the try and except within four lines of line 6")
	}
	if w.Beacon == "" {
		t.Fatal("expected the CRITICAL comment")
	}

	// Far away from the annotated block, neither matches.
	if w := ReadWindow(root, "app/x.py", 12); w.FaultLine || w.Beacon != "" {
		t.Fatalf("line 12 should be clear, got %+v", w)
	}
}

func TestReadWindowMissingFileIsNotAnError(t *testing.T) {
	if w := ReadWindow(t.TempDir(), "nope/missing.py", 5); w.FaultLine || w.Beacon != "" {
		t.Fatal("a missing file yields an empty window rather than an error")
	}
}

func TestReadWindowNoCallSite(t *testing.T) {
	if w := ReadWindow(t.TempDir(), "app/x.py", 0); w.FaultLine || w.Beacon != "" {
		t.Fatal("no call site means no window")
	}
}

func TestFarField(t *testing.T) {
	cases := []struct {
		node, source string
		want         bool
	}{
		{"tests/test_service.py", "app/service.py", true},
		{"app/api.py", "app/service.py", false},
		{"app/sub/deep.py", "app/service.py", false},
		{"service.py", "app/service.py", true},
	}
	for _, c := range cases {
		if got := FarField(c.node, c.source); got != c.want {
			t.Errorf("FarField(%q, %q) = %v, want %v", c.node, c.source, got, c.want)
		}
	}
}
