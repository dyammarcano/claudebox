# claudebox — Design Spec

Ephemeral, isolated sandbox for running Claude Code agents in headless mode inside a
throwaway Docker container. A Go wrapper builds an image (with an embedded two-stage
Dockerfile pinned to the host's Claude version), injects a scrubbed copy of the host
credentials/settings, runs `claude -p` headless against a mounted workspace, streams the
result back, then force-removes the container so every run is a clean instance with no
pollution of the host's Claude install.

## Goals
- **Isolation** — agent runs in a container, not on the host.
- **Safety** — limit blast radius via non-root user, RO credential mount, `--allowedTools`/`--max-turns`, ephemeral container.
- **Clean instance** — no carry-over state; host `~/.claude` is never modified.
- **Version parity** — container Claude == host Claude (currently `2.1.158`).

## Decisions (locked)
| Topic | Choice |
|-------|--------|
| Auth source | **Subscription OAuth file** — copy `~/.claude/.credentials.json` (Pro/Max login) into the container. Optional `--api-key`/`ANTHROPIC_API_KEY` override for API billing. |
| Credential injection | **Copy host creds+settings → ephemeral temp dir → bind-mount RO at `/seed` → entrypoint copies into writable `$HOME/.claude` → `defer RemoveAll`** |
| Docker control | **Docker Go SDK** (`github.com/docker/docker/client`) — `ImageBuild` (tar build ctx), `ContainerCreate`/`ContainerStart`, `ContainerLogs` stream, `ContainerRemove{Force:true}`. No shelling out. |
| Network | **Full outbound** for v1 (Claude needs the API). Harden later w/ egress allowlist. |
| Persistence | Workspace bind-mount **RW** at `/workspace`; Claude history is ephemeral (the "clean" part) |

## Why entrypoint-copies-seed (not RO-mount `~/.claude` directly)
OAuth access tokens **refresh** — Claude rewrites `.credentials.json`. A RO mount of `~/.claude`
breaks token refresh + history writes. So we mount the seed copy RO at `/seed`, and the
entrypoint copies it into a writable `$HOME/.claude` inside the container. Host creds stay
untouched and read-only; the container's copy is ephemeral and dies with the container.

## Two-stage Dockerfile (embedded via go:embed)
- **Stage 1 (builder)** — `debian:bookworm-slim` + curl/bash/ca-certificates.
  `ARG CLAUDE_VERSION` → `curl -fsSL https://claude.ai/install.sh | bash -s -- "$CLAUDE_VERSION"`.
  Installs to `/root/.local/bin/claude` + `/root/.local/share/claude`.
- **Stage 2 (runtime)** — `debian:bookworm-slim` + ca-certificates + git.
  `COPY --from=builder` the `.local/bin/claude` + `.local/share/claude`. Create non-root
  `agent` user (uid 1000). `ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]`.

The Go app resolves the version from `--claude-version` (default `stable`) and passes it as `--build-arg CLAUDE_VERSION=<version>`.

## entrypoint.sh (container)
1. `cp -a /seed/.claude "$HOME/.claude"` (if present); `chmod 600 .credentials.json`.
2. `exec claude -p "$PROMPT" --output-format stream-json --verbose --max-turns "$MAX_TURNS" $EXTRA_FLAGS`.
   (Auth: if `ANTHROPIC_API_KEY` env present, Claude uses it; else the copied OAuth file.)

## Go layout (hexagonal, Cobra, slog JSON)
```
cmd/claudebox/main.go         # Cobra root + run command, slog JSON setup
internal/config/config.go     # flags → Config struct + validation
internal/credentials/creds.go # locate host ~/.claude, copy seed (creds+settings) → temp, scrub
internal/engine/build.go      # Docker SDK: tar embedded build ctx, ImageBuild w/ CLAUDE_VERSION arg
internal/engine/run.go        # Docker SDK: ContainerCreate/Start, stream ContainerLogs, ContainerRemove
internal/sandbox/sandbox.go   # orchestrate lifecycle: detect → build → prepare creds → run → cleanup
assets/Dockerfile             # embedded via go:embed
assets/entrypoint.sh          # embedded via go:embed
```
SDK note: `ImageBuild` needs the build context as a tar stream — `internal/engine/build.go` builds an in-memory tar from the embedded `assets/` (Dockerfile + entrypoint.sh). Mounts are passed as `container.HostConfig.Binds` / `Mounts`.

## CLI (v1)
```
claudebox run --workspace ./proj --prompt "fix the failing test" \
  --max-turns 20 --allowed-tools "Edit,Bash,Read" [--api-key|auto] [--keep] [--image-tag]
```
Lifecycle: resolve version (--claude-version or "stable") → tar embedded assets → `ImageBuild(CLAUDE_VERSION=<version>)` →
copy host creds+settings to temp seed (0700/0600) → `ContainerCreate` with binds
(`ws:/workspace:rw`, `seed:/seed:ro`) + env (`PROMPT`, `MAX_TURNS`, optional `ANTHROPIC_API_KEY`) →
`ContainerStart` → stream `ContainerLogs` to stdout → wait → `ContainerRemove{Force:true}`
(skipped if `--keep`) → `defer os.RemoveAll(temp)`.

## Security posture (v1)
- Non-root container user; creds mounted RO; host `~/.claude` never written.
- `--allowedTools` allowlist + `--max-turns` cap bound agent behavior.
- Temp seed dir 0700, creds file 0600, scrubbed on exit.
- Documented hardening backlog: egress allowlist, `--read-only` rootfs + tmpfs, `--cap-drop ALL`, seccomp, memory/pids limits, `--network` policy.

## Out of scope (v1 / backlog)
Egress filtering, resource limits, multi-agent fan-out, session resume across runs, Windows-container target (Linux containers only).
