package report

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"strings"
)

//go:embed assets/umbra.html.tmpl assets/umbra.css assets/umbra.js
var assets embed.FS

// htmlData is what the template sees. Everything here is escaped by
// html/template except the CSS and JS, which are our own files, and the JSON,
// which is escaped by hand for a script block.
type htmlData struct {
	A   *Analysis
	CSS template.CSS
	JS  template.JS
	// JSON is typed as template.JS because html/template treats the contents
	// of any <script> element as JavaScript, and a template.HTML value there
	// would be escaped into a string literal rather than emitted as an object.
	JSON template.JS

	Rows      []DocketRow
	LitHidden int

	EclipseGlyph     template.HTML
	CoverageLine     string
	CoverageNote     string
	IlluminationText string
	ProbeText        string
	SweepText        string
	Reproduce        string
}

// HTML renders the self-contained report.
//
// There are no external requests: the CSS and JS are embedded, the fonts are
// system stacks, and the Content-Security-Policy in the template forbids
// anything else.
func HTML(w io.Writer, sd Sealed) error {
	if !sd.Valid() {
		return ErrUnsealed
	}
	a := sd.a
	tmpl, err := template.New("umbra.html.tmpl").Funcs(template.FuncMap{
		"orNone": orNone,
		"short":  shortSHA,
	}).ParseFS(assets, "assets/umbra.html.tmpl")
	if err != nil {
		return err
	}
	css, err := assets.ReadFile("assets/umbra.css")
	if err != nil {
		return err
	}
	js, err := assets.ReadFile("assets/umbra.js")
	if err != nil {
		return err
	}
	blob, err := MarshalJSON(sd)
	if err != nil {
		return err
	}

	d := htmlData{
		A:                a,
		CSS:              template.CSS(css),
		JS:               template.JS(js),
		JSON:             template.JS(escapeForScript(blob)),
		Rows:             BuildDocket(a),
		LitHidden:        a.Summary.Lit,
		EclipseGlyph:     eclipseGlyph(a),
		CoverageLine:     CoverageLine(a),
		CoverageNote:     CoverageNote(a),
		IlluminationText: illuminationText(a),
		ProbeText:        probeText(a),
		SweepText:        sweepText(a),
		Reproduce:        reproduce(a),
	}
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, d); err != nil {
		return err
	}
	_, err = w.Write(buf.Bytes())
	return err
}

// escapeForScript makes JSON safe inside a <script> block. A literal "</" or
// "<!--" inside a string would end the element or open a comment, so both are
// escaped in a way JSON.parse still reads correctly.
func escapeForScript(blob []byte) string {
	s := string(blob)
	s = strings.ReplaceAll(s, "</", `<\/`)
	s = strings.ReplaceAll(s, "<!--", `<!--`)
	return s
}

// eclipseGlyph draws the header disc: a lit circle overlapped by an umbra disc
// offset so the covered fraction equals what was not examined.
func eclipseGlyph(a *Analysis) template.HTML {
	frac, ok := a.Summary.Illumination()
	if !ok {
		return template.HTML(`<svg width="22" height="22" viewBox="0 0 22 22" role="img" ` +
			`aria-label="examined set unavailable for this transcript">` +
			`<circle cx="11" cy="11" r="9" fill="none" stroke="var(--unknown)" stroke-width="1.5" stroke-dasharray="3 3"/></svg>`)
	}
	// A fully lit disc is uncovered; a fully dark one is covered completely.
	offset := 18 * frac
	label := fmt.Sprintf("%d of %d dependents lit", a.Summary.Lit,
		a.Summary.Lit+a.Summary.Penumbra+a.Summary.Umbra)
	return template.HTML(fmt.Sprintf(
		`<svg width="22" height="22" viewBox="0 0 22 22" role="img" aria-label=%q>`+
			`<circle cx="11" cy="11" r="9" fill="var(--lit)"/>`+
			`<circle cx="%.2f" cy="11" r="9" fill="var(--umbra)"/></svg>`,
		label, 11-offset))
}

func illuminationText(a *Analysis) string {
	if _, ok := a.Summary.Illumination(); !ok {
		return "examined set unavailable for this transcript"
	}
	return fmt.Sprintf("%d of %d dependents lit", a.Summary.Lit,
		a.Summary.Lit+a.Summary.Penumbra+a.Summary.Umbra)
}

func probeText(a *Analysis) string {
	if a.Run == "none" {
		return "probes not run"
	}
	if len(a.Execution.Selected) == 0 {
		return "no probe reaches the shadow"
	}
	return fmt.Sprintf("%d probes, %d cracked", len(a.Execution.Selected), len(a.Execution.CrackedProbes()))
}

func sweepText(a *Analysis) string {
	switch {
	case a.Run == "none":
		return "no sweep"
	case !a.Audit:
		return "no sweep"
	case a.Execution.SweepCut:
		return "sweep cut short"
	case !a.Execution.Sweep:
		return "no sweep"
	}
	word := "leaks"
	if len(a.Execution.Leaks) == 1 {
		word = "leak"
	}
	return fmt.Sprintf("full sweep, %d %s", len(a.Execution.Leaks), word)
}
