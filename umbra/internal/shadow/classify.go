package shadow

import (
	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// Classify decides the state of one symbol given the examined set.
//
// The rules are the four states from PRD.md, applied in order:
//
//   - lit: a read of the file covering the symbol's span at or after the cut,
//     or an edit of the file. An edit counts because the agent changed the
//     file, so it saw it.
//   - penumbra: partial evidence, strongest tier first.
//   - umbra: nothing touched or mentioned the file.
//   - unknown: the transcript carried no evidence at all, so no state can be
//     computed for anything.
//
// The symbol name is passed separately so a mention of the name, with no event
// on its file, can reach the echo tier.
func Classify(e *Examined, file, symbolName string, span [2]int) (State, Tier) {
	if e == nil || !e.Any {
		return Unknown, TierNone
	}

	exposures := e.ByFile[file]
	mentions := e.BySymbol[symbolName]
	if len(exposures) == 0 && len(mentions) == 0 {
		return Umbra, TierNone
	}

	best := TierNone
	consider := func(t Tier) {
		if best == TierNone || t.Rank() < best.Rank() {
			best = t
		}
	}

	for _, x := range exposures {
		switch x.Kind {
		case transcript.Edit:
			return Lit, TierNone

		case transcript.Read:
			covers := x.Range == nil || (x.Range[0] <= span[0] && x.Range[1] >= span[1])
			switch {
			case covers && (!e.HasCut || x.Seq >= e.Cut):
				// A full read after the change began is the strongest
				// evidence there is.
				return Lit, TierNone
			case covers:
				// Read fully, but only before the cut. The picture the agent
				// holds is of the old code.
				consider(TierAfterimage)
			default:
				// The read happened but its range missed the symbol.
				consider(TierGlance)
			}

		case transcript.Grep, transcript.Glob:
			consider(TierGlimpse)

		case transcript.ResultFile:
			consider(TierQuoted)

		case transcript.Mention:
			consider(TierEcho)
		}
	}

	if len(mentions) > 0 {
		consider(TierEcho)
	}
	if best == TierNone {
		return Umbra, TierNone
	}
	return Penumbra, best
}

// StateSentence is the plain sentence the detail panel and the packet show as
// the evidence behind a state. It names the file, never a line of prose from
// the transcript.
func StateSentence(state State, tier Tier, file string) string {
	switch state {
	case Lit:
		return "the session read or edited " + file + " after the change began"
	case Umbra:
		return "nothing in the session touched or mentioned " + file
	case Unknown:
		return "the transcript carries no tool activity, so " + file + " cannot be judged"
	case Penumbra:
		switch tier {
		case TierGlance:
			return "the session read " + file + " with a range that missed this symbol"
		case TierGlimpse:
			return "the session saw " + file + " in a search result and never opened it"
		case TierQuoted:
			return "the content of " + file + " passed through a tool result and was never opened"
		case TierAfterimage:
			return "the session read " + file + " in full, but only before the change began"
		case TierEcho:
			return "the session named " + file + " in its own words and never opened it"
		}
	}
	return ""
}

// Vocabulary builds the mention vocabulary from the field.
func Vocabulary(f *graph.Field) transcript.Vocabulary {
	files, syms := f.Vocabulary()
	return transcript.Vocabulary{Files: files, Symbols: syms}
}
