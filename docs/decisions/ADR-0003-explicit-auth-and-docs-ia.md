# ADR-0003: Explicit Entra Credentials and Task-Oriented Documentation

- Status: accepted
- Date: 2026-02-19

## Context

MVP validated core functionality, but onboarding and operator discoverability were still too ambiguous for agent-centric usage.

Pain points identified:

- multiple docs with overlapping ownership
- README flow not fully linear for first-time setup
- managed default auth path introduced complexity and tenant confusion

## Decision

1. Require explicit Entra app credentials for all auth flows.
   - Remove managed default credentials fallback from runtime.
   - Remove `MO_MANAGED_CLIENT_ID` and `MO_MANAGED_TENANT` paths.

2. Restructure docs into task-oriented operator docs.
   - Add a canonical docs index (`docs/index.md`).
   - Create focused docs for setup/auth/permissions/commands/examples/automation/security/testing/release.

3. Keep CLI surface stable and align UX conservatively with gogcli style.
   - Preserve command groups.
   - Improve help/readme/examples consistency and linearity.

## Consequences

Positive:

- predictable auth behavior
- cleaner onboarding and reduced hidden magic
- stronger agent discoverability through explicit docs ownership

Negative:

- explicit app registration is now mandatory
- no fallback path for users expecting preconfigured shared app behavior

## Alternatives Considered

1. Keep managed auth as default.
   - Rejected: operational ambiguity and support burden.

2. Keep managed auth but de-emphasize in docs.
   - Rejected: retains complexity without clear value for target workflow.

3. Keep existing docs topology and only add links.
   - Rejected: insufficient for intent-based agent discovery.
