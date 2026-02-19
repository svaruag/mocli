# Testing

## Local Fast Loop

```bash
go test ./...
```

## Build Check

```bash
mkdir -p "$HOME/.local/bin"
go build -o "$HOME/.local/bin/mo" ./cmd/mo
```

## Live Smoke (Real Graph)

```bash
make smoke-live
```

Optional smoke env overrides:

- `SMOKE_TO_EMAIL`
- `SMOKE_TASK_LIST_ID`
- `SMOKE_TIMEOUT_SECONDS`
- `SMOKE_DRIVE_ENABLED`
- `SMOKE_DRIVE_PARENT`
- `SMOKE_DRIVE_SHARE_EMAIL`
- `SMOKE_DRIVE_SHARED_CHECK`

Detailed smoke behavior: `docs/smoke-tests.md`.

## Required Quality Gates

For behavior changes:

1. unit tests pass (`go test ./...`)
2. build passes (`go build ...`)
3. smoke checks pass (`make smoke-live`)
4. docs/examples updated to match final behavior

## Coverage Focus

- auth flow state transitions
- Graph error mapping and retries
- flag/argument validation
- JSON contract stability
- mail/calendar/tasks/drive core workflows
