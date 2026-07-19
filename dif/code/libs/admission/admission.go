// Package admission enforces DIF v1 uniformly readable corpus admission.
package admission

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/aaraminds/dif/libs/requestctx"
)

const (
	StatusOK                Status = "ok"
	StatusCorpusNotAdmitted Status = "corpus_not_admitted"
	StatusSourceNotAdmitted Status = "source_not_admitted"

	AdmissionPending  AdmissionStatus = "pending"
	AdmissionAdmitted AdmissionStatus = "admitted"
	AdmissionRejected AdmissionStatus = "rejected"
	AdmissionArchived AdmissionStatus = "archived"

	ReadabilityUniform ReadabilityModel = "uniform_readable"

	AuditOutcomeDenied = "denied"
)

// Status is the explicit fail-closed result returned by admission checks.
type Status string

// AdmissionStatus mirrors dif_meta.corpora/sources admission_status.
type AdmissionStatus string

// ReadabilityModel mirrors dif_meta.corpora readability_model.
type ReadabilityModel string

// Corpus is the v1 corpus admission record.
type Corpus struct {
	CorpusID          string
	ProjectID         string
	DisplayName       string
	AdmissionStatus   AdmissionStatus
	ReadabilityModel  ReadabilityModel
	AdmissionEvidence map[string]any
}

// Source is the v1 source admission record inside a corpus.
type Source struct {
	SourceID        string
	CorpusID        string
	SourceType      string
	SourceURI       string
	ScopePath       string
	AdmissionStatus AdmissionStatus
}

// Decision is the result of an admission check.
type Decision struct {
	Status        Status
	Allowed       bool
	CorpusID      string
	ProjectID     string
	SourceID      string
	AuditIntent   *AuditIntent
	FailureReason string
}

// AuditIntent captures the fields future audit writes need for denied access.
type AuditIntent struct {
	PrincipalID    string
	TenantID       string
	ProjectID      string
	CorpusID       string
	ToolName       string
	ParametersHash string
	Outcome        string
	ErrorClass     string
}

// Catalog is an in-memory admission catalog. A later persistence prompt can
// back the same semantics with dif_meta queries.
type Catalog struct {
	Corpora map[string]Corpus
	Sources map[string]Source
}

// LoadGoldenManifest loads corpus/source admission records from the evaluation
// golden manifest. It is intended for tests and scaffold validation.
func LoadGoldenManifest(path string) (Catalog, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Catalog{}, fmt.Errorf("read admission manifest %q: %w", path, err)
	}

	var manifest struct {
		ProjectID string `json:"project_id"`
		Corpora   []struct {
			CorpusID          string         `json:"corpus_id"`
			DisplayName       string         `json:"display_name"`
			AdmissionStatus   string         `json:"admission_status"`
			ReadabilityModel  string         `json:"readability_model"`
			AdmissionEvidence map[string]any `json:"admission_evidence"`
		} `json:"corpora"`
		Sources []struct {
			SourceID        string `json:"source_id"`
			CorpusID        string `json:"corpus_id"`
			SourceType      string `json:"source_type"`
			SourceURI       string `json:"source_uri"`
			ScopePath       string `json:"scope_path"`
			AdmissionStatus string `json:"admission_status"`
		} `json:"sources"`
	}
	if err := json.Unmarshal(content, &manifest); err != nil {
		return Catalog{}, fmt.Errorf("parse admission manifest %q: %w", path, err)
	}

	catalog := Catalog{
		Corpora: make(map[string]Corpus, len(manifest.Corpora)),
		Sources: make(map[string]Source, len(manifest.Sources)),
	}
	for _, corpus := range manifest.Corpora {
		record := Corpus{
			CorpusID:          strings.TrimSpace(corpus.CorpusID),
			ProjectID:         strings.TrimSpace(manifest.ProjectID),
			DisplayName:       strings.TrimSpace(corpus.DisplayName),
			AdmissionStatus:   AdmissionStatus(strings.TrimSpace(corpus.AdmissionStatus)),
			ReadabilityModel:  ReadabilityModel(strings.TrimSpace(corpus.ReadabilityModel)),
			AdmissionEvidence: corpus.AdmissionEvidence,
		}
		if err := record.Validate(); err != nil {
			return Catalog{}, err
		}
		catalog.Corpora[record.CorpusID] = record
	}
	for _, source := range manifest.Sources {
		record := Source{
			SourceID:        strings.TrimSpace(source.SourceID),
			CorpusID:        strings.TrimSpace(source.CorpusID),
			SourceType:      strings.TrimSpace(source.SourceType),
			SourceURI:       strings.TrimSpace(source.SourceURI),
			ScopePath:       strings.TrimSpace(source.ScopePath),
			AdmissionStatus: AdmissionStatus(strings.TrimSpace(source.AdmissionStatus)),
		}
		if err := record.Validate(); err != nil {
			return Catalog{}, err
		}
		catalog.Sources[record.SourceID] = record
	}
	return catalog, nil
}

