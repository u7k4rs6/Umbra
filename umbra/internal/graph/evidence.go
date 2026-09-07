package graph

import "sort"

// This file carries the two things every relation record already tells us and
// Umbra used to drop on the floor: how the relation was resolved, and how
// confident the provider is in it.
//
// Nothing here decides what a reader is shown. It reads what Graph reports and
// makes it available; `internal/shadow` turns it into the three evidence
// tiers, and the renderers print those.

// Resolution values the installed provider emits on a relation record,
// observed across 66,644 relations in a snapshot of this repository.
const (
	ResolutionExact          = "exact"
	ResolutionImportResolved = "import_resolved"
	ResolutionImportExternal = "import_external"
	ResolutionPackage        = "package"
	ResolutionTypeInferred   = "type_inferred"
	ResolutionNameOnly       = "name_only"
	ResolutionPattern        = "pattern"
	ResolutionGitHistory     = "git_history"
)

// resolutionRank orders resolutions strongest first, so a path can report its
// weakest hop. An unreported resolution is weaker than every reported one,
// because a record that did not say cannot be taken to have said "exact".
var resolutionRank = map[string]int{
	ResolutionExact:          7,
	ResolutionImportResolved: 6,
	ResolutionImportExternal: 5,
	ResolutionPackage:        4,
	ResolutionTypeInferred:   3,
	ResolutionNameOnly:       2,
	ResolutionPattern:        1,
	ResolutionGitHistory:     1,
}

// ResolutionRank returns the strength of a resolution. Unknown or absent
// resolutions rank 0, below everything the provider reports.
func ResolutionRank(r string) int { return resolutionRank[r] }

// ResolutionIsStructural reports whether the provider resolved this relation
// to a definition rather than inferring it.
//
// Only `exact` and `import_resolved` qualify. `package` and `import_external`
// are name matches across a boundary, `type_inferred` is inference,
// `name_only` is a unique-name match, `pattern` is a regex over a framework
// convention, and `git_history` is co-change. Every one of those can be right
// and none of them is a compiler.
func ResolutionIsStructural(r string) bool {
	return r == ResolutionExact || r == ResolutionImportResolved
}

// ResolutionMeaning is the plain sentence shown next to the provider's own
// word, so a reader who has never read the provider's schema can still read
// the report. FRONTEND_SPEC.md requires this of every coined word, and these
// are the provider's coinages rather than Umbra's.
func ResolutionMeaning(r string) string {
	switch r {
	case ResolutionExact:
		return "the provider resolved this to the definition itself"
	case ResolutionImportResolved:
		return "resolved through an import the provider followed"
	case ResolutionImportExternal:
		return "matched to something imported from outside this repository"
	case ResolutionPackage:
		return "matched by name within a package, not to a definition"
	case ResolutionTypeInferred:
		return "the receiver's type was inferred, not declared"
	case ResolutionNameOnly:
		return "matched because the name is unique, nothing more"
	case ResolutionPattern:
		return "matched by a framework pattern, not by a parse"
	case ResolutionGitHistory:
		return "taken from commit history, not from the code"
	case "":
		return "the provider reported no resolution for this relation"
	}
	return "a resolution this build of Umbra does not have a plain word for"
}

// Warning is one entry from the snapshot summary's `warnings` array, or the
// identical array `graph impact --format json` prints.
type Warning struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	FilePath string `json:"file_path,omitempty"`
	Effect   string `json:"effect_on_semantic_completeness"`
	Detail   string `json:"detail,omitempty"`
}

// PartialFailure is one entry from `partial_failures`: a file the provider
// could not fully analyse. The provider guarantees these are reported rather
// than silently dropped, and until now Umbra dropped them anyway.
type PartialFailure struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	FilePath string `json:"file_path,omitempty"`
	Effect   string `json:"effect_on_semantic_completeness"`
	Detail   string `json:"detail,omitempty"`
}

// Stats is the snapshot summary's own count of what it managed to read.
type Stats struct {
	Files             int    `json:"files"`
	ParsedFiles       int    `json:"parsed_files"`
	Symbols           int    `json:"symbols"`
	Relations         int    `json:"relations"`
	PartialFailures   int    `json:"partial_failures"`
	CompletenessLevel string `json:"completeness_level"`
}

