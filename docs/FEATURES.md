# Features

## Completed

- **Isolated headless runs** — execute `claude -p` inside a throwaway container.
- **Pinned version** — container Claude pinned (`stable` default, or `--claude-version X.Y.Z`).
- **Ephemeral credential seed** — scrubbed read-only copy; host `~/.claude` never modified.
- **Auth flexibility** — subscription OAuth by default; `--api-key`/`ANTHROPIC_API_KEY` override.
- **Guardrails** — `--allowed-tools`, `--max-turns`, `--permission-mode`.
- **`--no-settings`** — skip host `settings.json` to avoid hook/plugin/MCP breakage in-container.
- **Clean instance** — container force-removed every run (`--keep` to debug).
- **Self-contained binary** — Dockerfile + entrypoint embedded via `go:embed`.
- **Exit-code fidelity** — process exits with Claude's own exit code.
- **Streamed output** — de-multiplexed stdout/stderr via stdcopy.

## Proposed

| Feature | Priority | Notes |
|---------|----------|-------|
| Network egress allowlist | P1 | Anthropic API only; block exfiltration |
| Container hardening flags | P1 | cap-drop, read-only rootfs, resource limits |
| Session resume across runs | P3 | Opt-in named session volume |
| Alpine/musl image variant | P3 | Smaller image behind `--base` |
| BuildKit build | P2 | Cache mounts, faster rebuilds |
