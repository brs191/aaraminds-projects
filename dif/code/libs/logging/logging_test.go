package logging

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
	"time"
)

func TestStructuredLoggingAllowsOperationalFields(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelInfo,
		"dif.ingestion.completed",
		ID("project_id", "project-a"),
		ID("corpus_id", "engineering-docs"),
		ID("run_id", "run-001"),
		Path("source_path", "docs/service.md"),
		Hash("content_hash", "sha256:abcdef"),
		Count("block_count", 12),
		CaveatCode("caveat_code", "json_depth_capped"),
		Latency("latency_ms", 1500*time.Millisecond),
		Status("status", "complete"),
	)

	rendered := out.String()
	for _, expected := range []string{
		`"project_id":"project-a"`,
		`"corpus_id":"engineering-docs"`,
		`"run_id":"run-001"`,
		`"source_path":"docs/service.md"`,
		`"content_hash":"sha256:abcdef"`,
		`"block_count":12`,
		`"caveat_code":"json_depth_capped"`,
		`"latency_ms":1500`,
		`"status":"complete"`,
	} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered log to contain %s, got %s", expected, rendered)
		}
	}
}

func TestStructuredLoggingRedactsCredentialsTokensAndSecretLikeValues(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)
	rawPassword := "super-secret-password"
	rawBearer := "Bearer eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiJkaWYtdGVzdC1wcmluY2lwYWwifQ.signature-with-enough-length"
	rawDatabaseURL := "postgres://dif_user:super-secret-password@localhost:5432/rif_project"
	rawAPIKey := "AKIA1234567890ABCDEF"
	rawPrivateKey := "-----BEGIN PRIVATE KEY-----"

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelWarn,
		"dif.config.checked",
		ID("request_id", "request-001"),
		ID("token_id", rawBearer),
		Path("config_path", "settings.json?password="+rawPassword),
		Hash("api_key_hash", rawAPIKey),
		Status("database_url", rawDatabaseURL),
		Status("private_key_marker", rawPrivateKey),
	)

	rendered := out.String()
	for _, forbidden := range []string{
		rawPassword,
		rawBearer,
		rawAPIKey,
		rawPrivateKey,
		":super-secret-password@",
	} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("rendered log leaked secret-like value %q: %s", forbidden, rendered)
		}
	}
	if !IsSecretRedacted(rendered) {
		t.Fatalf("expected rendered log to include redaction marker, got %s", rendered)
	}
}

func TestStructuredLoggingRedactsRawDocumentTextFields(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)
	rawDocumentText := "This enterprise-only paragraph explains an unreleased acquisition workflow."

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelInfo,
		"dif.retrieval.candidate",
		ID("document_id", "doc-001"),
		slog.String("raw_document_text", rawDocumentText),
		slog.String("passage_text", rawDocumentText),
	)

	rendered := out.String()
	if strings.Contains(rendered, rawDocumentText) {
		t.Fatalf("rendered log leaked raw document text: %s", rendered)
	}
	if !strings.Contains(rendered, redactedDocumentText) {
		t.Fatalf("expected document text redaction marker, got %s", rendered)
	}
}

func TestStructuredLoggingRedactsArbitraryAnyValues(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)
	rawSecret := "super-secret-password"
	rawDocumentText := "This enterprise-only paragraph should never be logged."

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelInfo,
		"dif.any.checked",
		slog.Any("metadata", map[string]string{
			"password": rawSecret,
			"text":     rawDocumentText,
		}),
		slog.Any("raw_document_text", map[string]string{
			"body": rawDocumentText,
		}),
	)

	rendered := out.String()
	for _, forbidden := range []string{rawSecret, rawDocumentText, "password", "body"} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("rendered log leaked arbitrary structured value %q: %s", forbidden, rendered)
		}
	}
	if !IsSecretRedacted(rendered) || !strings.Contains(rendered, redactedDocumentText) {
		t.Fatalf("expected secret and document redaction markers, got %s", rendered)
	}
}

func TestStructuredLoggingRedactsNestedGroupValues(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)
	rawSecret := "super-secret-password"

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelInfo,
		"dif.group.checked",
		slog.Group(
			"context",
			slog.String("request_id", "request-001"),
			slog.String("database_url", "postgres://dif_user:"+rawSecret+"@localhost:5432/rif_project"),
			slog.Any("metadata", map[string]string{"token": rawSecret}),
		),
	)

	rendered := out.String()
	if strings.Contains(rendered, rawSecret) {
		t.Fatalf("rendered log leaked nested group secret: %s", rendered)
	}
	if !strings.Contains(rendered, `"request_id":"request-001"`) {
		t.Fatalf("expected safe nested request_id to be preserved, got %s", rendered)
	}
	if !IsSecretRedacted(rendered) {
		t.Fatalf("expected nested secret redaction marker, got %s", rendered)
	}
}

func TestLatencyAndCountClampInvalidNegativeValues(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	logger := NewJSONLogger(&out, slog.LevelDebug)

	LogAttrs(
		context.Background(),
		logger,
		slog.LevelInfo,
		"dif.metrics.checked",
		Count("document_count", -5),
		Latency("latency_ms", -time.Second),
	)

	rendered := out.String()
	for _, expected := range []string{`"document_count":0`, `"latency_ms":0`} {
		if !strings.Contains(rendered, expected) {
			t.Fatalf("expected rendered log to contain %s, got %s", expected, rendered)
		}
	}
}
