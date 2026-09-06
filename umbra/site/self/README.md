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

This report also carries no "session said" line, and that is correct. Entire
stored no summary for the checkpoint, and the session left no edit event to tie
any of its words to this commit, so there is nothing that can honestly be
quoted as the agent's account of the change. Umbra says so rather than reaching
for the last thing said in a session that ran for hours.

Files here: `umbra.html` is the map, `umbra.json` the report, and
`umbra.packet.md` the one-page packet. Everything is scrubbed: no absolute
paths, no user, host or author name, no email address, and no transcript prose
at all in this one.
