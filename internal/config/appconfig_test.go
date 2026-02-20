package config

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestLoadAppConfigRejectsInsecureFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks are skipped on windows")
	}

	tmp := t.TempDir()
	t.Setenv("MO_CONFIG_DIR", tmp)
	path := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(path, []byte(`{"default_client":"default"}`), 0o644); err != nil {
		t.Fatalf("write config file: %v", err)
	}

	_, err := LoadAppConfig()
	if err == nil {
		t.Fatalf("expected insecure-permissions error")
	}
}
