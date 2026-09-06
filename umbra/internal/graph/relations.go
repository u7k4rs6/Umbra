// Package graph reads the code graph through documented entire-graph output
// only: the NDJSON snapshot, graph commit, graph impact and graph verify. It
// never imports entire-graph packages and never reads the checkpoints branch.
package graph

import (
	"encoding/json"
	"sort"
)

// Family groups the relation names an installed Graph reports into the
// families Umbra traverses. The names are read from `graph capabilities` at
// startup rather than hard coded, so a Graph that renames or adds a relation
// does not silently change what the map means.
type Family string

const (
	FamilyCalls     Family = "calls"
	FamilyTypeUse   Family = "type use"
	FamilyDataFlow  Family = "data flow"
	FamilyCoChange  Family = "co-change"
	FamilyImports   Family = "imports"
	FamilyTests     Family = "tests"
	FamilyStructure Family = "structure"
)

// familyOf is the mapping table. Every relation name the Step 0 probe saw in
// `capabilities --json` appears here or is deliberately unmapped.
var familyOf = map[string]Family{
	"CALLS":       FamilyCalls,
	"ASYNC_CALLS": FamilyCalls,

	"USES_TYPE":    FamilyTypeUse,
	"PARAM_TYPE":   FamilyTypeUse,
	"RETURNS_TYPE": FamilyTypeUse,
	"EXTENDS":      FamilyTypeUse,
	"IMPLEMENTS":   FamilyTypeUse,
	"INHERITS":     FamilyTypeUse,
	"OVERRIDES":    FamilyTypeUse,

	"DATA_FLOWS":   FamilyDataFlow,
	"READS_FIELD":  FamilyDataFlow,
	"WRITES_FIELD": FamilyDataFlow,
	"ACCESSES":     FamilyDataFlow,

	"FILE_CHANGES_WITH": FamilyCoChange,

	"IMPORTS": FamilyImports,
	"TESTS":   FamilyTests,

	"DEFINES":  FamilyStructure,
	"CONTAINS": FamilyStructure,
}

// Capabilities is the part of `graph capabilities --json` Umbra reads.
type Capabilities struct {
	SchemaVersion          string              `json:"schema_version"`
	Provider               string              `json:"provider"`
	ProviderVersion        string              `json:"provider_version"`
	SupportedRelationTypes []string            `json:"supported_relation_types"`
	RelationByLanguage     map[string][]string `json:"relation_support_by_language"`
	HeuristicRelations     []string            `json:"heuristic_relation_types"`
	NetworkFeatures        map[string]bool     `json:"features_requiring_network_access"`
}

// ParseCapabilities reads the JSON `graph capabilities --json` prints.
func ParseCapabilities(blob []byte) (*Capabilities, error) {
	var c Capabilities
	if err := json.Unmarshal(blob, &c); err != nil {
		return nil, err
	}
	return &c, nil
}

// RelationMap is the resolved mapping for one installed Graph.
type RelationMap struct {
	byName  map[string]Family
	Used    []string
	Ignored []string
}

// NewRelationMap resolves the installed relation names into families.
//
// A relation Umbra does not traverse is not an error: it is listed as ignored
// and the report header names it, so a reader can see what the map does not
// account for.
func NewRelationMap(c *Capabilities) *RelationMap {
	m := &RelationMap{byName: map[string]Family{}}
	names := []string(nil)
	if c != nil {
		names = c.SupportedRelationTypes
	}
	if len(names) == 0 {
		// No capabilities available. Fall back to the documented default set
		// so the map still works, and say so through Ignored being empty.
		for name := range familyOf {
			names = append(names, name)
		}
	}
	for _, name := range names {
		if f, ok := familyOf[name]; ok {
			m.byName[name] = f
			if f != FamilyStructure {
				m.Used = append(m.Used, name)
			}
		} else {
			m.Ignored = append(m.Ignored, name)
		}
	}
	sort.Strings(m.Used)
	sort.Strings(m.Ignored)
	return m
}

// Family returns the family for a relation name, and whether it is one Umbra
// traverses at all.
func (m *RelationMap) Family(name string) (Family, bool) {
	f, ok := m.byName[name]
	return f, ok
}

// Traverses reports whether a relation contributes dependents to the field.
func (m *RelationMap) Traverses(name string) bool {
	f, ok := m.byName[name]
	if !ok {
		return false
	}
	switch f {
	case FamilyCalls, FamilyTypeUse, FamilyDataFlow:
		return true
	}
	return false
}

// Offline reports whether the installed Graph declares that nothing it does
// needs the network. SECURITY_AND_ACCESS.md claims the default path is
// offline; this is how the claim is checked rather than assumed.
func (c *Capabilities) Offline() bool {
	if c == nil {
		return false
	}
	for _, needs := range c.NetworkFeatures {
		if needs {
			return false
		}
	}
	return true
}
