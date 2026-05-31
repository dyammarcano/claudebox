# Implementation Tasks

Granular tasks for the planned work in `docs/ROADMAP.md`, `docs/BACKLOG.md`, and
`docs/FEATURES.md`, grouped by domain. IDs are referenced from those docs.

## Domain 1: Container hardening (ROADMAP Phase 3, BACKLOG P1)

| ID | What | Files | Environment | Depends on | Effort |
|----|------|-------|-------------|------------|--------|
| 1.1 | Add `--cap-drop ALL` + `--security-opt no-new-privileges` to HostConfig | `internal/engine/run.go`, `internal/config/config.go`, `cmd/claudebox/cmd_run.go` | Go | — | Small |
| 1.2 | Read-only root filesystem + tmpfs for `/tmp` and writable `~/.claude` | `internal/engine/run.go` (`HostConfig.ReadonlyRootfs`, `Tmpfs`), `assets/entrypoint.sh` | Go, shell | 1.1 | Medium |
| 1.3 | Resource limits: memory, cpus, pids | `internal/engine/run.go` (`HostConfig.Resources`), `internal/config` | Go | 1.1 | Small |
| 1.4 | Flags + config plumbing + validation for the above | `internal/config/config.go`, `cmd/claudebox/cmd_run.go` | Go | 1.1–1.3 | Small |
| 1.5 | Tests for HostConfig assembly (table-driven) | `internal/engine/run_test.go` | Go | 1.1–1.4 | Medium |

## Domain 2: Network egress allowlist (BACKLOG P1)

| ID | What | Files | Environment | Depends on | Effort |
|----|------|-------|-------------|------------|--------|
| 2.1 | Design egress approach (proxy sidecar vs firewall rules) | `docs/adr/0003-egress-allowlist.md` | Docs | — | Small |
| 2.2 | Implement allowlist transport (api.anthropic.com only) | `internal/engine/*`, possibly new `internal/netpolicy` | Go, Docker networking | 2.1 | Large |
| 2.3 | `--egress` flag (off|anthropic|custom list) | `internal/config`, `cmd/claudebox/cmd_run.go` | Go | 2.2 | Small |

## Domain 3: Credential delivery hardening (BACKLOG P2)

| ID | What | Files | Environment | Depends on | Effort |
|----|------|-------|-------------|------------|--------|
| 3.1 | Move API key off container env (file mount + helper) | `internal/sandbox/sandbox.go`, `assets/entrypoint.sh` | Go, shell | — | Medium |
| 3.2 | Document residual exposure + update flag help | `README.md`, `cmd/claudebox/cmd_run.go` | Docs/Go | 3.1 | Small |

## Domain 4: Test coverage (ROADMAP Phase 4)

| ID | What | Files | Environment | Depends on | Effort |
|----|------|-------|-------------|------------|--------|
| 4.1 | Engine integration tests against a real/faked daemon | `internal/engine/*_test.go` | Go, Docker | — | Large |
| 4.2 | Sandbox orchestration tests (engine interface + fake) | `internal/sandbox/sandbox_test.go` (extract engine interface) | Go | 4.1 | Medium |
| 4.3 | cmd smoke tests (flag wiring, validation, exit codes) | `cmd/claudebox/*_test.go` | Go | — | Small |

## Domain 5: Build & release (ROADMAP Phase 4)

| ID | What | Files | Environment | Depends on | Effort |
|----|------|-------|-------------|------------|--------|
| 5.1 | Move to BuildKit session build | `internal/engine/build.go` | Go, BuildKit | — | Medium |
| 5.2 | GoReleaser cross-platform release + tag | `.goreleaser.yaml`, git tag | CI | 4.x | Small |

## Suggested order

1. Domain 1 (1.1 → 1.5) — highest safety value, low risk.
2. Domain 3 (3.1 → 3.2) — closes the documented API-key gap.
3. Domain 4 (4.1 → 4.3) — raise coverage to the 80% target.
4. Domain 2 (2.1 → 2.3) — largest; isolates exfiltration risk.
5. Domain 5 (5.1 → 5.2) — performance + release.
