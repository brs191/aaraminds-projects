package cloneurl

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

var ErrInvalidCloneURL = errors.New("invalid clone_url")

// Validate returns a trimmed clone URL if it uses an allowed Git transport and
// host. HTTPS and SSH GitHub-style URLs are accepted; local/file paths are not.
func Validate(raw string, allowedHosts []string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", fmt.Errorf("%w: empty", ErrInvalidCloneURL)
	}
	host, err := cloneHost(trimmed)
	if err != nil {
		return "", err
	}
	if net.ParseIP(host) != nil {
		return "", fmt.Errorf("%w: IP hosts are not allowed", ErrInvalidCloneURL)
	}
	allowed := normalizeAllowedHosts(allowedHosts)
	if len(allowed) == 0 {
		return "", fmt.Errorf("%w: no allowed clone hosts configured", ErrInvalidCloneURL)
	}
	if _, ok := allowed[host]; !ok {
		return "", fmt.Errorf("%w: host %q is not allowed", ErrInvalidCloneURL, host)
	}
	return trimmed, nil
}

func cloneHost(raw string) (string, error) {
	if strings.HasPrefix(raw, "git@") {
		rest := strings.TrimPrefix(raw, "git@")
		parts := strings.SplitN(rest, ":", 2)
		if len(parts) != 2 || strings.TrimSpace(parts[1]) == "" {
			return "", fmt.Errorf("%w: malformed SSH scp-style URL", ErrInvalidCloneURL)
		}
		return normalizeHost(parts[0])
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidCloneURL, err)
	}
	switch parsed.Scheme {
	case "https", "ssh":
	default:
		return "", fmt.Errorf("%w: unsupported scheme %q", ErrInvalidCloneURL, parsed.Scheme)
	}
	if parsed.User != nil {
		if parsed.Scheme != "ssh" || parsed.User.Username() != "git" {
			return "", fmt.Errorf("%w: credentials in clone_url are not allowed", ErrInvalidCloneURL)
		}
		if _, hasPassword := parsed.User.Password(); hasPassword {
			return "", fmt.Errorf("%w: credentials in clone_url are not allowed", ErrInvalidCloneURL)
		}
	}
	if strings.TrimSpace(parsed.Path) == "" || parsed.Path == "/" {
		return "", fmt.Errorf("%w: repository path is required", ErrInvalidCloneURL)
	}
	return normalizeHost(parsed.Hostname())
}

func normalizeHost(host string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(host))
	if normalized == "" {
		return "", fmt.Errorf("%w: host is required", ErrInvalidCloneURL)
	}
	if strings.EqualFold(normalized, "localhost") {
		return "", fmt.Errorf("%w: localhost is not allowed", ErrInvalidCloneURL)
	}
	return normalized, nil
}

func normalizeAllowedHosts(hosts []string) map[string]struct{} {
	allowed := make(map[string]struct{}, len(hosts))
	for _, host := range hosts {
		normalized := strings.ToLower(strings.TrimSpace(host))
		if normalized != "" {
			allowed[normalized] = struct{}{}
		}
	}
	return allowed
}
