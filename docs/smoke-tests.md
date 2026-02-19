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

## What It Asserts

- Auth status has a usable token
- Mail list works, mail get works, send works, sent-item read-back works
- Tasks list works, create/read/update/complete/delete all work
- Calendar create/read/update/delete all work
- Created task/event are cleaned up by the script

## Recommended Dev Loop

```bash
make test
make smoke-live
```

If `mo` is not on `PATH`, point the script explicitly:

```bash
MO_BIN=./mo make smoke-live
```

Run the live smoke checks after behavior-changing updates in `internal/app`, auth flows, or Graph request logic.
