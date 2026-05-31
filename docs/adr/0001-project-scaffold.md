# ADR-0001: Project Scaffold and Tooling Choices

## Status
Accepted

## Context
`claudebox` is a CLI that drives Docker to run Claude Code in throwaway
containers. It needs a standard Go structure, a CLI framework, and a build/test
toolchain consistent with the owner's other projects.

## Decision
- **Structure:** Hexagonal-ish — `cmd/claudebox` for the CLI surface,
  `internal/` packages for isolated concerns (config, version, credentials,
  engine, sandbox), `assets/` for embedded build context.
- **CLI framework:** Cobra (via `omni scaffold cobra`), single `package main`
  under `cmd/claudebox` with `cmd_*.go` files.
- **Task runner:** Taskfile (cross-platform).
- **Lint/release:** golangci-lint v2, GoReleaser, GitHub Actions.
- **Module path:** `github.com/inovacc/claudebox`.
- **License:** BSD 3-Clause.
- **Logging:** `log/slog` JSON to stderr.

## Consequences
### Positive
- Each concern is independently testable; config/version/credentials reach
  85–95% coverage without Docker.
- Single self-contained binary (assets embedded).
### Negative
- Docker-touching code (`engine`, `sandbox`) is harder to unit-test and starts
  with low coverage; needs an engine interface + fakes (tracked: task 4.x).
