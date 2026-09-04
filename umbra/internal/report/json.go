package report

import (
	"encoding/json"
	"io"

	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// The JSON schema is the one in ARCHITECTURE.md section 7. It is the contract
// the HTML, the landing-page viewer and any machine consumer read, so the
// field names here are fixed and the plain state names are used throughout,
// never the coined words.

type jsonReport struct {
	UmbraVersion    string          `json:"umbra_version"`
	Checkpoint      jsonCheckpoint  `json:"checkpoint"`
	Inputs          jsonInputs      `json:"inputs"`
	SessionSaid     string          `json:"session_said,omitempty"`
	SessionSaidFrom string          `json:"session_said_from,omitempty"`
	Notes           []string        `json:"notes,omitempty"`
	Sources         []jsonSource    `json:"sources"`
	Nodes           []jsonNode      `json:"nodes"`
	Timeline        []TimelineEvent `json:"timeline"`
	T0              int             `json:"t0"`
	Execution       Execution       `json:"execution"`
	Summary         jsonSummary     `json:"summary"`
	Layout          *Layout         `json:"layout,omitempty"`
	Limitations     []string        `json:"limitations"`
	CommandsRun     []string        `json:"commands_run"`
}

type jsonCheckpoint struct {
	ID         string   `json:"id"`
	Commit     string   `json:"commit"`
	Parent     string   `json:"parent"`
	Agent      string   `json:"agent"`
	Route      string   `json:"route"`
	SessionIDs []string `json:"session_ids"`
}

type jsonInputs struct {
	Adapter          string          `json:"adapter"`
	Depth            int             `json:"depth"`
	TestRunner       string          `json:"test_runner"`
	Run              string          `json:"run"`
	Audit            bool            `json:"audit"`
	History          bool            `json:"history"`
	Snippets         bool            `json:"snippets"`
	RelationsUsed    []string        `json:"relations_used"`
	RelationsIgnored []string        `json:"relations_ignored"`
	Channels         map[string]bool `json:"channels"`
}

type jsonSource struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	File         string  `json:"file"`
	Span         [2]int  `json:"span"`
	Change       string  `json:"change"`
	Weight       float64 `json:"weight"`
	Dependents   int     `json:"dependents"`
	OldSignature string  `json:"old_signature,omitempty"`
	NewSignature string  `json:"new_signature,omitempty"`
}

type jsonNode struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	File       string             `json:"file"`
	Span       [2]int             `json:"span"`
	IsTest     bool               `json:"is_test"`
	Sources    []string           `json:"sources"`
	Relation   string             `json:"relation"`
	Depth      int                `json:"depth"`
	Path       []string           `json:"path"`
	State      string             `json:"state"`
	Tier       string             `json:"tier,omitempty"`
	Modifiers  []string           `json:"modifiers"`
	Exposures  []jsonExposure     `json:"exposures"`
	Sentence   string             `json:"sentence"`
	CallSite   int                `json:"call_site"`
	Dependents int                `json:"dependents"`
	Score      float64            `json:"score"`
	Factors    map[string]float64 `json:"factors"`
	Beacon     string             `json:"beacon,omitempty"`
	Signature  string             `json:"signature,omitempty"`
	Tests      []jsonTestRef      `json:"tests"`
	Result     string             `json:"result"`
	Failure    string             `json:"failure_excerpt,omitempty"`
}

type jsonExposure struct {
	Seq   int     `json:"seq"`
	Kind  string  `json:"kind"`
	Range *[2]int `json:"range,omitempty"`
}

type jsonTestRef struct {
	ID      string `json:"id"`
	Outcome string `json:"outcome"`
}

type jsonSummary struct {
	Lit          int      `json:"lit"`
	Penumbra     int      `json:"penumbra"`
	Umbra        int      `json:"umbra"`
	Unknown      int      `json:"unknown"`
	Illumination *float64 `json:"illumination"`
}

