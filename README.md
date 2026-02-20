# Mocli (`mo`)

Microsoft Outlook + OneDrive in your terminal.

`mo` is a fast, script-friendly CLI for Microsoft Graph delegated workflows: mail, calendar, tasks, and OneDrive. It is JSON-first, multi-account aware, and designed for agentic execution.

If `mo` is not installed in your `PATH`, use `./mo` in all examples.

## Features

- Mail: list messages, fetch message details, send email
- Calendar: list/create/update/delete events
- Tasks: list/create/update/complete/delete Microsoft To Do tasks
- OneDrive: list/search/upload/download files, create folders, move/rename/delete, manage sharing
- Auth: browser and device OAuth flows
- Multi-account + multi-client profiles
- Secure token storage via OS keyring or encrypted file backend
- Agent controls: JSON output, stable exit codes, command allowlist
- Live smoke script for end-to-end validation

## Installation

### Build from source (recommended)

```bash
git clone <your-repo-url>
cd mocli
go build -o ./mo ./cmd/mo
```

Use the binary directly:

```bash
./mo --help
```

Optional: install into PATH.

```bash
mkdir -p "$HOME/.local/bin"
cp ./mo "$HOME/.local/bin/mo"
export PATH="$HOME/.local/bin:$PATH"
hash -r
mo --help
```

## Help and command discovery

- Top-level help: `mo help`
- Group help: `mo auth help`, `mo mail help`, `mo drive help`
- Version: `mo version` or `mo --version`

## Quick Start

### 1. Create a Microsoft Entra app registration

Mocli uses your own app registration (recommended for reliability and scope control).

Follow:

- `docs/setup-entra-app.md`

You will need:

- `client_id` (Application / Client ID)
- `tenant` (`common`, `consumers`, or your tenant GUID)

### 2. Save app credentials for mocli

Create `entra-app.json`:

```json
{
  "client_id": "00000000-0000-0000-0000-000000000000",
  "tenant": "common"
}
```

Store it:

```bash
mo auth credentials ./entra-app.json
```

For multiple client profiles:

```bash
mo --client work auth credentials ./work-entra-app.json
mo auth credentials list
```

### 3. Configure token storage before `auth add`

Check backend resolution:

```bash
mo auth status
```

