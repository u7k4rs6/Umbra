package graph

import (
	"context"
	"encoding/json"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/u7k4rs6/Umbra/umbra/internal/runner"
)

// Impact is what `graph impact` knows about one changed symbol.
type Impact struct {
	Symbol string
	// CallSites maps a caller symbol id to the exact line of its call into
	// the source. The snapshot's own evidence spans the calling function, so
	// this is the only place the precise line comes from.
	CallSites map[string]int
	// CallSitesByName is the same keyed by "file:name", for callers the
	// snapshot did not give an id for.
	CallSitesByName map[string]int
	// CoChange lists files that historically change with the source's file.
	CoChange []string
	// Counts are Graph's own totals, kept as a cross-check on the BFS.
	//
	// They are heuristic, and every renderer says so where it prints them: the
	// provider derives a dependent count from the same mix of exact, name-only
	// and pattern resolutions the relations carry, so it is an estimate rather
	// than a compiler's answer.
	DirectCallers     int
	TransitiveCallers int

	// What `graph impact --format json` says about its own completeness. It
	// prints this block at the top of every answer and Umbra used to read past
	// it to reach the caller list.
	Warnings        []Warning
	PartialFailures []PartialFailure
	// CompletenessLevel is the snapshot-wide level, and ScopeLevel is the
	// level for this query alone. They differ, and the difference is the
	// useful part: a failure in another language cannot affect an answer about
	// a Python symbol, and completeness_scope is where the provider says so.
	CompletenessLevel string
	ScopeLevel        string
	ScopeLanguage     string
	// InScopeWarnings are the warning codes the provider says can affect this
	// answer. OutOfScopeFailures counts the diagnostics it says cannot.
	InScopeWarnings    []string
	OutOfScopeFailures int
	// InScopeSevere is the subset of InScopeWarnings the provider did NOT mark
	// `info`.
	//
	// The distinction matters more than it looks. On this repository the only
	// in-scope warning is W_DATA_FLOW_EVIDENCE_UNMERGED at severity `info`,
	// whose own text says "the relation, its confidence and its reason are
	// unaffected": it describes a merged evidence array on some DATA_FLOWS
	// edges and nothing else. Treating it as a reason to doubt every exact
	// CALLS edge in the field turned all thirteen dependents of the demo
	// commit into "needs verification", which is the crying-wolf failure that
	// makes an honesty feature worthless. An info warning is reported in the
	// header and does not promote a single node.
	InScopeSevere []string
}

// InScopeFailures is how many of this answer's partial failures the provider
// did NOT scope to another language.
func (i *Impact) InScopeFailures() int {
	if i == nil {
		return 0
	}
	n := len(i.PartialFailures) - i.OutOfScopeFailures
	if n < 0 {
		return 0
	}
	return n
}

// Degraded reports whether the provider said this answer may be incomplete.
// The query scope wins when it is set, because it is the narrower claim.
func (i *Impact) Degraded() bool {
	if i == nil {
		return false
	}
	level := i.ScopeLevel
	if level == "" {
		level = i.CompletenessLevel
	}
	return !CompletenessIsHealthy(level)
}

type impactJSON struct {
	Focus struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		FilePath string `json:"file_path"`
	} `json:"focus"`
	Callers struct {
		Total      int `json:"total"`
		Direct     int `json:"direct"`
		Transitive int `json:"transitive"`
		Entries    []struct {
			Endpoint struct {
				ID       string `json:"id"`
				Name     string `json:"name"`
				FilePath string `json:"file_path"`
			} `json:"endpoint"`
			Relation string `json:"relation"`
			Depth    int    `json:"depth"`
			CallSite struct {
				FilePath        string `json:"file_path"`
				Line            int    `json:"line"`
				AdditionalSites int    `json:"additional_sites"`
			} `json:"call_site"`
		} `json:"entries"`
	} `json:"callers"`
	CoChange struct {
		Entries []struct {
			Path string `json:"path"`
			File string `json:"file_path"`
		} `json:"entries"`
	} `json:"co_change"`

	Warnings        []Warning        `json:"warnings"`
	PartialFailures []PartialFailure `json:"partial_failures"`
	Stats           Stats            `json:"stats"`
	Completeness    struct {
		Relations map[string]int `json:"relations"`
	} `json:"completeness"`
	CompletenessScope struct {
		Language             string    `json:"language"`
		LanguageFiles        int       `json:"language_files"`
		OtherLanguageFailure int       `json:"other_language_failures"`
		InScopeWarnings      []Warning `json:"in_scope_warnings"`
		Level                string    `json:"level"`
	} `json:"completeness_scope"`
}

// LoadImpact runs `graph impact --format json` for one symbol.
//
// The JSON form is used because the Step 0 probe showed it carries
// call_site.line directly. The text parser below is kept as a fallback for a
// Graph that does not offer JSON.
func LoadImpact(ctx context.Context, run runner.Runner, repo, symbol, file string, line int) (*Impact, error) {
	args := []string{"graph", "impact", "--symbol", symbol, "--repo", repo, "--format", "json"}
	if file != "" {
		args = append(args, "--file", file)
	}
	if line > 0 {
		args = append(args, "--line", strconv.Itoa(line))
	}
	stdout, _, exit, err := run.Run(ctx, "entire", args, nil)
	if err != nil {
		return nil, err
	}
	if exit != 0 {
		return &Impact{Symbol: symbol, CallSites: map[string]int{}, CallSitesByName: map[string]int{}}, nil
	}
	if imp, err := ParseImpactJSON(stdout); err == nil {
		imp.Symbol = symbol
		return imp, nil
	}
	imp := ParseImpactText(string(stdout))
	imp.Symbol = symbol
	return imp, nil
}

