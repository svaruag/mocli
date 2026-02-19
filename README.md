# Mocli (`mo`)

Outlook and Microsoft To Do in your terminal.

Fast, script-friendly CLI for Microsoft Graph delegated workflows: mail, calendar, and tasks. JSON-first output, multi-account support, secure refresh-token storage, and agent-safe controls are built in.

## Features

- **Mail** - list messages, fetch message details, send mail
- **Calendar** - list/create/update/delete events
- **Tasks** - list/create/update/complete/delete Microsoft To Do tasks
- **Authentication** - browser and device OAuth flows
- **Multiple clients/accounts** - isolate credentials and tokens per client profile
- **Secure token storage** - OS keyring or encrypted file backend
- **Agent controls** - JSON-first output, stable exit codes, allowlisted command execution
- **Live smoke checks** - end-to-end validation script for mail/tasks/calendar workflows

## Installation

### Install from source (recommended)

```bash
git clone <your-fork-or-repo-url>
cd mocli
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/mo" ./cmd/mo
export PATH="$HOME/.local/bin:$PATH"
hash -r
```

Run:

```bash
mo --help
```

## Quick Start

### 1. Create OAuth app credentials (required)

Mocli requires your own Microsoft Entra app registration.

Follow:

- `docs/setup-entra-app.md`

You need these values:

- `client_id` (application ID)
- `tenant` (`common`, `consumers`, or tenant GUID)

### 2. Save credentials

Create `entra-app.json`:

```json
{
  "client_id": "00000000-0000-0000-0000-000000000000",
  "tenant": "common"
}
```

Store credentials:

```bash
mo auth credentials ./entra-app.json
```

For multiple client profiles:

```bash
mo --client work auth credentials ./work-entra-app.json
mo auth credentials list
```

### 3. Configure token storage (required before `auth add`)

Mocli stores refresh tokens in either:

- system keychain (`keychain` backend), or
- encrypted files (`file` backend).

Check what backend will be used:

```bash
mo auth status
```

If `keyring_backend.resolved` is `file`, set a password:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='choose-a-strong-password'
```

You can put these exports in your shell profile for local development.

### 4. Authorize account

Device flow (recommended for headless/remote):

```bash
mo auth add you@outlook.com --device
```

Browser flow:

```bash
mo auth add you@outlook.com
```

### 5. Test auth + run first commands

```bash
mo auth status
mo mail list --max 5
mo tasks list --max 5
```

## Authentication & Secrets

### OAuth client credentials

- Stored under config dir:
  - `credentials.json` (default client)
  - `credentials-<client>.json` (named client)
- Write command:
  - `mo auth credentials <path>`
- List command:
  - `mo auth credentials list`

### Refresh tokens

Stored via keyring backend:

- `auto` (default)
- `keychain`
- `file` (encrypted files, requires password)

If `auto` resolves to `file`, `MO_KEYRING_PASSWORD` is required before `mo auth add`.

File backend for non-interactive usage:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='strong-password'
```

### Account and client selection

Client selection order:

1. `--client` / `MO_CLIENT`
2. `default_client` in config
3. `default`

Account selection order:

1. `--account` / `MO_ACCOUNT`
2. `default_account` in config
3. only account for selected client (if exactly one)

See `docs/authentication.md` for details.

## Permissions

Required delegated Microsoft Graph permissions:

- `User.Read`
- `Mail.Read`
- `Mail.Send`
- `Calendars.ReadWrite`
- `Tasks.ReadWrite`

OIDC scopes:

- `openid`
- `profile`
- `offline_access`

See `docs/permissions.md` for scope-to-command mapping.

## Configuration

Config commands:

```bash
mo config list
mo config get <key>
mo config set <key> <value>
mo config unset <key>
mo config path
```

Config keys:

- `keyring_backend` (`auto|keychain|file`)
- `default_account`
- `default_client`

## Security

- Refresh tokens are never written in plaintext JSON files.
- Secrets are stored in keyring backend.
- Use command allowlist for sandboxed agents:

```bash
export MO_ENABLE_COMMANDS=mail,calendar,tasks
```

See `docs/security.md` for operational guidance.

## Commands

### Auth

