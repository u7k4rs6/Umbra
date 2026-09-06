package report

import (
	"bufio"
	"bytes"
	"regexp"
	"strconv"
	"strings"
)

// The output-wide check.
//
// The scrubber decides what leaves; this decides whether anything did. They
// are separate on purpose. The scrubber runs on values as they are built and
// can only remove what it was given; this runs on finished bytes and does not
// care which code path produced them, so a new output path is covered the day
// it is written rather than the day someone remembers to scrub it.

// Finding is one banned thing found in an artifact.
type Finding struct {
	// What names the category, in the words SECURITY_AND_ACCESS.md uses.
	What string
	// Match is the text that tripped it, so a failure says what to look for.
	Match string
	// Line is the one-based line it was on.
	Line int
}

func (f Finding) String() string {
	return f.What + " on line " + strconv.Itoa(f.Line) + ": " + f.Match
}

type bannedPattern struct {
	what string
	re   *regexp.Regexp
}

// bannedInArtifacts is the list from SECURITY_AND_ACCESS.md, plus the two
// categories the earlier leaks fell into: a home path and an address.
var bannedInArtifacts = []bannedPattern{
	{"an absolute home path", regexp.MustCompile(`/home/[A-Za-z0-9._-]+`)},
	{"an absolute home path", regexp.MustCompile(`/Users/[A-Za-z0-9._-]+`)},
	{"an absolute root path", regexp.MustCompile(`/root/[A-Za-z0-9._-]+`)},
	{"an email address", regexp.MustCompile(`[A-Za-z0-9._%+\-]+@[A-Za-z0-9.\-]+\.[A-Za-z]{2,}`)},
	{"a github token", regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{16,}`)},
	{"an api key", regexp.MustCompile(`\bsk-[A-Za-z0-9_-]{16,}`)},
	{"a slack token", regexp.MustCompile(`xox[abposr]-[A-Za-z0-9-]{10,}`)},
	{"an aws key", regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{"a private key", regexp.MustCompile(`BEGIN [A-Z ]*PRIVATE KEY`)},
	{"a jwt", regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{5,}`)},
}

// entities undoes the escaping the HTML renderer applies, so a leak cannot
// hide behind an ampersand. The report writes < and > escaped, and a home path
// that survived would appear as &lt;home&gt; only if it had been scrubbed, but
// an author name would appear verbatim in either form.
var entities = strings.NewReplacer("&lt;", "<", "&gt;", ">", "&amp;", "&", `\/`, "/", `<`, "<", `>`, ">")

// ScanArtifact returns everything banned in blob.
//
// names are display names to look for. A name cannot be recognised by shape,
// so the caller supplies the ones it knows about, exactly as the scrubber
// takes them.
func ScanArtifact(blob []byte, names []string) []Finding {
	var out []Finding
	sc := bufio.NewScanner(bytes.NewReader(blob))
	// A report is one long line of JSON in the HTML case, so the default
	// 64 KB limit is not enough.
	sc.Buffer(make([]byte, 0, 1<<20), 64<<20)
	line := 0
	for sc.Scan() {
		line++
		text := entities.Replace(sc.Text())
		for _, b := range bannedInArtifacts {
			if m := b.re.FindString(text); m != "" {
				out = append(out, Finding{What: b.what, Match: m, Line: line})
			}
		}
		for _, n := range names {
			if len(n) > 2 && strings.Contains(text, n) {
				out = append(out, Finding{What: "an author name", Match: n, Line: line})
			}
		}
	}
	return out
}
