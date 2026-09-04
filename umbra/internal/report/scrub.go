// Package report renders the analysis: the terminal table, the JSON, the HTML
// eclipse map and the Blind Spot Packet.
package report

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Token shapes that must never reach a report even though Entire redacts
// before Umbra sees anything. This is defence in depth, and the residual risk
// that it is pattern based is stated in the README.
var tokenREs = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(gh[pousr]_[A-Za-z0-9]{16,})`),
	regexp.MustCompile(`(?i)\b(sk-[A-Za-z0-9_-]{16,})`),
	regexp.MustCompile(`(?i)\b(xox[abposr]-[A-Za-z0-9-]{10,})`),
	regexp.MustCompile(`(?i)\bAKIA[0-9A-Z]{16}\b`),
	regexp.MustCompile(`(?i)\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{5,}`),
	regexp.MustCompile(`(?i)\b(api[_-]?key|secret|password|token)\s*[:=]\s*\S+`),
	regexp.MustCompile(`(?i)-----BEGIN [A-Z ]*PRIVATE KEY-----`),
	// A long unbroken run of base64-ish characters is almost never prose.
	regexp.MustCompile(`\b[A-Za-z0-9+/]{40,}={0,2}\b`),
}

// Scrubber removes identifying and secret-shaped text.
type Scrubber struct {
	// Home and Repo are replaced with stable placeholders so a recording or a
	// shared report does not carry the builder's paths.
	Home string
	Repo string
	User string
	Host string
}

// NewScrubber builds a scrubber for this machine.
func NewScrubber(repo string) *Scrubber {
	s := &Scrubber{Repo: repo}
	if h, err := os.UserHomeDir(); err == nil {
		s.Home = h
		s.User = filepath.Base(h)
	}
	if h, err := os.Hostname(); err == nil {
		s.Host = h
	}
	return s
}

// Clean applies every replacement.
func (s *Scrubber) Clean(text string) string {
	if text == "" {
		return ""
	}
	if s != nil {
		if s.Repo != "" {
			text = strings.ReplaceAll(text, s.Repo, "<repo>")
		}
		if s.Home != "" {
			text = strings.ReplaceAll(text, s.Home, "<home>")
		}
		if s.User != "" && len(s.User) > 2 {
			text = strings.ReplaceAll(text, s.User, "<user>")
		}
		if s.Host != "" && len(s.Host) > 2 {
			text = strings.ReplaceAll(text, s.Host, "<host>")
		}
	}
	for _, re := range tokenREs {
		text = re.ReplaceAllString(text, "<redacted>")
	}
	return text
}

// Sentence scrubs and caps the one transcript-derived sentence Umbra ever
// shows: the session-said line. The cap is 200 characters, from
// SECURITY_AND_ACCESS.md.
func (s *Scrubber) Sentence(text string) string {
	return Cap(s.Clean(strings.TrimSpace(collapse(text))), 200)
}

// Command scrubs a command line for the timeline. Only the first 120
// characters survive.
func (s *Scrubber) Command(text string) string {
	return Cap(s.Clean(strings.TrimSpace(collapse(text))), 120)
}

// Excerpt scrubs a test failure line. The cap is 300 characters.
func (s *Scrubber) Excerpt(text string) string {
	return Cap(s.Clean(strings.TrimSpace(collapse(text))), 300)
}

// Cap truncates with an ellipsis, counting runes so a multi-byte character is
// never split.
func Cap(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n]) + "..."
}

func collapse(s string) string { return strings.Join(strings.Fields(s), " ") }
