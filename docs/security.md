# Security

## Secret Handling

- OAuth client metadata (`client_id`, `tenant`) is stored in credentials files.
- Refresh tokens are stored in keyring backend, not plaintext config files.
- Access tokens are short-lived and obtained from refresh tokens at runtime.

## Keyring Backends

- `auto`: best available backend
- `keychain`: OS keychain backend
- `file`: encrypted file backend under config dir

For file backend in non-interactive environments:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='strong-password'
```

## Least Privilege

App registration should request only delegated permissions needed for implemented commands:

- `User.Read`
- `Mail.Read`
- `Mail.Send`
- `Calendars.ReadWrite`
- `Tasks.ReadWrite`
- `Files.ReadWrite`

## Operational Controls

- Restrict command groups with `MO_ENABLE_COMMANDS` in agent/sandbox runtimes.
- Use dedicated app registrations per environment when possible.
- Use dedicated test accounts for smoke validation.

## Threat-Model Notes (MVP)

- Scope is user-delegated actions for one signed-in account context at a time.
- This CLI does not perform tenant-wide admin operations.
- Security boundary is primarily token protection + constrained command surface.
