// Command gen-site builds the landing page.
//
// It copies the report's own CSS and JavaScript into site/ so the two surfaces
// cannot drift apart, and writes a sample umbra.json built from the recorded
// classic scenario, so the live map on the page is a real report rather than a
// mock-up. No real session prose reaches the sample: the scenario transcripts
// are authored fixtures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

func main() {
	root := flag.String("root", ".", "the umbra module directory")
	flag.Parse()

	if err := run(*root); err != nil {
		fmt.Fprintf(os.Stderr, "gen-site: %v\n", err)
		os.Exit(1)
	}
}

func run(root string) error {
	site := filepath.Join(root, "site")
	if err := os.MkdirAll(filepath.Join(site, "sample"), 0o755); err != nil {
		return err
	}

	// The page uses the report's own stylesheet and script.
	for _, name := range []string{"umbra.css", "umbra.js"} {
		src := filepath.Join(root, "internal", "report", "assets", name)
		blob, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(site, name), blob, 0o644); err != nil {
			return err
		}
		fmt.Printf("copied %s\n", name)
	}

	a, err := sampleAnalysis(root)
	if err != nil {
		return err
	}
	a.Layout = report.BuildLayout(a)

	blob, err := report.MarshalJSON(a)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(site, "sample", "umbra.json"), append(blob, '\n'), 0o644); err != nil {
		return err
	}
	fmt.Printf("wrote sample/umbra.json (%d nodes)\n", len(a.Nodes))

	// The sample is embedded in the page as well as written beside it. A page
	// opened from file:// cannot fetch its own sibling, and the landing page
	// has to work from file://.
	if err := embedSample(filepath.Join(site, "index.html"), blob); err != nil {
		return err
	}
	fmt.Println("embedded the sample in index.html")
	return nil
}

const (
	openTag  = `<script type="application/json" id="umbra-data">`
	closeTag = `</script>`
)

// embedSample replaces the contents of the page's data block.
func embedSample(path string, blob []byte) error {
	page, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	text := string(page)
	start := strings.Index(text, openTag)
	if start < 0 {
		return fmt.Errorf("%s has no umbra-data block", path)
	}
	from := start + len(openTag)
	end := strings.Index(text[from:], closeTag)
	if end < 0 {
		return fmt.Errorf("%s has an unclosed umbra-data block", path)
	}

	// The same escaping the report uses: a literal closing tag inside a string
	// would end the script element early.
	safe := strings.ReplaceAll(string(blob), "</", `<\/`)
	out := text[:from] + safe + text[from+end:]
	return os.WriteFile(path, []byte(out), 0o644)
}

