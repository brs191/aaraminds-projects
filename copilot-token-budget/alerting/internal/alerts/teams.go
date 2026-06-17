// Package alerts implements the Microsoft Teams Adaptive Card sender for the
// Copilot Token Budget alert engine.
//
// Design constraints (ADR-004, ADR-006):
//   - Webhook URL is read from the COPILOT_BUDGET_TEAMS_WEBHOOK environment variable —
//     never a CLI flag (visible in ps aux) and never logged.
//   - One retry on HTTP 429 or 5xx with jitter backoff to prevent stampedes
//     when 1,000+ engineers fire alerts at the same time.
package alerts

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/session"
)

// rng provides per-process retry jitter independent of the global math/rand source,
// so 1,000 engineers' alert processes do not back off in lockstep regardless of
// toolchain. *rand.Rand is not concurrency-safe, so guard it with rngMu.
var (
	rng   = rand.New(rand.NewSource(time.Now().UnixNano() ^ int64(os.Getpid())))
	rngMu sync.Mutex
)

// jitterMillis returns a random [0,1000) ms jitter using the per-process rng.
func jitterMillis() time.Duration {
	rngMu.Lock()
	n := rng.Intn(1000)
	rngMu.Unlock()
	return time.Duration(n) * time.Millisecond
}

// AdaptiveCard is the complete Teams webhook message payload (outer envelope + card).
// Construct with NewBudgetCard; post with PostAdaptiveCard.
type AdaptiveCard struct {
	Type        string       `json:"type"`
	Attachments []Attachment `json:"attachments"`
}

// Attachment wraps a single Adaptive Card within a Teams message.
type Attachment struct {
	ContentType string      `json:"contentType"`
	Content     CardContent `json:"content"`
}

// CardContent is the Adaptive Card v1.4 schema body.
type CardContent struct {
	Schema  string        `json:"$schema"`
	Type    string        `json:"type"`
	Version string        `json:"version"`
	Body    []CardElement `json:"body"`
}

// CardElement represents a single element in an Adaptive Card body.
// All optional fields use omitempty; booleans use pointer types so false is preserved.
type CardElement struct {
	Type      string        `json:"type"`
	Text      string        `json:"text,omitempty"`
	Size      string        `json:"size,omitempty"`
	Weight    string        `json:"weight,omitempty"`
	Color     string        `json:"color,omitempty"`
	Wrap      *bool         `json:"wrap,omitempty"`
	Separator *bool         `json:"separator,omitempty"`
	Style     string        `json:"style,omitempty"`
	Facts     []Fact        `json:"facts,omitempty"`
	Items     []CardElement `json:"items,omitempty"`
}

// Fact is a key-value pair within a FactSet element.
type Fact struct {
	Title string `json:"title"`
	Value string `json:"value"`
}

// HTTPError carries a typed HTTP status code so callers can decide whether to retry.
type HTTPError struct {
	StatusCode int
	Body       string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("alerts: HTTP %d: %s", e.StatusCode, e.Body)
}

// NewBudgetCard builds a complete AdaptiveCard from a budget state, session list,
// the projected month-end TOTAL credits, and model routing recommendations.
//
// projectedTotal is the full projected month-end consumption (credits already used
// plus projected remaining burn) — see forecast.ProjectedMonthEndTotal. Any positive
// total is shown; it does not collapse to zero on the last day of the month.
func NewBudgetCard(
	state budget.BudgetState,
	sessions []session.Session,
	projectedTotal float64,
	recommendations []string,
) AdaptiveCard {
	body := buildCardBody(state, sessions, projectedTotal, recommendations)
	return AdaptiveCard{
		Type: "message",
		Attachments: []Attachment{
			{
				ContentType: "application/vnd.microsoft.card.adaptive",
				Content: CardContent{
					Schema:  "http://adaptivecards.io/schemas/adaptive-card.json",
					Type:    "AdaptiveCard",
					Version: "1.4",
					Body:    body,
				},
			},
		},
	}
}

// PostAdaptiveCard marshals card and POSTs it to webhookURL.
// It retries once on HTTP 429 or 5xx with a 2s+jitter backoff that respects ctx
// cancellation. webhookURL is NEVER logged — error messages carry no URL.
func PostAdaptiveCard(ctx context.Context, webhookURL string, card AdaptiveCard) error {
	payload, err := json.Marshal(card)
	if err != nil {
		return fmt.Errorf("alerts: marshal card: %w", err)
	}

	const maxAttempts = 2
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		if attempt > 0 {
			// Per-process jitter prevents 1,000 engineers triggering a stampede
			// simultaneously. The backoff is cancellable via ctx.
			backoff := 2*time.Second + jitterMillis()
			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				return ctx.Err()
			}
		}
		lastErr = postOnce(ctx, webhookURL, payload)
		if lastErr == nil {
			return nil
		}
		var httpErr *HTTPError
		if !errors.As(lastErr, &httpErr) || !isRetryable(httpErr.StatusCode) {
			return lastErr
		}
	}
	return lastErr
}

// postOnce performs a single HTTP POST with a 10-second per-attempt timeout derived
// from the supplied ctx.
func postOnce(ctx context.Context, webhookURL string, payload []byte) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, webhookURL, bytes.NewReader(payload))
	if err != nil {
		// Do not include the webhook URL — http errors embed it in their string.
		return errors.New("alerts: invalid webhook URL (check COPILOT_BUDGET_TEAMS_WEBHOOK)")
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// http.DefaultClient.Do returns *url.Error which includes the full URL in
		// its Error() string. Extract only the underlying cause to prevent leakage.
		return fmt.Errorf("alerts: POST failed: %s", urlErrMessage(err))
	}
	defer resp.Body.Close()

	// Teams webhooks return 200, 202, or 204 on success.
	if resp.StatusCode != http.StatusOK &&
		resp.StatusCode != http.StatusAccepted &&
		resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		return &HTTPError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	// Drain the body so the connection can be reused, then it is closed by the defer.
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func isRetryable(code int) bool {
	return code == http.StatusTooManyRequests || (code >= 500 && code < 600)
}

