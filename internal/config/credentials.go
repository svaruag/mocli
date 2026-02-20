package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

type Credentials struct {
	ClientID string `json:"client_id"`
	Tenant   string `json:"tenant"`
}

func (c *Credentials) Normalize() {
	c.ClientID = strings.TrimSpace(c.ClientID)
	c.Tenant = strings.TrimSpace(c.Tenant)
	if c.Tenant == "" {
		c.Tenant = "common"
	}
}

func (c Credentials) Validate() error {
	if strings.TrimSpace(c.ClientID) == "" {
		return errors.New("missing client_id/appId")
	}
	return nil
}

func LoadCredentials(client string) (Credentials, error) {
	path, err := CredentialsPath(client)
	if err != nil {
		return Credentials{}, err
	}
	info, err := os.Stat(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("read credentials: %w", err)
	}
	if err := validatePrivateFile(path, info.Mode()); err != nil {
		return Credentials{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Credentials{}, fmt.Errorf("read credentials: %w", err)
	}
	cred, err := ParseCredentialsJSON(data)
	if err != nil {
		return Credentials{}, err
	}
	return cred, nil
}

func SaveCredentials(client string, cred Credentials) error {
	if err := cred.Validate(); err != nil {
		return err
	}
	cred.Normalize()
	_, err := EnsureBaseDir()
	if err != nil {
		return err
	}
	path, err := CredentialsPath(client)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cred, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	if err := os.WriteFile(path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	return nil
}

func ParseCredentialsJSON(data []byte) (Credentials, error) {
	var direct Credentials
	if err := json.Unmarshal(data, &direct); err == nil {
		if direct.ClientID != "" {
			direct.Normalize()
			if err := direct.Validate(); err != nil {
				return Credentials{}, err
			}
			return direct, nil
		}
	}

	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return Credentials{}, fmt.Errorf("parse credentials json: %w", err)
	}

	clientID := pickString(raw,
		"client_id",
		"clientId",
		"appId",
		"app_id",
	)
	tenant := pickString(raw,
		"tenant",
		"tenant_id",
		"tenantId",
	)

	if clientID == "" {
		if nested, ok := raw["installed"].(map[string]any); ok {
			clientID = pickString(nested, "client_id")
		}
	}
	if clientID == "" {
		if nested, ok := raw["web"].(map[string]any); ok {
			clientID = pickString(nested, "client_id")
		}
	}

	cred := Credentials{ClientID: clientID, Tenant: tenant}
	cred.Normalize()
	if err := cred.Validate(); err != nil {
		return Credentials{}, err
	}
	return cred, nil
}

func pickString(m map[string]any, keys ...string) string {
	for _, k := range keys {
		v, ok := m[k]
		if !ok {
			continue
		}
		s, ok := v.(string)
		if !ok {
			continue
		}
		s = strings.TrimSpace(s)
		if s != "" {
			return s
		}
	}
	return ""
}
