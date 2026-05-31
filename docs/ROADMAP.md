# Roadmap

## Current Status
**Overall Progress:** ~70% to a hardened v1 — the core sandbox flow is built,
tested, and validated against real Docker.

## Phases

### Phase 1: Foundation [COMPLETE]
- [x] Project scaffold (Cobra, hexagonal layout, CI config)
- [x] Config model with Normalize/Validate
- [x] Host Claude version detection
- [x] Ephemeral credential seed (copy + scrub + cleanup)

### Phase 2: Sandbox core [COMPLETE]
- [x] Two-stage embedded Dockerfile, version-pinned install
- [x] Entrypoint: seed credentials, exec headless `claude -p`
- [x] Docker SDK engine: build, run, stdcopy stream, force-remove
- [x] Lifecycle orchestration (`internal/sandbox`)
- [x] CLI `run` command with guardrail flags
- [x] Real `docker build` validated end-to-end (incl. runtime smoke test)

### Phase 3: Security hardening [IN PROGRESS]
- [ ] `--cap-drop ALL`, `--security-opt no-new-privileges`
- [ ] Read-only root filesystem + tmpfs
- [ ] Resource limits (memory, cpus, pids)
- [ ] Network egress allowlist (Anthropic API only)
- [ ] Move API key off env (docker-inspect exposure)
- See `docs/BACKLOG.md` for details.

### Phase 4: Polish & release [NOT STARTED]
- [ ] Increase coverage of `engine`/`sandbox` (integration tests w/ Docker)
- [ ] `cmd` smoke tests
- [ ] v1.0.0 release via GoReleaser

## Test Coverage
**Current:** 13.4% total  |  **Target:** 80%

The total is dragged down by the large, Docker-dependent and CLI-glue packages
that have no unit tests yet; the pure-logic packages are well covered.

| Package | Coverage | Status |
|---------|----------|--------|
| internal/config | 96.8% | Good |
| internal/credentials | 69.8% | OK (copyFile/DefaultClaudeHome error paths untested) |
| internal/version | 25.0% | `Parse` covered; `Detect` needs host claude |
| internal/engine | 19.1% | Needs improvement (Docker-dependent paths) |
| internal/sandbox | 0.0% | No tests (orchestration; needs Docker fake) |
| cmd/claudebox | 0.0% | No tests |
| assets | n/a | Embed-only |