// SnapshotMeta is everything the snapshot said about its own reliability.
//
// The provider writes a header line first and a `summary` record last. The
// header carries the profile and the relation set; the counts, warnings,
// partial failures and completeness are only filled in on the summary, because
// they are not known until the walk finishes. Both are read and merged.
type SnapshotMeta struct {
	SchemaVersion   string            `json:"schema_version,omitempty"`
	ProviderVersion string            `json:"provider_version,omitempty"`
	Profile         string            `json:"profile,omitempty"`
	Commit          string            `json:"commit,omitempty"`
	LanguageTiers   map[string]string `json:"language_tiers,omitempty"`
	Warnings        []Warning         `json:"warnings,omitempty"`
	PartialFailures []PartialFailure  `json:"partial_failures,omitempty"`
	Stats           Stats             `json:"stats"`

	// SawSummary records whether the trailing summary record arrived at all.
	// A snapshot cut short has no summary, and an absent summary is not the
	// same claim as a summary reporting nothing wrong.
	SawSummary bool `json:"saw_summary"`

	// Malformed counts lines that were not valid JSON, and Dropped counts
	// records that parsed but lacked the ids they need to be usable.
	//
	// Both used to be `continue` with no counter. A snapshot whose relation
	// lines were all malformed produced an empty field, and an empty field
	// reads exactly like a clean one. That is the false-zero shape the
	// external review already found in the sweep.
	Malformed int `json:"malformed_records"`
	Dropped   int `json:"dropped_records"`
}

// CompletenessIsHealthy reports whether a completeness level is one the
// provider uses for a clean reading.
//
// The levels the installed provider emits are "ok", "degraded" and "unsafe".
// It never emits "complete", which is worth stating because assuming it did
// was a real bug in this file: treating anything other than "complete" as a
// degradation put a warning on every healthy snapshot, which is exactly the
// crying-wolf failure that makes an honesty feature worthless. It was caught
// by running the provider against the new fixture rather than by reading.
//
// "complete" is accepted anyway, so a future build that adopts the word is not
// reported as broken.
func CompletenessIsHealthy(level string) bool {
	return level == "" || level == "ok" || level == "complete"
}

// Degraded reports whether the provider itself said this snapshot is not a
// complete reading of the repository.
func (m *SnapshotMeta) Degraded() bool {
	if m == nil {
		return false
	}
	if !CompletenessIsHealthy(m.Stats.CompletenessLevel) {
		return true
	}
	return len(m.PartialFailures) > 0 || m.Malformed > 0 || m.Dropped > 0
}

// FailedFiles is the set of files the provider could not fully analyse.
func (m *SnapshotMeta) FailedFiles() map[string]bool {
	out := map[string]bool{}
	if m == nil {
		return out
	}
	for _, pf := range m.PartialFailures {
		if pf.FilePath != "" {
			out[pf.FilePath] = true
		}
	}
	for _, w := range m.Warnings {
		if w.FilePath != "" {
			out[w.FilePath] = true
		}
	}
	return out
}

// InventoryOnlyFiles is the set of files whose language the snapshot tiers as
// inventory-only, meaning file discovery and basic symbols with no relations.
// A dependent in one of those files can never be confirmed structurally,
// because no structure was extracted for it.
func (m *SnapshotMeta) InventoryOnly(language string) bool {
	if m == nil || language == "" {
		return false
	}
	return m.LanguageTiers[language] == "inventory-only"
}

// Codes lists the warning and partial-failure codes present, sorted, for a
// header that has one line to say what went wrong.
func (m *SnapshotMeta) Codes() []string {
	if m == nil {
		return nil
	}
	seen := map[string]bool{}
	for _, w := range m.Warnings {
		seen[w.Code] = true
	}
	for _, pf := range m.PartialFailures {
		seen[pf.Code] = true
	}
	out := make([]string, 0, len(seen))
	for c := range seen {
		out = append(out, c)
	}
	sort.Strings(out)
	return out
}
