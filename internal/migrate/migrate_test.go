package migrate

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestRunMigratesLegacyConfigTree(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	newBase := filepath.Join(root, "mocli")
	t.Setenv("MO_CONFIG_DIR", newBase)

	legacy := filepath.Join(root, "legacy-cli")
	if err := os.MkdirAll(filepath.Join(legacy, "keyring"), 0o700); err != nil {
		t.Fatalf("mkdir legacy: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "credentials.json"), []byte(`{"client_id":"abc","tenant":"common"}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "config.json"), []byte(`{"default_account":"a@example.com","default_client":"default","accounts":[{"email":"a@example.com","client":"default"}]}`), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacy, "keyring", "token-file"), []byte("ciphertext"), 0o600); err != nil {
		t.Fatalf("write keyring: %v", err)
	}

	if err := Run(io.Discard); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}

	for _, p := range []string{
		filepath.Join(newBase, "credentials.json"),
		filepath.Join(newBase, "config.json"),
		filepath.Join(newBase, "keyring", "token-file"),
		filepath.Join(newBase, markerFile),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Fatalf("expected %s to exist: %v", p, err)
		}
	}
}

func TestRunNoLegacyCandidateNoop(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	newBase := filepath.Join(root, "mocli")
	t.Setenv("MO_CONFIG_DIR", newBase)

	if err := Run(io.Discard); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if _, err := os.Stat(newBase); !os.IsNotExist(err) {
		t.Fatalf("expected new base to remain absent when no legacy candidate")
	}
}

func TestRunMarksExistingCurrentState(t *testing.T) {
	root := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", root)
	newBase := filepath.Join(root, "mocli")
	t.Setenv("MO_CONFIG_DIR", newBase)

	if err := os.MkdirAll(newBase, 0o700); err != nil {
		t.Fatalf("mkdir new base: %v", err)
	}
	if err := os.WriteFile(filepath.Join(newBase, "credentials.json"), []byte(`{"client_id":"abc","tenant":"common"}`), 0o600); err != nil {
		t.Fatalf("write credentials: %v", err)
	}

	if err := Run(io.Discard); err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(newBase, markerFile)); err != nil {
		t.Fatalf("expected marker file: %v", err)
	}
}
