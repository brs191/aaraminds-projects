// Package rifcompat implements DIF's RIF compatibility status contract.
package rifcompat

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	StatusNotDeployed  Status = "rif_not_deployed"
	StatusIncompatible Status = "rif_incompatible"
	StatusShadowEmpty  Status = "rif_shadow_empty"
	StatusCompatible   Status = "rif_compatible"

	ShadowEmpty     ShadowStatus = "rif_shadow_empty"
	ShadowPopulated ShadowStatus = "rif_shadow_populated"

	LookupResolved   LookupStatus = "resolved"
	LookupUnresolved LookupStatus = "unresolved"

	ConfidenceExact    Confidence = "exact"
	ConfidenceInferred Confidence = "inferred"
)

// RequiredFields are the minimum fields DIF requires before enabling
// cross-graph behavior.
var RequiredFields = []string{"node_id", "repo_id", "kind", "qualified_name", "source_ref", "origin", "confidence"}

var kindRank = map[string]int{
	"METHOD":    0,
	"CLASS":     1,
	"INTERFACE": 2,
	"RECORD":    3,
	"ENUM":      4,
	"FILE":      5,
}

// Status is the top-level RIF compatibility state.
type Status string

// ShadowStatus describes the relational shadow readiness.
type ShadowStatus string

// LookupStatus is a code entity lookup result status.
type LookupStatus string

// Confidence is exact or inferred match confidence.
type Confidence string

// Surface is the detected RIF compatibility surface.
type Surface struct {
	Schemas         []string
	ShadowAvailable bool
	ShadowEntities  []Entity
	AGEAvailable    bool
	AGEEntities     []Entity
	MissingFields   []string
}

// Entity is the minimum code-entity shape exposed to DIF.
type Entity struct {
	EntityAlias   string     `json:"entity_alias,omitempty"`
	NodeID        string     `json:"node_id"`
	RepoID        string     `json:"repo_id"`
	Kind          string     `json:"kind"`
	QualifiedName string     `json:"qualified_name"`
	SimpleName    string     `json:"simple_name,omitempty"`
	SourceRef     string     `json:"source_ref"`
	Origin        string     `json:"origin"`
	Confidence    Confidence `json:"confidence"`
}

// Report is the persisted/runtime compatibility report.
type Report struct {
	Status              Status       `json:"rif_status"`
	ShadowStatus        ShadowStatus `json:"shadow_status,omitempty"`
	Matches             []Entity     `json:"matches"`
	MissingCapabilities []string     `json:"missing_capabilities"`
	Caveats             []string     `json:"caveats"`
}

// LookupMode selects how a code entity should be resolved.
type LookupMode string

const (
	LookupQualifiedName LookupMode = "qualified_name"
	LookupSourcePath    LookupMode = "source_path"
	LookupSimpleName    LookupMode = "simple_name"
)

// LookupResult is a non-success-shaped resolver response.
type LookupResult struct {
	Status     LookupStatus `json:"status"`
	Confidence Confidence   `json:"confidence,omitempty"`
	Matches    []Entity     `json:"matches"`
	Caveats    []string     `json:"caveats"`
}

// StatusStore persists RIF compatibility status snapshots.
type StatusStore interface {
	WriteStatus(context.Context, string, Report) error
}

// SQLStatusStore writes to dif_meta.rif_compatibility_status. It does not
// mutate RIF-owned schemas.
type SQLStatusStore struct {
	Execer       Execer
	DatabaseName string
}

// Execer is implemented by *sql.DB and *sql.Tx.
type Execer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Queryer is implemented by *sql.DB and *sql.Tx.
type Queryer interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Relation names a read-only compatibility relation.
type Relation struct {
	Schema string
	Name   string
}

// SQLInspector detects configured RIF compatibility surfaces from Postgres.
// It only reads metadata and explicitly configured relations; it never creates,
// alters, or drops RIF-owned objects.
type SQLInspector struct {
	Queryer         Queryer
	ShadowRelations []Relation
	AGERelations    []Relation
	MaxEntities     int
}