// urlErrMessage extracts the underlying error message from a *url.Error without
// including the URL field, preventing accidental webhook URL leakage in logs.
func urlErrMessage(err error) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) && urlErr.Err != nil {
		return urlErr.Err.Error()
	}
	return err.Error()
}

// --- card building helpers ---

func buildCardBody(
	state budget.BudgetState,
	sessions []session.Session,
	projectedTotal float64,
	recommendations []string,
) []CardElement {
	color := statusColor(state.Status)
	var body []CardElement

	// Title
	body = append(body, CardElement{
		Type:   "TextBlock",
		Text:   fmt.Sprintf("🤖 Copilot Budget Alert — %s", state.Status),
		Size:   "ExtraLarge",
		Weight: "Bolder",
		Color:  color,
	})

	// Progress bar (ASCII, 20 blocks)
	bar := progressBar(state.UsedPct, 20)
	body = append(body, CardElement{
		Type:  "TextBlock",
		Text:  fmt.Sprintf("`%s` %.1f%%", bar, state.UsedPct),
		Wrap:  boolP(false),
		Color: color,
	})

	// Budget overview facts
	body = append(body, CardElement{
		Type:      "FactSet",
		Separator: boolP(true),
		Facts: []Fact{
			{Title: "Used Credits", Value: fmt.Sprintf("%.0f cr", state.UsedCredits)},
			{Title: "Monthly Allowance", Value: fmt.Sprintf("%d cr", state.AllowedCredits)},
			{Title: "Usage", Value: fmt.Sprintf("%.1f%%", state.UsedPct)},
			{Title: "Remaining", Value: fmt.Sprintf("%.0f cr", state.RemainingCredit)},
			{Title: "Status", Value: string(state.Status)},
		},
	})

	// Top 3 sessions
	top := topSessions(sessions, 3)
	if len(top) > 0 {
		body = append(body, CardElement{
			Type:      "TextBlock",
			Text:      "**Top Sessions by Consumption**",
			Weight:    "Bolder",
			Separator: boolP(true),
		})
		facts := make([]Fact, 0, len(top))
		for _, s := range top {
			credits := budget.FromNanoAIU(s.TotalNanoAIU)
			facts = append(facts, Fact{
				Title: s.ProjectName,
				Value: fmt.Sprintf("%.1f cr (%s)", credits, s.PrimaryModel),
			})
		}
		body = append(body, CardElement{
			Type:  "FactSet",
			Facts: facts,
		})
	}

	// Projected month-end forecast. projectedTotal is the full projected month-end
	// consumption; any positive total shows (it does not vanish on the last day).
	if projectedTotal > 0 {
		forecastColor := "Good"
		var trendText string
		overUnder := projectedTotal - float64(state.AllowedCredits)
		if overUnder > 0 {
			forecastColor = "Attention"
			trendText = fmt.Sprintf("+%.0f cr over allowance", overUnder)
		} else {
			trendText = fmt.Sprintf("%.0f cr under allowance", -overUnder)
		}
		body = append(body, CardElement{
			Type:      "TextBlock",
			Text:      "**Projected Month-End**",
			Weight:    "Bolder",
			Separator: boolP(true),
		})
		body = append(body, CardElement{
			Type: "FactSet",
			Facts: []Fact{
				{Title: "Projected Month-End Total", Value: fmt.Sprintf("%.0f cr", projectedTotal)},
				{Title: "vs Allowance", Value: trendText},
			},
		})
		body = append(body, CardElement{
			Type:  "TextBlock",
			Text:  fmt.Sprintf("Projected to use **%.0f cr** by month end", projectedTotal),
			Color: forecastColor,
			Wrap:  boolP(true),
		})
	}

	// Model routing recommendations
	if len(recommendations) > 0 {
		body = append(body, CardElement{
			Type:      "TextBlock",
			Text:      "**Model Routing Recommendations**",
			Weight:    "Bolder",
			Separator: boolP(true),
		})
		for _, rec := range recommendations {
			body = append(body, CardElement{
				Type: "TextBlock",
				Text: "• " + rec,
				Wrap: boolP(true),
			})
		}
	}

	return body
}

func statusColor(s budget.BudgetStatus) string {
	switch s {
	case budget.StatusCritical:
		return "Attention"
	case budget.StatusWarning:
		return "Warning"
	default:
		return "Good"
	}
}

// progressBar returns a string of width characters: filled blocks followed by empty blocks.
func progressBar(pct float64, width int) string {
	filled := int(pct / 100.0 * float64(width))
	if filled < 0 {
		filled = 0
	}
	if filled > width {
		filled = width
	}
	return strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
}

// topSessions returns the top n sessions by TotalNanoAIU (descending).
// Returns a copy — original slice is not modified.
func topSessions(sessions []session.Session, n int) []session.Session {
	sorted := make([]session.Session, len(sessions))
	copy(sorted, sessions)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].TotalNanoAIU > sorted[j].TotalNanoAIU
	})
	if len(sorted) > n {
		sorted = sorted[:n]
	}
	return sorted
}

func boolP(v bool) *bool { return &v }
