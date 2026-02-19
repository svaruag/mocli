# Troubleshooting

## `auth_required` Errors

Symptoms:

- `{"error":{"code":"auth_required",...}}`

Checks:

1. Save credentials:
   - `mo auth credentials <path>`
2. Authorize account:
   - `mo auth add <email> --device`
3. Verify status:
   - `mo auth status`

## Missing Credentials File

Symptoms:

- `missing credentials for client ...`

Fix:

```bash
mo auth credentials ./entra-app.json
```

## New Scope Added, Command Still Fails

Symptoms:

- drive commands fail with `permission_denied` after app permissions were updated

Fix:

```bash
mo auth add <email> --device --force-consent
```

## `AADSTS50020` Tenant Mismatch

Symptoms:

- error contains `AADSTS50020`

Fix:

- set `tenant` in credentials JSON to `common` (or `consumers` for personal-only)
- retry device auth:

```bash
mo auth add <email> --device
```

## `AADSTS50059` No Tenant-Identifying Information

Symptoms:

- error contains `AADSTS50059`

Fix:

- ensure credentials JSON includes explicit `tenant`
- use one of:
  - `common`
  - `consumers`
  - tenant GUID

## `AADSTS700016` App Not Found

Symptoms:

- error contains `AADSTS700016`

Fix:

- verify `client_id` in credentials file matches app registration

## File Keyring Password Missing

Symptoms:

- error mentions `MO_KEYRING_PASSWORD`

Fix:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='<strong-password>'
```

## Command Blocked by Allowlist

Symptoms:

- `command_disabled`

Fix:

```bash
export MO_ENABLE_COMMANDS=mail,calendar,tasks,drive,auth,config
```

## Drive Comments Not Implemented

Symptoms:

- `mo drive comments ...` returns `not_implemented`

Cause:

- Microsoft Graph v1.0 does not expose general file-comments endpoints for drive items.

## Drive Shared Endpoint Warnings

Symptoms:

- `mo drive shared` includes a deprecation warning

Cause:

- Graph `sharedWithMe` is deprecated by Microsoft and may degrade.

## Browser Redirect Issues

If browser callback does not complete:

1. retry browser flow:
   - `mo auth add <email>`
2. on headless/non-browser environments use device flow:
   - `mo auth add <email> --device`

## Device Flow Times Out

If `--device` polls until timeout:

1. complete code entry before expiry at verification URL
2. retry login
3. confirm app registration has required delegated permissions

## Transient Graph Failures (`429`/`5xx`)

Mocli retries transient responses with bounded backoff.

If failures persist:

1. retry after delay
2. reduce request volume
3. check Microsoft service health

## Restricted Config Directory

Use custom config path when default user config is unavailable:

```bash
export MO_CONFIG_DIR=/tmp/mocli-dev
```
