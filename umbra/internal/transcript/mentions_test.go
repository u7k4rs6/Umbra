package transcript

import "testing"

func vocab() Vocabulary {
	return Vocabulary{
		Files: []string{
			"app/service.py", "app/api.py", "app/refunds.py", "app/models.py",
			"tests/test_service.py", "tests/test_api.py",
		},
		Symbols: []string{"compute_total", "apply_refund", "handle_order", "round_money", "RefundLine"},
	}
}

// The session names apply_refund in its own words and never opens
// app/refunds.py. That is the echo tier: attention, not reading.
func TestResolveMentionsFindsTheNamedSymbol(t *testing.T) {
	s := load(t, "classic.jsonl")
	s.ResolveMentions(vocab())

	var syms []string
	for _, e := range s.Events {
		if e.Kind == Mention {
			syms = append(syms, e.Symbols...)
		}
	}
	found := false
	for _, x := range syms {
		if x == "apply_refund" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected apply_refund among mentions, got %v", syms)
	}
}

// A file the session actually opened gains nothing from being named: a real
// tool event is always stronger evidence.
func TestResolveMentionsSkipsFilesAlreadyTouched(t *testing.T) {
	s := &Session{texts: []textAt{{seq: 1, text: "I updated app/service.py just now."}}}
	s.Events = []Event{{Seq: 0, Kind: Read, Path: "app/service.py"}}
	s.ResolveMentions(vocab())

	for _, e := range s.Events {
		if e.Kind == Mention {
			for _, p := range e.Paths {
				if p == "app/service.py" {
					t.Fatal("a file that was read must not also produce a mention")
				}
			}
		}
	}
}

func TestResolveMentionsMatchesBareFileName(t *testing.T) {
	s := &Session{texts: []textAt{{seq: 1, text: "The logic lives in refunds.py somewhere."}}}
	s.ResolveMentions(vocab())

	found := false
	for _, e := range s.Events {
		if e.Kind == Mention {
			for _, p := range e.Paths {
				if p == "app/refunds.py" {
					found = true
				}
			}
		}
	}
	if !found {
		t.Fatal("a bare file name should match its unique path")
	}
}

// Symbol matching is exact and case sensitive so ordinary prose cannot be
// mistaken for a symbol name.
func TestResolveMentionsSymbolMatchIsExact(t *testing.T) {
	s := &Session{texts: []textAt{{seq: 1, text: "I will Compute_Total the numbers and compute totals."}}}
	s.ResolveMentions(vocab())

	for _, e := range s.Events {
		if e.Kind == Mention && len(e.Symbols) > 0 {
			t.Fatalf("expected no symbol mentions, got %v", e.Symbols)
		}
	}
}

func TestResolveMentionsIgnoresUnknownNames(t *testing.T) {
	s := &Session{texts: []textAt{{seq: 1, text: "I refactored frobnicate in widget.py."}}}
	s.ResolveMentions(vocab())
	for _, e := range s.Events {
		if e.Kind == Mention {
			t.Fatalf("a name outside the vocabulary must not become evidence: %v", e)
		}
	}
}

func TestResolveMentionsRecordsNoProse(t *testing.T) {
	s := load(t, "classic.jsonl")
	s.ResolveMentions(vocab())
	for _, e := range s.Events {
		if e.Kind == Mention && e.Cmd != "" {
			t.Fatal("a mention must carry names only, never the sentence")
		}
	}
}

func TestResolveMentionsKeepsEventsOrdered(t *testing.T) {
	s := load(t, "classic.jsonl")
	s.ResolveMentions(vocab())
	last := 0
	for _, e := range s.Events {
		if e.Seq < last {
			t.Fatalf("events out of order at %v", e)
		}
		last = e.Seq
	}
}

// The probe found imported checkpoints never carry a stored summary, so the
// last sentence of agent prose is the fallback for the session-said line.
func TestLastSentence(t *testing.T) {
	s := load(t, "classic.jsonl")
	got := s.LastSentence(0)
	if got != "All tests pass." {
		t.Fatalf("LastSentence = %q, want %q", got, "All tests pass.")
	}
}

func TestLastSentenceBeforeACut(t *testing.T) {
	s := &Session{texts: []textAt{
		{seq: 2, text: "Reading the service module."},
		{seq: 9, text: "All done."},
	}}
	if got := s.LastSentence(5); got != "Reading the service module." {
		t.Fatalf("LastSentence(5) = %q", got)
	}
}

func TestLastSentenceEmptyWhenNoProse(t *testing.T) {
	s := load(t, "no-reads.jsonl")
	// The only prose is "Done.", which is a complete sentence.
	if got := s.LastSentence(0); got != "Done." {
		t.Fatalf("LastSentence = %q", got)
	}
	empty := &Session{}
	if got := empty.LastSentence(0); got != "" {
		t.Fatalf("LastSentence on an empty session = %q", got)
	}
}

func TestExtractPathsShapes(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"grep hit", "app/refunds.py:23:  refundable = x\n", "app/refunds.py"},
		{"grep context", "app/api.py:16-  total = x\n", "app/api.py"},
		{"bare line", "tests/test_service.py\n", "tests/test_service.py"},
		{"diff header", "--- a/app/models.py\n+++ b/app/models.py\n", "app/models.py"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := extractPaths(c.in, "/repo")
			found := false
			for _, p := range got {
				if p == c.want {
					found = true
				}
			}
			if !found {
				t.Fatalf("extractPaths(%q) = %v, want %s", c.in, got, c.want)
			}
		})
	}
}

func TestExtractPathsIgnoresProse(t *testing.T) {
	got := extractPaths("This sentence has no paths in it at all.\n", "/repo")
	if len(got) != 0 {
		t.Fatalf("expected no paths, got %v", got)
	}
}
