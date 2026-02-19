# ADR-0002: Go Baseline Version

- Status: accepted
- Date: 2026-02-18

## Context

The local environment has Go `1.22.2` installed via Ubuntu packages.
Installing newer versions through privileged or networked paths may vary by machine.

## Decision

Use Go `>=1.22` as the initial project baseline.

Revisit upgrades after MVP when release packaging and CI pinning are implemented.

## Consequences

Positive:

- immediate local implementation start
- lower environment friction

Negative:

- cannot depend on newer Go features beyond 1.22 initially

