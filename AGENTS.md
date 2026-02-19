# AGENTS Guide

This file is intentionally short. Use it as a navigation map.

## Repo Map

- `README.md`: linear onboarding and command overview
- `docs/index.md`: docs router for humans and agents
- `docs/setup-entra-app.md`: required Entra app registration flow
- `docs/authentication.md`: auth flows, token storage, account/client selection
- `docs/permissions.md`: Graph scope requirements and mapping
- `docs/commands.md`: CLI command signatures
- `docs/examples.md`: practical workflows and copy-paste examples
- `docs/automation.md`: agent-safe operation and JSON/exit-code contract
- `docs/security.md`: secret handling and operational security notes
- `docs/troubleshooting.md`: failure diagnosis and fixes
- `docs/testing.md`: test loop and quality gates
- `docs/smoke-tests.md`: live smoke script behavior and env overrides
- `docs/release.md`: release readiness checklist
- `docs/spec.md`: canonical MVP runtime/contract specification
- `docs/vision.md`: product goals and non-goals
- `docs/roadmap.md`: delivery phase record
- `docs/architecture.md`: technical layout
- `docs/decisions/`: ADRs for material design decisions

## Source-of-Truth Guidelines

- Coding behavior and simplicity rules: `../AGENT.md`
- Runtime command contract: `docs/commands.md`, `docs/automation.md`
- Security and auth constraints: `docs/authentication.md`, `docs/security.md`, `docs/permissions.md`
- Testing requirements: `docs/testing.md`, `docs/smoke-tests.md`
- Scope boundaries: `docs/vision.md`, `docs/roadmap.md`

## Working Rules

- Keep changes scoped to a single behavior change per commit.
- Add/adjust tests for every behavior change.
- Record major design decisions in `docs/decisions/`.
- Install toolchains/dependencies only when needed.
