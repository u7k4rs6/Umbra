package graph

import "testing"

// The provider tags every relation record with a resolution and a confidence.
// Umbra read the type and dropped both, so a name-only guess and a resolved
// call arrived at the map indistinguishable from each other.
func TestLoadSnapshotKeepsResolutionAndConfidence(t *testing.T) {
	f := loadField(t)
	if len(f.Resolutions) == 0 {
		t.Fatal("no resolutions counted; the loader is still dropping them")
	}

	src := symbolID(t, f, "app/service.py", "compute_total")
	var seen int
	for _, e := range f.In[src] {
		if e.Relation != "CALLS" {
			continue
		}
		seen++
		if e.Quality.Resolution == "" {
			t.Errorf("CALLS edge from %s carries no resolution", e.From)
		}
		if e.Quality.Confidence <= 0 {
			t.Errorf("CALLS edge from %s carries no confidence", e.From)
		}
	}
	if seen == 0 {
		t.Fatal("compute_total has no incoming CALLS in the fixture")
	}
}

// The fixture's own resolutions, asserted by value, so a change in what the
// provider reports for this snapshot is a test failure rather than a silent
// change in what the report claims.
func TestFixtureResolutionsAreTheOnesTheProviderReported(t *testing.T) {
	f := loadField(t)
	for _, want := range []struct {
		resolution string
		atLeast    int
	}{
		{ResolutionExact, 1},
		{ResolutionImportResolved, 1},
		{ResolutionTypeInferred, 1},
		{ResolutionNameOnly, 1},
		{ResolutionPattern, 1},
		{ResolutionGitHistory, 1},
	} {
		if f.Resolutions[want.resolution] < want.atLeast {
			t.Errorf("resolution %q counted %d, want at least %d",
				want.resolution, f.Resolutions[want.resolution], want.atLeast)
		}
	}
}

func TestResolutionIsStructuralOnlyForAParse(t *testing.T) {
	for _, r := range []string{ResolutionExact, ResolutionImportResolved} {
		if !ResolutionIsStructural(r) {
			t.Errorf("%q should be structural", r)
		}
	}
	for _, r := range []string{
		ResolutionNameOnly, ResolutionPattern, ResolutionTypeInferred,
		ResolutionPackage, ResolutionImportExternal, ResolutionGitHistory, "",
	} {
		if ResolutionIsStructural(r) {
			t.Errorf("%q should not be structural", r)
		}
		if ResolutionMeaning(r) == "" {
			t.Errorf("%q has no plain meaning beside the provider's word", r)
		}
	}
}

// A path is only as resolved as its weakest link. The walk used to keep the
// first hop's relation and nothing about how any hop was resolved.
func TestDependentsCarriesTheWeakestHop(t *testing.T) {
	f := &Field{
		Symbols: map[string]*Symbol{
			"src": {ID: "src", Name: "src", File: "a.py"},
			"mid": {ID: "mid", Name: "mid", File: "a.py"},
			"far": {ID: "far", Name: "far", File: "a.py"},
		},
		In: map[string][]Edge{
			"src": {{From: "mid", To: "src", Relation: "CALLS", Quality: EdgeQuality{Resolution: ResolutionExact, Confidence: 1}}},
			"mid": {{From: "far", To: "mid", Relation: "CALLS", Quality: EdgeQuality{Resolution: ResolutionNameOnly, Confidence: 0.68}}},
		},
		Out:       map[string][]Edge{},
		ByFile:    map[string][]string{},
		Relations: map[string]int{},
	}
	rm := NewRelationMap(&Capabilities{
		SupportedRelationTypes: []string{"CALLS"},
		HeuristicRelations:     []string{"TESTS"},
	})

	got := map[string]Reach{}
	for _, r := range f.Dependents([]string{"src"}, 2, rm) {
		got[r.ID] = r
	}

	mid, ok := got["mid"]
	if !ok {
		t.Fatal("mid was not reached")
	}
	if mid.Weakest.Resolution != ResolutionExact || mid.Heuristic {
		t.Errorf("mid = %q heuristic=%v, want exact and not heuristic", mid.Weakest.Resolution, mid.Heuristic)
	}

	far, ok := got["far"]
	if !ok {
		t.Fatal("far was not reached")
	}
	if far.Weakest.Resolution != ResolutionNameOnly {
		t.Errorf("far resolution = %q, want the weakest hop %q", far.Weakest.Resolution, ResolutionNameOnly)
	}
	if !far.Heuristic {
		t.Error("far was reached through a name-only hop and is not marked heuristic")
	}
	if far.HeuristicVia != "far" {
		t.Errorf("far heuristic via = %q, want the hop that made it so", far.HeuristicVia)
	}
	if far.Weakest.Confidence != 0.68 {
		t.Errorf("far confidence = %v, want the lowest on the path", far.Weakest.Confidence)
	}
}

