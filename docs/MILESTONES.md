# Milestones

## v0.1.0 — Working sandbox [In Progress]
- **Status:** Core complete, hardening pending
- **Goals:**
  - [x] Pin the in-container Claude version (stable default; --claude-version override)
  - [x] Two-stage embedded Dockerfile (install.sh version target honored)
  - [x] Ephemeral, read-only credential seed; host ~/.claude untouched
  - [x] Docker SDK build/run/stream/force-remove lifecycle
  - [x] `claudebox run` with guardrails (--allowed-tools, --max-turns, --no-settings)
  - [x] End-to-end real `docker build` validated (with runtime smoke test)
- **Test Coverage:** config 97%, credentials 70%, engine 19%, sandbox 0%, cmd 0%

## v0.2.0 — Hardened sandbox [Not Started]
- **Status:** Not Started
- **Goals:**
  - [ ] `--cap-drop ALL`, no-new-privileges, read-only rootfs + tmpfs
  - [ ] Resource limits (memory / cpus / pids)
  - [ ] Network egress allowlist (Anthropic API only)
  - [ ] API key off env (avoid docker-inspect exposure)
  - **Coverage target:** 80%+

## v1.0.0 — First stable release [Not Started]
- **Status:** Not Started
- **Goals:**
  - [ ] Integration tests for engine/sandbox against a real/faked daemon
  - [ ] cmd smoke tests
  - [ ] Documentation complete
  - [ ] GoReleaser cross-platform release
  - **Coverage target:** 80%+
