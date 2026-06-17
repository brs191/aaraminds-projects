// Package export provides stable, deterministic serialization of the tool's
// computed state to JSON and CSV for reporting and downstream consumption.
//
// The shapes here are an explicit public contract: field names and column order
// are stable so saved reports diff cleanly across runs. All functions are pure
// with respect to their inputs and never touch the file system directly (CSV
// writers take an io.Writer).
package export

import (
	"encoding/csv"
	"encoding/json"
	"io"
	"strconv"
	"time"

	"github.com/aaraminds/copilot-session-manager/internal/analytics"
	"github.com/aaraminds/copilot-session-manager/internal/budget"
	"github.com/aaraminds/copilot-session-manager/internal/session"
)

// Report is the top-level aggregate serialized by ToJSON. It bundles the budget
// summary, time series, leaderboards, and a flattened per-session view.
type Report struct {
	GeneratedAt time.Time          `json:"generatedAt"`
	BudgetState budget.BudgetState `json:"budgetState"`
	// PremiumRequests is the total premium-request count across this month's
	// settled sessions (sum of session.shutdown.totalPremiumRequests).
	PremiumRequests int64                `json:"premiumRequests"`
	Daily           []analytics.Bucket   `json:"daily"`
	TopSessions     []analytics.Consumer `json:"topSessions"`
	TopModels       []analytics.Consumer `json:"topModels"`
	TopProjects     []analytics.Consumer `json:"topProjects"`
	Sessions        []SessionView        `json:"sessions"`
}

// SessionView is the flattened, serialization-friendly projection of a session.
// It exposes derived figures (credits, token totals, billing date) rather than
// the raw event-shaped Session, so consumers do not depend on internal fields.
type SessionView struct {
	ID               string    `json:"id"`
	Source           string    `json:"source"`
	Project          string    `json:"project"`
	Model            string    `json:"model"`
	BillingDate      string    `json:"billingDate"` // "2006-01-02"
	Credits          float64   `json:"credits"`
	InputTokens      int64     `json:"inputTokens"`
	OutputTokens     int64     `json:"outputTokens"`
	SystemTokens     int64     `json:"systemTokens"`
	CacheReadTokens  int64     `json:"cacheReadTokens"`  // prompt-cache reads, summed over the session's models
	CacheWriteTokens int64     `json:"cacheWriteTokens"` // prompt-cache writes, summed over the session's models
	ReasoningTokens  int64     `json:"reasoningTokens"`  // extended-thinking tokens, summed over the session's models
	PremiumRequests  int64     `json:"premiumRequests"`  // session.shutdown.totalPremiumRequests (0 until finalized)
	IsActive         bool      `json:"isActive"`
	IsFinal          bool      `json:"isFinal"`
	StartTime        time.Time `json:"startTime"`
	EndTime          time.Time `json:"endTime,omitempty"`
}

// NewSessionView builds the flattened view for one session.
func NewSessionView(s session.Session) SessionView {
	return SessionView{
		ID:               s.ID,
		Source:           s.Source,
		Project:          s.ProjectName,
		Model:            s.PrimaryModel,
		BillingDate:      s.BillingTime().Format("2006-01-02"),
		Credits:          budget.FromNanoAIU(s.TotalNanoAIU),
		InputTokens:      s.TotalInputTokens(),
		OutputTokens:     s.TotalOutputTokens(),
		SystemTokens:     s.Tokens.SystemTokens,
		CacheReadTokens:  s.TotalCacheReadTokens(),
		CacheWriteTokens: s.TotalCacheWriteTokens(),
		ReasoningTokens:  s.TotalReasoningTokens(),
		PremiumRequests:  s.TotalPremiumRequests,
		IsActive:         s.IsActive,
		IsFinal:          s.IsFinal,
		StartTime:        s.StartTime,
		EndTime:          s.EndTime,
	}
}

// SessionViews maps a slice of sessions to their flattened views, preserving order.
func SessionViews(sessions []session.Session) []SessionView {
	out := make([]SessionView, 0, len(sessions))
	for _, s := range sessions {
		out = append(out, NewSessionView(s))
	}
	return out
}

// ToJSON serializes a Report as indented, deterministic JSON. Determinism comes
// from the report being built from already-sorted analytics slices; encoding/json
// preserves slice order and emits struct fields in declaration order.
func ToJSON(r Report) ([]byte, error) {
	return json.MarshalIndent(r, "", "  ")
}

// SessionsToCSV writes one row per session with a stable header. Columns:
// date,project,model,source,credits,inputTokens,outputTokens,systemTokens,
// cacheReadTokens,cacheWriteTokens,reasoningTokens,premiumRequests,isActive,isFinal.
func SessionsToCSV(w io.Writer, sessions []session.Session) error {
	cw := csv.NewWriter(w)
	header := []string{
		"date", "project", "model", "source",
		"credits", "inputTokens", "outputTokens", "systemTokens",
		"cacheReadTokens", "cacheWriteTokens", "reasoningTokens", "premiumRequests",
		"isActive", "isFinal",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, s := range sessions {
		row := []string{
			s.BillingTime().Format("2006-01-02"),
			s.ProjectName,
			s.PrimaryModel,
			s.Source,
			formatCredits(budget.FromNanoAIU(s.TotalNanoAIU)),
			strconv.FormatInt(s.TotalInputTokens(), 10),
			strconv.FormatInt(s.TotalOutputTokens(), 10),
			strconv.FormatInt(s.Tokens.SystemTokens, 10),
			strconv.FormatInt(s.TotalCacheReadTokens(), 10),
			strconv.FormatInt(s.TotalCacheWriteTokens(), 10),
			strconv.FormatInt(s.TotalReasoningTokens(), 10),
			strconv.FormatInt(s.TotalPremiumRequests, 10),
			strconv.FormatBool(s.IsActive),
			strconv.FormatBool(s.IsFinal),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// DailyToCSV writes one row per daily bucket with a stable header. Columns:
// date,sessions,credits,inputTokens,outputTokens.
func DailyToCSV(w io.Writer, daily []analytics.Bucket) error {
	cw := csv.NewWriter(w)
	header := []string{"date", "sessions", "credits", "inputTokens", "outputTokens"}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, b := range daily {
		row := []string{
			b.Key,
			strconv.Itoa(b.Sessions),
			formatCredits(b.Credits),
			strconv.FormatInt(b.InputTokens, 10),
			strconv.FormatInt(b.OutputTokens, 10),
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// formatCredits renders a credit value at full precision using the shortest decimal
// representation that round-trips ('f' format, precision -1), trimming trailing zeros
// and never using exponential notation, for compact and stable CSV output.
func formatCredits(c float64) string {
	return strconv.FormatFloat(c, 'f', -1, 64)
}