// capabilities has always reported which relation types are heuristic and
// nothing read the field.
func TestRelationMapReadsHeuristicRelationTypes(t *testing.T) {
	rm := loadRelMap(t)
	if len(rm.Heuristic) == 0 {
		t.Fatal("no heuristic relation types read from capabilities")
	}
	for _, name := range []string{"HANDLES_ROUTE", "HANDLES_TOOL", "TESTS"} {
		if !rm.IsHeuristic(name) {
			t.Errorf("%s should be heuristic; capabilities lists it", name)
		}
	}
	if rm.IsHeuristic("CALLS") {
		t.Error("CALLS is not in the provider's heuristic list")
	}
	// Co-change is heuristic by construction rather than by declaration.
	if !rm.IsHeuristic("FILE_CHANGES_WITH") {
		t.Error("FILE_CHANGES_WITH comes from commit history and is never structural")
	}
}

// A snapshot whose records are all malformed produced an empty field, and an
// empty field reads exactly like a clean one. This is the same shape as the
// false zero the external review found in the sweep.
func TestLoadSnapshotCountsMalformedRecordsRatherThanDroppingThem(t *testing.T) {
	blob := []byte("not json\n" +
		"{unterminated\n" +
		`{"record_type":"symbol","id":"a","kind":"function","name":"f","file_path":"x.py","start_line":1,"end_line":2}` + "\n" +
		`{"record_type":"symbol","kind":"function","name":"g","file_path":"x.py"}` + "\n" +
		`{"record_type":"relation","to_id":"a","type":"CALLS"}` + "\n")
	f, err := LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	if len(f.Symbols) != 1 {
		t.Fatalf("symbols = %d, want 1", len(f.Symbols))
	}
	if f.Meta.Malformed != 2 {
		t.Errorf("malformed = %d, want 2", f.Meta.Malformed)
	}
	// One symbol with no id, one relation with no from_id.
	if f.Meta.Dropped != 2 {
		t.Errorf("dropped = %d, want 2", f.Meta.Dropped)
	}
	if !f.Meta.Degraded() {
		t.Error("a snapshot with unreadable records is not a complete reading of the repository")
	}
	if f.Meta.SawSummary {
		t.Error("this snapshot carried no summary record")
	}
}

