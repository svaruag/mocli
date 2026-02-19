package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

const appDirName = "mocli"

var safeNameRE = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

func BaseDir() (string, error) {
	if v := strings.TrimSpace(os.Getenv("MO_CONFIG_DIR")); v != "" {
		return v, nil
	}
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, appDirName), nil
}

func EnsureBaseDir() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	if mkErr := os.MkdirAll(dir, 0o700); mkErr != nil {
		return "", fmt.Errorf("ensure base dir: %w", mkErr)
	}
	return dir, nil
}

func ConfigPath() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

func CredentialsPath(client string) (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	if normalizeClientName(client) == "default" {
		return filepath.Join(dir, "credentials.json"), nil
	}
	return filepath.Join(dir, "credentials-"+normalizeClientName(client)+".json"), nil
}

func KeyringDir() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "keyring"), nil
}

func OAuthStateDir() (string, error) {
	dir, err := BaseDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "state", "oauth"), nil
}

func normalizeClientName(v string) string {
	v = safeNameRE.ReplaceAllString(v, "-")
	if v == "" {
		return "default"
	}
	return v
}
