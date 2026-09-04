# Umbra run on its own build

`entire umbra 13adcdd --run none` on the commit where Umbra changed
`shadow.Build` and `shadow.AddCoChange` during phase 9.

Every one of the twelve dependents came back **umbra**, including `runScenario`
and all eight scenario tests, which are exactly the tests that would catch a
mistake in that function.

That is a true reading, and the reason is worth stating plainly: the change was
made with a shell command rather than with the editing tool, so the session
produced no read and no edit event for `build.go`. Umbra's examined set is
built from tool activity. An agent that edits through the shell leaves no
trace Umbra can see, and everything downstream looks unexamined.

It is the same shape of blind spot the product exists to find, pointed at the
product. It is disclosed in the README's limitations rather than tuned away.

Files here: `umbra.html` is the map, `umbra.json` the report, and
`umbra.packet.md` the one-page packet. Everything is scrubbed: no absolute
paths, no user or host name, and no transcript prose beyond the single
"session said" sentence.
