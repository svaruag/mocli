# Mocli Spec (MVP)

## Goal

Provide a small, reliable, JSON-first CLI for Outlook-centric Microsoft Graph workflows used by agents.

Implemented command groups:

- `auth`
- `mail`
- `calendar`
- `tasks`
- `config`
- `version`

## Non-Goals

- broad Microsoft 365 tenant administration
- workload coverage beyond mail/calendar/tasks
- backwards compatibility guarantees prior to first public release

## Runtime

- Language: Go (`>=1.22`)
- Binary: `mo`

## Global Flags

- `--json`
- `--plain`
- `--force`
- `--no-input`
- `--account`
- `--client`
- `--color=auto|always|never`
- `--version`

## Auth Model

- Delegated OAuth only
- Explicit app credentials required (`mo auth credentials <path>`)
- Refresh-token based operation after one-time login

## Output Contract

- JSON output is default
- Plain output available via `--plain`
- Error schema:

```json
{
  "error": {
    "code": "...",
    "message": "...",
    "hint": "..."
  }
}
```

## Exit Codes

- `0`: success
- `2`: usage error
- `3`: auth required
- `4`: permission denied
- `5`: not found
- `6`: command disabled
- `10`: transient error
- `12`: not implemented

## Security Model

- refresh tokens in keyring backend (`auto|keychain|file`)
- no plaintext token files
- allowlist support for restricted agent execution (`MO_ENABLE_COMMANDS`)