// CheckCorpus enforces v1 corpus admission for the corpus in execution context.
func (c Catalog) CheckCorpus(ctx context.Context, toolName, parametersHash string) Decision {
	exec, err := requestctx.RequireFromContext(ctx, requestctx.OperationRetrieval)
	if err != nil {
		return deniedDecision(requestctx.ExecutionContext{}, toolName, parametersHash, "missing_execution_context", err.Error())
	}
	corpusID := strings.TrimSpace(exec.CorpusID)
	corpus, ok := c.Corpora[corpusID]
	if !ok {
		return deniedDecision(exec, toolName, parametersHash, string(StatusCorpusNotAdmitted), "corpus not found")
	}
	if err := corpus.Admit(); err != nil {
		return deniedDecision(exec, toolName, parametersHash, string(StatusCorpusNotAdmitted), err.Error())
	}
	if corpus.ProjectID != "" && corpus.ProjectID != exec.ProjectID {
		return deniedDecision(exec, toolName, parametersHash, string(StatusCorpusNotAdmitted), "corpus project does not match request project")
	}
	return Decision{
		Status:    StatusOK,
		Allowed:   true,
		CorpusID:  corpus.CorpusID,
		ProjectID: corpus.ProjectID,
	}
}

// CheckSource enforces corpus admission and source-level admission.
func (c Catalog) CheckSource(ctx context.Context, sourceID, toolName, parametersHash string) Decision {
	exec, err := requestctx.RequireFromContext(ctx, requestctx.OperationRetrieval)
	if err != nil {
		return deniedDecision(requestctx.ExecutionContext{}, toolName, parametersHash, "missing_execution_context", err.Error())
	}

	corpusDecision := c.CheckCorpus(ctx, toolName, parametersHash)
	if !corpusDecision.Allowed {
		return corpusDecision
	}

	source, ok := c.Sources[strings.TrimSpace(sourceID)]
	if !ok {
		return sourceDenied(corpusDecision, exec, toolName, parametersHash, "", "source not found")
	}
	if source.CorpusID != corpusDecision.CorpusID {
		return sourceDenied(corpusDecision, exec, toolName, parametersHash, source.SourceID, "source corpus does not match request corpus")
	}
	if source.AdmissionStatus != AdmissionAdmitted {
		return sourceDenied(corpusDecision, exec, toolName, parametersHash, source.SourceID, fmt.Sprintf("source admission_status is %q", source.AdmissionStatus))
	}
	corpusDecision.SourceID = source.SourceID
	return corpusDecision
}

