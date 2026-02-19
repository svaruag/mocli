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
