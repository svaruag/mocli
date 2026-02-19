# Examples

All examples assume:

- credentials already stored (`mo auth credentials ...`)
- account already authorized (`mo auth add ...`)

## Mail: Read and Send

```bash
# list recent mail
mo mail list --max 10

# get one message
mo mail get <message-id>

# send mail
mo mail send \
  --to you@outlook.com \
  --subject "Mocli test" \
  --body "hello from mo"
```

## Calendar: Create and Update Event

```bash
mo calendar create \
  --summary "Mocli demo" \
  --from 2026-02-19T09:00:00Z \
  --to 2026-02-19T09:30:00Z \
  --description "calendar smoke check"

mo calendar update <event-id> --summary "Mocli demo (updated)"
mo calendar delete <event-id> --force
```

## Tasks: Full Lifecycle

```bash
# list tasks
mo tasks list --max 20

# create
mo tasks create --title "Mocli follow up" --body "check API response"

# update
mo tasks update <task-id> --status inProgress --importance high

# complete
mo tasks complete <task-id>

# delete
mo tasks delete <task-id> --force
```

## Use Explicit To Do List

```bash
mo tasks list --list-id Tasks --max 50
mo tasks create --list-id Tasks --title "From explicit list"
```

## Non-Interactive/Agent Pattern

```bash
mo --json --no-input --force tasks list --max 20
```

## Live Smoke Verification

```bash
make test
make smoke-live
```
