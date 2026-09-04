package report

import (
	"strings"
	"testing"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

func htmlOf(t *testing.T, a *Analysis) string {
	t.Helper()
	a.Layout = BuildLayout(a)
	var b strings.Builder
	if err := HTML(&b, a); err != nil {
		t.Fatalf("HTML: %v", err)
	}
	return b.String()
}

// The report is one self-contained file: no fonts, no scripts, no images
// fetched from anywhere.
func TestHTMLMakesNoExternalRequests(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	for _, bad := range []string{
		"src=\"http", "href=\"http", "@import", "url(http", "fonts.googleapis", "cdn.",
	} {
		if strings.Contains(out, bad) {
			t.Errorf("the report must not reference %q", bad)
		}
	}
	// The SVG namespace is a name, not a request, and appears only in script.
	if strings.Count(out, "http://www.w3.org/2000/svg") > 2 {
		t.Error("unexpected number of namespace references")
	}
}

func TestHTMLCarriesTheContentSecurityPolicy(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	if !strings.Contains(out, "default-src 'none'") {
		t.Fatal("the policy must forbid everything by default")
	}
	for _, want := range []string{"style-src 'unsafe-inline'", "script-src 'unsafe-inline'", "img-src data:"} {
		if !strings.Contains(out, want) {
			t.Errorf("the policy is missing %q", want)
		}
	}
}

// html/template treats a script element as JavaScript, so the embedded report
// has to be emitted as an object rather than escaped into a string literal.
func TestHTMLEmbedsTheReportAsAnObject(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	i := strings.Index(out, `id="umbra-data">`)
	if i < 0 {
		t.Fatal("the data block is missing")
	}
	body := out[i+len(`id="umbra-data">`):]
	body = body[:strings.Index(body, "</script>")]
	if !strings.HasPrefix(strings.TrimSpace(body), "{") {
		t.Fatalf("the data block must start with an object, got %.40s", body)
	}
	if strings.Contains(body, `\"umbra_version\"`) {
		t.Fatal("the data block was escaped into a string literal")
	}
}

// A closing tag inside a string would end the script element early. Go's JSON
// encoder escapes angle brackets to \u003c and \u003e, so the literal never
// reaches the page; escapeForScript is a second layer for the case where that
// encoder setting ever changes. This checks the outcome rather than which
// layer produced it.
func TestHTMLEscapesClosingTagsInsideTheData(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Symbol.Name = "</script><img src=x onerror=alert(1)>"
	out := htmlOf(t, a)
	if strings.Contains(out, "</script><img") {
		t.Fatal("a closing tag inside the data must never appear literally")
	}
	if !strings.Contains(out, `\u003c/script\u003e`) {
		t.Fatal("expected the escaped form in the embedded data")
	}
	// The script element must still close exactly where the template put it.
	if strings.Count(out, "</script>") != 2 {
		t.Fatalf("expected two script closers, got %d", strings.Count(out, "</script>"))
	}
}

// escapeForScript is the second layer, checked on its own.
func TestEscapeForScript(t *testing.T) {
	got := escapeForScript([]byte(`{"a":"</script>"}`))
	if strings.Contains(got, "</script>") {
		t.Fatalf("escapeForScript left a literal closer: %s", got)
	}
}

// A symbol name that looks like markup renders as text everywhere.
func TestHTMLEscapesHostileNamesInTheDocket(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Symbol.Name = `<img src=x onerror=alert(1)>`
	out := htmlOf(t, a)
	if strings.Contains(out, "<img src=x") {
		t.Fatal("the docket must render a hostile name as text")
	}
	if !strings.Contains(out, "&lt;img src=x") {
		t.Fatal("expected the escaped name in the docket")
	}
}

// The docket is a real table so it survives with scripting off.
func TestHTMLDocketIsPresentWithoutJavaScript(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	if !strings.Contains(out, "<table class=\"docket\"") {
		t.Fatal("the docket must be a real table in the markup")
	}
	if !strings.Contains(out, "test_rounding") {
		t.Fatal("the rows must be rendered by the template, not by script")
	}
	if !strings.Contains(out, "The map needs JavaScript") {
		t.Fatal("the no-script fallback text is missing")
	}
	if !strings.Contains(out, "<caption>") {
		t.Fatal("the table needs a caption for screen readers")
	}
	if !strings.Contains(out, `scope="col"`) {
		t.Fatal("the table needs column scopes")
	}
}

// Every coined word on the page sits next to its plain meaning.
func TestHTMLChipsCarryTheirPlainMeaning(t *testing.T) {
	a := mkAnalysis()
	a.Nodes[0].Modifiers = []string{"echo", "far field", "fault line", "beacon"}
	out := htmlOf(t, a)
	for _, pair := range [][2]string{
		{"echo", "named, never opened"},
		{"far field", "across a package boundary"},
		{"fault line", "call site inside error handling"},
		{"beacon", "annotated call site"},
	} {
		if !strings.Contains(out, pair[0]) {
			t.Errorf("chip %q missing", pair[0])
		}
		if !strings.Contains(out, pair[1]) {
			t.Errorf("chip %q has no plain meaning next to it", pair[0])
		}
	}
}

func TestHTMLHeaderSaysWhenTheExaminedSetIsUnavailable(t *testing.T) {
	a := mkAnalysis()
	a.Nodes = []*shadow.Node{mkNode("x", "app/a.py", 1, shadow.Unknown, shadow.TierNone, 1)}
	a.Summary = shadow.Summarize(a.Nodes)
	out := htmlOf(t, a)
	if !strings.Contains(out, "examined set unavailable for this transcript") {
		t.Fatal("the header must say the set is unavailable")
	}
	if !strings.Contains(out, "stroke-dasharray") {
		t.Fatal("the eclipse glyph should be a dashed outline when nothing can be judged")
	}
}

func TestHTMLEclipseGlyphHasAnAccessibleLabel(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	if !strings.Contains(out, `role="img"`) {
		t.Fatal("the eclipse glyph needs role=img")
	}
	if !strings.Contains(out, "dependents lit") {
		t.Fatal("the glyph's label must carry the same words as the text")
	}
}

func TestHTMLOmitsTheSessionLineWhenThereIsNone(t *testing.T) {
	a := mkAnalysis()
	a.SessionSaid = ""
	out := htmlOf(t, a)
	if strings.Contains(out, "Session said") {
		t.Fatal("the line is omitted rather than invented when there is no summary")
	}
}

func TestHTMLFooterCarriesTheReproduceCommand(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	if !strings.Contains(out, "entire umbra b20f84567474") {
		t.Fatal("the footer must carry the reproduce command")
	}
	if !strings.Contains(out, "no model was involved") {
		t.Fatal("the footer must say no model was involved")
	}
}

func TestHTMLIsValidEnoughToParse(t *testing.T) {
	out := htmlOf(t, mkAnalysis())
	if !strings.HasPrefix(out, "<!doctype html>") {
		t.Fatal("missing doctype")
	}
	if strings.Count(out, "<html") != 1 || strings.Count(out, "</html>") != 1 {
		t.Fatal("unbalanced html element")
	}
	if strings.Count(out, "<table") != strings.Count(out, "</table>") {
		t.Fatal("unbalanced table")
	}
}