// BuildJSON turns the analysis into the report structure.
//
// Every value that came from a transcript has already been scrubbed and
// capped. Signature lines appear only when --snippets is on.
func BuildJSON(a *Analysis) jsonReport {
	r := jsonReport{
		UmbraVersion: a.Version,
		Checkpoint: jsonCheckpoint{
			ID: a.CheckpointID, Commit: a.Commit, Parent: a.Parent,
			Agent: a.Agent, Route: a.Route, SessionIDs: nonNilStrings(a.SessionIDs),
		},
		Inputs: jsonInputs{
			Adapter: a.Adapter, Depth: a.Depth, TestRunner: a.TestRunner,
			Run: a.Run, Audit: a.Audit, History: a.History, Snippets: a.Snippets,
			RelationsUsed: nonNilStrings(a.RelationsUsed), RelationsIgnored: nonNilStrings(a.RelationsIgnored),
			Channels: a.Channels,
		},
		SessionSaid:     a.SessionSaid,
		SessionSaidFrom: a.SessionSaidFrom,
		Notes:           a.Notes,
		Timeline:        a.Timeline,
		T0:              a.Cut,
		Execution:       a.Execution,
		Layout:          a.Layout,
		Limitations:     nonNilStrings(a.Limitations),
		CommandsRun:     nonNilStrings(a.Commands),
	}

	for _, s := range a.Sources {
		r.Sources = append(r.Sources, jsonSource{
			ID: s.Symbol, Name: s.Name, File: s.File, Span: s.Span,
			Change: s.Change, Weight: s.Weight, Dependents: s.Dependents,
			OldSignature: snippet(a, s.OldSignature), NewSignature: snippet(a, s.NewSignature),
		})
	}
	if r.Sources == nil {
		r.Sources = []jsonSource{}
	}

	for _, n := range a.Nodes {
		jn := jsonNode{
			ID: n.Symbol.ID, Name: n.Symbol.Name, File: n.Symbol.File, Span: n.Symbol.Span,
			IsTest: n.Symbol.IsTest, Sources: nonNilStrings(n.Sources), Relation: n.Relation,
			Depth: n.Depth, Path: nonNilStrings(n.Path), State: n.State.String(),
			Tier: string(n.Tier), Modifiers: nonNilStrings(n.Modifiers),
			Sentence:   shadow.StateSentence(n.State, n.Tier, n.Symbol.File),
			CallSite:   n.CallSite,
			Dependents: n.Dependents, Score: n.Score, Factors: n.Factors,
			Beacon:    n.Beacon,
			Signature: snippet(a, n.Symbol.Signature),
			Result:    string(n.Result),
			Failure:   n.Failure,
		}
		if jn.Result == "" {
			jn.Result = string(shadow.OutcomeNotRun)
		}
		for _, x := range n.Exposures {
			jn.Exposures = append(jn.Exposures, jsonExposure{Seq: x.Seq, Kind: x.Kind.String(), Range: x.Range})
		}
		if jn.Exposures == nil {
			jn.Exposures = []jsonExposure{}
		}
		for _, tr := range n.Tests {
			jn.Tests = append(jn.Tests, jsonTestRef{ID: tr.ID, Outcome: string(tr.Outcome)})
		}
		if jn.Tests == nil {
			jn.Tests = []jsonTestRef{}
		}
		r.Nodes = append(r.Nodes, jn)
	}
	if r.Nodes == nil {
		r.Nodes = []jsonNode{}
	}

	r.Summary = jsonSummary{
		Lit: a.Summary.Lit, Penumbra: a.Summary.Penumbra,
		Umbra: a.Summary.Umbra, Unknown: a.Summary.Unknown,
	}
	if frac, ok := a.Summary.Illumination(); ok {
		r.Summary.Illumination = &frac
	}

	if r.Timeline == nil {
		r.Timeline = []TimelineEvent{}
	}
	if r.Execution.Selected == nil {
		r.Execution.Selected = []string{}
	}
	if r.Execution.NewFailures == nil {
		r.Execution.NewFailures = []string{}
	}
	if r.Execution.Fixed == nil {
		r.Execution.Fixed = []string{}
	}
	if r.Execution.Leaks == nil {
		r.Execution.Leaks = []Leak{}
	}
	return r
}

// snippet returns a declaration line only when --snippets is on, capped and
// scrubbed, as SECURITY_AND_ACCESS.md requires.
func snippet(a *Analysis, s string) string {
	if !a.Snippets || s == "" {
		return ""
	}
	return Cap(s, 200)
}

// WriteJSON renders the report as indented JSON.
func WriteJSON(w io.Writer, a *Analysis) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(true)
	return enc.Encode(BuildJSON(a))
}

// MarshalJSON returns the report bytes, for embedding in the HTML.
func MarshalJSON(a *Analysis) ([]byte, error) {
	return json.MarshalIndent(BuildJSON(a), "", "  ")
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
