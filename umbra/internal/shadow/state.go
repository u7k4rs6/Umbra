// Package shadow decides what the session did and did not examine, ranks what
// it missed, and selects the tests that reach it.
package shadow

// State is one of the four answers. Missing data is a state, never an error.
type State int

const (
	Unknown State = iota
	Umbra
	Penumbra
	Lit
)

var stateNames = map[State]string{
	Lit: "lit", Penumbra: "penumbra", Umbra: "umbra", Unknown: "unknown",
}

// String is the plain name used in the JSON and the terminal.
func (s State) String() string {
	if n, ok := stateNames[s]; ok {
		return n
	}
	return "unknown"
}

// Glyph is the map and table marker.
func (s State) Glyph() string {
	switch s {
	case Lit:
		return "●" // filled circle
	case Penumbra:
		return "◐" // half filled circle
	case Umbra:
		return "○" // open circle
	}
	return "?"
}

// ASCII is the fallback marker for a terminal that is not UTF-8.
func (s State) ASCII() string {
	switch s {
	case Lit:
		return "L"
	case Penumbra:
		return "P"
	case Umbra:
		return "U"
	}
	return "?"
}

// Tier names the strength of partial evidence, strongest first. The order here
// is the precedence the classifier applies.
type Tier string

const (
	TierNone       Tier = ""
	TierGlance     Tier = "glance"
	TierGlimpse    Tier = "glimpse"
	TierQuoted     Tier = "quoted"
	TierAfterimage Tier = "afterimage"
	TierEcho       Tier = "echo"
)

// tierOrder is the precedence from ARCHITECTURE.md section 5: a partial read
// beats a search hit, which beats quoted content, which beats a read that
// happened only before the cut, which beats a bare mention.
var tierOrder = []Tier{TierGlance, TierGlimpse, TierQuoted, TierAfterimage, TierEcho}

// Rank returns the precedence of a tier, lower being stronger.
func (t Tier) Rank() int {
	for i, x := range tierOrder {
		if x == t {
			return i
		}
	}
	return len(tierOrder)
}

// Meaning is the plain sentence shown next to the coined word, so a reader who
// has never seen Umbra can still read the page. FRONTEND_SPEC.md requires
// every coined word to sit next to its plain meaning.
func (t Tier) Meaning() string {
	switch t {
	case TierGlance:
		return "a partial read whose range missed the symbol"
	case TierGlimpse:
		return "only a search hit on the file"
	case TierQuoted:
		return "the file's content appeared inside a tool result"
	case TierAfterimage:
		return "read fully, but only before the change began"
	case TierEcho:
		return "named in the agent's own words and never opened"
	}
	return ""
}

// Meaning is the plain sentence for a state.
func (s State) Meaning() string {
	switch s {
	case Lit:
		return "read fully after the change began, or edited"
	case Penumbra:
		return "half seen"
	case Umbra:
		return "never touched or mentioned"
	case Unknown:
		return "the transcript cannot tell"
	}
	return ""
}
