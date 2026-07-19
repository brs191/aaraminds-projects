// Package logging provides DIF's safe structured logging baseline.
//
// It intentionally exposes small typed helpers for operational metadata only:
// identifiers, paths, hashes, counts, caveat/status codes, and latency. Raw
// document text is not a supported helper and obvious secret-like values are
// redacted by the JSON handler before output.
package logging

import (
	"context"
	"io"
	"log/slog"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	redactedSecret       = "[REDACTED_SECRET]"
	redactedDocumentText = "[REDACTED_DOCUMENT_TEXT]"
)

var (
	secretKeyPattern       = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api[_-]?key|credential|authorization|client[_-]?secret|private[_-]?key|connection[_-]?string|database[_-]?url|db[_-]?url)`)
	bearerPattern          = regexp.MustCompile(`(?i)\bbearer\s+[a-z0-9._~+/=-]+`)
	keyValuePattern        = regexp.MustCompile(`(?i)(password|passwd|pwd|secret|token|api[_-]?key|client[_-]?secret|authorization)=([^&\s]+)`)
	jwtPattern             = regexp.MustCompile(`\b[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\.[A-Za-z0-9_-]{20,}\b`)
	awsKeyPattern          = regexp.MustCompile(`\bAKIA[0-9A-Z]{16}\b`)
	privateKeyPattern      = regexp.MustCompile(`-----BEGIN [A-Z ]*PRIVATE KEY-----`)
	documentTextKeyPattern = regexp.MustCompile(`(?i)(raw[_-]?document|document[_-]?text|raw[_-]?text|passage[_-]?text|chunk[_-]?text|snippet)`)
)

// NewJSONLogger returns a slog logger configured with safe JSON output.
func NewJSONLogger(w io.Writer, level slog.Leveler) *slog.Logger {
	return slog.New(NewJSONHandler(w, level))
}

// NewJSONHandler returns a slog JSON handler with DIF redaction enabled.
func NewJSONHandler(w io.Writer, level slog.Leveler) slog.Handler {
	return slog.NewJSONHandler(w, &slog.HandlerOptions{
		Level:       level,
		ReplaceAttr: redactAttr,
	})
}

// LogAttrs emits a structured record using the safe handler and typed attrs.
func LogAttrs(ctx context.Context, logger *slog.Logger, level slog.Level, event string, attrs ...slog.Attr) {
	safeAttrs := make([]slog.Attr, 0, len(attrs))
	for _, attr := range attrs {
		safeAttrs = append(safeAttrs, sanitizeAttr(attr))
	}
	logger.LogAttrs(ctx, level, event, safeAttrs...)
}

// ID records a stable identifier such as project_id, corpus_id, request_id, or run_id.
func ID(key, value string) slog.Attr {
	return safeString(key, value)
}

// Path records a source or local path. Do not use it for raw document text.
func Path(key, value string) slog.Attr {
	return safeString(key, value)
}

// Hash records a content, parameter, or source hash.
func Hash(key, value string) slog.Attr {
	return safeString(key, value)
}

// Count records a non-negative count.
func Count(key string, value int64) slog.Attr {
	if value < 0 {
		value = 0
	}
	return slog.Int64(key, value)
}

// CaveatCode records an extraction, retrieval, or policy caveat code.
func CaveatCode(key, value string) slog.Attr {
	return safeString(key, value)
}

// Status records an explicit operational status or outcome.
func Status(key, value string) slog.Attr {
	return safeString(key, value)
}

// Latency records duration as milliseconds under the supplied key.
func Latency(key string, value time.Duration) slog.Attr {
	if value < 0 {
		value = 0
	}
	return slog.Int64(key, value.Milliseconds())
}

func safeString(key, value string) slog.Attr {
	return slog.String(key, redactValue(key, value))
}

func redactAttr(groups []string, attr slog.Attr) slog.Attr {
	return sanitizeAttr(attr)
}

func sanitizeAttr(attr slog.Attr) slog.Attr {
	if isDocumentTextKey(attr.Key) {
		return slog.String(attr.Key, redactedDocumentText)
	}
	if secretKeyPattern.MatchString(attr.Key) {
		return slog.String(attr.Key, redactedSecret)
	}
	if attr.Value.Kind() == slog.KindGroup {
		group := attr.Value.Group()
		safeGroup := make([]slog.Attr, 0, len(group))
		for _, nested := range group {
			safeGroup = append(safeGroup, sanitizeAttr(nested))
		}
		return slog.Group(attr.Key, attrsToAny(safeGroup)...)
	}
	if attr.Value.Kind() == slog.KindString {
		attr.Value = slog.StringValue(redactValue(attr.Key, attr.Value.String()))
	}
	if attr.Value.Kind() == slog.KindAny {
		return slog.String(attr.Key, redactedSecret)
	}
	return attr
}

func attrsToAny(attrs []slog.Attr) []any {
	values := make([]any, 0, len(attrs))
	for _, attr := range attrs {
		values = append(values, attr)
	}
	return values
}

func redactValue(key, value string) string {
	if value == "" {
		return value
	}
	if isDocumentTextKey(key) {
		return redactedDocumentText
	}
	if secretKeyPattern.MatchString(key) {
		return redactedSecret
	}

	value = redactDatabaseURL(value)
	value = bearerPattern.ReplaceAllString(value, "Bearer "+redactedSecret)
	value = keyValuePattern.ReplaceAllString(value, "${1}="+redactedSecret)
	value = jwtPattern.ReplaceAllString(value, redactedSecret)
	value = awsKeyPattern.ReplaceAllString(value, redactedSecret)
	value = privateKeyPattern.ReplaceAllString(value, redactedSecret)
	return value
}

func isDocumentTextKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	switch normalized {
	case "body", "content", "document", "text":
		return true
	default:
		return documentTextKeyPattern.MatchString(normalized)
	}
}

func redactDatabaseURL(value string) string {
	for _, scheme := range []string{"postgres://", "postgresql://"} {
		index := strings.Index(value, scheme)
		if index == -1 {
			continue
		}
		end := len(value)
		if space := strings.IndexAny(value[index:], " \t\n\r"); space != -1 {
			end = index + space
		}
		candidate := value[index:end]
		parsed, err := url.Parse(candidate)
		if err != nil || parsed.User == nil {
			continue
		}
		username := parsed.User.Username()
		if username == "" {
			parsed.User = nil
		} else {
			parsed.User = url.UserPassword(username, redactedSecret)
		}
		value = value[:index] + parsed.String() + value[end:]
	}
	return value
}

// IsSecretRedacted reports whether a rendered log line contains the redaction marker.
// It is intended for tests and local harnesses, not runtime policy checks.
func IsSecretRedacted(rendered string) bool {
	return strings.Contains(rendered, strconv.Quote(redactedSecret)) || strings.Contains(rendered, redactedSecret)
}
