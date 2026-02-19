package app

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	"github.com/svaruag/mocli/internal/config"
	"github.com/svaruag/mocli/internal/exitcode"
	"github.com/svaruag/mocli/internal/outfmt"
	"github.com/svaruag/mocli/internal/version"
)

var errUsage = errors.New("usage")

type Globals struct {
	JSON    bool
	Plain   bool
	Force   bool
	NoInput bool
	Help    bool
	Account string
	Client  string
	Color   string
	Version bool
}

type parseResult struct {
	Globals Globals
	Rest    []string
}

type runtimeState struct {
	globals Globals
	stdout  io.Writer
	stderr  io.Writer
	lookup  config.LookupFunc
}

func Run(args []string, stdout, stderr io.Writer, lookup config.LookupFunc) int {
	parsed, err := parseGlobals(args, lookup)
	rt := &runtimeState{globals: parsed.Globals, stdout: stdout, stderr: stderr, lookup: lookup}
	if err != nil {
		return rt.fail("usage_error", err.Error(), "Run 'mo help' for usage.", exitcode.UsageError)
	}

	if parsed.Globals.Version {
		_, _ = fmt.Fprintln(stdout, version.Value)
		return exitcode.Success
	}
	if parsed.Globals.Help {
		_, _ = fmt.Fprint(stdout, rootHelp())
		return exitcode.Success
	}

	if len(parsed.Rest) == 0 || isHelpToken(parsed.Rest[0]) {
		_, _ = fmt.Fprint(stdout, rootHelp())
		return exitcode.Success
	}

	cmd := strings.ToLower(strings.TrimSpace(parsed.Rest[0]))
	if !isKnownCommand(cmd) {
		return rt.fail("unknown_command", fmt.Sprintf("unknown command %q", cmd), "Run 'mo help' to list commands.", exitcode.UsageError)
	}
	if !commandAllowed(cmd, lookup) {
		return rt.fail("command_disabled", fmt.Sprintf("command %q is disabled by MO_ENABLE_COMMANDS", cmd), "Update MO_ENABLE_COMMANDS to include this command.", exitcode.CommandDisabled)
	}

	rest := parsed.Rest[1:]
	switch cmd {
	case "version":
		_, _ = fmt.Fprintln(stdout, version.Value)
		return exitcode.Success
	case "auth":
		return runAuth(rt, rest)
	case "mail":
		return runMail(rt, rest)
	case "calendar":
		return runCalendar(rt, rest)
	case "tasks":
		return runTasks(rt, rest)
	case "config":
		return runConfig(rt, rest)
	default:
		return rt.fail("usage_error", fmt.Sprintf("unsupported command %q", cmd), "Run 'mo help' to list commands.", exitcode.UsageError)
	}
}

func parseGlobals(args []string, lookup config.LookupFunc) (parseResult, error) {
	defaults := Globals{
		JSON:    config.Bool(lookup, "MO_JSON", false),
		Plain:   config.Bool(lookup, "MO_PLAIN", false),
		Account: config.String(lookup, "MO_ACCOUNT", ""),
		Client:  config.String(lookup, "MO_CLIENT", ""),
		Color:   strings.ToLower(config.String(lookup, "MO_COLOR", "auto")),
	}

	fs := flag.NewFlagSet("mo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	g := defaults
	fs.BoolVar(&g.JSON, "json", defaults.JSON, "Output JSON")
	fs.BoolVar(&g.Plain, "plain", defaults.Plain, "Output plain text")
	fs.BoolVar(&g.Force, "force", false, "Skip confirmations")
	fs.BoolVar(&g.NoInput, "no-input", false, "Disable interactive prompts")
	fs.BoolVar(&g.Help, "help", false, "Show help")
	fs.BoolVar(&g.Help, "h", false, "Show help")
	fs.StringVar(&g.Account, "account", defaults.Account, "Account identifier")
	fs.StringVar(&g.Client, "client", defaults.Client, "OAuth client name")
	fs.StringVar(&g.Color, "color", defaults.Color, "Color mode: auto|always|never")
	fs.BoolVar(&g.Version, "version", false, "Print version")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			g.Help = true
			return parseResult{Globals: g}, nil
		}
		return parseResult{Globals: g}, fmt.Errorf("parse flags: %w", errUsage)
	}

	g.Color = strings.ToLower(strings.TrimSpace(g.Color))
	if g.Color == "" {
		g.Color = "auto"
	}
	if g.Color != "auto" && g.Color != "always" && g.Color != "never" {
		return parseResult{Globals: g}, fmt.Errorf("invalid --color value %q (allowed: auto|always|never)", g.Color)
	}
	if g.JSON && g.Plain {
		return parseResult{Globals: g}, fmt.Errorf("--json and --plain cannot be used together")
	}

	return parseResult{Globals: g, Rest: fs.Args()}, nil
}

