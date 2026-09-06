package report

import (
	"fmt"
	"strings"
)

// The coverage line reports how the session worked, not what it examined.
//
// One question decides how much weight a reader should put on a mostly-umbra
// report: did the agent examine nothing, or did it work in a way Umbra cannot
// see? The examined set is built from file tool events. An agent that reads and
// edits through the shell produces none, and its report is thin for a reason
// that has nothing to do with the change. Saying so is honest; leaving the
// reader to infer it is not.
//
// The wording lives here so the terminal, the JSON, the HTML and the packet
// cannot disagree about it.

// CoverageLine is the counts, in one line of plain words.
func CoverageLine(a *Analysis) string {
	c := a.Coverage
	if !c.Any() {
		return "no tool events in this transcript"
	}
	parts := []string{
		plural(c.Reads, "file read", "file reads"),
		plural(c.Edits, "edit", "edits"),
		plural(c.Searches, "search", "searches"),
		plural(c.Commands, "shell command", "shell commands"),
	}
	return strings.Join(parts, ", ")
}

// CoverageNote is the sentence that follows the counts when the session did
// nearly all of its work through the shell. It is empty otherwise.
func CoverageNote(a *Analysis) string {
	c := a.Coverage
	if !c.Any() {
		return "The examined set cannot be built, so every dependent is unknown rather than judged."
	}
	if !c.Thin() {
		return ""
	}
	return fmt.Sprintf(
		"This session worked almost entirely through the shell: %s against %s. "+
			"Umbra builds the examined set from file tool events, so a mostly umbra "+
			"report is the expected outcome here and says little about the change.",
		plural(c.FileEvents(), "file tool event", "file tool events"),
		plural(c.Commands, "shell command", "shell commands"))
}

func plural(n int, one, many string) string {
	if n == 1 {
		return fmt.Sprintf("%d %s", n, one)
	}
	return fmt.Sprintf("%d %s", n, many)
}
