package report

import (
	"strings"
	"testing"
)

func TestScrubRemovesTokenShapes(t *testing.T) {
	s := &Scrubber{}
	cases := []string{
		"ghp_0123456789abcdefghijABCDEF",
		"sk-0123456789abcdefghijklmnop",
		"xoxb-1234567890-abcdefghij",
		"AKIAIOSFODNN7EXAMPLE",
		"api_key = supersecretvalue",
		"-----BEGIN RSA PRIVATE KEY-----",
	}
	for _, in := range cases {
		got := s.Clean("the value is " + in)
		if got == "the value is "+in {
			t.Errorf("scrub left %q untouched", in)
		}
	}
}

func TestScrubReplacesPaths(t *testing.T) {
	s := &Scrubber{Home: "/home/someone", Repo: "/home/someone/work/repo", User: "someone", Host: "laptop"}
	got := s.Clean("read /home/someone/work/repo/app/x.py on laptop")
	if got != "read <repo>/app/x.py on <host>" {
		t.Fatalf("Clean = %q", got)
	}
}

func TestSentenceIsCappedAt200(t *testing.T) {
	long := ""
	for i := 0; i < 60; i++ {
		long += "word "
	}
	got := (&Scrubber{}).Sentence(long)
	if len([]rune(got)) > 203 {
		t.Fatalf("sentence length = %d, want it capped near 200", len([]rune(got)))
	}
}

func TestCommandIsCappedAt120(t *testing.T) {
	long := ""
	for i := 0; i < 60; i++ {
		long += "arg "
	}
	got := (&Scrubber{}).Command(long)
	if len([]rune(got)) > 123 {
		t.Fatalf("command length = %d, want it capped near 120", len([]rune(got)))
	}
}

func TestExcerptIsCappedAt300(t *testing.T) {
	long := ""
	for i := 0; i < 200; i++ {
		long += "x "
	}
	got := (&Scrubber{}).Excerpt(long)
	if len([]rune(got)) > 303 {
		t.Fatalf("excerpt length = %d, want it capped near 300", len([]rune(got)))
	}
}

func TestCapCountsRunesNotBytes(t *testing.T) {
	// Three-byte runes must not be split down the middle.
	in := "●●●●●"
	got := Cap(in, 3)
	if got != "●●●..." {
		t.Fatalf("Cap = %q", got)
	}
}

func TestSentenceCollapsesWhitespace(t *testing.T) {
	got := (&Scrubber{}).Sentence("  Checked  the\n callers.\t ")
	if got != "Checked the callers." {
		t.Fatalf("Sentence = %q", got)
	}
}

func TestScrubEmptyInput(t *testing.T) {
	if got := (&Scrubber{}).Clean(""); got != "" {
		t.Fatalf("Clean(\"\") = %q", got)
	}
}

// A git object id is forty hex characters and matches the base64 shape. The
// reproduce list is only useful if a reader can check the commits in it, so
// hex runs survive while real base64 does not.
func TestScrubKeepsGitObjectIDs(t *testing.T) {
	s := &Scrubber{}
	sha := "00634433c2bc2ac6d1da8e76f45847e7f6702a36"
	got := s.Clean("git log -1 --format=%B " + sha)
	if !strings.Contains(got, sha) {
		t.Fatalf("a git object id must survive the scrub, got %q", got)
	}
}

func TestScrubRemovesRealBase64(t *testing.T) {
	s := &Scrubber{}
	blob := "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVphYmNkZWZnaGlqaw=="
	got := s.Clean("value " + blob)
	if strings.Contains(got, blob) {
		t.Fatalf("a long base64 run must be redacted, got %q", got)
	}
}

func TestScrubLeavesOrdinaryProseAlone(t *testing.T) {
	s := &Scrubber{}
	in := "Checked the callers of compute_total and updated them."
	if got := s.Clean(in); got != in {
		t.Fatalf("Clean rewrote ordinary prose: %q", got)
	}
}

// A file path is a long run of letters, digits and slashes. Treating the slash
// as a base64 character redacted every worktree path in the reproduce list.
func TestScrubKeepsLongPaths(t *testing.T) {
	s := &Scrubber{}
	path := "/var/lib/entire/plugins/data/umbra/wt/00634433c2bc2ac6/head"
	got := s.Clean("entire graph snapshot --repo " + path)
	if !strings.Contains(got, path) {
		t.Fatalf("a long path must survive the scrub, got %q", got)
	}
}

// An email address is personal data and Umbra never needs one. It reaches a
// report through the author line that `checkpoint explain --short` prints.
func TestScrubRemovesEmailAddresses(t *testing.T) {
	s := &Scrubber{}
	got := s.Clean("author   Some Person <someone@example.com>")
	if strings.Contains(got, "someone@example.com") {
		t.Fatalf("an email address must be redacted, got %q", got)
	}
}

// A display name cannot be recognised by shape, so the caller supplies it.
func TestScrubRemovesSuppliedNames(t *testing.T) {
	s := &Scrubber{Names: []string{"Some Person"}}
	got := s.Clean("author   Some Person <someone@example.com>")
	if strings.Contains(got, "Some Person") {
		t.Fatalf("a supplied name must be redacted, got %q", got)
	}
	if !strings.Contains(got, "<author>") {
		t.Fatalf("expected the author placeholder, got %q", got)
	}
}

func TestScrubLeavesOrdinaryAtSignsAlone(t *testing.T) {
	s := &Scrubber{}
	in := "run the test with pytest -k not_slow"
	if got := s.Clean(in); got != in {
		t.Fatalf("Clean rewrote ordinary text: %q", got)
	}
}