func (rt *runtimeState) jsonMode() bool {
	if rt.globals.Plain {
		return false
	}
	if rt.globals.JSON {
		return true
	}
	// JSON-first default.
	return true
}

func (rt *runtimeState) fail(code, message, hint string, ec int) int {
	_ = outfmt.WriteError(rt.stderr, rt.jsonMode(), outfmt.ErrorPayload{
		Code:    code,
		Message: message,
		Hint:    hint,
	})
	return ec
}

func (rt *runtimeState) writeJSON(v any) int {
	if err := outfmt.WriteJSON(rt.stdout, v); err != nil {
		return rt.fail("write_failed", "failed to write output", err.Error(), exitcode.UsageError)
	}
	return exitcode.Success
}

func (rt *runtimeState) failErr(err error) int {
	if err == nil {
		return exitcode.Success
	}
	ae, ok := err.(*appError)
	if ok {
		return rt.fail(ae.Code, ae.Message, ae.Hint, ae.Exit)
	}
	return rt.fail("command_failed", err.Error(), "", exitcode.UsageError)
}

func commandAllowed(cmd string, lookup config.LookupFunc) bool {
	if cmd == "help" || cmd == "version" {
		return true
	}
	raw := config.String(lookup, "MO_ENABLE_COMMANDS", "")
	if strings.TrimSpace(raw) == "" {
		return true
	}
	allow := parseAllowlist(raw)
	_, ok := allow[cmd]
	return ok
}

func parseAllowlist(raw string) map[string]struct{} {
	out := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		k := strings.ToLower(strings.TrimSpace(part))
		if k != "" {
			out[k] = struct{}{}
		}
	}
	return out
}

func isKnownCommand(cmd string) bool {
	switch cmd {
	case "auth", "mail", "calendar", "tasks", "config", "version":
		return true
	default:
		return false
	}
}

func isHelpToken(v string) bool {
	s := strings.TrimSpace(strings.ToLower(v))
	return s == "help" || s == "--help" || s == "-h"
}

func rootHelp() string {
	return strings.TrimSpace(`mo - Outlook-focused CLI for agents

Usage:
  mo [global flags] <command> [args...]
  mo help

Commands:
  auth       Authentication and account setup
  mail       Mailbox commands
  calendar   Calendar commands
  tasks      Task commands
  config     Local configuration commands
  version    Print version

Help:
  mo <command> help
  mo <command> --help

Global flags:
  --json
  --plain
  --force
  --no-input
  --account <id>
  --client <name>
  --color auto|always|never
  --version
`) + "\n"
}

func groupHelp(group string) string {
	switch group {
	case "auth":
		return strings.TrimSpace(`auth commands: credentials, add, status, list, remove

Usage:
  mo auth credentials <path>
  mo auth credentials list
  mo auth add <email> [--device]
  mo auth status
  mo auth list
  mo auth remove <email>`) + "\n"
	case "mail":
		return strings.TrimSpace(`mail commands: list, get, send

Usage:
  mo mail list [--max N] [--page TOKEN] [--from RFC3339] [--to RFC3339] [--folder ID_OR_NAME]
  mo mail get <message-id>
  mo mail send --to <emails> --subject <text> --body <text> [--cc ...] [--bcc ...] [--body-html]`) + "\n"
	case "calendar":
		return strings.TrimSpace(`calendar commands: list, create, update, delete

Usage:
  mo calendar list [--from RFC3339 --to RFC3339] [--max N] [--page TOKEN]
  mo calendar create --summary <text> --from <RFC3339> --to <RFC3339> [--description ...] [--location ...] [--attendees ...]
  mo calendar update <event-id> [--summary ...] [--from ...] [--to ...] [--description ...] [--location ...] [--attendees ...]
  mo calendar delete <event-id>`) + "\n"
	case "tasks":
		return strings.TrimSpace(`tasks commands: list, create, update, complete, delete

Usage:
  mo tasks list [--list-id ID] [--max N] [--page TOKEN]
  mo tasks create --title <text> [--list-id ID] [--body ...] [--due RFC3339] [--status ...] [--importance ...]
  mo tasks update <task-id> [--list-id ID] [--title ...] [--body ...] [--due ...] [--status ...] [--importance ...]
  mo tasks complete <task-id> [--list-id ID]
  mo tasks delete <task-id> [--list-id ID]`) + "\n"
	case "config":
		return strings.TrimSpace(`config commands: list, get, set, unset, path

Usage:
  mo config list
  mo config get <key>
  mo config set <key> <value>
  mo config unset <key>
  mo config path`) + "\n"
	default:
		return "help\n"
	}
}