// Inspect reads the configured database surface and returns an abstract
// compatibility surface for Assess.
func (i SQLInspector) Inspect(ctx context.Context) (Surface, error) {
	if i.Queryer == nil {
		return Surface{}, errors.New("rifcompat SQL inspector requires a queryer")
	}
	schemas, err := i.schemas(ctx)
	if err != nil {
		return Surface{}, err
	}
	shadowEntities, shadowAvailable, shadowMissing, err := i.entitiesFromRelations(ctx, i.ShadowRelations)
	if err != nil {
		return Surface{}, err
	}
	ageEntities, ageAvailable, ageMissing, err := i.entitiesFromRelations(ctx, i.AGERelations)
	if err != nil {
		return Surface{}, err
	}
	return Surface{
		Schemas:         schemas,
		ShadowAvailable: shadowAvailable,
		ShadowEntities:  shadowEntities,
		AGEAvailable:    ageAvailable,
		AGEEntities:     ageEntities,
		MissingFields:   fatalMissingFields(shadowEntities, ageEntities, shadowMissing, ageMissing),
	}, nil
}

// Assess evaluates the detected RIF surface without treating empty shadows as
// success unless an AGE/API fallback provides complete entities.
func Assess(surface Surface) Report {
	schemas := set(surface.Schemas)
	if !schemas["rif"] && !schemas["rif_meta"] {
		return Report{
			Status:  StatusNotDeployed,
			Caveats: []string{"No RIF compatibility surface is available."},
		}
	}

	shadowEntities := completeEntities(surface.ShadowEntities)
	ageEntities := completeEntities(surface.AGEEntities)
	shadowMissing := missingCapabilities(shadowEntities, nil)
	ageMissing := missingCapabilities(ageEntities, nil)

	if len(shadowEntities) > 0 && len(shadowMissing) == 0 {
		return Report{
			Status:       StatusCompatible,
			ShadowStatus: ShadowPopulated,
			Matches:      sortedEntities(shadowEntities),
		}
	}
	if len(ageEntities) > 0 && len(ageMissing) == 0 {
		return Report{
			Status:       StatusCompatible,
			ShadowStatus: ShadowEmpty,
			Matches:      sortedEntities(ageEntities),
			Caveats:      []string{"RIF relational shadows are empty; AGE fallback is active."},
		}
	}
	if surface.ShadowAvailable {
		return Report{
			Status:       StatusIncompatible,
			ShadowStatus: ShadowEmpty,
			Caveats:      []string{"RIF shadows are empty and no AGE/API fallback is available."},
		}
	}
	missing := missingCapabilities(append(shadowEntities, ageEntities...), surface.MissingFields)
	if len(missing) > 0 {
		return Report{
			Status:              StatusIncompatible,
			MissingCapabilities: missing,
			Caveats:             []string{"Required RIF compatibility fields are unavailable."},
		}
	}
	return Report{
		Status:  StatusIncompatible,
		Caveats: []string{"RIF compatibility surface has no usable code entities."},
	}
}

// ResolveLookup resolves against a compatible report and returns explicit
// non-success statuses for incompatible or unresolved cases.
func ResolveLookup(report Report, mode LookupMode, query string) LookupResult {
	if report.Status != StatusCompatible {
		return LookupResult{Status: LookupStatus(report.Status), Matches: []Entity{}, Caveats: append([]string{}, report.Caveats...)}
	}
	query = strings.TrimSpace(query)
	var matches []Entity
	confidence := ConfidenceExact
	switch mode {
	case LookupQualifiedName:
		for _, entity := range report.Matches {
			if entity.QualifiedName == query {
				matches = append(matches, entity)
			}
		}
	case LookupSourcePath:
		for _, entity := range report.Matches {
			if entity.Kind == "FILE" && entity.QualifiedName == query {
				matches = append(matches, entity)
			}
		}
	case LookupSimpleName:
		confidence = ConfidenceInferred
		for _, entity := range report.Matches {
			if entity.SimpleName == query {
				matches = append(matches, entity)
			}
		}
	default:
		return LookupResult{Status: LookupUnresolved, Confidence: confidence, Matches: []Entity{}}
	}
	matches = sortedLookupMatches(matches)
	if len(matches) == 0 {
		return LookupResult{Status: LookupUnresolved, Confidence: confidence, Matches: []Entity{}}
	}
	result := LookupResult{Status: LookupResolved, Confidence: confidence, Matches: matches}
	if mode == LookupSimpleName && len(matches) > 1 {
		result.Caveats = []string{"ambiguous_simple_name"}
	}
	return result
}

