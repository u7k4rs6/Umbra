package report

import (
	"encoding/json"
	"io"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
)

// The JSON schema is the one in ARCHITECTURE.md section 7. It is the contract
// the HTML, the landing-page viewer and any machine consumer read, so the
// field names here are fixed and the plain state names are used throughout,
// never the coined words.

type jsonReport struct {
	UmbraVersion    string         `json:"umbra_version"`
	Checkpoint      jsonCheckpoint `json:"checkpoint"`
	Inputs          jsonInputs     `json:"inputs"`
	SessionSaid     string         `json:"session_said,omitempty"`
	SessionSaidFrom string         `json:"session_said_from,omitempty"`
	Coverage        jsonCoverage   `json:"coverage"`
	Reach           ReachSummary   `json:"reach"`
	// GraphEvidence is what the graph said about its own reliability, and
	// EvidenceLegend is the three coined words with their plain meanings, so
	// a reader of the JSON alone can tell what a tier claims.
	GraphEvidence  jsonGraphEvidence  `json:"graph_evidence"`
	EvidenceLegend []jsonLegendEntry  `json:"evidence_legend"`
	Notes          []string           `json:"notes,omitempty"`
	Sources        []jsonSource       `json:"sources"`
	Unresolved     []UnresolvedSource `json:"unresolved,omitempty"`
	Nodes          []jsonNode         `json:"nodes"`
	Timeline       []TimelineEvent    `json:"timeline"`
	T0             int                `json:"t0"`
	Execution      Execution          `json:"execution"`
	Summary        jsonSummary        `json:"summary"`
	Layout         *Layout            `json:"layout,omitempty"`
	Limitations    []string           `json:"limitations"`
	CommandsRun    []string           `json:"commands_run"`
}

// jsonCoverage says how the session worked, so a consumer can tell a thin
// report from a real finding without re-reading the timeline.
type jsonCoverage struct {
	Reads    int    `json:"reads"`
	Edits    int    `json:"edits"`
	Searches int    `json:"searches"`
	Quoted   int    `json:"quoted"`
	Mentions int    `json:"mentions"`
	Commands int    `json:"shell_commands"`
	Thin     bool   `json:"shell_only"`
	Line     string `json:"line"`
	Note     string `json:"note,omitempty"`
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
	KindLabel    string  `json:"kind_label"`
	Weight       float64 `json:"weight"`
	Dependents   int     `json:"dependents"`
	OldSignature string  `json:"old_signature,omitempty"`
	NewSignature string  `json:"new_signature,omitempty"`
}

type jsonNode struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	File      string         `json:"file"`
	Span      [2]int         `json:"span"`
	IsTest    bool           `json:"is_test"`
	Sources   []string       `json:"sources"`
	Relation  string         `json:"relation"`
	Depth     int            `json:"depth"`
	Path      []string       `json:"path"`
	State     string         `json:"state"`
	Tier      string         `json:"tier,omitempty"`
	Modifiers []string       `json:"modifiers"`
	Exposures []jsonExposure `json:"exposures"`
	Sentence  string         `json:"sentence"`
	// Evidence is the plain word for how much of this node the graph can
	// vouch for, with its meaning and the reason, so the JSON needs no key.
	Evidence        string `json:"evidence"`
	EvidenceMeaning string `json:"evidence_meaning"`
	EvidenceWhy     string `json:"evidence_why"`
	// DependentsAre says, on every node that carries a count, that the count
	// is the graph's estimate rather than a compiler's.
	DependentsAre   string             `json:"dependents_are"`
	Resolution      string             `json:"resolution,omitempty"`
	ResolutionMeans string             `json:"resolution_means,omitempty"`
	Confidence      float64            `json:"confidence,omitempty"`
	GraphWarnings   []string           `json:"graph_warnings,omitempty"`
	Verify          *VerifyPath        `json:"verify,omitempty"`
	CallSite        int                `json:"call_site"`
	Dependents      int                `json:"dependents"`
	Score           float64            `json:"score"`
	Factors         map[string]float64 `json:"factors"`
	Beacon          string             `json:"beacon,omitempty"`
	Signature       string             `json:"signature,omitempty"`
	Tests           []jsonTestRef      `json:"tests"`
	Result          string             `json:"result"`
	Failure         string             `json:"failure_excerpt,omitempty"`
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

