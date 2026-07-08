package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"regexp"
	"slices"
	"strings"

	"github.com/aaraminds/rif/phase5/ingestion/diff"
	"github.com/aaraminds/rif/phase5/ingestion/queue"
)

type githubPushPayload struct {
	Before     string `json:"before"`
	After      string `json:"after"`
	Ref        string `json:"ref"`
	Compare    string `json:"compare"`
	Created    bool   `json:"created"`
	Deleted    bool   `json:"deleted"`
	Forced     bool   `json:"forced"`
	BaseRef    string `json:"base_ref"`
	Repository struct {
		Name          string `json:"name"`
		FullName      string `json:"full_name"`
		DefaultBranch string `json:"default_branch"`
	} `json:"repository"`
	Commits []struct {
		Added    []string `json:"added"`
		Modified []string `json:"modified"`
		Removed  []string `json:"removed"`
	} `json:"commits"`
}

// GithubWebhook handles push events and enqueues Phase 5 incremental jobs.
func GithubWebhook(qs *queue.Store, secret string) http.HandlerFunc {
	webhookSecret := strings.TrimSpace(secret)
	return func(w http.ResponseWriter, r *http.Request) {
		eventType := strings.TrimSpace(r.Header.Get("X-GitHub-Event"))
		if eventType != "" && eventType != "push" {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"status": "ignored",
				"reason": "unsupported_event",
				"event":  eventType,
			})
			return
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, errResponse("failed to read webhook payload: "+err.Error()))
			return
		}
		if err := verifyGitHubSignature(body, r.Header.Get("X-Hub-Signature-256"), webhookSecret); err != nil {
			writeJSON(w, http.StatusUnauthorized, errResponse("invalid webhook signature: "+err.Error()))
			return
		}

		var payload githubPushPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			writeJSON(w, http.StatusBadRequest, errResponse("invalid webhook JSON: "+err.Error()))
			return
		}

		repoID := strings.TrimSpace(r.Header.Get("X-RIF-Repo-ID"))
		if repoID == "" {
			repoID = strings.TrimSpace(payload.Repository.FullName)
		}
		if repoID == "" {
			repoID = strings.TrimSpace(payload.Repository.Name)
		}
		if repoID == "" {
			writeJSON(w, http.StatusBadRequest, errResponse("repo_id not found in webhook payload/header"))
			return
		}
		defaultBranch := strings.TrimSpace(payload.Repository.DefaultBranch)
		if defaultBranch == "" {
			defaultBranch = "main"
		}
		targetRef := "refs/heads/" + defaultBranch
		if strings.TrimSpace(payload.Ref) != targetRef {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"status":         "ignored",
				"reason":         "non_default_branch",
				"ref":            payload.Ref,
				"default_branch": defaultBranch,
			})
			return
		}

		var changedFiles []string
		for _, commit := range payload.Commits {
			changedFiles = append(changedFiles, commit.Added...)
			changedFiles = append(changedFiles, commit.Modified...)
			changedFiles = append(changedFiles, commit.Removed...)
		}
		changedFiles = dedupeStrings(changedFiles)
		classification := diff.Classify(changedFiles)
		forceReindex := payload.Forced || payload.Deleted || payload.Created || strings.HasPrefix(strings.TrimSpace(payload.Before), "0000000")
		if forceReindex {
			classification.ForceReindex = true
		}
		queuedSHA := selectQueuedSHA(payload.After, payload.Before)
		if queuedSHA == "" {
			writeJSON(w, http.StatusAccepted, map[string]any{
				"status":        "ignored",
				"reason":        "no_resolvable_sha",
				"before":        payload.Before,
				"after":         payload.After,
				"forced":        payload.Forced,
				"created":       payload.Created,
				"deleted":       payload.Deleted,
				"force_reindex": classification.ForceReindex,
			})
			return
		}

		enqueued := 0
		if classification.ForceReindex {
			if err := qs.Enqueue(r.Context(), repoID, queuedSHA, queue.LaneFullReindex, payload.Before, payload.After); err != nil {
				writeJSON(w, http.StatusInternalServerError, errResponse("failed to enqueue full reindex: "+err.Error()))
				return
			}
			enqueued++
		} else {
			if len(classification.LaneA) > 0 {
				if err := qs.Enqueue(r.Context(), repoID, queuedSHA, queue.LaneA, payload.Before, payload.After); err != nil {
					writeJSON(w, http.StatusInternalServerError, errResponse("failed to enqueue lane A: "+err.Error()))
					return
				}
				enqueued++
			}
			if len(classification.LaneB) > 0 {
				if err := qs.Enqueue(r.Context(), repoID, queuedSHA, queue.LaneB, payload.Before, payload.After); err != nil {
					writeJSON(w, http.StatusInternalServerError, errResponse("failed to enqueue lane B: "+err.Error()))
					return
				}
				enqueued++
			}
			if len(classification.LaneC) > 0 {
				if err := qs.Enqueue(r.Context(), repoID, queuedSHA, queue.LaneC, payload.Before, payload.After); err != nil {
					writeJSON(w, http.StatusInternalServerError, errResponse("failed to enqueue lane C: "+err.Error()))
					return
				}
				enqueued++
			}
		}

		slog.InfoContext(r.Context(), "github webhook enqueued",
			slog.String("repo_id", repoID),
			slog.String("ref", payload.Ref),
			slog.String("default_branch", defaultBranch),
			slog.String("before", payload.Before),
			slog.String("after", payload.After),
			slog.String("queued_sha", queuedSHA),
			slog.Bool("forced_push", payload.Forced),
			slog.Bool("ref_created", payload.Created),
			slog.Bool("ref_deleted", payload.Deleted),
			slog.String("compare", payload.Compare),
			slog.Bool("force_reindex", classification.ForceReindex),
			slog.Int("lane_a_files", len(classification.LaneA)),
			slog.Int("lane_b_files", len(classification.LaneB)),
			slog.Int("lane_c_files", len(classification.LaneC)),
			slog.Int("enqueued_jobs", enqueued),
		)
		writeJSON(w, http.StatusAccepted, map[string]any{
			"status":         "queued",
			"repo_id":        repoID,
			"ref":            payload.Ref,
			"default_branch": defaultBranch,
			"force_reindex":  classification.ForceReindex,
			"lane_a_files":   len(classification.LaneA),
			"lane_b_files":   len(classification.LaneB),
			"lane_c_files":   len(classification.LaneC),
			"queued_sha":     queuedSHA,
			"enqueued_jobs":  enqueued,
		})
	}
}