// ParseImpactJSON reads `graph impact --format json`.
func ParseImpactJSON(blob []byte) (*Impact, error) {
	text := strings.TrimSpace(string(blob))
	i := strings.Index(text, "{")
	if i < 0 {
		return nil, errNoJSON
	}
	var j impactJSON
	if err := json.Unmarshal([]byte(text[i:]), &j); err != nil {
		return nil, err
	}
	imp := &Impact{
		CallSites:          map[string]int{},
		CallSitesByName:    map[string]int{},
		DirectCallers:      j.Callers.Direct,
		TransitiveCallers:  j.Callers.Transitive,
		Warnings:           j.Warnings,
		PartialFailures:    j.PartialFailures,
		CompletenessLevel:  j.Stats.CompletenessLevel,
		ScopeLevel:         j.CompletenessScope.Level,
		ScopeLanguage:      j.CompletenessScope.Language,
		OutOfScopeFailures: j.CompletenessScope.OtherLanguageFailure,
	}
	for _, w := range j.CompletenessScope.InScopeWarnings {
		if !containsString(imp.InScopeWarnings, w.Code) {
			imp.InScopeWarnings = append(imp.InScopeWarnings, w.Code)
		}
		if w.Severity != "info" && !containsString(imp.InScopeSevere, w.Code) {
			imp.InScopeSevere = append(imp.InScopeSevere, w.Code)
		}
	}
	sort.Strings(imp.InScopeWarnings)
	sort.Strings(imp.InScopeSevere)
	for _, e := range j.Callers.Entries {
		if e.CallSite.Line <= 0 {
			continue
		}
		if e.Endpoint.ID != "" {
			imp.CallSites[e.Endpoint.ID] = e.CallSite.Line
		}
		key := e.CallSite.FilePath + ":" + e.Endpoint.Name
		imp.CallSitesByName[key] = e.CallSite.Line
	}
	seen := map[string]bool{}
	for _, c := range j.CoChange.Entries {
		p := c.Path
		if p == "" {
			p = c.File
		}
		if p != "" && !seen[p] {
			seen[p] = true
			imp.CoChange = append(imp.CoChange, p)
		}
	}
	sort.Strings(imp.CoChange)
	return imp, nil
}

var errNoJSON = jsonError("graph impact produced no JSON")

type jsonError string

func (e jsonError) Error() string { return string(e) }

// The documented text form of a caller line:
//
//   - handle_order (umbra/fixtures/app/app/api.py:16, def :14)
//
// where 16 is the call site and 14 the definition.
var (
	callerLineRE = regexp.MustCompile(`^-\s+(\S+)\s+\(([^:]+):(\d+),\s*def\s*:(\d+)\)`)
	coChangeRE   = regexp.MustCompile(`^-\s+(\S+)\s+\[files changed together`)
	sectionRE    = regexp.MustCompile(`^([A-Za-z].*?)\s*\(`)
	// The text form opens with a line such as
	//   Completeness: degraded for Go (0 of 492 Go files failed to parse; ...)
	// and then lists the codes as "- warning W_X" and "- partial E_Y".
	completenessRE = regexp.MustCompile(`^Completeness:\s+(\S+)\s+for\s+(.+?)\s*\(`)
	diagnosticRE   = regexp.MustCompile(`^-\s+(warning|partial)\s+([A-Z_]+)`)
)

// ParseImpactText reads the human form of `graph impact`. It is the fallback
// for a Graph without --format json.
func ParseImpactText(text string) *Impact {
	imp := &Impact{CallSites: map[string]int{}, CallSitesByName: map[string]int{}}
	section := ""
	for _, raw := range strings.Split(text, "\n") {
		line := strings.TrimRight(raw, " \t\r")
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "-") {
			if m := completenessRE.FindStringSubmatch(line); m != nil {
				imp.ScopeLevel = m[1]
				imp.ScopeLanguage = m[2]
			}
			if m := sectionRE.FindStringSubmatch(line); m != nil {
				section = strings.ToLower(m[1])
			}
			continue
		}
		if m := diagnosticRE.FindStringSubmatch(line); m != nil {
			if m[1] == "warning" {
				imp.Warnings = append(imp.Warnings, Warning{Code: m[2]})
				if !containsString(imp.InScopeWarnings, m[2]) {
					imp.InScopeWarnings = append(imp.InScopeWarnings, m[2])
				}
			} else {
				imp.PartialFailures = append(imp.PartialFailures, PartialFailure{Code: m[2]})
			}
			continue
		}
		switch {
		case strings.HasPrefix(section, "callers"):
			if m := callerLineRE.FindStringSubmatch(line); m != nil {
				callSite, _ := strconv.Atoi(m[3])
				imp.CallSitesByName[m[2]+":"+m[1]] = callSite
				if strings.Contains(line, "[via ") {
					imp.TransitiveCallers++
				} else {
					imp.DirectCallers++
				}
			}
		case strings.HasPrefix(section, "co-change"):
			if m := coChangeRE.FindStringSubmatch(line); m != nil {
				imp.CoChange = append(imp.CoChange, m[1])
			}
		}
	}
	sort.Strings(imp.CoChange)
	return imp
}

// CallSiteFor returns the exact call line for a caller, preferring the id and
// falling back to the file and name pair.
func (i *Impact) CallSiteFor(id, file, name string) int {
	if i == nil {
		return 0
	}
	if line, ok := i.CallSites[id]; ok {
		return line
	}
	if line, ok := i.CallSitesByName[file+":"+name]; ok {
		return line
	}
	return 0
}
