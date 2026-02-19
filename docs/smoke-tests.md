# Live Smoke Tests

Use this after CLI changes to verify real Microsoft Graph behavior.

## Prerequisites

- Built binary available as `mo` on `PATH`
- A signed-in account with delegated Graph permissions
- `jq` installed
- If file keyring is used: `MO_KEYRING_PASSWORD` set

## Run

```bash
cd mocli
make smoke-live
```

Optional environment overrides:

- `SMOKE_TO_EMAIL`: recipient for smoke send-mail check (defaults to current account)
- `SMOKE_TASK_LIST_ID`: force a specific To Do list id
- `SMOKE_TIMEOUT_SECONDS`: polling timeout per eventual-consistency check (default `60`)

Drive smoke options:

- `SMOKE_DRIVE_ENABLED=1`: enable drive smoke lifecycle
- `SMOKE_DRIVE_PARENT=<item-id>`: create/upload under a specific folder
- `SMOKE_DRIVE_SHARE_EMAIL=<addr>`: optionally run share/unshare check
- `SMOKE_DRIVE_SHARED_CHECK=1`: also call `drive shared` (deprecated Graph endpoint)

## What It Asserts

Base checks:

- Auth status has a usable token
- Mail list works, mail get works, send works, sent-item read-back works
- Tasks list works, create/read/update/complete/delete all work
- Calendar create/read/update/delete all work

Drive checks (when enabled):

- `drive drives` works
- upload/download/list/get/move/rename/delete lifecycle works
- permissions listing works
- comments command currently returns `not_implemented`

## Recommended Dev Loop

```bash
make test
make smoke-live
```

Run the drive smoke checks for behavior-changing updates in `internal/app/drive_cmd.go`, auth scopes, or Graph request logic.

If `mo` is not on `PATH`, point the script explicitly:

```bash
MO_BIN=./mo make smoke-live
```
