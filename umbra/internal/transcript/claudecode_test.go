package transcript

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func load(t *testing.T, name string) *Session {
	t.Helper()
	blob, err := os.ReadFile(filepath.Join("testdata", name))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	s, err := ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	return s
}

func firstOf(s *Session, k Kind, path string) (Event, bool) {
	for _, e := range s.Events {
		if e.Kind == k && (path == "" || e.Path == path) {
			return e, true
		}
	}
	return Event{}, false
}

func TestParseFullReadHasNoRange(t *testing.T) {
	s := load(t, "classic.jsonl")
	ev, ok := firstOf(s, Read, "app/service.py")
	if !ok {
		t.Fatal("no Read event for app/service.py")
	}
	if ev.Range != nil {
		t.Fatalf("a full read must have a nil range, got %v", *ev.Range)
	}
	if !ev.Covers(20, 30) {
		t.Fatal("a full read covers every span")
	}
}

// The probe confirmed offset and limit are stored. This is what makes the
// glance tier possible at all.
func TestParsePartialReadCarriesRange(t *testing.T) {
	s := load(t, "classic.jsonl")
	ev, ok := firstOf(s, Read, "app/api.py")
	if !ok {
		t.Fatal("no Read event for app/api.py")
	}
	if ev.Range == nil {
		t.Fatal("a partial read must carry a range")
	}
	if got := *ev.Range; got != [2]int{1, 12} {
		t.Fatalf("range = %v, want [1 12]", got)
	}
	// handle_order spans lines 14 to 23, past the end of the read.
	if ev.Covers(14, 23) {
		t.Fatal("the read stopped at line 12 and must not cover line 14")
	}
	if !ev.Covers(1, 11) {
		t.Fatal("the read should cover lines inside its range")
	}
}

func TestParseAbsolutePathsBecomeRepositoryRelative(t *testing.T) {
	s := load(t, "classic.jsonl")
	for _, e := range s.Events {
		if e.Path == "" {
			continue
		}
		if filepath.IsAbs(e.Path) {
			t.Fatalf("path %q should be repository relative", e.Path)
		}
	}
	if _, ok := firstOf(s, Read, "app/service.py"); !ok {
		t.Fatal("expected app/service.py after relativizing")
	}
}

func TestParseEdits(t *testing.T) {
	s := load(t, "classic.jsonl")
	var edited []string
	for _, e := range s.Events {
		if e.Kind == Edit {
			edited = append(edited, e.Path)
		}
	}
	if len(edited) != 2 {
		t.Fatalf("edits = %v, want two", edited)
	}
	if edited[0] != "app/service.py" || edited[1] != "app/api.py" {
		t.Fatalf("edits = %v", edited)
	}
}

// Search in this harness runs through Bash, so the hits arrive as result text.
// The adapter recovers the paths, which is the quoted tier.
func TestParseRecoversPathsFromResultText(t *testing.T) {
	s := load(t, "classic.jsonl")
	var got []string
	for _, e := range s.Events {
		if e.Kind == ResultFile {
			got = append(got, e.Paths...)
		}
	}
	want := map[string]bool{
		"app/service.py":        false,
		"app/refunds.py":        false,
		"app/api.py":            false,
		"tests/test_service.py": false,
	}
	for _, p := range got {
		if _, ok := want[p]; ok {
			want[p] = true
		}
	}
	for p, seen := range want {
		if !seen {
			t.Fatalf("expected %s among the quoted paths, got %v", p, got)
		}
	}
}

func TestParseCommandsAreRecordedButNeverRun(t *testing.T) {
	s := load(t, "classic.jsonl")
	var cmds []string
	for _, e := range s.Events {
		if e.Kind == Command {
			cmds = append(cmds, e.Cmd)
		}
	}
	if len(cmds) != 2 {
		t.Fatalf("commands = %v, want two", cmds)
	}
	if cmds[1] != "pytest tests/test_api.py" {
		t.Fatalf("second command = %q", cmds[1])
	}
}

func TestSequenceNumbersAreMonotonic(t *testing.T) {
	s := load(t, "classic.jsonl")
	last := 0
	for _, e := range s.Events {
		if e.Seq < last {
			t.Fatalf("sequence went backwards at %v", e)
		}
		last = e.Seq
	}
}

