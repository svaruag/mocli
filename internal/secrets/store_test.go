package secrets

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/svaruag/mocli/internal/config"
)

func TestFileStoreTokenRoundTrip(t *testing.T) {
	t.Setenv("MO_CONFIG_DIR", t.TempDir())

	cfg := config.AppConfig{KeyringBackend: "file"}
	lookup := func(key string) (string, bool) {
		if key == "MO_KEYRING_PASSWORD" {
			return "test-password", true
		}
		return "", false
	}

	store, info, err := OpenStore(lookup, cfg)
	if err != nil {
		t.Fatalf("OpenStore returned error: %v", err)
	}
	if info.Resolved != "file" {
		t.Fatalf("expected file backend, got %q", info.Resolved)
	}

	client := "default"
	email := "user@example.com"
	refresh := "refresh-token-abc"

	if err := store.PutToken(client, email, Token{RefreshToken: refresh}); err != nil {
		t.Fatalf("PutToken returned error: %v", err)
	}

	tok, err := store.GetToken(client, email)
	if err != nil {
		t.Fatalf("GetToken returned error: %v", err)
	}
	if tok.RefreshToken != refresh {
		t.Fatalf("unexpected refresh token %q", tok.RefreshToken)
	}

	dir, err := config.KeyringDir()
	if err != nil {
		t.Fatalf("KeyringDir returned error: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("ReadDir returned error: %v", err)
	}
	if len(entries) == 0 {
		t.Fatalf("expected encrypted keyring files")
	}
	content, err := os.ReadFile(filepath.Join(dir, entries[0].Name()))
	if err != nil {
		t.Fatalf("ReadFile returned error: %v", err)
	}
	if strings.Contains(string(content), refresh) {
		t.Fatalf("refresh token should not be stored in plaintext")
	}

	if err := store.DeleteToken(client, email); err != nil {
		t.Fatalf("DeleteToken returned error: %v", err)
	}
	if _, err := store.GetToken(client, email); err == nil {
		t.Fatalf("expected missing token after delete")
	}
}

func TestOpenStoreFileBackendRequiresPassword(t *testing.T) {
	t.Setenv("MO_CONFIG_DIR", t.TempDir())

	cfg := config.AppConfig{KeyringBackend: "file"}
	lookup := func(string) (string, bool) {
		return "", false
	}

	_, _, err := OpenStore(lookup, cfg)
	if err == nil {
		t.Fatalf("expected OpenStore to fail without MO_KEYRING_PASSWORD")
	}
	if !errors.Is(err, ErrFileBackendPasswordRequired) {
		t.Fatalf("expected ErrFileBackendPasswordRequired, got %v", err)
	}
}
