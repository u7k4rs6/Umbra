package main

import (
	"context"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/checkpoint"
	"github.com/u7k4rs6/Umbra/umbra/internal/graph"
	"github.com/u7k4rs6/Umbra/umbra/internal/report"
	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
	"github.com/u7k4rs6/Umbra/umbra/internal/shadow"
	"github.com/u7k4rs6/Umbra/umbra/internal/transcript"
)

// pipelineState is what the later steps need from the earlier ones.
type pipelineState struct {
	Field         *graph.Field
	RelMap        *graph.RelationMap
	SourceIDs     []string
	CoChangeFiles map[string]bool
	// TestRoot is the repository-relative directory the test runner runs in.
	// A repository whose tests live at the root has an empty TestRoot.
	TestRoot string
}

// pipeline runs the seven steps from ARCHITECTURE.md and returns the analysis.
func pipeline(ctx context.Context, o *Options, run runner.Runner, res *checkpoint.Resolution, pair *checkpoint.Pair) (*report.Analysis, *pipelineState, error) {
	scrub := report.NewScrubber(o.Repo)

	a := &report.Analysis{
		Version:      Version,
		CheckpointID: res.CheckpointID,
		Commit:       res.Commit,
		Parent:       res.Parent,
		Agent:        res.Agent,
		Adapter:      "claude-code",
		Route:        res.Route,
		SessionIDs:   res.SessionIDs,
		Depth:        o.Depth,
		TestRunner:   o.Test,
		Run:          o.Run,
		Audit:        !o.NoAudit,
		History:      o.History,
		Snippets:     o.Snippets,
		Channels:     map[string]bool{},
	}
	if res.Note != "" {
		a.Notes = append(a.Notes, res.Note)
	}

	// Step 3a: capabilities, so relation names are read rather than assumed.
	caps := loadCapabilities(ctx, run)
	relMap := graph.NewRelationMap(caps)
	a.RelationsUsed = relMap.Used
	a.RelationsIgnored = relMap.Ignored
	a.Channels["graph"] = true

	// Step 3b: the field.
	head := pair.Head
	if head == "" {
		head = o.Repo
	}
	snap, err := runGraph(ctx, run, []string{"graph", "snapshot", "--repo", head, "--format", "ndjson"})
	if err != nil {
		return nil, nil, fmt.Errorf("loading the graph snapshot: %w", err)
	}
	field, err := graph.LoadSnapshot(snap)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the graph snapshot: %w", err)
	}

	// Step 2: the sources.
	sr, err := graph.LoadSources(ctx, run, head, res.CheckpointID, res.Commit)
	if err != nil {
		return nil, nil, fmt.Errorf("asking Graph what changed: %w", err)
	}
	if sr.Note != "" {
		a.Notes = append(a.Notes, sr.Note)
	}
	bound := graph.Bind(sr.Sources, field)
	var dropped []graph.Source
	a.Sources, dropped = graph.FilterSources(bound, field, caps)
	if len(dropped) > 0 {
		a.Notes = append(a.Notes, fmt.Sprintf(
			"%d changed entity(s) in languages the graph cannot resolve calls for were not treated as sources",
			len(dropped)))
	}
	if len(a.Sources) == 0 && len(bound) > 0 {
		a.Notes = append(a.Notes, "nothing in this commit changed code the graph can follow, so there is no field to light")
	}

	sourceIDs := make([]string, 0, len(a.Sources))
	sourceFiles := map[string]bool{}
	for _, s := range a.Sources {
		sourceFiles[s.File] = true
		if s.Symbol != "" {
			sourceIDs = append(sourceIDs, s.Symbol)
		}
	}

	// Step 3c: impact per source, for exact call-site lines and co-change.
	impacts := map[string]*graph.Impact{}
	for _, s := range a.Sources {
		if s.Symbol == "" {
			continue
		}
		imp, err := graph.LoadImpact(ctx, run, head, s.Name, s.File, s.Span[0])
		if err != nil {
			continue
		}
		impacts[s.Symbol] = imp
	}
	a.Channels["callsites"] = len(impacts) > 0

	// Step 4: the exposure.
	raw, err := checkpoint.RawTranscript(ctx, run, res.CheckpointID)
	if err != nil {
		return nil, nil, fmt.Errorf("reading the checkpoint transcript: %w", err)
	}
	session, err := transcript.ClaudeCode{RepoRoot: o.Repo}.Parse(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing the transcript: %w", err)
	}
	session.ResolveMentions(shadow.Vocabulary(field))
	a.Channels["reads"] = session.HasExposure()
	a.Channels["mentions"] = true
	a.Coverage = session.Coverage()

	examined := shadow.BuildExamined(session, sourceFiles)
	a.Cut, a.HasCut = examined.Cut, examined.HasCut
	a.Timeline = buildTimeline(session, scrub, field)

	if !examined.Any {
		a.Notes = append(a.Notes, "the transcript carries no tool activity, so the examined set is unavailable and every node is unknown")
	}

	// The session's own account of itself.
	a.SessionSaid, a.SessionSaidFrom = sessionSaid(ctx, run, res.CheckpointID, session, examined, scrub)

	// Step 5: classify and rank.
	reach := field.Dependents(sourceIDs, o.Depth, relMap)

	var scars map[string]int
	if o.History {
		files := map[string]bool{}
		for _, r := range reach {
			if s := field.Symbols[r.ID]; s != nil {
				files[s.File] = true
			}
		}
		list := make([]string, 0, len(files))
		for f := range files {
			list = append(list, f)
		}
		sort.Strings(list)
		if scars, err = shadow.LoadScars(ctx, run, head, list); err != nil {
			return nil, nil, fmt.Errorf("reading git history for the scar factor: %w", err)
		}
	}

	in := shadow.BuildInput{
		Field: field, RelMap: relMap, Sources: a.Sources, Reach: reach,
		Examined: examined, Impacts: impacts, HeadRoot: head, Scars: scars,
	}
	a.Nodes = shadow.Build(in)
	a.Nodes = shadow.AddCoChange(a.Nodes, in)
	shadow.SortDocket(a.Nodes)
	a.Summary = shadow.Summarize(a.Nodes)

	a.Limitations = limitations(a, caps, examined)
	if ex, ok := run.(*runner.Exec); ok {
		for _, c := range ex.Calls() {
			a.Commands = append(a.Commands, scrub.Command(c.String()))
		}
	}

	st := &pipelineState{
		Field: field, RelMap: relMap, SourceIDs: sourceIDs,
		CoChangeFiles: map[string]bool{},
		TestRoot:      testRoot(o, head, a.Nodes),
	}
	for _, imp := range impacts {
		for _, f := range imp.CoChange {
			st.CoChangeFiles[f] = true
		}
	}
	return a, st, nil
}

