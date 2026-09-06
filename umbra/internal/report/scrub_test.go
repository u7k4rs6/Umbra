package report

import "testing"

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
