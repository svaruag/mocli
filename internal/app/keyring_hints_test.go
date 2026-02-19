package app

import (
	"errors"
	"strings"
	"testing"

	"github.com/svaruag/mocli/internal/secrets"
)

func TestSecretsBackendHintForMissingFilePassword(t *testing.T) {
	hint := secretsBackendHint(secrets.ErrFileBackendPasswordRequired)
	if !strings.Contains(hint, "MO_KEYRING_PASSWORD") {
		t.Fatalf("expected MO_KEYRING_PASSWORD in hint, got %q", hint)
	}
	if !strings.Contains(hint, "MO_KEYRING_BACKEND=file") {
		t.Fatalf("expected MO_KEYRING_BACKEND=file in hint, got %q", hint)
	}
}

func TestSecretsBackendHintFallbackToErrorText(t *testing.T) {
	err := errors.New("backend unavailable")
	hint := secretsBackendHint(err)
	if hint != err.Error() {
		t.Fatalf("expected fallback to original error text, got %q", hint)
	}
}