// NodeID computes the shared RIF/DIF code node ID algorithm.
func NodeID(repoID, qualifiedName, kind string) string {
	return sha256Text(strings.Join([]string{repoID, qualifiedName, kind}, "\x00"))
}

// EdgeID computes the shared RIF/DIF edge ID algorithm.
func EdgeID(fromNodeID, label, toNodeID string) string {
	return sha256Text(strings.Join([]string{fromNodeID, label, toNodeID}, "\x00"))
}

// WriteStatus persists one compatibility snapshot.
func (s SQLStatusStore) WriteStatus(ctx context.Context, projectID string, report Report) error {
	if s.Execer == nil {
		return errors.New("rifcompat SQL status store requires an execer")
	}
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return errors.New("project_id is required")
	}
	if !validStatus(report.Status) {
		return fmt.Errorf("invalid RIF compatibility status %q", report.Status)
	}
	capabilities, err := json.Marshal(capabilitiesFromReport(report))
	if err != nil {
		return err
	}
	missing, err := json.Marshal(report.MissingCapabilities)
	if err != nil {
		return err
	}
	caveats, err := json.Marshal(report.Caveats)
	if err != nil {
		return err
	}
	_, err = s.Execer.ExecContext(ctx, `
INSERT INTO dif_meta.rif_compatibility_status (
    project_id, rif_status, database_name, capabilities, missing_capabilities, caveats
) VALUES ($1, $2, $3, $4::jsonb, $5::jsonb, $6::jsonb)`,
		projectID,
		string(report.Status),
		emptyToNil(s.DatabaseName),
		string(capabilities),
		string(missing),
		string(caveats),
	)
	return err
}

func completeEntities(entities []Entity) []Entity {
	out := make([]Entity, 0, len(entities))
	for _, entity := range entities {
		normalized := normalizeEntity(entity)
		out = append(out, normalized)
	}
	return out
}

func normalizeEntity(entity Entity) Entity {
	entity.EntityAlias = strings.TrimSpace(entity.EntityAlias)
	entity.NodeID = strings.TrimSpace(entity.NodeID)
	entity.RepoID = strings.TrimSpace(entity.RepoID)
	entity.Kind = strings.TrimSpace(entity.Kind)
	entity.QualifiedName = strings.TrimSpace(entity.QualifiedName)
	entity.SimpleName = strings.TrimSpace(entity.SimpleName)
	entity.SourceRef = strings.TrimSpace(entity.SourceRef)
	entity.Origin = strings.TrimSpace(entity.Origin)
	entity.Confidence = Confidence(strings.TrimSpace(string(entity.Confidence)))
	return entity
}

func missingCapabilities(entities []Entity, explicit []string) []string {
	missing := map[string]bool{}
	for _, field := range explicit {
		if trimmed := strings.TrimSpace(field); trimmed != "" {
			missing[trimmed] = true
		}
	}
	for _, entity := range entities {
		values := map[string]string{
			"node_id":        entity.NodeID,
			"repo_id":        entity.RepoID,
			"kind":           entity.Kind,
			"qualified_name": entity.QualifiedName,
			"source_ref":     entity.SourceRef,
			"origin":         entity.Origin,
			"confidence":     string(entity.Confidence),
		}
		for _, field := range RequiredFields {
			if strings.TrimSpace(values[field]) == "" {
				missing[field] = true
			}
		}
	}
	result := make([]string, 0, len(missing))
	for field := range missing {
		result = append(result, field)
	}
	sort.Strings(result)
	return result
}

func sortedEntities(entities []Entity) []Entity {
	out := append([]Entity(nil), entities...)
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].NodeID < out[j].NodeID
	})
	return out
}

func sortedLookupMatches(matches []Entity) []Entity {
	out := append([]Entity(nil), matches...)
	sort.SliceStable(out, func(i, j int) bool {
		left, right := out[i], out[j]
		if left.Confidence != right.Confidence {
			return left.Confidence == ConfidenceExact
		}
		if kindRank[left.Kind] != kindRank[right.Kind] {
			return kindRank[left.Kind] < kindRank[right.Kind]
		}
		if len(left.QualifiedName) != len(right.QualifiedName) {
			return len(left.QualifiedName) < len(right.QualifiedName)
		}
		if left.QualifiedName != right.QualifiedName {
			return left.QualifiedName < right.QualifiedName
		}
		return left.NodeID < right.NodeID
	})
	return out
}

