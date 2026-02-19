# Authentication

## Overview

Mocli uses delegated OAuth with refresh tokens.

Supported login flows:

- Browser (default)
- Device code (`--device`)

## Credential Files

Before `mo auth add`, store app credentials once:

```bash
mo auth credentials ./entra-app.json
```

Storage paths:

- default client: `credentials.json`
- named client: `credentials-<client>.json`

List stored credentials:

```bash
mo auth credentials list
```

## Multiple Clients

Use `--client` (or `MO_CLIENT`) to separate OAuth client profiles and token buckets.

```bash
mo --client work auth credentials ./work-entra-app.json
mo --client work auth add you@company.com --device
```

## Account Selection

Account resolution order:

1. `--account` / `MO_ACCOUNT`
2. `default_account` from config
3. only account for selected client (if exactly one)

List accounts:

```bash
mo auth list
```

Show current state:

```bash
mo auth status
```

## Token Storage

Refresh tokens are stored in keyring backend (`auto`, `keychain`, or `file`).

Inspect resolved backend:

```bash
mo auth status
```

If backend resolves to `file`, set password before `mo auth add`:

Force file backend for non-interactive hosts:

```bash
export MO_KEYRING_BACKEND=file
export MO_KEYRING_PASSWORD='strong-password'
```

## Auth Commands

```bash
mo auth add <email>
mo auth add <email> --device
mo auth remove <email>
```

## Failure Behavior

- Missing credentials file: `auth_required` with hint to run `mo auth credentials <path>`
- Missing stored token for account: `auth_required` with hint to run `mo auth add <email>`
- Refresh/token exchange failures: `auth_required` with Graph/OAuth diagnostic hint
