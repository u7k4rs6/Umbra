package shadow

import (
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// exam builds an examined set directly, so each row of the state table can be
// exercised on its own.
func exam(cut int, hasCut bool, byFile map[string][]Exposure, bySymbol map[string][]Exposure) *Examined {
	if byFile == nil {
		byFile = map[string][]Exposure{}
	}
	if bySymbol == nil {
		bySymbol = map[string][]Exposure{}
	}
	return &Examined{ByFile: byFile, BySymbol: bySymbol, Cut: cut, HasCut: hasCut, Any: true}
}

func rng(a, b int) *[2]int { return &[2]int{a, b} }

const (
	file = "app/refunds.py"
	name = "apply_refund"
)

var span = [2]int{17, 28}

// One test per row of the state definition in PRD.md.

func TestLitFromFullReadAfterTheCut(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 7, Kind: transcript.Read, Range: nil}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Lit || tier != TierNone {
		t.Fatalf("got %v/%v, want lit", got, tier)
	}
}

func TestLitFromCoveringRangeAfterTheCut(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 9, Kind: transcript.Read, Range: rng(10, 40)}},
	}, nil)
	if got, _ := Classify(e, file, name, span); got != Lit {
		t.Fatalf("a range covering the span after the cut is lit, got %v", got)
	}
}

// An edit means the agent changed the file, so it saw it, whenever it happened.
func TestLitFromEdit(t *testing.T) {
	e := exam(9, true, map[string][]Exposure{
		file: {{Seq: 2, Kind: transcript.Edit}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Lit || tier != TierNone {
		t.Fatalf("got %v/%v, want lit", got, tier)
	}
}

func TestPenumbraGlanceFromRangeThatMissesTheSymbol(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 7, Kind: transcript.Read, Range: rng(1, 12)}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierGlance {
		t.Fatalf("got %v/%v, want penumbra/glance", got, tier)
	}
}

func TestPenumbraGlimpseFromSearchHit(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 3, Kind: transcript.Grep}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierGlimpse {
		t.Fatalf("got %v/%v, want penumbra/glimpse", got, tier)
	}
}

func TestPenumbraGlimpseFromGlob(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{file: {{Seq: 3, Kind: transcript.Glob}}}, nil)
	if _, tier := Classify(e, file, name, span); tier != TierGlimpse {
		t.Fatalf("tier = %v, want glimpse", tier)
	}
}

func TestPenumbraQuotedFromToolResult(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 3, Kind: transcript.ResultFile}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierQuoted {
		t.Fatalf("got %v/%v, want penumbra/quoted", got, tier)
	}
}