// buildJSON turns the analysis into the report structure.
//
// It is unexported and takes an already sealed analysis, so no caller outside
// this package can turn an unscrubbed Analysis into bytes.
func buildJSON(a *Analysis) jsonReport {
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
		Coverage: jsonCoverage{
			Reads: a.Coverage.Reads, Edits: a.Coverage.Edits,
			Searches: a.Coverage.Searches, Quoted: a.Coverage.Quoted,
			Mentions: a.Coverage.Mentions, Commands: a.Coverage.Commands,
			Thin: a.Coverage.Thin(), Line: CoverageLine(a), Note: CoverageNote(a),
		},
		Reach:          a.Reach,
		EvidenceLegend: evidenceLegendEntries(),
		Notes:          a.Notes,
		Timeline:       a.Timeline,
		T0:             a.Cut,
		Execution:      a.Execution,
		Layout:         a.Layout,
		Limitations:    nonNilStrings(a.Limitations),
		CommandsRun:    nonNilStrings(a.Commands),
	}

	for _, s := range a.Sources {
		r.Sources = append(r.Sources, jsonSource{
			ID: s.Symbol, Name: s.Name, File: s.File, Span: s.Span,
			Change: s.Change, KindLabel: s.KindLabel(), Weight: s.Weight, Dependents: s.Dependents,
			OldSignature: snippet(a, s.OldSignature), NewSignature: snippet(a, s.NewSignature),
		})
	}
	es := shadow.SummarizeEvidence(a.Nodes)
	r.GraphEvidence = jsonGraphEvidence{
		Confirmed:         es.Confirmed,
		Heuristic:         es.Heuristic,
		NeedsVerification: es.NeedsVerification,
		MayBeIncomplete:   a.Graph.Degraded(),
		Why:               GraphCompletenessNote(a),
		GraphEvidence:     a.Graph,
	}

	if r.Sources == nil {
		r.Sources = []jsonSource{}
	}
	r.Unresolved = a.Unresolved

	for _, n := range a.Nodes {
		jn := jsonNode{
			ID: n.Symbol.ID, Name: n.Symbol.Name, File: n.Symbol.File, Span: n.Symbol.Span,
			IsTest: n.Symbol.IsTest, Sources: nonNilStrings(n.Sources), Relation: n.Relation,
			Depth: n.Depth, Path: nonNilStrings(n.Path), State: n.State.String(),
			Tier: string(n.Tier), Modifiers: nonNilStrings(n.Modifiers),
			Sentence:        shadow.StateSentence(n.State, n.Tier, n.Symbol.File),
			Evidence:        string(n.Evidence),
			EvidenceMeaning: n.Evidence.Meaning(),
			EvidenceWhy:     n.EvidenceWhy,
			DependentsAre:   shadow.DependentCountIsHeuristic,
			Resolution:      n.Resolution,
			ResolutionMeans: graph.ResolutionMeaning(n.Resolution),
			Confidence:      n.Confidence,
			GraphWarnings:   nonNilStrings(n.EdgeWarnings),
			Verify:          verifyForNode(a, n),
			CallSite:        n.CallSite,
			Dependents:      n.Dependents, Score: n.Score, Factors: n.Factors,
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
func WriteJSON(w io.Writer, sd Sealed) error {
	if !sd.Valid() {
		return ErrUnsealed
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(true)
	return enc.Encode(buildJSON(sd.a))
}

// MarshalJSON returns the report bytes, for embedding in the HTML.
func MarshalJSON(sd Sealed) ([]byte, error) {
	if !sd.Valid() {
		return nil, ErrUnsealed
	}
	return json.MarshalIndent(buildJSON(sd.a), "", "  ")
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// jsonLegendEntry is one of the three evidence words with its plain meaning, so
// a reader of the JSON alone never has to guess what a tier claims.
type jsonLegendEntry struct {
	Word    string `json:"word"`
	Mark    string `json:"mark"`
	Meaning string `json:"meaning"`
}

func evidenceLegendEntries() []jsonLegendEntry {
	var out []jsonLegendEntry
	for _, e := range []shadow.Evidence{shadow.Confirmed, shadow.Heuristic, shadow.NeedsVerification} {
		out = append(out, jsonLegendEntry{Word: string(e), Mark: e.Mark(), Meaning: e.Meaning()})
	}
	return out
}

// jsonGraphEvidence is the plain-word version of what the provider reported.
type jsonGraphEvidence struct {
	Confirmed         int    `json:"confirmed"`
	Heuristic         int    `json:"heuristic"`
	NeedsVerification int    `json:"needs_verification"`
	MayBeIncomplete   bool   `json:"may_be_incomplete"`
	Why               string `json:"why,omitempty"`
	GraphEvidence
}

// verifyForNode returns the commands that settle a node, and nothing for one
// the graph already confirmed.
func verifyForNode(a *Analysis, n *shadow.Node) *VerifyPath {
	if n.Evidence != shadow.NeedsVerification {
		return nil
	}
	v := VerifyFor(a, n)
	return &v
}
