// Package transcript turns an agent's stored session into a normalized list of
// events: what the session read, searched, edited, ran and named.
//
// Nothing here decides what a node's state is. This package only reports what
// happened, in order. It also never carries transcript prose into its output:
// assistant text is held in memory to find mentions and the session's own
// closing sentence, and is dropped after that.
package transcript

import (
	"sort"
	"strings"
	"time"
)

// Kind is the sort of thing that happened.
type Kind int

const (
	Prompt Kind = iota
	AssistantText
	Read
	Grep
	Glob
	Edit
	Command
	ResultFile
	Mention
)

var kindNames = map[Kind]string{
	Prompt:        "prompt",
	AssistantText: "text",
	Read:          "read",
	Grep:          "grep",
	Glob:          "glob",
	Edit:          "edit",
	Command:       "command",
	ResultFile:    "quoted",
	Mention:       "mention",
}

// String is the plain name used in the JSON and the terminal.
func (k Kind) String() string {
	if s, ok := kindNames[k]; ok {
		return s
	}
	return "unknown"
}

// Event is one moment in the session.
type Event struct {
	Seq   int
	TS    time.Time
	Kind  Kind
	Path  string
	Range *[2]int
	Paths []string
	// Symbols holds the symbol names named in agent text, for Mention events.
	Symbols []string
	// Cmd is the command line for Command events, already capped and scrubbed
	// by the renderer before it reaches any output.
	Cmd string
	// ToolID links a tool_use to its tool_result while parsing.
	ToolID string
}

// Covers reports whether a Read event's range covers the span [a, b].
// A nil range means the whole file was read, which covers everything.
func (e Event) Covers(a, b int) bool {
	if e.Range == nil {
		return true
	}
	return e.Range[0] <= a && e.Range[1] >= b
}

// Session is a parsed transcript.
type Session struct {
	Events []Event
	// Agent names the adapter that produced this session.
	Agent string
	// SessionIDs are the agent session identifiers seen in the records.
	SessionIDs []string

	// texts holds assistant prose in order. It never leaves this package
	// except as the single "session said" sentence, which the caller scrubs
	// and caps before it reaches a report.
	texts []textAt
}

type textAt struct {
	seq  int
	text string
}

// HasExposure reports whether the transcript carried any evidence at all.
//
// When this is false every node is unknown, which is the correct answer for a
// transcript with no tool records rather than an error.
func (s *Session) HasExposure() bool {
	for _, e := range s.Events {
		switch e.Kind {
		case Read, Grep, Glob, Edit, ResultFile, Mention:
			return true
		}
	}
	return false
}

// Files lists every path the session touched through a tool, sorted.
func (s *Session) Files() []string {
	seen := map[string]bool{}
	for _, e := range s.Events {
		if e.Path != "" {
			seen[e.Path] = true
		}
		for _, p := range e.Paths {
			seen[p] = true
		}
	}
	out := make([]string, 0, len(seen))
	for p := range seen {
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

// FirstEditSeq returns the sequence number of the earliest edit to any file in
// the given set. This is the cut: the moment the change begins. It returns 0
// and false when the session edited none of them.
func (s *Session) FirstEditSeq(files map[string]bool) (int, bool) {
	for _, e := range s.Events {
		if e.Kind == Edit && files[e.Path] {
			return e.Seq, true
		}
	}
	return 0, false
}

// LastSentence returns the last complete sentence of agent prose at or before
// seq, or the empty string. It is the fallback for the "session said" line
// when Entire has no stored summary, which the Step 0 probe found is always
// the case for imported checkpoints.
//
// The caller applies the scrub and the 200 character cap from
// SECURITY_AND_ACCESS.md before this reaches any output.
func (s *Session) LastSentence(seq int) string {
	for i := len(s.texts) - 1; i >= 0; i-- {
		if seq > 0 && s.texts[i].seq > seq {
			continue
		}
		if sent := lastSentenceOf(s.texts[i].text); sent != "" {
			return sent
		}
	}
	return ""
}

// SentenceAfter returns the first complete sentence of agent prose at or after
// seq. In a session that ends in a commit this is the agent's summing-up about
// the change it just made, which is what the "session said" line wants.
func (s *Session) SentenceAfter(seq int) string {
	for _, t := range s.texts {
		if t.seq < seq {
			continue
		}
		if sent := lastSentenceOf(t.text); sent != "" {
			return sent
		}
	}
	return ""
}

// Texts returns the agent prose in order, for mention matching. It is not
// output; callers must not place it in a report.
func (s *Session) Texts() []string {
	out := make([]string, 0, len(s.texts))
	for _, t := range s.texts {
		out = append(out, t.text)
	}
	return out
}

func lastSentenceOf(text string) string {
	text = strings.TrimSpace(collapseSpace(text))
	if text == "" {
		return ""
	}
	// Walk back to the start of the final sentence.
	end := len(text)
	for end > 0 && isTerminator(text[end-1]) {
		end--
	}
	if end == 0 {
		return ""
	}
	start := 0
	for i := end - 1; i > 0; i-- {
		if isTerminator(text[i-1]) && i < end {
			start = i
			break
		}
	}
	out := strings.TrimSpace(text[start:])
	if out == "" {
		return ""
	}
	// Keep the terminator so the sentence reads as one.
	if end < len(text) {
		out = strings.TrimSpace(text[start : end+1])
	}
	return out
}

func isTerminator(c byte) bool { return c == '.' || c == '!' || c == '?' }

func collapseSpace(s string) string { return strings.Join(strings.Fields(s), " ") }
