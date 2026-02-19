# Architecture (Planned)

## Top-Level Design

`mocli` is a single-process Go binary with clear boundaries:

- command parsing and validation
- auth and token lifecycle
- secure secret storage
- Graph request execution
- output/error formatting

## CLI Shape

- Binary name: `mo`
- Command groups: `auth`, `mail`, `calendar`, `tasks`, `drive`, `config`, `version`
- Style target: close to `gogcli` command ergonomics
- Improvement over baseline: stricter cross-command flag consistency (`--max`, `--page`, `--from`, `--to`)

## Package Shape (Current)

- `cmd/mo`: executable entrypoint
- `internal/app`: command routing, flag handling, Graph request orchestration
- `internal/auth`: OAuth PKCE flows, token exchange/refresh
- `internal/secrets`: keyring backend resolution, secret-tool integration, encrypted file backend
- `internal/config`: config/env resolution, app config persistence, credential files, paths
- `internal/outfmt`: JSON/plain output and error contract formatting
- `internal/exitcode`: stable exit code map

## Command Contract

- JSON-first output for all commands
- Stable top-level keys
- Deterministic non-zero exit codes by error class
- Similar option naming patterns across command groups

## Data Flow

1. Parse command + validate input
2. Resolve account/client context
3. Acquire access token (refresh when needed)
4. Execute Graph request(s)
5. Normalize response/errors into CLI contract

## Constraints

- Keep design small and readable
- Avoid framework-heavy abstractions
- Add indirection only when it removes real duplication
