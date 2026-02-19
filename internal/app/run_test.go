package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/svaruag/mocli/internal/exitcode"
)

func TestParseGlobalsPrecedence(t *testing.T) {
	env := fakeEnv(map[string]string{
		"MO_ACCOUNT": "env-account",
		"MO_CLIENT":  "env-client",
		"MO_JSON":    "1",
		"MO_COLOR":   "always",
	})

	parsed, err := parseGlobals([]string{"--account", "flag-account", "--color", "never", "version"}, env)
	if err != nil {
		t.Fatalf("parseGlobals returned error: %v", err)
	}

	if parsed.Globals.Account != "flag-account" {
		t.Fatalf("expected flag account to override env, got %q", parsed.Globals.Account)
	}
	if parsed.Globals.Client != "env-client" {
		t.Fatalf("expected env client, got %q", parsed.Globals.Client)
	}
	if parsed.Globals.Color != "never" {
		t.Fatalf("expected color never, got %q", parsed.Globals.Color)
	}
	if !parsed.Globals.JSON {
		t.Fatalf("expected MO_JSON=1 to enable JSON mode")
	}
}

func TestRunUnknownCommandJSONMode(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"--json", "bogus"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.UsageError {
		t.Fatalf("expected usage exit code, got %d", exit)
	}

	var payload map[string]any
	if err := json.Unmarshal(errOut.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON error payload, got %q (%v)", errOut.String(), err)
	}

	errObj, ok := payload["error"].(map[string]any)
	if !ok {
		t.Fatalf("expected error object in payload: %v", payload)
	}

	if errObj["code"] != "unknown_command" {
		t.Fatalf("expected unknown_command, got %v", errObj["code"])
	}
}

func TestRunUnknownCommandPlainMode(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"--plain", "bogus"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.UsageError {
		t.Fatalf("expected usage exit code, got %d", exit)
	}

	got := errOut.String()
	if !strings.Contains(got, "error:") {
		t.Fatalf("expected plain error output, got %q", got)
	}
	if strings.Contains(got, "\"error\"") {
		t.Fatalf("expected non-JSON plain output, got %q", got)
	}
}

func TestRunAllowlistBlocksCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"--json", "mail", "list"}, &out, &errOut, fakeEnv(map[string]string{
		"MO_ENABLE_COMMANDS": "calendar,tasks",
	}))
	if exit != exitcode.CommandDisabled {
		t.Fatalf("expected command disabled exit code, got %d", exit)
	}

	var payload map[string]any
	if err := json.Unmarshal(errOut.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON error payload, got %q (%v)", errOut.String(), err)
	}
	errObj := payload["error"].(map[string]any)
	if errObj["code"] != "command_disabled" {
		t.Fatalf("expected command_disabled error code, got %v", errObj["code"])
	}
}

func TestRunVersionCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"version"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.Success {
		t.Fatalf("expected success exit code, got %d", exit)
	}
	if strings.TrimSpace(out.String()) == "" {
		t.Fatalf("expected version output, got empty output")
	}
}

func TestRunGroupHelp(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"mail"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.Success {
		t.Fatalf("expected success exit code, got %d", exit)
	}
	if !strings.Contains(out.String(), "mail commands") {
		t.Fatalf("expected group help output, got %q", out.String())
	}
}

func TestRunRootHelpFlag(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"--help"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.Success {
		t.Fatalf("expected success exit code, got %d", exit)
	}
	if !strings.Contains(out.String(), "Commands:") {
		t.Fatalf("expected root help output, got %q", out.String())
	}
}

func TestRunRootHelpCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"help"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.Success {
		t.Fatalf("expected success exit code, got %d", exit)
	}
	if !strings.Contains(out.String(), "Commands:") {
		t.Fatalf("expected root help output, got %q", out.String())
	}
}

func TestRunMailHelpCommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"mail", "help"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.Success {
		t.Fatalf("expected success exit code, got %d", exit)
	}
	if !strings.Contains(out.String(), "mail commands:") {
		t.Fatalf("expected mail help output, got %q", out.String())
	}
}

func TestRunUnknownSubcommand(t *testing.T) {
	var out bytes.Buffer
	var errOut bytes.Buffer

	exit := Run([]string{"--json", "mail", "nope"}, &out, &errOut, fakeEnv(nil))
	if exit != exitcode.UsageError {
		t.Fatalf("expected usage exit code, got %d", exit)
	}

	var payload map[string]any
	if err := json.Unmarshal(errOut.Bytes(), &payload); err != nil {
		t.Fatalf("expected JSON error payload, got %q (%v)", errOut.String(), err)
	}
	errObj := payload["error"].(map[string]any)
	if errObj["code"] != "usage_error" {
		t.Fatalf("expected usage_error code, got %v", errObj["code"])
	}
}

func TestParseGlobalsRejectsInvalidColor(t *testing.T) {
	_, err := parseGlobals([]string{"--color", "rainbow", "version"}, fakeEnv(nil))
	if err == nil {
		t.Fatalf("expected invalid color to fail parsing")
	}
}

func fakeEnv(values map[string]string) func(string) (string, bool) {
	return func(key string) (string, bool) {
		v, ok := values[key]
		return v, ok
	}
}
