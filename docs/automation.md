# Automation

## JSON Contract

Mocli defaults to JSON output for command results and errors.

Error shape:

```json
{
  "error": {
    "code": "auth_required",
    "message": "...",
    "hint": "..."
  }
}
```

Use `--plain` only when human-readable terminal output is preferred.

## Exit Codes

- `0`: success
- `2`: usage error
- `3`: auth required
- `4`: permission denied
- `5`: not found
- `6`: command disabled
- `10`: transient error
- `12`: not implemented

## Non-Interactive Execution

Use:

- `--no-input` to fail instead of prompting
- `--force` to skip confirmations

Example:

```bash
mo --json --no-input --force calendar delete <event-id>
```

## Restrict Command Surface

Limit top-level command groups available to an agent:

```bash
export MO_ENABLE_COMMANDS=mail,calendar,tasks
```

Blocked command attempts return `command_disabled` with exit code `6`.

## Stable Runtime Inputs

Environment variables commonly set by automation:

- `MO_ACCOUNT`
- `MO_CLIENT`
- `MO_CONFIG_DIR`
- `MO_KEYRING_BACKEND`
- `MO_KEYRING_PASSWORD`
- `MO_ENABLE_COMMANDS`

## Recommended Agent Loop

```bash
make test
make smoke-live
```

Run smoke checks after behavior changes in auth, Graph request handling, or command wiring.