// testRoot decides the directory the test runner runs in.
//
// A repository whose tests sit at the root needs nothing. A project nested in
// a subdirectory, which is how the fixture app is laid out, needs the runner
// to start there or imports fail. The directory is found by walking up from
// the test files to the nearest ancestor holding a recognised project marker.
// --test-root overrides the search.
func testRoot(o *Options, headRoot string, nodes []*shadow.Node) string {
	if o.TestRoot != "" {
		return strings.Trim(filepath.ToSlash(o.TestRoot), "/")
	}
	markers := []string{"pytest.ini", "pyproject.toml", "setup.cfg", "tox.ini", "go.mod", "package.json", "Cargo.toml"}

	best := ""
	for _, n := range nodes {
		if n.Symbol == nil || !n.Symbol.IsTest {
			continue
		}
		dir := path.Dir(n.Symbol.File)
		for dir != "." && dir != "/" && dir != "" {
			for _, m := range markers {
				if _, err := os.Stat(filepath.Join(headRoot, filepath.FromSlash(dir), m)); err == nil {
					if best == "" || len(dir) < len(best) {
						best = dir
					}
					dir = "."
					break
				}
			}
			if dir == "." {
				break
			}
			dir = path.Dir(dir)
		}
	}
	return best
}

func loadCapabilities(ctx context.Context, run runner.Runner) *graph.Capabilities {
	blob, err := runGraph(ctx, run, []string{"graph", "capabilities", "--json"})
	if err != nil {
		return nil
	}
	caps, err := graph.ParseCapabilities(blob)
	if err != nil {
		return nil
	}
	return caps
}

func runGraph(ctx context.Context, run runner.Runner, args []string) ([]byte, error) {
	stdout, stderr, exit, err := run.Run(ctx, "entire", args, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return nil, fmt.Errorf("entire %s exited %d: %s",
			strings.Join(args, " "), exit, strings.TrimSpace(string(stderr)))
	}
	return stdout, nil
}

