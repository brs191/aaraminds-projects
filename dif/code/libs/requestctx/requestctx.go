// Package requestctx carries DIF request and execution scope through service
// calls without global mutable state.
package requestctx

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// Operation identifies the DIF operation boundary being validated.
type Operation string

const (
	OperationIngestion  Operation = "ingestion"
	OperationRetrieval  Operation = "retrieval"
	OperationMCPTool    Operation = "mcp_tool"
	OperationAuditWrite Operation = "audit_write"
	OperationUsageWrite Operation = "usage_write"
)

// ExecutionContext is the explicit scope attached to a request or background
// execution path.
type ExecutionContext struct {
	RequestID   string
	PrincipalID string
	TenantID    string
	ProjectID   string
	CorpusID    string
	ToolName    string
	RunID       string
}

// MissingFieldsError reports all fields required for an operation that were
// absent or blank.
type MissingFieldsError struct {
	Operation Operation
	Fields    []string
}

// Error returns a stable structured validation message.
func (e MissingFieldsError) Error() string {
	if len(e.Fields) == 0 {
		return fmt.Sprintf("missing required execution context for %s", e.Operation)
	}
	return fmt.Sprintf("missing required execution context for %s: %s", e.Operation, strings.Join(e.Fields, ", "))
}

// IsMissingFields reports whether err is a MissingFieldsError.
func IsMissingFields(err error) bool {
	var missing MissingFieldsError
	return errors.As(err, &missing)
}

// Validate checks the context fields required by an operation. It never fills
// scope from defaults because tenant, project, and corpus boundaries must stay
// explicit at every operation boundary.
func (c ExecutionContext) Validate(operation Operation) error {
	required, ok := requiredFields(operation)
	if !ok {
		return fmt.Errorf("unsupported execution context operation %q", operation)
	}

	values := c.fields()
	var missing []string
	for _, field := range required {
		if strings.TrimSpace(values[field]) == "" {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		return MissingFieldsError{Operation: operation, Fields: missing}
	}
	return nil
}

// WithExecutionContext validates and attaches execution context to ctx.
func WithExecutionContext(ctx context.Context, exec ExecutionContext, operation Operation) (context.Context, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := exec.Validate(operation); err != nil {
		return nil, err
	}
	return context.WithValue(ctx, contextKey{}, exec.trimmed()), nil
}

// FromContext extracts execution context from ctx.
func FromContext(ctx context.Context) (ExecutionContext, bool) {
	if ctx == nil {
		return ExecutionContext{}, false
	}
	exec, ok := ctx.Value(contextKey{}).(ExecutionContext)
	return exec, ok
}

// RequireFromContext extracts execution context and validates it for operation.
func RequireFromContext(ctx context.Context, operation Operation) (ExecutionContext, error) {
	if _, ok := requiredFields(operation); !ok {
		return ExecutionContext{}, fmt.Errorf("unsupported execution context operation %q", operation)
	}
	exec, ok := FromContext(ctx)
	if !ok {
		return ExecutionContext{}, MissingFieldsError{Operation: operation, Fields: requiredFieldsOrEmpty(operation)}
	}
	if err := exec.Validate(operation); err != nil {
		return ExecutionContext{}, err
	}
	return exec, nil
}

// Attrs returns log-safe structured attributes for operational metadata.
func (c ExecutionContext) Attrs() []slog.Attr {
	trimmed := c.trimmed()
	attrs := []slog.Attr{
		slog.String("request_id", trimmed.RequestID),
		slog.String("principal_id", trimmed.PrincipalID),
		slog.String("tenant_id", trimmed.TenantID),
		slog.String("project_id", trimmed.ProjectID),
		slog.String("corpus_id", trimmed.CorpusID),
	}
	if trimmed.ToolName != "" {
		attrs = append(attrs, slog.String("tool_name", trimmed.ToolName))
	}
	if trimmed.RunID != "" {
		attrs = append(attrs, slog.String("run_id", trimmed.RunID))
	}
	return attrs
}

type contextKey struct{}

func requiredFields(operation Operation) ([]string, bool) {
	switch operation {
	case OperationIngestion:
		return []string{"request_id", "principal_id", "tenant_id", "project_id", "corpus_id", "run_id"}, true
	case OperationRetrieval, OperationAuditWrite:
		return []string{"request_id", "principal_id", "tenant_id", "project_id", "corpus_id"}, true
	case OperationMCPTool:
		return []string{"request_id", "principal_id", "tenant_id", "project_id", "corpus_id", "tool_name"}, true
	case OperationUsageWrite:
		return []string{"request_id", "tenant_id", "project_id", "corpus_id"}, true
	default:
		return nil, false
	}
}

func requiredFieldsOrEmpty(operation Operation) []string {
	fields, ok := requiredFields(operation)
	if !ok {
		return nil
	}
	return fields
}

func (c ExecutionContext) fields() map[string]string {
	return map[string]string{
		"request_id":   c.RequestID,
		"principal_id": c.PrincipalID,
		"tenant_id":    c.TenantID,
		"project_id":   c.ProjectID,
		"corpus_id":    c.CorpusID,
		"tool_name":    c.ToolName,
		"run_id":       c.RunID,
	}
}

func (c ExecutionContext) trimmed() ExecutionContext {
	return ExecutionContext{
		RequestID:   strings.TrimSpace(c.RequestID),
		PrincipalID: strings.TrimSpace(c.PrincipalID),
		TenantID:    strings.TrimSpace(c.TenantID),
		ProjectID:   strings.TrimSpace(c.ProjectID),
		CorpusID:    strings.TrimSpace(c.CorpusID),
		ToolName:    strings.TrimSpace(c.ToolName),
		RunID:       strings.TrimSpace(c.RunID),
	}
}
