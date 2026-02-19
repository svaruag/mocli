# Roadmap

## Phase 0: Tooling Bootstrap (Done)

- Confirm/install required tools:
  - Go toolchain
  - `jq`
  - `unzip`
- Verify local build/test loop can run without manual environment fixes

Exit criteria:

- Tooling script executed successfully
- Tool versions captured in plan verification notes

## Phase 1: Docs Bootstrap (Done)

- Create repository, policies, and executable plans
- Define MVP boundaries, architecture direction, and testing strategy

Exit criteria:

- All seed docs exist
- Five active implementation plans exist with explicit scope and tests

## Phase 2: Foundation (Done)

- CLI skeleton and global flags
- Config and output foundations
- Command allowlist mechanism for agent safety

Exit criteria:

- Minimal command tree compiles
- JSON output + exit code baseline in place

## Phase 3: Auth + Secrets (Done)

- Delegated OAuth login flows (browser/device)
- Secure token persistence via keyring
- Auto-refresh and auth status/introspection commands

Exit criteria:

- Non-interactive command execution works after one-time login
- Revoked/expired token behavior is deterministic and actionable

## Phase 4: Mail/Calendar/Tasks Core (Done)

- Core 12 command set implemented
- Input validation and Graph error normalization
- Stable output schemas for agent use

Exit criteria:

- End-to-end command tests pass
- Core workflows validated from clean environment

## Phase 5: Hardening + Release (Done)

- Reliability hardening and retry tuning
- Documentation polish and examples
- Release packaging and smoke checks

Exit criteria:

- Regression suite green
- Release checklist complete