// sessionSaid returns the agent's own sentence and where it came from.
//
// Entire's stored summary is preferred. The Step 0 probe found imported
// checkpoints never carry one, so the fallback is the last assistant sentence
// at or before the cut. Both are scrubbed and capped; neither is verified.
func sessionSaid(ctx context.Context, run runner.Runner, id string, s *transcript.Session, e *shadow.Examined, scrub *report.Scrubber) (string, string) {
	if stored, err := checkpoint.StoredSummary(ctx, run, id); err == nil && stored != "" {
		return scrub.Sentence(stored), "the checkpoint summary Entire stored"
	}
	// Bound the search to the change itself. A checkpoint that spans more
	// than one piece of work would otherwise offer its final sentence, which
	// may have nothing to do with this commit.
	if e != nil && e.LastSourceTouch > 0 {
		if sent := s.SentenceAfter(e.LastSourceTouch); sent != "" {
			return scrub.Sentence(sent), "what the agent said after its last edit, because Entire stored no summary"
		}
	}
	// No stored summary and no edit to bound the search. Taking the last
	// sentence of the whole transcript would put an arbitrary line of the
	// session's conversation into a report that people share, and in a long
	// session that line has nothing to do with the commit. Omit it: the
	// header already says the summary is absent, and the coverage line says
	// when a session left no file tool events to bound it by.
	return "", ""
}

// buildTimeline reduces the session to what a report may carry.
//
// Paths are filtered against the repository's real file list. A tool result is
// mined for path-shaped text, and not everything path-shaped is a file: a
// domain and an email address both match the shape, and one of each reached a
// committed report before this filter existed. A path that is not a file in
// this repository can say nothing about the change, so it is dropped.
func buildTimeline(s *transcript.Session, scrub *report.Scrubber, field *graph.Field) []report.TimelineEvent {
	known := map[string]bool{}
	if field != nil {
		for _, f := range field.Files {
			known[f] = true
		}
	}
	keep := func(paths []string) []string {
		if len(known) == 0 {
			return paths
		}
		var out []string
		for _, p := range paths {
			if known[p] {
				out = append(out, p)
			}
		}
		return out
	}

	var out []report.TimelineEvent
	for _, ev := range s.Events {
		switch ev.Kind {
		case transcript.AssistantText, transcript.Prompt:
			continue
		}
		te := report.TimelineEvent{
			Seq:     ev.Seq,
			Kind:    ev.Kind.String(),
			Path:    ev.Path,
			Range:   ev.Range,
			Paths:   keep(ev.Paths),
			Symbols: ev.Symbols,
		}
		// An event whose only content was paths outside the repository has
		// nothing left to say.
		if len(ev.Paths) > 0 && len(te.Paths) == 0 && te.Path == "" && len(te.Symbols) == 0 && te.Cmd == "" {
			continue
		}
		if !ev.TS.IsZero() {
			te.TS = ev.TS.UTC().Format("2006-01-02T15:04:05Z")
		}
		if ev.Cmd != "" {
			te.Cmd = scrub.Command(ev.Cmd)
		}
		out = append(out, te)
	}
	return out
}

// limitations states, in the report itself, what this run could not see.
func limitations(a *report.Analysis, caps *graph.Capabilities, e *shadow.Examined) []string {
	var out []string
	out = append(out, "graph edges are incomplete: reflection, dynamic dispatch and configuration are invisible, so umbra is a lower bound on what was unexamined")
	out = append(out, "read evidence is file and line-range based; a mention in the agent's text is attention, not reading, and is the weakest tier")
	out = append(out, "fault lines and beacons are matched by a regex window around the call site, not by a parser")
	out = append(out, "ranking weights are hand set and printed; they are a heuristic, not a measurement")
	if !e.Any {
		out = append(out, "this transcript carried no tool activity, so every node is unknown rather than judged")
	}
	if len(a.RelationsIgnored) > 0 {
		out = append(out, "relations not traversed: "+strings.Join(a.RelationsIgnored, ", "))
	}
	if caps == nil {
		out = append(out, "graph capabilities were unavailable, so relation names fell back to the built-in table")
	}
	return out
}

func writeTable(a *report.Analysis, all bool) error {
	return report.Table(os.Stdout, a, report.DetectTableOptions(all))
}
