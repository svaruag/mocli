# Vision

## Problem

Agents need a reliable command-line interface for Outlook-centric workflows without the overhead of a broad, tenant-admin CLI.

## Target Users

- Agent runtimes (primary)
- Engineers debugging or scripting agent behavior (secondary)

## Product Direction

Build a focused CLI with:

- strict, predictable machine-readable output
- stable exit codes
- explicit scope boundaries
- low operational complexity

## Non-Goals

- Full Microsoft 365 tenant management
- Broad workload support beyond mail/calendar/tasks (v1)
- Feature parity with large multi-domain CLIs

## Quality Bar

- Correctness over feature count
- Reliability over speed of adding commands
- Security-first token handling

## Success Criteria (MVP)

- Agent can authenticate once and run continuously via token refresh
- Agent can execute core mail/calendar/tasks commands without manual intervention
- Reproducible behavior under retries, paging, and common Graph errors
