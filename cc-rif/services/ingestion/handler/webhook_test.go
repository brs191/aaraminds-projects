package handler

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestVerifyGitHubSignature(t *testing.T) {
	t.Parallel()

	body := []byte(`{"ref":"refs/heads/main"}`)
	secret := "test-secret"

	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	signature := "sha256=" + hex.EncodeToString(mac.Sum(nil))

	tests := []struct {
		name      string
		header    string
		secret    string
		wantError bool
	}{
		{
			name:      "valid signature",
			header:    signature,
			secret:    secret,
			wantError: false,
		},
		{
			name:      "missing signature when secret configured",
			header:    "",
			secret:    secret,
			wantError: true,
		},
		{
			name:      "bad prefix",
			header:    "sha1=abc",
			secret:    secret,
			wantError: true,
		},
		{
			name:      "signature mismatch",
			header:    "sha256=" + hex.EncodeToString([]byte("wrong")),
			secret:    secret,
			wantError: true,
		},
		{
			name:      "verification disabled when secret empty",
			header:    "",
			secret:    "",
			wantError: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := verifyGitHubSignature(body, tc.header, tc.secret)
			if tc.wantError && err == nil {
				t.Fatalf("expected error but got nil")
			}
			if !tc.wantError && err != nil {
				t.Fatalf("expected no error but got %v", err)
			}
		})
	}
}
