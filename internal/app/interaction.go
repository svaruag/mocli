package app

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

func confirmAction(rt *runtimeState, prompt string) (bool, error) {
	if rt.globals.Force {
		return true, nil
	}
	if rt.globals.NoInput {
		return false, usageError("confirmation required", "Re-run with --force to skip confirmations.")
	}
	_, _ = fmt.Fprintf(rt.stderr, "%s [y/N]: ", prompt)
	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}
	answer := strings.ToLower(strings.TrimSpace(line))
	return answer == "y" || answer == "yes", nil
}
