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
mo --force calendar delete <event-id>
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
mo --force tasks delete <task-id>
```

## Drive: Files and Folders

```bash
# list/search
mo drive ls --max 20
mo drive search "invoice" --max 20

# upload/download
mo drive upload ./report.txt --conflict rename
mo drive download <item-id> --out ./report.txt

# organize
mo drive mkdir "Agent Artifacts"
mo drive move <item-id> --parent <folder-id>
mo drive rename <item-id> "report-final.txt"
mo --force drive delete <item-id>
```

## Drive: Sharing and Permissions

```bash
# list permissions
mo drive permissions <item-id>

# share to user
mo drive share <item-id> --to user --email user@example.com --role read

# revoke permission
mo --force drive unshare <item-id> <permission-id>
```

## Drive: Discover Drives and Shared Items

```bash
mo drive drives --max 50
mo drive shared --max 50
```

## Comments Limitation

```bash
mo drive comments <item-id>
```

`drive comments` currently returns `not_implemented` because Graph v1.0 does not expose general drive-item comments endpoints.

## Non-Interactive/Agent Pattern

```bash
mo --json --no-input --force tasks list --max 20
mo --json --no-input --force drive ls --max 20
```

## Live Smoke Verification

```bash
make test
make smoke-live
```
