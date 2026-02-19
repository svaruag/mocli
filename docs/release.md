# Release

## Release Readiness Checklist

### Core Checks

- [ ] `go test ./...` is green
- [ ] `go build -o "$HOME/.local/bin/mo" ./cmd/mo` is green
- [ ] docs are updated for any user-visible behavior changes
- [ ] command examples in docs are valid current syntax

### Functional Smoke

- [ ] `make smoke-live` passes
- [ ] mail send/read flow verified
- [ ] calendar create/update/delete flow verified
- [ ] tasks create/update/complete/delete flow verified

### Contract Checks

- [ ] JSON error shape unchanged (`error.code`, `error.message`, `error.hint`)
- [ ] exit codes unchanged unless intentionally revised
- [ ] command allowlist behavior unchanged

### Documentation Checks

- [ ] `README.md` quick start works from clean environment
- [ ] `docs/setup-entra-app.md` matches current auth requirements
- [ ] `docs/permissions.md` maps to implemented command scopes

## Release Notes Template

- user-visible features
- bug fixes
- breaking changes (if any)
- migration/upgrade notes
- known limitations