If backend resolves to `file`, set:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='choose-a-strong-password'
```

Without `MO_KEYRING_PASSWORD`, `mo auth add` will fail on file backend.

### 4. Authorize your account

Device flow (recommended for headless/remote):

```bash
mo auth add you@outlook.com --device
```

Browser flow:

```bash
mo auth add you@outlook.com
```

If permissions were recently changed in Entra, re-consent:

```bash
mo auth add you@outlook.com --device --force-consent
```

### 5. Verify and run first commands

```bash
mo auth status
mo mail list --max 5
mo calendar list --max 5
mo tasks list --max 5
mo drive ls --max 5
```

## Required permissions

Delegated Graph permissions:

- `User.Read`
- `Mail.Read`
- `Mail.Send`
- `Calendars.ReadWrite`
- `Tasks.ReadWrite`
- `Files.ReadWrite`

OIDC scopes:

- `openid`
- `profile`
- `offline_access`

Details and mapping: `docs/permissions.md`.

## Command groups

### Auth

```bash
mo auth credentials <path> [--client-id ...] [--tenant ...]
mo auth credentials list
mo auth add <email> [--device] [--timeout ...] [--force-consent]
mo auth status
mo auth list
mo auth remove <email>
```

### Mail

```bash
mo mail list [--max N] [--page TOKEN] [--from RFC3339] [--to RFC3339] [--folder ID_OR_NAME]
mo mail get <message-id>
mo mail send --to <emails> --subject <text> --body <text> [--cc ...] [--bcc ...] [--body-html] [--save-to-sent=false]
```

### Calendar

```bash
mo calendar list [--max N] [--page TOKEN] [--from RFC3339 --to RFC3339]
mo calendar create --summary <text> --from <RFC3339> --to <RFC3339> [--description ...] [--location ...] [--attendees ...]
mo calendar update <event-id> [--summary ...] [--from ...] [--to ...] [--description ...] [--location ...] [--attendees ...]
mo calendar delete <event-id>
```

### Tasks

```bash
mo tasks list [--list-id ID] [--max N] [--page TOKEN]
mo tasks create --title <text> [--list-id ID] [--body ...] [--due RFC3339] [--status ...] [--importance ...]
mo tasks update <task-id> [--list-id ID] [--title ...] [--body ...] [--due ...] [--status ...] [--importance ...]
mo tasks complete <task-id> [--list-id ID]
mo tasks delete <task-id> [--list-id ID]
```

### Drive

```bash
mo drive ls [--parent ID] [--max N] [--page TOKEN] [--drive DRIVE_ID]
mo drive search <text> [--max N] [--page TOKEN] [--drive DRIVE_ID]
mo drive get <item-id> [--drive DRIVE_ID]
mo drive upload <local-path> [--parent ID] [--name NAME] [--conflict fail|rename|replace] [--drive DRIVE_ID]
mo drive download <item-id> [--out PATH] [--drive DRIVE_ID]
mo drive mkdir <name> [--parent ID] [--drive DRIVE_ID]
mo drive rename <item-id> <new-name> [--drive DRIVE_ID]
mo drive move <item-id> --parent <dest-id> [--drive DRIVE_ID]
mo drive delete <item-id> [--permanent] [--drive DRIVE_ID]
mo drive permissions <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]
mo drive share <item-id> --to user|domain|anyone [--email ...] [--domain ...] --role read|write [--send-invite] [--drive DRIVE_ID]
mo drive unshare <item-id> <permission-id> [--drive DRIVE_ID]
mo drive comments <item-id> [--max N] [--page TOKEN] [--drive DRIVE_ID]
mo drive comment add <item-id> --text <value> [--drive DRIVE_ID]
mo drive comment delete <item-id> <comment-id> [--drive DRIVE_ID]
mo drive drives [--max N] [--page TOKEN]
mo drive shared [--max N] [--page TOKEN]
```

Full reference: `docs/commands.md`.

## Examples

### Mail

```bash
mo mail list --max 10
mo mail get <message-id>
mo mail send --to you@outlook.com --subject "hello" --body "hi"
```

### Calendar

```bash
mo calendar list --from 2026-02-19T00:00:00Z --to 2026-02-20T00:00:00Z
mo calendar create --summary "Demo" --from 2026-02-19T09:00:00Z --to 2026-02-19T09:30:00Z
mo calendar update <event-id> --summary "Updated Demo"
mo --force calendar delete <event-id>
```

### Tasks

```bash
mo tasks list --max 20
mo tasks create --title "Follow up"
mo tasks update <task-id> --status inProgress
mo tasks complete <task-id>
mo --force tasks delete <task-id>
```

### Drive

```bash
mo drive ls --max 20
mo drive search "invoice" --max 20
mo drive upload ./report.txt --conflict rename
mo drive download <item-id> --out ./report.txt
mo drive mkdir "Agent Artifacts"
mo drive move <item-id> --parent <folder-id>
mo drive share <item-id> --to user --email user@example.com --role read
```

More examples: `docs/examples.md`.

## Output contract

JSON is default.

```bash
mo mail list
```

Plain output:

```bash
mo mail list --plain
```

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

## Agent and automation usage

Non-interactive run style:

```bash
mo --no-input --force tasks list --max 20
```

Restrict allowed command groups:

```bash
export MO_ENABLE_COMMANDS=mail,calendar,tasks,drive
```

Automation details: `docs/automation.md`.

## Security model

- Refresh tokens are stored in configured keyring backend (not plaintext config)
- OAuth client metadata is stored in credentials JSON
- Supports account/client isolation for agent contexts

Security guide: `docs/security.md`.

## Testing

```bash
make test
make smoke-live
```

- Smoke script: `scripts/smoke-live.sh`
- Smoke guide and env vars: `docs/smoke-tests.md`
- Testing strategy: `docs/testing.md`

## Global flags

- `--json`
- `--plain`
- `--force`
- `--no-input`
- `--account <id>`
- `--client <name>`
- `--color auto|always|never`
- `--version`

## Environment variables

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

## Exit codes

- `0`: success
- `2`: usage error
- `3`: auth required
- `4`: permission denied
- `5`: not found
- `6`: command disabled
- `10`: transient error
- `12`: not implemented

## Current limitations

- `mo drive comments` and `mo drive comment ...` return `not_implemented` because Graph v1.0 does not provide a general drive-item comments API.
- `mo drive shared` uses Graph `sharedWithMe`, which Microsoft has deprecated and may degrade over time.

Design note: `docs/decisions/ADR-0004-onedrive-scope-and-limitations.md`.

## Credits

This project is inspired by `gogcli` by Peter Steinberger:

- `https://github.com/steipete/gogcli`

## License

MIT License. See `LICENSE`.

## Documentation map

Start with `docs/index.md`.

Operator docs:

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

Project internals:

- `docs/spec.md`
- `docs/vision.md`
- `docs/roadmap.md`
- `docs/architecture.md`
- `docs/decisions/`
