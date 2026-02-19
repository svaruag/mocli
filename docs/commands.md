# Commands

## Root

```bash
mo [global flags] <command> [args...]
mo help
```

Top-level commands:

- `auth`
- `mail`
- `calendar`
- `tasks`
- `drive`
- `config`
- `version`

Help drill-down:

```bash
mo <command> help
mo <command> --help
```

## Auth

```bash
mo auth credentials <path> [--client-id ...] [--tenant ...]
mo auth credentials list
mo auth add <email> [--device] [--timeout ...] [--force-consent]
mo auth status
mo auth list
mo auth remove <email>
```

## Mail

```bash
mo mail list [--max N] [--page TOKEN] [--from RFC3339] [--to RFC3339] [--folder ID_OR_NAME]
mo mail get <message-id>
mo mail send --to <emails> --subject <text> --body <text> [--cc ...] [--bcc ...] [--body-html] [--save-to-sent=false]
```

## Calendar

```bash
mo calendar list [--max N] [--page TOKEN] [--from RFC3339 --to RFC3339]
mo calendar create --summary <text> --from <RFC3339> --to <RFC3339> [--description ...] [--location ...] [--attendees ...]
mo calendar update <event-id> [--summary ...] [--from ...] [--to ...] [--description ...] [--location ...] [--attendees ...]
mo calendar delete <event-id>
```

## Tasks

```bash
mo tasks list [--list-id ID] [--max N] [--page TOKEN]
mo tasks create --title <text> [--list-id ID] [--body ...] [--due RFC3339] [--status ...] [--importance ...]
mo tasks update <task-id> [--list-id ID] [--title ...] [--body ...] [--due ...] [--status ...] [--importance ...]
mo tasks complete <task-id> [--list-id ID]
mo tasks delete <task-id> [--list-id ID]
```

## Drive

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

Notes:

- `drive comments` and `drive comment ...` currently return `not_implemented` because Graph v1.0 does not expose general drive item comments endpoints.
- `drive shared` uses Graph `sharedWithMe`, which is deprecated by Microsoft and may degrade.

## Config

```bash
mo config list
mo config get <key>
mo config set <key> <value>
mo config unset <key>
mo config path
```

Config keys:

- `keyring_backend`: `auto|keychain|file`
- `default_account`
- `default_client`