// The header carries the profile and the summary carries the counts. Reading
// only one of them loses half of what the provider said about itself.
func TestLoadSnapshotReadsTheHeaderAndTheSummary(t *testing.T) {
	blob := []byte(`{"schema_version":"1.1","provider":"entire-graph","provider_version":"dev","profile":"full","commit":"abc","warnings":[],"partial_failures":[],"stats":{"files":0}}` + "\n" +
		`{"record_type":"symbol","id":"a","kind":"function","name":"f","file_path":"x.py","start_line":1,"end_line":2}` + "\n" +
		`{"record_type":"summary","language_tiers":{"Python":"semantic","CSS":"inventory-only"},` +
		`"warnings":[{"code":"W_DATA_FLOW_EVIDENCE_UNMERGED","severity":"info","effect_on_semantic_completeness":"evidence explains the edge from one side only"}],` +
		`"partial_failures":[{"code":"E_MINIFIED","severity":"warning","file_path":"x.py","effect_on_semantic_completeness":"file record emitted but symbol parsing skipped"}],` +
		`"stats":{"files":3,"parsed_files":2,"symbols":1,"relations":0,"partial_failures":1,"completeness_level":"degraded"}}` + "\n")
	f, err := LoadSnapshot(blob)
	if err != nil {
		t.Fatalf("LoadSnapshot: %v", err)
	}
	m := f.Meta
	if m.Profile != "full" {
		t.Errorf("profile = %q, want full; the header line carries no record_type and used to be skipped", m.Profile)
	}
	if !m.SawSummary {
		t.Error("the summary record was not read")
	}
	if m.Stats.CompletenessLevel != "degraded" {
		t.Errorf("completeness = %q, want degraded", m.Stats.CompletenessLevel)
	}
	if m.Stats.ParsedFiles != 2 || m.Stats.Files != 3 {
		t.Errorf("stats = %d of %d files parsed, want 2 of 3", m.Stats.ParsedFiles, m.Stats.Files)
	}
	if len(m.Warnings) != 1 || len(m.PartialFailures) != 1 {
		t.Fatalf("warnings = %d, partial failures = %d, want 1 and 1", len(m.Warnings), len(m.PartialFailures))
	}
	if !m.FailedFiles()["x.py"] {
		t.Error("the partial failure names x.py and FailedFiles does not")
	}
	if !m.InventoryOnly("CSS") || m.InventoryOnly("Python") {
		t.Error("language tiers were not read from the summary")
	}
	if got, want := m.Codes(), []string{"E_MINIFIED", "W_DATA_FLOW_EVIDENCE_UNMERGED"}; len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Errorf("codes = %v, want %v", got, want)
	}
	if !m.Degraded() {
		t.Error("the provider said degraded and Degraded reports otherwise")
	}
}

// graph impact prints its own completeness block above every answer, including
// a per-query scope saying whether the degradation can affect that answer.
func TestParseImpactJSONKeepsTheCompletenessBlock(t *testing.T) {
	imp, err := ParseImpactJSON(read(t, "impact.json"))
	if err != nil {
		t.Fatalf("ParseImpactJSON: %v", err)
	}
	_ = imp

	blob := []byte(`{"callers":{"direct":1,"transitive":0,"entries":[]},` +
		`"warnings":[{"code":"W_DATA_FLOW_EVIDENCE_UNMERGED","severity":"info","effect_on_semantic_completeness":"one side only"}],` +
		`"partial_failures":[{"code":"E_PARSE_ERROR","severity":"warning","file_path":"a.h","effect_on_semantic_completeness":"parsed with syntax errors"}],` +
		`"stats":{"completeness_level":"degraded"},` +
		`"completeness_scope":{"language":"Python","other_language_failures":8,` +
		`"in_scope_warnings":[{"code":"W_DATA_FLOW_EVIDENCE_UNMERGED"}],"level":"degraded"}}`)
	got, err := ParseImpactJSON(blob)
	if err != nil {
		t.Fatalf("ParseImpactJSON: %v", err)
	}
	if !got.Degraded() {
		t.Error("the provider said degraded for this query and Degraded reports otherwise")
	}
	if got.ScopeLanguage != "Python" || got.OutOfScopeFailures != 8 {
		t.Errorf("scope = %q with %d out-of-scope failures, want Python and 8",
			got.ScopeLanguage, got.OutOfScopeFailures)
	}
	if len(got.InScopeWarnings) != 1 || got.InScopeWarnings[0] != "W_DATA_FLOW_EVIDENCE_UNMERGED" {
		t.Errorf("in-scope warnings = %v", got.InScopeWarnings)
	}
	if len(got.Warnings) != 1 || len(got.PartialFailures) != 1 {
		t.Errorf("warnings = %d, partial failures = %d", len(got.Warnings), len(got.PartialFailures))
	}
}

