package runner

import (
	"context"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// fixtures/recorded/minimal is what `entire umbra record` writes: every Runner
// call of a real analysis with its real output. These tests prove the file is
// replayable, and that the scrub the recorder applied actually held.

const minimalDir = "../../fixtures/recorded/minimal"

func loadMinimal(t *testing.T) *Recording {
	t.Helper()
	rec, err := LoadRecording(minimalDir)
	if err != nil {
		t.Fatalf("loading the minimal recording: %v", err)
	}
	return rec
}

func TestMinimalRecordingReplays(t *testing.T) {
	rec := loadMinimal(t)
	if rec.Scenario != "minimal" {
		t.Fatalf("scenario = %q", rec.Scenario)
	}
	if len(rec.Entries) < 10 {
		t.Fatalf("entries = %d, want a whole analysis", len(rec.Entries))
	}

	r := NewReplay(rec)
	// Every recorded call must be served back without starting a process.
	for _, e := range rec.Entries {
		out, _, _, err := r.Run(context.Background(), e.Call.Name, e.Call.Args, nil)
		if err != nil {
			t.Fatalf("replaying %s: %v", e.Call.String(), err)
		}
		if len(out) == 0 && len(e.Result.Stdout) > 0 {
			t.Fatalf("replay of %s returned nothing", e.Call.String())
		}
	}
}

// The recording has to contain the calls that matter, or it is not a recording
// of an analysis.
func TestMinimalRecordingCoversThePipeline(t *testing.T) {
	rec := loadMinimal(t)
	var joined []string
	for _, e := range rec.Entries {
		joined = append(joined, e.Call.String())
	}
	all := strings.Join(joined, "\n")

	for _, want := range []string{
		"entire checkpoint list --json",
		"entire graph capabilities --json",
		"entire graph snapshot",
		"entire graph commit",
		"entire checkpoint explain",
		"git",
	} {
		if !strings.Contains(all, want) {
			t.Errorf("the recording is missing a %q call", want)
		}
	}
}

// The scrub is the only thing standing between a recording and the machine it
// was taken on. These are the categories SECURITY_AND_ACCESS.md names.
func TestMinimalRecordingIsScrubbed(t *testing.T) {
	blob, err := readFile(filepath.Join(minimalDir, "recording.json"))
	if err != nil {
		t.Fatal(err)
	}
	// Go's JSON encoder escapes angle brackets, so the placeholders are
	// written as \u003crepo\u003e. Unescape before looking for them.
	text := strings.NewReplacer(`\u003c`, "<", `\u003e`, ">").Replace(string(blob))

	banned := []struct {
		what string
		re   *regexp.Regexp
	}{
		{"an absolute home path", regexp.MustCompile(`/home/[a-z]`)},
		{"an email address", regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},
		{"a github token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{16,}`)},
		{"an api key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`)},
		{"an aws key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
		{"a private key", regexp.MustCompile(`BEGIN [A-Z ]*PRIVATE KEY`)},
	}
	for _, b := range banned {
		if m := b.re.FindString(text); m != "" {
			t.Errorf("the recording contains %s: %q", b.what, m)
		}
	}

	// And the placeholders that prove the scrub ran rather than found nothing.
	for _, want := range []string{"<repo>", "<home>"} {
		if !strings.Contains(text, want) {
			t.Errorf("expected %s in a scrubbed recording", want)
		}
	}
}