// sampleAnalysis rebuilds the classic scenario from the recorded transcript
// and the captured snapshot, with no process started and nothing read from a
// live repository.
func sampleAnalysis(root string) (*report.Analysis, error) {
	snap, err := os.ReadFile(filepath.Join(root, "internal", "graph", "testdata", "snapshot.ndjson"))
	if err != nil {
		return nil, err
	}
	field, err := graph.LoadSnapshot(snap)
	if err != nil {
		return nil, err
	}
	capsBlob, err := os.ReadFile(filepath.Join(root, "internal", "graph", "testdata", "capabilities.json"))
	if err != nil {
		return nil, err
	}
	caps, err := graph.ParseCapabilities(capsBlob)
	if err != nil {
		return nil, err
	}
	relMap := graph.NewRelationMap(caps)

	var src *graph.Symbol
	for _, id := range field.ByFile["app/service.py"] {
		if field.Symbols[id].Name == "compute_total" {
			src = field.Symbols[id]
		}
	}
	if src == nil {
		return nil, fmt.Errorf("compute_total is missing from the captured snapshot")
	}
	sources := []graph.Source{{
		Symbol: src.ID, Name: src.Name, File: src.File, Span: src.Span,
		Change: "signature", Weight: graph.Weight("signature"), Dependents: 7,
		OldSignature: "def compute_total(items)", NewSignature: "def compute_total(items, tax_rate)",
	}}

	jsonl, err := os.ReadFile(filepath.Join(root, "fixtures", "recorded", "classic", "transcript.jsonl"))
	if err != nil {
		return nil, err
	}
	session, err := transcript.ClaudeCode{RepoRoot: "/repo"}.Parse(jsonl)
	if err != nil {
		return nil, err
	}
	// The scenario paths carry the project prefix; the captured snapshot was
	// scrubbed without it.
	const prefix = "umbra/fixtures/app/"
	for i := range session.Events {
		session.Events[i].Path = trim(session.Events[i].Path, prefix)
		for j := range session.Events[i].Paths {
			session.Events[i].Paths[j] = trim(session.Events[i].Paths[j], prefix)
		}
	}
	session.ResolveMentions(shadow.Vocabulary(field))
	examined := shadow.BuildExamined(session, map[string]bool{"app/service.py": true})

	impacts := map[string]*graph.Impact{
		src.ID: {
			CallSites: map[string]int{},
			CallSitesByName: map[string]int{
				"app/refunds.py:apply_refund":         23,
				"app/api.py:handle_order":             16,
				"app/api.py:quote":                    29,
				"tests/test_service.py:test_rounding": 9,
			},
			CoChange: []string{"app/api.py"},
		},
	}

	in := shadow.BuildInput{
		Field: field, RelMap: relMap, Sources: sources,
		Reach:    field.Dependents([]string{src.ID}, 2, relMap),
		Examined: examined, Impacts: impacts,
		HeadRoot: filepath.Join(root, "fixtures", "app"),
	}
	nodes := shadow.Build(in)
	shadow.SortDocket(nodes)

	a := &report.Analysis{
		Version:          "0.1.0",
		CheckpointID:     "sample000class",
		Commit:           "0000000000000000000000000000000000000001",
		Parent:           "0000000000000000000000000000000000000000",
		Agent:            "claude-code",
		Adapter:          "claude-code",
		Route:            "checkpoint id",
		Depth:            2,
		TestRunner:       "pytest -v",
		Run:              "shadow",
		Audit:            true,
		Channels:         map[string]bool{"reads": true, "mentions": true, "callsites": true, "graph": true},
		RelationsUsed:    relMap.Used,
		RelationsIgnored: relMap.Ignored,
		SessionSaid:      "Checked the callers. All tests pass.",
		SessionSaidFrom:  "the checkpoint summary Entire stored",
		Sources:          sources,
		Nodes:            nodes,
		Cut:              examined.Cut,
		HasCut:           examined.HasCut,
		Notes: []string{
			"this is the seeded fixture from the repository, recorded so the page has a real report to draw",
		},
		Limitations: []string{
			"graph edges are incomplete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined",
			"read evidence is file and line-range based; a mention in the agent's text is attention, not reading, and is the weakest tier",
			"fault lines and beacons are matched by a regex window around the call site, not by a parser",
			"ranking weights are hand set and printed; they are a heuristic, not a measurement",
			"the fixture is seeded to produce lit, penumbra and umbra nodes and failing shadowed tests",
		},
		Commands: []string{
			"entire graph snapshot --repo . --format ndjson",
			"entire graph commit HEAD --repo . --json",
			"entire graph impact --symbol compute_total --repo . --format json",
			"entire checkpoint explain <id> --raw-transcript",
		},
	}
	a.Summary = shadow.Summarize(nodes)
	a.Timeline = timelineOf(session)

	// The probes that reach the shadow, with the outcomes the fixture really
	// produces when the signature changes.
	cands, _ := shadow.SelectTests(nodes, field, relMap, []string{src.ID}, "shadow", "")
	a.Execution.Selected = shadow.IDs(cands)
	sort.Strings(a.Execution.Selected)
	a.Execution.NewFailures = []string{
		"tests/test_service.py::test_empty_is_zero",
		"tests/test_service.py::test_negative",
		"tests/test_service.py::test_rounding",
	}
	a.Execution.Sweep = true
	a.Execution.Verdict = "REGRESSION in 3 tests"
	for _, n := range nodes {
		for _, id := range a.Execution.NewFailures {
			if id == "tests/"+base(n.Symbol.File)+"::"+n.Symbol.Name {
				n.Result = shadow.OutcomeFail
			}
		}
	}
	return a, nil
}

func timelineOf(s *transcript.Session) []report.TimelineEvent {
	var out []report.TimelineEvent
	for _, ev := range s.Events {
		switch ev.Kind {
		case transcript.AssistantText, transcript.Prompt:
			continue
		}
		te := report.TimelineEvent{
			Seq: ev.Seq, Kind: ev.Kind.String(), Path: ev.Path,
			Range: ev.Range, Paths: ev.Paths, Symbols: ev.Symbols, Cmd: ev.Cmd,
		}
		if !ev.TS.IsZero() {
			te.TS = ev.TS.UTC().Format("2006-01-02T15:04:05Z")
		}
		out = append(out, te)
	}
	return out
}

func trim(s, prefix string) string {
	if len(s) > len(prefix) && s[:len(prefix)] == prefix {
		return s[len(prefix):]
	}
	return s
}

func base(p string) string {
	return filepath.Base(p)
}

var _ = json.Marshal
