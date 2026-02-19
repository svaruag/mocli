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
