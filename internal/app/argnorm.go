package app

import "strings"

func normalizeOnePositionalArgs(args []string) []string {
	if len(args) < 2 {
		return args
	}
	first := strings.TrimSpace(args[0])
	if strings.HasPrefix(first, "-") {
		return args
	}
	hasFlagAfter := false
	for _, part := range args[1:] {
		if strings.HasPrefix(strings.TrimSpace(part), "-") {
			hasFlagAfter = true
			break
		}
	}
	if !hasFlagAfter {
		return args
	}
	out := make([]string, 0, len(args))
	out = append(out, args[1:]...)
	out = append(out, args[0])
	return out
}

func normalizeTwoPositionalArgs(args []string) []string {
	if len(args) < 3 {
		return args
	}
	first := strings.TrimSpace(args[0])
	second := strings.TrimSpace(args[1])
	if strings.HasPrefix(first, "-") || strings.HasPrefix(second, "-") {
		return args
	}
	hasFlagAfter := false
	for _, part := range args[2:] {
		if strings.HasPrefix(strings.TrimSpace(part), "-") {
			hasFlagAfter = true
			break
		}
	}
	if !hasFlagAfter {
		return args
	}
	out := make([]string, 0, len(args))
	out = append(out, args[2:]...)
	out = append(out, args[0], args[1])
	return out
}
