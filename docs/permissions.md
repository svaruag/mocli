# Permissions

## Delegated Graph Permissions

Mocli requires these Microsoft Graph delegated permissions:

- `User.Read`
- `Mail.Read`
- `Mail.Send`
- `Calendars.ReadWrite`
- `Tasks.ReadWrite`
- `Files.ReadWrite`

OIDC scopes used during login:

- `openid`
- `profile`
- `offline_access`

## Scope-to-Command Mapping

- `User.Read`
  - `auth add` identity resolution via `/me`
- `Mail.Read`
  - `mail list`, `mail get`
- `Mail.Send`
  - `mail send`
- `Calendars.ReadWrite`
  - `calendar list`, `calendar create`, `calendar update`, `calendar delete`
- `Tasks.ReadWrite`
  - `tasks list`, `tasks create`, `tasks update`, `tasks complete`, `tasks delete`
- `Files.ReadWrite`
  - `drive ls`, `drive search`, `drive get`
  - `drive upload`, `drive download`
  - `drive mkdir`, `drive rename`, `drive move`, `drive delete`
  - `drive permissions`, `drive share`, `drive unshare`
  - `drive drives`, `drive shared`

## Consent Guidance

- Ask only for scopes required by implemented commands.
- If tenant policy requires admin consent, grant it before account authorization.
- For personal Microsoft accounts, prefer `tenant: "common"` (or `"consumers"` when required).
- After adding new scopes to the app registration, re-run auth consent:

```bash
mo auth add <email> --device --force-consent
```

## Operational Checks

After consent/login:

```bash
mo auth status
mo mail list --max 1
mo calendar list --max 1
mo tasks list --max 1
mo drive ls --max 1
```

If a command fails with `permission_denied`, verify the corresponding Graph permission in app registration.

Drive comments note:

- `drive comments` commands currently return `not_implemented` due Graph v1.0 API limitations for general drive-item comments.