func verifyGitHubSignature(body []byte, signatureHeader, secret string) error {
	if strings.TrimSpace(secret) == "" {
		return nil
	}

	signature := strings.TrimSpace(signatureHeader)
	if signature == "" {
		return errors.New("missing X-Hub-Signature-256 header")
	}
	if !strings.HasPrefix(signature, "sha256=") {
		return errors.New("expected sha256 signature prefix")
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return errors.New("signature is not valid hex")
	}

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := mac.Sum(nil)
	if !hmac.Equal(provided, expected) {
		return errors.New("signature mismatch")
	}
	return nil
}

var sha40HexRe = regexp.MustCompile(`^[0-9a-f]{40}$`)

func selectQueuedSHA(after, before string) string {
	if normalized := normalizedNonZeroSHA(after); normalized != "" {
		return normalized
	}
	return normalizedNonZeroSHA(before)
}

func normalizedNonZeroSHA(v string) string {
	normalized := strings.ToLower(strings.TrimSpace(v))
	if !sha40HexRe.MatchString(normalized) {
		return ""
	}
	if normalized == strings.Repeat("0", 40) {
		return ""
	}
	return normalized
}

func dedupeStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := make([]string, 0, len(values))
	for _, value := range values {
		v := strings.TrimSpace(value)
		if v == "" {
			continue
		}
		if slices.Contains(seen, v) {
			continue
		}
		seen = append(seen, v)
		out = append(out, v)
	}
	return out
}
