package requestctx

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestValidateRequiresExplicitScopeByOperation(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		operation Operation
		ctx       ExecutionContext
		missing   []string
	}{
		"ingestion-requires-run": {
			operation: OperationIngestion,
			ctx:       baseContext(func(c *ExecutionContext) { c.RunID = "" }),
			missing:   []string{"run_id"},
		},
		"retrieval-requires-principal": {
			operation: OperationRetrieval,
			ctx:       baseContext(func(c *ExecutionContext) { c.PrincipalID = " " }),
			missing:   []string{"principal_id"},
		},
		"mcp-tool-requires-tool-name": {
			operation: OperationMCPTool,
			ctx:       baseContext(func(c *ExecutionContext) { c.ToolName = "" }),
			missing:   []string{"tool_name"},
		},
		"audit-requires-principal": {
			operation: OperationAuditWrite,
			ctx:       baseContext(func(c *ExecutionContext) { c.PrincipalID = "" }),
			missing:   []string{"principal_id"},
		},
		"usage-does-not-require-principal": {
			operation: OperationUsageWrite,
			ctx:       baseContext(func(c *ExecutionContext) { c.PrincipalID = ""; c.ToolName = ""; c.RunID = "" }),
			missing:   nil,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			err := tc.ctx.Validate(tc.operation)
			if len(tc.missing) == 0 {
				if err != nil {
					t.Fatalf("expected context to validate: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatal("expected missing field error")
			}
			var missing MissingFieldsError
			if !errors.As(err, &missing) {
				t.Fatalf("expected MissingFieldsError, got %T", err)
			}
			for _, field := range tc.missing {
				if !strings.Contains(err.Error(), field) {
					t.Fatalf("expected error to include %q, got %q", field, err.Error())
				}
			}
		})
	}
}

func TestWithExecutionContextRejectsMissingRequiredFields(t *testing.T) {
	t.Parallel()

	_, err := WithExecutionContext(context.Background(), ExecutionContext{
		RequestID: "request-001",
		TenantID:  "tenant-a",
		ProjectID: "project-a",
		CorpusID:  "engineering-docs",
	}, OperationMCPTool)
	if err == nil {
		t.Fatal("expected missing context fields to be rejected")
	}
	if !IsMissingFields(err) {
		t.Fatalf("expected missing fields error, got %T", err)
	}
	for _, expected := range []string{"principal_id", "tool_name"} {
		if !strings.Contains(err.Error(), expected) {
			t.Fatalf("expected error to include %s, got %q", expected, err.Error())
		}
	}
}

func TestExecutionContextPropagatesThroughNestedCalls(t *testing.T) {
	t.Parallel()

	parent := context.WithValue(context.Background(), localTestKey{}, "parent-value")
	ctx, err := WithExecutionContext(parent, baseContext(nil), OperationMCPTool)
	if err != nil {
		t.Fatalf("expected context to attach: %v", err)
	}

	got, err := nestedLookup(ctx)
	if err != nil {
		t.Fatalf("expected nested lookup to validate: %v", err)
	}
	if got.RequestID != "request-001" || got.TenantID != "tenant-a" || got.ToolName != "search_docs" {
		t.Fatalf("unexpected propagated context: %+v", got)
	}
	if parentValue, ok := ctx.Value(localTestKey{}).(string); !ok || parentValue != "parent-value" {
		t.Fatalf("expected unrelated parent context values to be preserved")
	}
}

func TestExecutionContextIsTrimmedBeforePropagation(t *testing.T) {
	t.Parallel()

	ctx, err := WithExecutionContext(context.Background(), baseContext(func(c *ExecutionContext) {
		c.RequestID = " request-001 "
		c.ToolName = " search_docs "
	}), OperationMCPTool)
	if err != nil {
		t.Fatalf("expected context to attach: %v", err)
	}

	got, ok := FromContext(ctx)
	if !ok {
		t.Fatal("expected execution context")
	}
	if got.RequestID != "request-001" || got.ToolName != "search_docs" {
		t.Fatalf("expected trimmed context, got %+v", got)
	}
}

func TestRequireFromContextReportsMissingContext(t *testing.T) {
	t.Parallel()

	_, err := RequireFromContext(context.Background(), OperationRetrieval)
	if err == nil {
		t.Fatal("expected missing execution context to fail")
	}
	if !IsMissingFields(err) {
		t.Fatalf("expected missing fields error, got %T", err)
	}
}

func TestAttrsIncludesOnlyPresentOptionalFields(t *testing.T) {
	t.Parallel()

	attrs := baseContext(func(c *ExecutionContext) {
		c.RunID = ""
	}).Attrs()

	var sawTool, sawRun bool
	for _, attr := range attrs {
		switch attr.Key {
		case "tool_name":
			sawTool = true
		case "run_id":
			sawRun = true
		}
	}
	if !sawTool {
		t.Fatal("expected tool_name attr")
	}
	if sawRun {
		t.Fatal("did not expect empty run_id attr")
	}
}

type localTestKey struct{}

func nestedLookup(ctx context.Context) (ExecutionContext, error) {
	return RequireFromContext(ctx, OperationMCPTool)
}

func baseContext(mutators ...func(*ExecutionContext)) ExecutionContext {
	exec := ExecutionContext{
		RequestID:   "request-001",
		PrincipalID: "principal-001",
		TenantID:    "tenant-a",
		ProjectID:   "project-a",
		CorpusID:    "engineering-docs",
		ToolName:    "search_docs",
		RunID:       "run-001",
	}
	for _, mutate := range mutators {
		if mutate != nil {
			mutate(&exec)
		}
	}
	return exec
}
