package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

type AccountRecord struct {
	Email     string `json:"email"`
	Client    string `json:"client"`
	CreatedAt string `json:"created_at,omitempty"`
}

type AppConfig struct {
	KeyringBackend string          `json:"keyring_backend,omitempty"`
	DefaultAccount string          `json:"default_account,omitempty"`
	DefaultClient  string          `json:"default_client,omitempty"`
	Accounts       []AccountRecord `json:"accounts,omitempty"`
}

func LoadAppConfig() (AppConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return AppConfig{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return AppConfig{}, nil
		}
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}
	if err := validatePrivateFile(path, info.Mode()); err != nil {
		return AppConfig{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return AppConfig{}, fmt.Errorf("read config: %w", err)
	}
	var cfg AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return AppConfig{}, fmt.Errorf("parse config: %w", err)
	}
	cfg.normalize()
	return cfg, nil
}

func SaveAppConfig(cfg AppConfig) error {
	dir, err := EnsureBaseDir()
	if err != nil {
		return err
	}
	cfg.normalize()
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	path := dir + string(os.PathSeparator) + "config.json"
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	return nil
}

func (c *AppConfig) normalize() {
	c.KeyringBackend = strings.ToLower(strings.TrimSpace(c.KeyringBackend))
	c.DefaultAccount = strings.ToLower(strings.TrimSpace(c.DefaultAccount))
	c.DefaultClient = normalizeClientName(strings.TrimSpace(c.DefaultClient))
	if c.DefaultClient == "" {
		c.DefaultClient = "default"
	}

	clean := make([]AccountRecord, 0, len(c.Accounts))
	seen := map[string]struct{}{}
	for _, a := range c.Accounts {
		email := strings.ToLower(strings.TrimSpace(a.Email))
		client := normalizeClientName(strings.TrimSpace(a.Client))
		if email == "" {
			continue
		}
		k := client + ":" + email
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		clean = append(clean, AccountRecord{Email: email, Client: client, CreatedAt: a.CreatedAt})
	}
	sort.Slice(clean, func(i, j int) bool {
		if clean[i].Client == clean[j].Client {
			return clean[i].Email < clean[j].Email
		}
		return clean[i].Client < clean[j].Client
	})
	c.Accounts = clean
}

func (c *AppConfig) UpsertAccount(client, email string) {
	client = normalizeClientName(client)
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return
	}
	for i := range c.Accounts {
		if c.Accounts[i].Client == client && c.Accounts[i].Email == email {
			if c.Accounts[i].CreatedAt == "" {
				c.Accounts[i].CreatedAt = time.Now().UTC().Format(time.RFC3339)
			}
			return
		}
	}
	c.Accounts = append(c.Accounts, AccountRecord{
		Email:     email,
		Client:    client,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
	c.normalize()
}

func (c *AppConfig) RemoveAccount(client, email string) {
	client = normalizeClientName(client)
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" {
		return
	}
	keep := c.Accounts[:0]
	for _, a := range c.Accounts {
		if !(a.Client == client && a.Email == email) {
			keep = append(keep, a)
		}
	}
	c.Accounts = keep
	if c.DefaultAccount == email {
		c.DefaultAccount = ""
	}
}

func (c *AppConfig) AccountsForClient(client string) []AccountRecord {
	client = normalizeClientName(client)
	out := make([]AccountRecord, 0, len(c.Accounts))
	for _, a := range c.Accounts {
		if a.Client == client {
			out = append(out, a)
		}
	}
	return out
}
