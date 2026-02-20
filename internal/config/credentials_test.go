package config

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseCredentialsJSONDirect(t *testing.T) {
	cred, err := ParseCredentialsJSON([]byte(`{"client_id":"abc","tenant":"contoso"}`))
	if err != nil {
		t.Fatalf("ParseCredentialsJSON returned error: %v", err)
	}
	if cred.ClientID != "abc" {
		t.Fatalf("expected client_id abc, got %q", cred.ClientID)
	}
	if cred.Tenant != "contoso" {
		t.Fatalf("expected tenant contoso, got %q", cred.Tenant)
	}
}

func TestParseCredentialsJSONAppID(t *testing.T) {
	cred, err := ParseCredentialsJSON([]byte(`{"appId":"xyz","tenantId":"tenant-1"}`))
	if err != nil {
		t.Fatalf("ParseCredentialsJSON returned error: %v", err)
	}
	if cred.ClientID != "xyz" {
		t.Fatalf("expected client_id xyz, got %q", cred.ClientID)
	}
	if cred.Tenant != "tenant-1" {
		t.Fatalf("expected tenant tenant-1, got %q", cred.Tenant)
	}
}

func TestParseCredentialsJSONMissingClientID(t *testing.T) {
	_, err := ParseCredentialsJSON([]byte(`{"tenant":"x"}`))
	if err == nil {
		t.Fatalf("expected error for missing client id")
	}
}

func TestLoadCredentialsMissingFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("MO_CONFIG_DIR", tmp)

	_, err := LoadCredentials("default")
	if err == nil {
		t.Fatalf("expected error when default client has no credentials file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestLoadCredentialsUsesFileWhenPresent(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("MO_CONFIG_DIR", tmp)
	if err := os.WriteFile(filepath.Join(tmp, "credentials.json"), []byte(`{"client_id":"file-client","tenant":"file-tenant"}`), 0o600); err != nil {
		t.Fatalf("write credentials file: %v", err)
	}

	cred, err := LoadCredentials("default")
	if err != nil {
		t.Fatalf("LoadCredentials returned error: %v", err)
	}
	if cred.ClientID != "file-client" || cred.Tenant != "file-tenant" {
		t.Fatalf("unexpected file credentials: %+v", cred)
	}
}

func TestLoadCredentialsReturnsNotExistForNamedClientWithoutFile(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("MO_CONFIG_DIR", tmp)

	_, err := LoadCredentials("work")
	if err == nil {
		t.Fatalf("expected error when named client has no credentials file")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected os.ErrNotExist, got %v", err)
	}
}

func TestLoadCredentialsRejectsInsecureFilePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission checks are skipped on windows")
	}

	tmp := t.TempDir()
	t.Setenv("MO_CONFIG_DIR", tmp)
	path := filepath.Join(tmp, "credentials.json")
	if err := os.WriteFile(path, []byte(`{"client_id":"file-client","tenant":"common"}`), 0o644); err != nil {
		t.Fatalf("write credentials file: %v", err)
	}

	_, err := LoadCredentials("default")
	if err == nil {
		t.Fatalf("expected insecure-permissions error")
	}
}
