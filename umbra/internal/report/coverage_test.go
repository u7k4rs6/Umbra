package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// coverageFor parses a recorded scenario and returns the analysis the
// renderers would see, so the line is checked against a real transcript rather
// than against hand-made counts.
func coverageFor(t *testing.T, scenario string) *Analysis {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("..", "..", "fixtures", "recorded", scenario, "transcript.jsonl"))
	if err != nil {
		t.Fatalf("read scenario %s: %v", scenario, err)
	}
	s, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("parse scenario %s: %v", scenario, err)
	}
	a := mkAnalysis()
	a.Coverage = s.Coverage()
	return a
}

// The classic scenario reads twice, edits once and searches once, with no
// shell command at all.
func TestCoverageLineOnTheClassicScenario(t *testing.T) {
	a := coverageFor(t, "classic")
	c := a.Coverage
	if c.Reads != 2 || c.Edits != 1 || c.Searches != 1 || c.Commands != 0 {
		t.Fatalf("counts = %+v", c)
	}

	line := CoverageLine(a)
	for _, want := range []string{"2 file reads", "1 edit", "1 search", "0 shell commands"} {
		if !strings.Contains(line, want) {
			t.Errorf("the line is missing %q: %s", want, line)
		}
	}
	if note := CoverageNote(a); note != "" {
		t.Errorf("an ordinary session needs no note, got %q", note)
	}
}

// The no-reads scenario has no tool events at all, so the line says that and
// the note explains what follows from it.
func TestCoverageLineOnTheNoReadsScenario(t *testing.T) {
	a := coverageFor(t, "no-reads")
	c := a.Coverage
	if c.Reads != 0 || c.Edits != 0 || c.Commands != 0 {
		t.Fatalf("counts = %+v, want nothing at all", c)
	}

	line := CoverageLine(a)
	if line != "no tool events in this transcript" {
		t.Fatalf("line = %q", line)
	}
	note := CoverageNote(a)
	if !strings.Contains(note, "unknown") {
		t.Fatalf("the note must say every dependent is unknown, got %q", note)
	}
}

// A session that worked through the shell gets the plain sentence, naming both
// counts and saying that a mostly umbra report is expected.
func TestCoverageNoteForAShellOnlySession(t *testing.T) {
	a := mkAnalysis()
	a.Coverage = transcript.Coverage{Reads: 2, Edits: 4, Commands: 234}

	line := CoverageLine(a)
	if !strings.Contains(line, "2 file reads") || !strings.Contains(line, "234 shell commands") {
		t.Fatalf("line = %q", line)
	}
	note := CoverageNote(a)
	for _, want := range []string{
		"almost entirely through the shell",
		"6 file tool events",
		"234 shell commands",
		"mostly umbra report is the expected outcome",
	} {
		if !strings.Contains(note, want) {
			t.Errorf("the note is missing %q:\n%s", want, note)
		}
	}
}

func TestCoverageSingularWording(t *testing.T) {
	a := mkAnalysis()
	a.Coverage = transcript.Coverage{Reads: 1, Edits: 1, Searches: 1, Commands: 1}
	line := CoverageLine(a)
	for _, want := range []string{"1 file read,", "1 edit,", "1 search,", "1 shell command"} {
		if !strings.Contains(line, want) {
			t.Errorf("the line is missing %q: %s", want, line)
		}
	}
}

// The line has to reach every surface, or a reader of one of them is missing
// the context the others have.

func TestCoverageAppearsInTheTable(t *testing.T) {
	a := coverageFor(t, "classic")
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "coverage  2 file reads") {
		t.Fatalf("the table has no coverage line:\n%s", out)
	}
}

func TestCoverageNoteAppearsInTheTable(t *testing.T) {
	a := mkAnalysis()
	a.Coverage = transcript.Coverage{Reads: 1, Commands: 60}
	out := render(t, a, TableOptions{UTF8: true})
	if !strings.Contains(out, "almost entirely through the shell") {
		t.Fatalf("the table must carry the note:\n%s", out)
	}
}

func TestCoverageAppearsInTheJSON(t *testing.T) {
	a := coverageFor(t, "classic")
	got := jsonOf(t, a)
	raw, ok := got["coverage"].(map[string]any)
	if !ok {
		t.Fatalf("the report has no coverage object: %v", got["coverage"])
	}
	if raw["reads"].(float64) != 2 {
		t.Errorf("reads = %v, want 2", raw["reads"])
	}
	if raw["shell_commands"].(float64) != 0 {
		t.Errorf("shell_commands = %v, want 0", raw["shell_commands"])
	}
	if raw["shell_only"].(bool) {
		t.Error("the classic session is not shell only")
	}
	if !strings.Contains(raw["line"].(string), "2 file reads") {
		t.Errorf("line = %v", raw["line"])
	}
}

func TestCoverageThinFlagReachesTheJSON(t *testing.T) {
	a := mkAnalysis()
	a.Coverage = transcript.Coverage{Reads: 2, Edits: 4, Commands: 234}
	got := jsonOf(t, a)
	raw := got["coverage"].(map[string]any)
	if !raw["shell_only"].(bool) {
		t.Error("shell_only should be true")
	}
	if !strings.Contains(raw["note"].(string), "mostly umbra") {
		t.Errorf("note = %v", raw["note"])
	}
	// The whole report must still be valid JSON with the new object in it.
	var round map[string]any
	blob, _ := json.Marshal(got)
	if err := json.Unmarshal(blob, &round); err != nil {
		t.Fatalf("round trip: %v", err)
	}
}

func TestCoverageAppearsInThePacket(t *testing.T) {
	a := coverageFor(t, "classic")
	out := packetOf(t, a)
	if !strings.Contains(out, "How the session worked: 2 file reads") {
		t.Fatalf("the packet has no coverage line:\n%s", out)
	}
}

func TestCoverageAppearsInTheHTML(t *testing.T) {
	a := coverageFor(t, "classic")
	out := htmlOf(t, a)
	if !strings.Contains(out, "How the session worked: 2 file reads") {
		t.Fatal("the report page has no coverage line")
	}

	a = mkAnalysis()
	a.Coverage = transcript.Coverage{Reads: 1, Commands: 60}
	out = htmlOf(t, a)
	if !strings.Contains(out, "almost entirely through the shell") {
		t.Fatal("the report page must carry the note")
	}
}
