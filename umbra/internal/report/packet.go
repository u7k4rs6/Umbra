package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// PacketLimit is how many shadowed nodes the packet lists. The point is a page
// a reviewer reads in a minute, not a complete dump.
const PacketLimit = 20

// Packet writes the Blind Spot Packet: five sections, markdown, suitable as a
// pull request comment. It is the same data as every other renderer.
func Packet(w io.Writer, sd Sealed) error {
	if !sd.Valid() {
		return ErrUnsealed
	}
	a := sd.a
	b := &strings.Builder{}

	fmt.Fprintf(b, "# Blind spot packet\n\n")
	fmt.Fprintf(b, "Checkpoint `%s`, commit `%s`, parent `%s`, agent %s, resolved via %s.\n\n",
		orNone(a.CheckpointID), shortSHA(a.Commit), shortSHA(a.Parent), orNone(a.Agent), a.Route)

	frac, ok := a.Summary.Illumination()
	if ok {
		fmt.Fprintf(b, "**%d of %d dependents were examined (%.0f%%).** %d are in penumbra, %d are in full shadow.\n\n",
			a.Summary.Lit, a.Summary.Lit+a.Summary.Penumbra+a.Summary.Umbra, frac*100,
			a.Summary.Penumbra, a.Summary.Umbra)
	} else {
		fmt.Fprintf(b, "**The examined set is unavailable**, so every dependent is unknown rather than judged.\n\n")
	}
	for _, n := range a.Notes {
		fmt.Fprintf(b, "> %s\n\n", n)
	}

	// 1. What changed.
	fmt.Fprintf(b, "## 1. What changed\n\n")
	if len(a.Sources) == 0 {
		fmt.Fprintf(b, "No changed entity in this commit is code the graph can follow.\n\n")
	} else {
		fmt.Fprintf(b, "| Symbol | Change | Where | Dependents |\n|---|---|---|---|\n")
		for _, s := range a.Sources {
			fmt.Fprintf(b, "| `%s` | %s | `%s:%d` | %d |\n",
				s.Name, s.KindLabel(), s.File, s.Span[0], s.Dependents)
		}
		b.WriteString("\n")
	}

	// 2. What was examined.
	fmt.Fprintf(b, "## 2. What was examined\n\n")
	if a.SessionSaid != "" {
		fmt.Fprintf(b, "The session's own account, %s:\n\n> %s\n\n", a.SessionSaidFrom, a.SessionSaid)
		fmt.Fprintf(b, "That sentence is displayed, never checked. This packet reports what the session did.\n\n")
	}
	fmt.Fprintf(b, "How the session worked: %s.\n\n", CoverageLine(a))
	if note := CoverageNote(a); note != "" {
		fmt.Fprintf(b, "> %s\n\n", note)
	}

	counts := map[string]int{}
	for _, e := range a.Timeline {
		counts[e.Kind]++
	}
	if len(counts) == 0 {
		fmt.Fprintf(b, "The transcript carried no tool activity.\n\n")
	} else {
		fmt.Fprintf(b, "Tool activity in the session: ")
		var parts []string
		for _, k := range []string{"read", "grep", "glob", "edit", "quoted", "mention", "command"} {
			if counts[k] > 0 {
				parts = append(parts, fmt.Sprintf("%d %s", counts[k], k))
			}
		}
		fmt.Fprintf(b, "%s.\n\n", strings.Join(parts, ", "))
	}

	// 3. Ranked shadow.
	fmt.Fprintf(b, "## 3. Ranked shadow\n\n")
	shadowed := a.Shadowed()
	if len(shadowed) == 0 {
		fmt.Fprintf(b, "Nothing is in shadow: every dependent the graph found was examined.\n\n")
	} else {
		fmt.Fprintf(b, "The %d highest ranked, of %d in shadow. Each score is the product of its printed factors.\n\n",
			minInt(PacketLimit, len(shadowed)), len(shadowed))
		for i, n := range shadowed {
			if i >= PacketLimit {
				fmt.Fprintf(b, "\nand %d more.\n", len(shadowed)-PacketLimit)
				break
			}
			pin := ""
			if n.Pinned() {
				pin = " **pinned by a beacon**"
			}
			fmt.Fprintf(b, "### %d. `%s` %s%s\n\n", i+1, n.Symbol.Name, n.State.Glyph(), pin)
			fmt.Fprintf(b, "- `%s:%d`, %s at depth %d, score %.1f\n",
				n.Symbol.File, n.Symbol.Span[0], n.Relation, n.Depth, n.Score)
			fmt.Fprintf(b, "- %s\n", shadow.StateSentence(n.State, n.Tier, n.Symbol.File))
			if len(n.Modifiers) > 0 {
				fmt.Fprintf(b, "- %s\n", strings.Join(describeModifiers(n.Modifiers), "; "))
			}
			fmt.Fprintf(b, "- factors: %s\n", factorLine(n))
			if n.Beacon != "" {
				fmt.Fprintf(b, "- the call site is annotated: `%s`\n", n.Beacon)
			}
			if n.Result == shadow.OutcomeFail {
				fmt.Fprintf(b, "- **a test covering this failed**\n")
			}
			b.WriteString("\n")
		}
	}

	// 4. Tests reaching it.
	fmt.Fprintf(b, "## 4. Tests reaching the shadow\n\n")
	switch {
	case a.Run == "none":
		fmt.Fprintf(b, "Tests were not run (`--run none`).\n\n")
	case len(a.Execution.Selected) == 0:
		fmt.Fprintf(b, "No test in the graph reaches the shadow.\n\n")
	default:
		fmt.Fprintf(b, "%d selected:\n\n", len(a.Execution.Selected))
		for _, id := range a.Execution.Selected {
			fmt.Fprintf(b, "- `%s`\n", id)
		}
		b.WriteString("\n")
		if len(a.Execution.NotRunnable) > 0 {
			fmt.Fprintf(b, "%d test id(s) were dropped as not runnable.\n\n", len(a.Execution.NotRunnable))
		}
	}

	// 5. Results and leaks.
	fmt.Fprintf(b, "## 5. Results\n\n")
	if a.Execution.Degraded {
		fmt.Fprintf(b, "The runner printed no per-test ids, so the verdict is suite level. Add `-v` for per-test results.\n\n")
	}
	if len(a.Execution.NewFailures) > 0 {
		fmt.Fprintf(b, "**%d test(s) newly failing:**\n\n", len(a.Execution.NewFailures))
		for _, id := range a.Execution.NewFailures {
			fmt.Fprintf(b, "- `%s`\n", id)
		}
		b.WriteString("\n")
	} else if a.Run != "none" {
		fmt.Fprintf(b, "No test changed from passing to failing.\n\n")
	}
	if len(a.Execution.PreExisting) > 0 {
		fmt.Fprintf(b, "%d test(s) were already failing before this change and are not counted against it.\n\n",
			len(a.Execution.PreExisting))
	}
	if a.Execution.Sweep {
		word := "leaks"
		if len(a.Execution.Leaks) == 1 {
			word = "leak"
		}
		if a.Execution.SweepNamedEverything() {
			fmt.Fprintf(b, "The full suite was swept after the selected tests: **%d %s**.\n\n", len(a.Execution.Leaks), word)
		} else {
			fmt.Fprintf(b, "The full suite was swept after the selected tests: **%d %s named, and the list is incomplete**.\n\n",
				len(a.Execution.Leaks), word)
		}
		for _, l := range a.Execution.Leaks {
			fmt.Fprintf(b, "- `%s`: %s\n", l.Test, l.Reason)
		}
		if len(a.Execution.Leaks) > 0 {
			b.WriteString("\n")
		}
		if note := a.Execution.UnnamedNote(); note != "" {
			fmt.Fprintf(b, "%s.\n\n", note)
		}
	} else if a.Run != "none" {
		fmt.Fprintf(b, "The sweep was skipped, so the selection is unaudited.\n\n")
	}

	// Reproduce and limitations.
	fmt.Fprintf(b, "## Reproduce\n\n```\n%s\n```\n\n", reproduce(a))
	fmt.Fprintf(b, "## Limitations\n\n")
	for _, l := range a.Limitations {
		fmt.Fprintf(b, "- %s\n", l)
	}
	b.WriteString("\n")
	if len(a.Commands) > 0 {
		fmt.Fprintf(b, "<details><summary>Commands run</summary>\n\n")
		for _, c := range a.Commands {
			fmt.Fprintf(b, "- `%s`\n", c)
		}
		fmt.Fprintf(b, "\n</details>\n")
	}

	_, err := io.WriteString(w, b.String())
	return err
}

