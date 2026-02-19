# ADR-0001: Initial Project Scope

- Status: accepted
- Date: 2026-02-18

## Context

A large Microsoft 365 CLI exists locally for reference but is intentionally out of scope due to breadth and complexity.

The project needs a smaller, reliable CLI tuned for agent usage with Outlook-centric operations.

## Decision

Adopt a focused MVP:

- Language/runtime: Go
- Integration surface: CLI JSON-first
- Auth: delegated single-user first
- Cloud target: public cloud only
- Command surface: core 12 mail/calendar/tasks commands

Use keyring-backed secure token storage with encrypted fallback only when required.

## Consequences

Positive:

- Faster time to stable MVP
- Lower maintenance burden
- Better reliability focus

Negative:

- No sovereign cloud support in v1
- No app-only auth in v1
- Smaller initial command surface

## Alternatives Considered

1. Build on top of large multi-workload CLI architecture
   - Rejected: too broad and heavy for target use case
2. Include app-only and multi-cloud from day one
   - Rejected: increased complexity without immediate MVP need
