package app

import (
	"fmt"
	"strings"

	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/exitcode"
)

func runConfig(rt *runtimeState, args []string) int {
	if len(args) == 0 || isHelpToken(args[0]) {
		_, _ = fmt.Fprint(rt.stdout, groupHelp("config"))
		return exitcode.Success
	}

	sub := strings.ToLower(strings.TrimSpace(args[0]))
	rest := args[1:]
	switch sub {
	case "list":
		cfg, err := config.LoadAppConfig()
		if err != nil {
			return rt.failErr(err)
		}
		return rt.writeJSON(map[string]any{
			"keyring_backend": cfg.KeyringBackend,
			"default_account": cfg.DefaultAccount,
			"default_client":  cfg.DefaultClient,
		})
	case "get":
		if len(rest) != 1 {
			return rt.failErr(usageError("config key is required", "Usage: mo config get <key>"))
		}
		cfg, err := config.LoadAppConfig()
		if err != nil {
			return rt.failErr(err)
		}
		key := strings.ToLower(strings.TrimSpace(rest[0]))
		value, ok := getConfigKey(cfg, key)
		if !ok {
			return rt.failErr(usageError("unknown config key", "Allowed: keyring_backend, default_account, default_client"))
		}
		return rt.writeJSON(map[string]any{"key": key, "value": value})
	case "set":
		if len(rest) != 2 {
			return rt.failErr(usageError("config key and value are required", "Usage: mo config set <key> <value>"))
		}
		cfg, err := config.LoadAppConfig()
		if err != nil {
			return rt.failErr(err)
		}
		if err := setConfigKey(&cfg, rest[0], rest[1]); err != nil {
			return rt.failErr(err)
		}
		if err := config.SaveAppConfig(cfg); err != nil {
			return rt.failErr(err)
		}
		return rt.writeJSON(map[string]any{"saved": true, "key": strings.ToLower(rest[0]), "value": rest[1]})
	case "unset":
		if len(rest) != 1 {
			return rt.failErr(usageError("config key is required", "Usage: mo config unset <key>"))
		}
		cfg, err := config.LoadAppConfig()
		if err != nil {
			return rt.failErr(err)
		}
		if err := setConfigKey(&cfg, rest[0], ""); err != nil {
			return rt.failErr(err)
		}
		if err := config.SaveAppConfig(cfg); err != nil {
			return rt.failErr(err)
		}
		return rt.writeJSON(map[string]any{"unset": true, "key": strings.ToLower(rest[0])})
	case "path":
		path, err := config.ConfigPath()
		if err != nil {
			return rt.failErr(err)
		}
		base, err := config.BaseDir()
		if err != nil {
			return rt.failErr(err)
		}
		return rt.writeJSON(map[string]any{"config_path": path, "base_dir": base})
	default:
		return rt.failErr(usageError(fmt.Sprintf("unknown config subcommand %q", sub), "Run 'mo config help' for usage."))
	}
}

func getConfigKey(cfg config.AppConfig, key string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "keyring_backend":
		return cfg.KeyringBackend, true
	case "default_account":
		return cfg.DefaultAccount, true
	case "default_client":
		return cfg.DefaultClient, true
	default:
		return "", false
	}
}

func setConfigKey(cfg *config.AppConfig, key, value string) error {
	k := strings.ToLower(strings.TrimSpace(key))
	v := strings.TrimSpace(value)
	switch k {
	case "keyring_backend":
		if v != "" {
			switch strings.ToLower(v) {
			case "auto", "keychain", "file":
			default:
				return usageError("invalid keyring_backend", "Allowed values: auto, keychain, file")
			}
		}
		cfg.KeyringBackend = strings.ToLower(v)
	case "default_account":
		cfg.DefaultAccount = strings.ToLower(v)
	case "default_client":
		if v == "" {
			cfg.DefaultClient = ""
		} else {
			cfg.DefaultClient = v
		}
	default:
		return usageError("unknown config key", "Allowed: keyring_backend, default_account, default_client")
	}
	return nil
}
