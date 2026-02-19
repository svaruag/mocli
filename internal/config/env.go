package config

import "strings"

// LookupFunc returns (value, true) when the key is set.
type LookupFunc func(string) (string, bool)

func String(lookup LookupFunc, key, defaultValue string) string {
	if lookup == nil {
		return defaultValue
	}
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	v = strings.TrimSpace(v)
	if v == "" {
		return defaultValue
	}
	return v
}

func Bool(lookup LookupFunc, key string, defaultValue bool) bool {
	if lookup == nil {
		return defaultValue
	}
	v, ok := lookup(key)
	if !ok {
		return defaultValue
	}
	parsed, ok := parseBool(v)
	if !ok {
		return defaultValue
	}
	return parsed
}

func parseBool(v string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true, true
	case "0", "false", "no", "off":
		return false, true
	default:
		return false, false
	}
}
