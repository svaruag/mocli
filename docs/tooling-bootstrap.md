# Tooling Bootstrap

## Required Tools

- Go toolchain (current baseline: `>=1.22`)
- `git`
- `make`
- `jq`
- `unzip`
- `rg` (ripgrep)

## Current Machine Status (2026-02-18)

- `go version`: `go1.22.2 linux/amd64`
- `jq --version`: `jq-1.7`
- `unzip`: present

## Install Strategy

Use system package manager with sudo (run by user).

Reference script:

- `/tmp/mocli-bootstrap-tools.sh`

Run:

```bash
sudo /tmp/mocli-bootstrap-tools.sh
```

## Notes

- This repository avoids auto-running privileged installs.
- If additional tools are needed, create a `/tmp` script and ask the user to run it with sudo.
- Use `MO_CONFIG_DIR` to override the default config location in restricted/sandboxed environments.