// Read in full, but only before the change began: the picture the agent holds
// is of the old code.
func TestPenumbraAfterimageFromReadBeforeTheCut(t *testing.T) {
	e := exam(9, true, map[string][]Exposure{
		file: {{Seq: 2, Kind: transcript.Read, Range: nil}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierAfterimage {
		t.Fatalf("got %v/%v, want penumbra/afterimage", got, tier)
	}
}

func TestPenumbraEchoFromSymbolMention(t *testing.T) {
	e := exam(5, true, nil, map[string][]Exposure{
		name: {{Seq: 4, Kind: transcript.Mention}},
	})
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierEcho {
		t.Fatalf("got %v/%v, want penumbra/echo", got, tier)
	}
}

func TestPenumbraEchoFromPathMention(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{
		file: {{Seq: 4, Kind: transcript.Mention}},
	}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Penumbra || tier != TierEcho {
		t.Fatalf("got %v/%v, want penumbra/echo", got, tier)
	}
}

func TestUmbraFromNothingAtAll(t *testing.T) {
	e := exam(5, true, map[string][]Exposure{"app/other.py": {{Seq: 1, Kind: transcript.Read}}}, nil)
	got, tier := Classify(e, file, name, span)
	if got != Umbra || tier != TierNone {
		t.Fatalf("got %v/%v, want umbra", got, tier)
	}
}

// A transcript with no evidence at all makes every node unknown. This is a
// state, never an error.
func TestUnknownWhenTheTranscriptCarriesNothing(t *testing.T) {
	e := &Examined{ByFile: map[string][]Exposure{}, BySymbol: map[string][]Exposure{}, Any: false}
	got, tier := Classify(e, file, name, span)
	if got != Unknown || tier != TierNone {
		t.Fatalf("got %v/%v, want unknown", got, tier)
	}
}

func TestUnknownOnNilExaminedSet(t *testing.T) {
	if got, _ := Classify(nil, file, name, span); got != Unknown {
		t.Fatalf("got %v, want unknown", got)
	}
}

// Tier precedence: the strongest evidence present wins.
func TestTierPrecedence(t *testing.T) {
	all := map[string][]Exposure{
		file: {
			{Seq: 1, Kind: transcript.Mention},
			{Seq: 2, Kind: transcript.ResultFile},
			{Seq: 3, Kind: transcript.Grep},
			{Seq: 4, Kind: transcript.Read, Range: rng(1, 5)},
		},
	}
	_, tier := Classify(exam(9, true, all, nil), file, name, span)
	if tier != TierGlance {
		t.Fatalf("tier = %v, want glance to win over glimpse, quoted and echo", tier)
	}

	withoutRead := map[string][]Exposure{
		file: {
			{Seq: 1, Kind: transcript.Mention},
			{Seq: 2, Kind: transcript.ResultFile},
			{Seq: 3, Kind: transcript.Grep},
		},
	}
	_, tier = Classify(exam(9, true, withoutRead, nil), file, name, span)
	if tier != TierGlimpse {
		t.Fatalf("tier = %v, want glimpse to win over quoted and echo", tier)
	}

	quotedAndEcho := map[string][]Exposure{
		file: {
			{Seq: 1, Kind: transcript.Mention},
			{Seq: 2, Kind: transcript.ResultFile},
		},
	}
	_, tier = Classify(exam(9, true, quotedAndEcho, nil), file, name, span)
	if tier != TierQuoted {
		t.Fatalf("tier = %v, want quoted to win over echo", tier)
	}
}

func TestTierRankOrder(t *testing.T) {
	order := []Tier{TierGlance, TierGlimpse, TierQuoted, TierAfterimage, TierEcho}
	for i := 1; i < len(order); i++ {
		if order[i-1].Rank() >= order[i].Rank() {
			t.Fatalf("%v should rank stronger than %v", order[i-1], order[i])
		}
	}
}

// With no cut at all, a covering read cannot be an afterimage because there is
// no "before" to be on the wrong side of.
func TestNoCutMakesCoveringReadLit(t *testing.T) {
	e := exam(0, false, map[string][]Exposure{
		file: {{Seq: 2, Kind: transcript.Read, Range: nil}},
	}, nil)
	if got, _ := Classify(e, file, name, span); got != Lit {
		t.Fatalf("got %v, want lit when there is no cut", got)
	}
}

func TestStateSentencesNameTheFileNeverTheProse(t *testing.T) {
	cases := []struct {
		state State
		tier  Tier
	}{
		{Lit, TierNone}, {Umbra, TierNone}, {Unknown, TierNone},
		{Penumbra, TierGlance}, {Penumbra, TierGlimpse}, {Penumbra, TierQuoted},
		{Penumbra, TierAfterimage}, {Penumbra, TierEcho},
	}
	for _, c := range cases {
		got := StateSentence(c.state, c.tier, file)
		if got == "" {
			t.Fatalf("no sentence for %v/%v", c.state, c.tier)
		}
		if !contains(got, file) {
			t.Fatalf("sentence %q should name the file", got)
		}
	}
}

func contains(hay, needle string) bool {
	return len(hay) >= len(needle) && (hay == needle || indexOf(hay, needle) >= 0)
}

func indexOf(hay, needle string) int {
	for i := 0; i+len(needle) <= len(hay); i++ {
		if hay[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}

func TestEveryTierHasAPlainMeaning(t *testing.T) {
	for _, tier := range []Tier{TierGlance, TierGlimpse, TierQuoted, TierAfterimage, TierEcho} {
		if tier.Meaning() == "" {
			t.Fatalf("tier %v has no plain meaning, which the frontend spec requires", tier)
		}
	}
	for _, s := range []State{Lit, Penumbra, Umbra, Unknown} {
		if s.Meaning() == "" {
			t.Fatalf("state %v has no plain meaning", s)
		}
	}
}

func TestGlyphsAndASCIIFallback(t *testing.T) {
	cases := map[State][2]string{
		Lit:      {"●", "L"},
		Penumbra: {"◐", "P"},
		Umbra:    {"○", "U"},
		Unknown:  {"?", "?"},
	}
	for s, want := range cases {
		if s.Glyph() != want[0] {
			t.Errorf("%v glyph = %q, want %q", s, s.Glyph(), want[0])
		}
		if s.ASCII() != want[1] {
			t.Errorf("%v ascii = %q, want %q", s, s.ASCII(), want[1])
		}
	}
}
