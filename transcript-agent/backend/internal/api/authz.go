package api

import (
	"context"
	"net/http"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

func canAccessJob(id Identity, job *domain.Job) bool {
	if id.Role == domain.RoleReviewer || id.Role == domain.RoleAdmin {
		return true
	}
	return id.Role == domain.RoleProducer && id.UserID == job.SubmittedBy
}

func requireJobAccess(id Identity, job *domain.Job) error {
	if canAccessJob(id, job) {
		return nil
	}
	return domain.E(domain.CodeUserNotAuthorized,
		"user %q is not permitted to access job %s", id.UserID, job.JobID)
}

func (s *Server) loadAuthorizedJob(r *http.Request) (*domain.Job, error) {
	job, err := s.loadJob(r)
	if err != nil {
		return nil, err
	}
	if err := requireJobAccess(identityFrom(r.Context()), job); err != nil {
		return nil, err
	}
	return job, nil
}

func (s *Server) requireVersionAccess(ctx context.Context, id Identity, versionID string) (*domain.TranscriptVersion, error) {
	parsed, err := parseUUID(versionID, "transcript_version_id")
	if err != nil {
		return nil, err
	}
	version, err := s.Tools.Stores.Transcripts.GetVersion(ctx, parsed)
	if err != nil {
		return nil, err
	}
	job, err := s.Tools.Stores.Jobs.GetJob(ctx, version.JobID)
	if err != nil {
		return nil, err
	}
	if err := requireJobAccess(id, job); err != nil {
		return nil, err
	}
	return version, nil
}