// The text fallback has to read the same thing out of the human form.
func TestParseImpactTextReadsTheCompletenessLine(t *testing.T) {
	text := "Index: cache-miss (100ms)\n" +
		"Completeness: degraded for Go (0 of 492 Go files failed to parse; 10 total diagnostics)\n" +
		"- warning W_DATA_FLOW_EVIDENCE_UNMERGED\n" +
		"- partial E_PARSE_ERROR: internal/sem/grammars/csharp/tree_sitter/array.h\n" +
		"Callers (1 direct, 0 transitive; who breaks if behavior changes):\n" +
		"- handle_order (app/api.py:16, def :14)\n"
	imp := ParseImpactText(text)
	if imp.ScopeLevel != "degraded" || imp.ScopeLanguage != "Go" {
		t.Errorf("scope = %q for %q, want degraded for Go", imp.ScopeLevel, imp.ScopeLanguage)
	}
	if !imp.Degraded() {
		t.Error("Degraded is false after reading a degraded completeness line")
	}
	if len(imp.Warnings) != 1 || len(imp.PartialFailures) != 1 {
		t.Errorf("warnings = %d, partial failures = %d, want 1 and 1",
			len(imp.Warnings), len(imp.PartialFailures))
	}
	if imp.DirectCallers != 1 {
		t.Errorf("direct callers = %d, want 1; the diagnostic lines must not be read as callers",
			imp.DirectCallers)
	}
}

// The provider marks W_DATA_FLOW_EVIDENCE_UNMERGED severity `info` and its own
// text says the relation, its confidence and its reason are unaffected. On the
// demo commit it is the only in-scope warning, and treating it as a reason to
// doubt every exact CALLS edge turned all thirteen dependents into "needs
// verification": a tier that fires on everything carries no information.
//
// This is the known-positive for that. Without the severity split the first
// case below reports a severe in-scope warning and the whole field is promoted.
func TestAnInfoWarningIsReportedAndDoesNotPromoteAnything(t *testing.T) {
	blob := []byte(`{"callers":{"direct":1,"transitive":0,"entries":[]},` +
		`"partial_failures":[{"code":"E_PARSE_ERROR","severity":"warning","file_path":"vendor/a.h"},` +
		`{"code":"E_MINIFIED","severity":"warning","file_path":"b.json"}],` +
		`"stats":{"completeness_level":"degraded"},` +
		`"completeness_scope":{"language":"Python","other_language_failures":2,` +
		`"in_scope_warnings":[{"code":"W_DATA_FLOW_EVIDENCE_UNMERGED","severity":"info"}],` +
		`"level":"degraded"}}`)
	imp, err := ParseImpactJSON(blob)
	if err != nil {
		t.Fatalf("ParseImpactJSON: %v", err)
	}
	if !imp.Degraded() {
		t.Error("the provider said degraded and it is not being reported")
	}
	if len(imp.InScopeWarnings) != 1 {
		t.Errorf("in-scope warnings = %v, want the info one reported", imp.InScopeWarnings)
	}
	if len(imp.InScopeSevere) != 0 {
		t.Errorf("severe in-scope warnings = %v, want none: the only one is severity info", imp.InScopeSevere)
	}
	if imp.InScopeFailures() != 0 {
		t.Errorf("in-scope failures = %d, want 0: the provider scoped both to other languages",
			imp.InScopeFailures())
	}

	// The opposite half. A warning the provider did not mark info, and a
	// failure it did not scope away, both have to promote.
	blob = []byte(`{"callers":{"direct":1,"transitive":0,"entries":[]},` +
		`"partial_failures":[{"code":"E_PARSE_ERROR","severity":"warning","file_path":"app/x.py"}],` +
		`"stats":{"completeness_level":"degraded"},` +
		`"completeness_scope":{"language":"Python","other_language_failures":0,` +
		`"in_scope_warnings":[{"code":"W_SOMETHING_REAL","severity":"warning"}],"level":"degraded"}}`)
	imp, err = ParseImpactJSON(blob)
	if err != nil {
		t.Fatalf("ParseImpactJSON: %v", err)
	}
	if len(imp.InScopeSevere) != 1 || imp.InScopeSevere[0] != "W_SOMETHING_REAL" {
		t.Errorf("severe in-scope warnings = %v, want the warning-severity one", imp.InScopeSevere)
	}
	if imp.InScopeFailures() != 1 {
		t.Errorf("in-scope failures = %d, want 1: the provider scoped none away", imp.InScopeFailures())
	}
}
