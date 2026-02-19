package migrate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/secrets"
)

const markerFile = ".mocli-migrated"

type legacyConfig struct {
	DefaultAccount string `json:"default_account"`
	DefaultClient  string `json:"default_client"`
	Accounts       []struct {
		Email  string `json:"email"`
		Client string `json:"client"`
	} `json:"accounts"`
}

// Run performs a best-effort one-time migration from a legacy config directory
// into the current MO_CONFIG_DIR / default mocli base directory.
func Run(stderr io.Writer) error {
	newBase, err := config.BaseDir()
	if err != nil {
		return err
	}
	newBase = filepath.Clean(newBase)

	if migratedAlready(newBase) {
		return nil
	}
	if hasCurrentState(newBase) {
		if err := writeMarker(newBase); err != nil {
			return err
		}
		return nil
	}

	legacyBase, err := discoverLegacyBase(newBase)
	if err != nil {
		return err
	}
	if legacyBase == "" {
		return nil
	}

	if err := copyTree(legacyBase, newBase); err != nil {
		return fmt.Errorf("copy legacy config: %w", err)
	}
	if err := migrateSecretToolTokens(legacyBase, newBase); err != nil {
		// best-effort: file backend migration is already covered by copyTree
		_, _ = fmt.Fprintf(stderr, "warning: token migration via secret-tool was incomplete: %v\n", err)
	}
	if err := writeMarker(newBase); err != nil {
		return err
	}

	_, _ = fmt.Fprintf(stderr, "mocli: migrated local config from %s to %s\n", legacyBase, newBase)
	return nil
}

func migratedAlready(newBase string) bool {
	if newBase == "" {
		return false
	}
	_, err := os.Stat(filepath.Join(newBase, markerFile))
	return err == nil
}

func hasCurrentState(newBase string) bool {
	checks := []string{
		filepath.Join(newBase, "config.json"),
		filepath.Join(newBase, "credentials.json"),
		filepath.Join(newBase, "keyring"),
	}
	for _, p := range checks {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

func writeMarker(newBase string) error {
	if err := os.MkdirAll(newBase, 0o700); err != nil {
		return err
	}
	path := filepath.Join(newBase, markerFile)
	return os.WriteFile(path, []byte("ok\n"), 0o600)
}

func discoverLegacyBase(newBase string) (string, error) {
	baseConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	entries, err := os.ReadDir(baseConfigDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", nil
		}
		return "", err
	}

	newName := filepath.Base(newBase)
	candidates := make([]candidate, 0, len(entries))
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := strings.TrimSpace(e.Name())
		if name == "" || name == newName {
			continue
		}
		dir := filepath.Join(baseConfigDir, name)
		score, ok := scoreLegacyDir(dir)
		if !ok {
			continue
		}
		candidates = append(candidates, candidate{Path: dir, Score: score})
	}

	if len(candidates) == 0 {
		return "", nil
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].Score == candidates[j].Score {
			return candidates[i].Path < candidates[j].Path
		}
		return candidates[i].Score > candidates[j].Score
	})
	return candidates[0].Path, nil
}

type candidate struct {
	Path  string
	Score int
}

func scoreLegacyDir(dir string) (int, bool) {
	score := 0

	credPath := filepath.Join(dir, "credentials.json")
	credData, err := os.ReadFile(credPath)
	if err != nil {
		return 0, false
	}
	if _, err := config.ParseCredentialsJSON(credData); err != nil {
		return 0, false
	}
	score += 4

	if _, err := os.Stat(filepath.Join(dir, "config.json")); err == nil {
		score += 2
	}
	if info, err := os.Stat(filepath.Join(dir, "keyring")); err == nil && info.IsDir() {
		score += 2
	}
	if matches, _ := filepath.Glob(filepath.Join(dir, "credentials-*.json")); len(matches) > 0 {
		score += 1
	}
	return score, true
}

func copyTree(src, dst string) error {
	if err := os.MkdirAll(dst, 0o700); err != nil {
		return err
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if rel == "." {
			return nil
		}
		target := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		if _, err := os.Stat(target); err == nil {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := fs.FileMode(0o600)
		if info, err := d.Info(); err == nil {
			mode = info.Mode() & 0o777
			if mode == 0 {
				mode = 0o600
			}
		}
		return os.WriteFile(target, b, mode)
	})
}

func migrateSecretToolTokens(legacyBase, newBase string) error {
	if !secretToolAvailable() {
		return nil
	}
	cfg, err := loadLegacyConfig(filepath.Join(legacyBase, "config.json"))
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}

	legacyService := filepath.Base(legacyBase)
	newService := filepath.Base(newBase)
	if legacyService == "" || newService == "" || legacyService == newService {
		return nil
	}

	keys := tokenKeysFromConfig(cfg)
	for _, key := range keys {
		value, err := secretLookup(legacyService, key)
		if err != nil {
			continue
		}
		if err := secretStore(newService, key, value); err != nil {
			return err
		}
	}
	return nil
}

func loadLegacyConfig(path string) (legacyConfig, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return legacyConfig{}, err
	}
	var cfg legacyConfig
	if err := json.Unmarshal(b, &cfg); err != nil {
		return legacyConfig{}, err
	}
	return cfg, nil
}

func tokenKeysFromConfig(cfg legacyConfig) []string {
	seen := map[string]struct{}{}
	keys := make([]string, 0, len(cfg.Accounts)+1)
	add := func(client, email string) {
		client = strings.TrimSpace(client)
		email = strings.ToLower(strings.TrimSpace(email))
		if client == "" {
			client = "default"
		}
		if email == "" {
			return
		}
		k := secrets.TokenKey(client, email)
		if _, ok := seen[k]; ok {
			return
		}
		seen[k] = struct{}{}
		keys = append(keys, k)
	}
	for _, a := range cfg.Accounts {
		add(a.Client, a.Email)
	}
	add(cfg.DefaultClient, cfg.DefaultAccount)
	sort.Strings(keys)
	return keys
}

func secretToolAvailable() bool {
	_, err := exec.LookPath("secret-tool")
	return err == nil
}

func secretLookup(service, key string) (string, error) {
	cmd := exec.Command("secret-tool", "lookup", "service", service, "key", key)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	value := strings.TrimSpace(string(out))
	if value == "" {
		return "", errors.New("empty value")
	}
	return value, nil
}

func secretStore(service, key, value string) error {
	cmd := exec.Command("secret-tool", "store", "--label", "mocli token", "service", service, "key", key)
	cmd.Stdin = strings.NewReader(value)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("secret-tool store failed: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return nil
}