// Validate verifies required corpus fields and allowed enum values.
func (c Corpus) Validate() error {
	var errs []error
	if strings.TrimSpace(c.CorpusID) == "" {
		errs = append(errs, errors.New("corpus_id is required"))
	}
	if strings.TrimSpace(c.ProjectID) == "" {
		errs = append(errs, errors.New("project_id is required"))
	}
	if strings.TrimSpace(c.DisplayName) == "" {
		errs = append(errs, errors.New("display_name is required"))
	}
	if !validAdmissionStatus(c.AdmissionStatus) {
		errs = append(errs, fmt.Errorf("invalid corpus admission_status %q", c.AdmissionStatus))
	}
	if c.ReadabilityModel != ReadabilityUniform {
		errs = append(errs, fmt.Errorf("invalid corpus readability_model %q: v1 requires %q", c.ReadabilityModel, ReadabilityUniform))
	}
	return errors.Join(errs...)
}

// Admit returns nil only when the corpus is admitted and uniformly readable.
func (c Corpus) Admit() error {
	if err := c.Validate(); err != nil {
		return err
	}
	if c.AdmissionStatus != AdmissionAdmitted {
		return fmt.Errorf("corpus admission_status is %q", c.AdmissionStatus)
	}
	return nil
}

// Validate verifies required source fields and allowed enum values.
func (s Source) Validate() error {
	var errs []error
	if strings.TrimSpace(s.SourceID) == "" {
		errs = append(errs, errors.New("source_id is required"))
	}
	if strings.TrimSpace(s.CorpusID) == "" {
		errs = append(errs, errors.New("corpus_id is required"))
	}
	if strings.TrimSpace(s.SourceType) == "" {
		errs = append(errs, errors.New("source_type is required"))
	} else if !validSourceType(s.SourceType) {
		errs = append(errs, fmt.Errorf("invalid source_type %q", s.SourceType))
	}
	if strings.TrimSpace(s.SourceURI) == "" {
		errs = append(errs, errors.New("source_uri is required"))
	}
	if !validAdmissionStatus(s.AdmissionStatus) {
		errs = append(errs, fmt.Errorf("invalid source admission_status %q", s.AdmissionStatus))
	}
	return errors.Join(errs...)
}

func deniedDecision(exec requestctx.ExecutionContext, toolName, parametersHash, errorClass, reason string) Decision {
	corpusID := strings.TrimSpace(exec.CorpusID)
	projectID := strings.TrimSpace(exec.ProjectID)
	return Decision{
		Status:        StatusCorpusNotAdmitted,
		Allowed:       false,
		CorpusID:      corpusID,
		ProjectID:     projectID,
		FailureReason: reason,
		AuditIntent: &AuditIntent{
			PrincipalID:    strings.TrimSpace(exec.PrincipalID),
			TenantID:       strings.TrimSpace(exec.TenantID),
			ProjectID:      projectID,
			CorpusID:       corpusID,
			ToolName:       firstNonEmpty(toolName, exec.ToolName),
			ParametersHash: strings.TrimSpace(parametersHash),
			Outcome:        AuditOutcomeDenied,
			ErrorClass:     errorClass,
		},
	}
}

func sourceDenied(base Decision, exec requestctx.ExecutionContext, toolName, parametersHash, sourceID, reason string) Decision {
	base.Status = StatusSourceNotAdmitted
	base.Allowed = false
	base.SourceID = sourceID
	base.FailureReason = reason
	base.AuditIntent = &AuditIntent{
		PrincipalID:    strings.TrimSpace(exec.PrincipalID),
		TenantID:       strings.TrimSpace(exec.TenantID),
		ProjectID:      base.ProjectID,
		CorpusID:       base.CorpusID,
		ToolName:       firstNonEmpty(toolName, exec.ToolName),
		ParametersHash: strings.TrimSpace(parametersHash),
		Outcome:        AuditOutcomeDenied,
		ErrorClass:     string(StatusSourceNotAdmitted),
	}
	return base
}

func validAdmissionStatus(status AdmissionStatus) bool {
	switch status {
	case AdmissionPending, AdmissionAdmitted, AdmissionRejected, AdmissionArchived:
		return true
	default:
		return false
	}
}

func validSourceType(sourceType string) bool {
	switch sourceType {
	case "local_tree", "git", "sharepoint", "onedrive":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
