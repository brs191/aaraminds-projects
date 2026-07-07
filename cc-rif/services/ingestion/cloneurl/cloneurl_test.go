package cloneurl

import "testing"

func TestValidateAllowsConfiguredGitHubURLs(t *testing.T) {
	t.Parallel()

	hosts := []string{"github.com", "github.example.com"}
	for _, raw := range []string{
		"https://github.com/acme/repo.git",
		"ssh://git@github.example.com/acme/repo.git",
		"git@github.com:acme/repo.git",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			got, err := Validate(raw, hosts)
			if err != nil {
				t.Fatalf("Validate(%q): %v", raw, err)
			}
			if got != raw {
				t.Fatalf("Validate(%q) = %q", raw, got)
			}
		})
	}
}

func TestValidateRejectsUnsafeCloneURLs(t *testing.T) {
	t.Parallel()

	hosts := []string{"github.com"}
	for _, raw := range []string{
		"file:///tmp/repo",
		"http://github.com/acme/repo.git",
		"https://evil.example.com/acme/repo.git",
		"https://token@github.com/acme/repo.git",
		"https://localhost/acme/repo.git",
		"https://127.0.0.1/acme/repo.git",
		"/tmp/repo",
	} {
		t.Run(raw, func(t *testing.T) {
			t.Parallel()
			if _, err := Validate(raw, hosts); err == nil {
				t.Fatalf("Validate(%q) expected error", raw)
			}
		})
	}
}