func TestFirstEditSeqIsTheCut(t *testing.T) {
	s := load(t, "classic.jsonl")
	cut, ok := s.FirstEditSeq(map[string]bool{"app/service.py": true})
	if !ok {
		t.Fatal("expected a cut")
	}
	readSeq := 0
	for _, e := range s.Events {
		if e.Kind == Read && e.Path == "app/service.py" {
			readSeq = e.Seq
		}
	}
	if cut <= readSeq {
		t.Fatalf("the cut (%d) must come after the read (%d)", cut, readSeq)
	}
}

func TestFirstEditSeqAbsent(t *testing.T) {
	s := load(t, "classic.jsonl")
	if _, ok := s.FirstEditSeq(map[string]bool{"app/nothing.py": true}); ok {
		t.Fatal("expected no cut for a file the session never edited")
	}
}

// A transcript with no tool records and no mentions yields no exposure at all,
// which is what makes every node unknown rather than an error.
func TestNoReadsHasNoExposure(t *testing.T) {
	s := load(t, "no-reads.jsonl")
	if s.HasExposure() {
		t.Fatal("a transcript with no tool events must report no exposure")
	}
}

func TestClassicHasExposure(t *testing.T) {
	if !load(t, "classic.jsonl").HasExposure() {
		t.Fatal("the classic transcript has plenty of exposure")
	}
}

func TestParseSkipsUnparseableLines(t *testing.T) {
	blob := []byte("{not json\n" +
		`{"type":"assistant","cwd":"/repo","timestamp":"2026-09-04T17:50:00.000Z","message":{"content":[{"type":"tool_use","id":"a","name":"Read","input":{"file_path":"/repo/app/x.py"}}]}}` + "\n" +
		"\n")
	s, err := ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if _, ok := firstOf(s, Read, "app/x.py"); !ok {
		t.Fatal("a bad line must not lose the good ones")
	}
}

func TestParseEmptyTranscript(t *testing.T) {
	s, err := ClaudeCode{}.Parse(nil)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if s.HasExposure() {
		t.Fatal("an empty transcript has no exposure")
	}
}

func TestSessionIDsCollected(t *testing.T) {
	s := load(t, "classic.jsonl")
	if len(s.SessionIDs) != 1 || s.SessionIDs[0] != "0000-session" {
		t.Fatalf("session ids = %v", s.SessionIDs)
	}
}

func TestFilesListsEverythingTouched(t *testing.T) {
	files := load(t, "classic.jsonl").Files()
	want := []string{"app/api.py", "app/refunds.py", "app/service.py", "tests/test_service.py"}
	for _, w := range want {
		found := false
		for _, f := range files {
			if f == w {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected %s in %v", w, files)
		}
	}
}

// A search that happens to list files elsewhere on the machine must not put
// those paths in the report. They can never match a symbol in the field, and
// carrying them out would leak the layout of the machine the session ran on.
func TestPathsOutsideTheRepositoryAreDropped(t *testing.T) {
	blob := []byte(
		`{"type":"assistant","cwd":"/repo","timestamp":"2026-09-05T10:00:00.000Z","message":{"content":[{"type":"tool_use","id":"a","name":"Bash","input":{"command":"find /home"}}]}}` + "\n" +
			`{"type":"user","cwd":"/repo","timestamp":"2026-09-05T10:00:01.000Z","message":{"content":[{"type":"tool_result","tool_use_id":"a","content":"/home/someone/other/HANDOFF.md\n/home/someone/Downloads/notes.md\napp/service.py\n"}]}}` + "\n")

	s, err := ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, e := range s.Events {
		for _, p := range e.Paths {
			if strings.HasPrefix(p, "/") || strings.Contains(p, "someone") {
				t.Fatalf("a path outside the repository reached the events: %q", p)
			}
		}
	}
	// The one path that is inside the repository survives.
	found := false
	for _, e := range s.Events {
		for _, p := range e.Paths {
			if p == "app/service.py" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("a repository path should still be recovered")
	}
}

// A read of a file outside the repository is outside the field, so it produces
// no event rather than an event nothing can match.
func TestReadOutsideTheRepositoryProducesNoEvent(t *testing.T) {
	blob := []byte(`{"type":"assistant","cwd":"/repo","timestamp":"2026-09-05T10:00:00.000Z","message":{"content":[{"type":"tool_use","id":"a","name":"Read","input":{"file_path":"/etc/passwd"}}]}}` + "\n")
	s, err := ClaudeCode{RepoRoot: "/repo"}.Parse(blob)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	for _, e := range s.Events {
		if e.Kind == Read {
			t.Fatalf("expected no read event, got %q", e.Path)
		}
	}
}
