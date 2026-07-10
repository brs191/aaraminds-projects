// Package objectstore abstracts artifact byte storage (PRD 12.2: object
// storage holds media, audio, caption and export artifacts). The MVP ships a
// local-filesystem implementation; a cloud blob implementation can be swapped
// in behind the same interface.
package objectstore

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aaraminds/transcript-agent/internal/domain"
)

// ObjectStore stores and retrieves artifact bytes by opaque URI.
type ObjectStore interface {
	// Put stores data under key and returns a durable URI.
	Put(ctx context.Context, key string, data []byte) (string, error)
	// PutStream stores everything read from r under key without buffering the
	// full payload in memory. Returns the durable URI and the byte count.
	PutStream(ctx context.Context, key string, r io.Reader) (string, int64, error)
	// Get retrieves the bytes for a URI previously returned by Put/PutStream.
	Get(ctx context.Context, uri string) ([]byte, error)
	// Open returns a seekable reader for a stored URI (Range-capable
	// streaming), plus the artifact's size and modification time.
	Open(ctx context.Context, uri string) (io.ReadSeekCloser, int64, time.Time, error)
	// Delete removes the bytes for a stored URI (retention sweep). Deleting a
	// URI that no longer exists is not an error.
	Delete(ctx context.Context, uri string) error
}

const localScheme = "local://"

// Local is a filesystem-backed ObjectStore rooted at BaseDir
// (default data/artifacts/).
type Local struct {
	BaseDir string
}

// NewLocal creates the base directory if needed.
func NewLocal(baseDir string) (*Local, error) {
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return nil, domain.E(domain.CodeArtifactWriteFailed, "create artifact dir: %v", err)
	}
	return &Local{BaseDir: baseDir}, nil
}

func (l *Local) path(key string) (string, error) {
	clean := filepath.Clean(key)
	if strings.HasPrefix(clean, "..") || filepath.IsAbs(clean) {
		return "", domain.E(domain.CodeValidationError, "invalid artifact key %q", key)
	}
	return filepath.Join(l.BaseDir, clean), nil
}

func (l *Local) Put(_ context.Context, key string, data []byte) (string, error) {
	p, err := l.path(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", domain.E(domain.CodeArtifactWriteFailed, "mkdir: %v", err)
	}
	if err := os.WriteFile(p, data, 0o644); err != nil {
		return "", domain.E(domain.CodeArtifactWriteFailed, "write %s: %v", key, err)
	}
	return localScheme + key, nil
}

func (l *Local) PutStream(_ context.Context, key string, r io.Reader) (string, int64, error) {
	p, err := l.path(key)
	if err != nil {
		return "", 0, err
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return "", 0, domain.E(domain.CodeArtifactWriteFailed, "mkdir: %v", err)
	}
	f, err := os.OpenFile(p, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return "", 0, domain.E(domain.CodeArtifactWriteFailed, "create %s: %v", key, err)
	}
	n, copyErr := io.Copy(f, r)
	closeErr := f.Close()
	if copyErr != nil {
		_ = os.Remove(p)
		return "", 0, copyErr // preserve the reader error (e.g. MaxBytesError)
	}
	if closeErr != nil {
		_ = os.Remove(p)
		return "", 0, domain.E(domain.CodeArtifactWriteFailed, "close %s: %v", key, closeErr)
	}
	return localScheme + key, n, nil
}

// PathFor resolves a local:// URI to its absolute filesystem path. Used by
// the ffmpeg media processor so media resolution goes through the object
// store only (never user-supplied raw paths).
func (l *Local) PathFor(uri string) (string, error) {
	if !strings.HasPrefix(uri, localScheme) {
		return "", domain.E(domain.CodeValidationError, "unsupported artifact uri scheme: %s", uri)
	}
	return l.path(strings.TrimPrefix(uri, localScheme))
}

func (l *Local) Get(_ context.Context, uri string) ([]byte, error) {
	p, err := l.PathFor(uri)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, domain.E(domain.CodeMediaNotFound, "artifact not found: %s", uri)
	}
	return data, nil
}

func (l *Local) Open(_ context.Context, uri string) (io.ReadSeekCloser, int64, time.Time, error) {
	p, err := l.PathFor(uri)
	if err != nil {
		return nil, 0, time.Time{}, err
	}
	f, err := os.Open(p)
	if err != nil {
		return nil, 0, time.Time{}, domain.E(domain.CodeMediaNotFound, "artifact not found: %s", uri)
	}
	st, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, 0, time.Time{}, domain.E(domain.CodeMediaNotFound, "artifact not found: %s", uri)
	}
	return f, st.Size(), st.ModTime(), nil
}

func (l *Local) Delete(_ context.Context, uri string) error {
	p, err := l.PathFor(uri)
	if err != nil {
		return err
	}
	if err := os.Remove(p); err != nil && !os.IsNotExist(err) {
		return domain.E(domain.CodeArtifactWriteFailed, "delete %s: %v", uri, err)
	}
	return nil
}

var _ ObjectStore = (*Local)(nil)

// KeyFor builds a conventional artifact key.
func KeyFor(jobID, kind, filename string) string {
	return fmt.Sprintf("%s/%s/%s", jobID, kind, filename)
}