// describeModifiers puts the plain meaning next to each coined word, which is
// what FRONTEND_SPEC.md requires of every surface.
func describeModifiers(mods []string) []string {
	meanings := map[string]string{
		"glance":         "a partial read whose range missed the symbol",
		"glimpse":        "only a search hit on the file",
		"quoted":         "the file's content appeared inside a tool result",
		"afterimage":     "read fully, but only before the change began",
		"echo":           "named in the agent's own words and never opened",
		"far field":      "across a package boundary from its source",
		"fault line":     "the call site sits inside error handling",
		"beacon":         "the call site carries a SAFETY, CRITICAL or INVARIANT note",
		"scar":           "the file has several recent fix commits",
		"test":           "a test function",
		"transitive":     "reached through another symbol",
		"co-change only": "no code path; the file historically changes with the source",
	}
	out := make([]string, 0, len(mods))
	for _, m := range mods {
		if meaning, ok := meanings[m]; ok {
			out = append(out, fmt.Sprintf("%s (%s)", m, meaning))
		} else {
			out = append(out, m)
		}
	}
	return out
}

func factorLine(n *shadow.Node) string {
	order := []string{"relation", "dependents", "state", "farfield", "faultline", "scar", "test"}
	var parts []string
	for _, k := range order {
		if v, ok := n.Factors[k]; ok {
			parts = append(parts, fmt.Sprintf("%s %g", k, v))
		}
	}
	return strings.Join(parts, ", ")
}

func reproduce(a *Analysis) string {
	cmd := "entire umbra " + orNone(a.CheckpointID)
	if a.TestRunner != "" {
		cmd += fmt.Sprintf(" --test %q", a.TestRunner)
	}
	if a.Depth != 2 {
		cmd += fmt.Sprintf(" --depth %d", a.Depth)
	}
	if a.Run != "shadow" {
		cmd += " --run " + a.Run
	}
	if !a.Audit {
		cmd += " --no-audit"
	}
	if a.History {
		cmd += " --history"
	}
	return cmd
}

func orNone(s string) string {
	if s == "" {
		return "(none)"
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