- `mo auth credentials <path> [--client-id ...] [--tenant ...]`
- `mo auth credentials list`
- `mo auth add <email> [--device] [--timeout ...] [--force-consent]`
- `mo auth status`
- `mo auth list`
- `mo auth remove <email>`

### Mail

- `mo mail list [--max N] [--page TOKEN] [--from RFC3339] [--to RFC3339] [--folder ID_OR_NAME]`
- `mo mail get <message-id>`
- `mo mail send --to <emails> --subject <text> --body <text> [--cc ...] [--bcc ...] [--body-html] [--save-to-sent=false]`

### Calendar

- `mo calendar list [--max N] [--page TOKEN] [--from RFC3339 --to RFC3339]`
- `mo calendar create --summary <text> --from <RFC3339> --to <RFC3339> [--description ...] [--location ...] [--attendees ...]`
- `mo calendar update <event-id> [--summary ...] [--from ...] [--to ...] [--description ...] [--location ...] [--attendees ...]`
- `mo calendar delete <event-id>`

### Tasks

- `mo tasks list [--list-id ID] [--max N] [--page TOKEN]`
- `mo tasks create --title <text> [--list-id ID] [--body ...] [--due RFC3339] [--status ...] [--importance ...]`
- `mo tasks update <task-id> [--list-id ID] [--title ...] [--body ...] [--due ...] [--status ...] [--importance ...]`
- `mo tasks complete <task-id> [--list-id ID]`
- `mo tasks delete <task-id> [--list-id ID]`

See `docs/commands.md` for expanded command reference.

## Output Formats

JSON is default:

```bash
mo mail list
```

Plain output:

```bash
mo mail list --plain
```

Error contract:

```json
{
  "error": {
    "code": "auth_required",
    "message": "...",
    "hint": "..."
  }
}
```

## Examples

Mail:

```bash
mo mail list --max 10
mo mail get <message-id>
mo mail send --to you@outlook.com --subject "hello" --body "hi"
```

Calendar:

```bash
mo calendar list --from 2026-02-19T00:00:00Z --to 2026-02-20T00:00:00Z
mo calendar create --summary "Demo" --from 2026-02-19T09:00:00Z --to 2026-02-19T09:30:00Z
mo calendar update <event-id> --summary "Updated Demo"
mo calendar delete <event-id> --force
```

Tasks:

```bash
mo tasks list --max 20
mo tasks create --title "Follow up"
mo tasks update <task-id> --status inProgress
mo tasks complete <task-id>
mo tasks delete <task-id> --force
```

More workflows: `docs/examples.md`.

## Agent / Automation Usage

Non-interactive mode:

```bash
mo --no-input --force tasks list --max 20
```

Restrict allowed command groups:

```bash
export MO_ENABLE_COMMANDS=tasks,calendar
```

See `docs/automation.md`.

## Global Flags

- `--json`
- `--plain`
- `--force`
- `--no-input`
- `--account <id>`
- `--client <name>`
- `--color auto|always|never`
- `--version`

## Environment Variables

- `MO_ACCOUNT`
- `MO_CLIENT`
- `MO_ENABLE_COMMANDS`
- `MO_KEYRING_BACKEND`
- `MO_KEYRING_PASSWORD`
- `MO_CONFIG_DIR`
- `MO_JSON`
- `MO_PLAIN`
- `MO_COLOR`
- `MO_AUTH_BASE_URL` (advanced)
- `MO_GRAPH_BASE_URL` (advanced)

## Exit Codes

- `0`: success
- `2`: usage error
- `3`: auth required
- `4`: permission denied
- `5`: not found
- `6`: command disabled
- `10`: transient error
- `12`: not implemented

## Testing

```bash
make test
make smoke-live
```

See `docs/testing.md` and `docs/smoke-tests.md`.

## Documentation

Start here:

- `docs/index.md`

Core operator docs:

- `docs/setup-entra-app.md`
- `docs/authentication.md`
- `docs/permissions.md`
- `docs/commands.md`
- `docs/examples.md`
- `docs/automation.md`
- `docs/security.md`
- `docs/troubleshooting.md`
- `docs/testing.md`
- `docs/release.md`

Project internals and planning docs:

- `docs/spec.md`
- `docs/vision.md`
- `docs/roadmap.md`
- `docs/architecture.md`
- `docs/decisions/`