func set(values []string) map[string]bool {
	result := map[string]bool{}
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			result[trimmed] = true
		}
	}
	return result
}

func capabilitiesFromReport(report Report) map[string]any {
	return map[string]any{
		"match_count":   len(report.Matches),
		"shadow_status": report.ShadowStatus,
	}
}

func validStatus(status Status) bool {
	switch status {
	case StatusNotDeployed, StatusIncompatible, StatusShadowEmpty, StatusCompatible:
		return true
	default:
		return false
	}
}

func emptyToNil(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return strings.TrimSpace(value)
}

func sha256Text(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func (i SQLInspector) schemas(ctx context.Context) ([]string, error) {
	rows, err := i.Queryer.QueryContext(ctx, `
SELECT schema_name
FROM information_schema.schemata
WHERE schema_name IN ('rif', 'rif_meta')`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var schemas []string
	for rows.Next() {
		var schema string
		if err := rows.Scan(&schema); err != nil {
			return nil, err
		}
		schemas = append(schemas, schema)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	sort.Strings(schemas)
	return schemas, nil
}

func (i SQLInspector) entitiesFromRelations(ctx context.Context, relations []Relation) ([]Entity, bool, []string, error) {
	var all []Entity
	var missing []string
	available := false
	for _, relation := range relations {
		fields, ok, err := i.relationFields(ctx, relation)
		if err != nil {
			return nil, false, nil, err
		}
		if !ok {
			continue
		}
		available = true
		missing = append(missing, requiredFieldGaps(fields)...)
		if len(requiredFieldGaps(fields)) > 0 {
			continue
		}
		entities, err := i.relationEntities(ctx, relation)
		if err != nil {
			return nil, false, nil, err
		}
		all = append(all, entities...)
	}
	return all, available, missing, nil
}

func (i SQLInspector) relationFields(ctx context.Context, relation Relation) (map[string]bool, bool, error) {
	if err := relation.validate(); err != nil {
		return nil, false, err
	}
	rows, err := i.Queryer.QueryContext(ctx, `
SELECT column_name
FROM information_schema.columns
WHERE table_schema = $1 AND table_name = $2`, relation.Schema, relation.Name)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	fields := map[string]bool{}
	for rows.Next() {
		var field string
		if err := rows.Scan(&field); err != nil {
			return nil, false, err
		}
		fields[field] = true
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return fields, len(fields) > 0, nil
}

func (i SQLInspector) relationEntities(ctx context.Context, relation Relation) ([]Entity, error) {
	limit := i.MaxEntities
	if limit <= 0 {
		limit = 1000
	}
	query := fmt.Sprintf(`
SELECT node_id, repo_id, kind, qualified_name, COALESCE(simple_name, ''), source_ref, origin, confidence
FROM %s.%s
ORDER BY node_id
LIMIT $1`, quoteIdent(relation.Schema), quoteIdent(relation.Name))
	rows, err := i.Queryer.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var entities []Entity
	for rows.Next() {
		var entity Entity
		var confidence string
		if err := rows.Scan(&entity.NodeID, &entity.RepoID, &entity.Kind, &entity.QualifiedName, &entity.SimpleName, &entity.SourceRef, &entity.Origin, &confidence); err != nil {
			return nil, err
		}
		entity.Confidence = Confidence(confidence)
		entities = append(entities, normalizeEntity(entity))
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entities, nil
}

func requiredFieldGaps(fields map[string]bool) []string {
	var missing []string
	for _, field := range RequiredFields {
		if !fields[field] {
			missing = append(missing, field)
		}
	}
	return missing
}

func fatalMissingFields(shadowEntities, ageEntities []Entity, shadowMissing, ageMissing []string) []string {
	if len(shadowEntities) > 0 && len(shadowMissing) == 0 {
		return nil
	}
	if len(ageEntities) > 0 && len(ageMissing) == 0 {
		return nil
	}
	return append(append([]string{}, shadowMissing...), ageMissing...)
}

func (r Relation) validate() error {
	if !validIdent(r.Schema) || !validIdent(r.Name) {
		return fmt.Errorf("invalid relation %q.%q", r.Schema, r.Name)
	}
	return nil
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

func validIdent(value string) bool {
	if value == "" {
		return false
	}
	for index, r := range value {
		if r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z' || r == '_' || index > 0 && r >= '0' && r <= '9' {
			continue
		}
		return false
	}
	return true
}
