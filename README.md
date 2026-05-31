# claudebox

Run Claude Code agents in **isolated, throwaway Docker containers**.

`claudebox` is a small Go wrapper that builds an ephemeral container image with a
version-pinned Claude Code install (`stable` by default, or any version via
`--claude-version`), injects a **scrubbed, read-only copy** of your credentials,
runs Claude **headlessly** against a mounted workspace, and then **force-removes
the container** so every run is a clean instance with no pollution of your own
Claude install.

## Why

- **Isolation** — the agent runs inside a container, not on your host.
- **Safety** — non-root user, read-only credential seed, `--allowed-tools`
  allowlist and `--max-turns` cap bound what the agent can do.
- **Clean instance** — your host `~/.claude` is never written to; container
  history dies with the container.
- **Pinned version** — the in-container Claude is pinned (`stable` by default, or
  `--claude-version X.Y.Z`), so runs are reproducible.

## How it works

```
claudebox run                       host                          container (throwaway)
  │
  ├─ resolve version (--claude-version, default "stable")
  ├─ build image (embedded 2-stage Dockerfile, CLAUDE_VERSION=<version>)
  │     stage 1: curl install.sh | bash -s -- <version>  (pinned)
  │     stage 2: minimal debian, non-root `agent`, COPY ~/.local
  ├─ copy ~/.claude/{.credentials.json[,settings.json]} ─► temp seed (0700/0600)
  ├─ docker run  -v workspace:/workspace:rw
  │              -v seed:/seed:ro
  │              -e PROMPT,MAX_TURNS,...
  │                                   entrypoint: seed ► ~/.claude (writable)
  │                                                 exec claude -p "$PROMPT" ...
  ├─ stream stdout/stderr ◄───────────────────────────  (de-multiplexed)
  └─ force-remove container; delete temp seed
```

The Dockerfile and entrypoint are **embedded in the binary** (`go:embed`), so
`claudebox` is a single self-contained executable.

## Requirements

- Docker (Docker Desktop on Windows/macOS, or the engine on Linux)
- A logged-in Claude subscription (`~/.claude/.credentials.json`) **or** an
  `ANTHROPIC_API_KEY`

The in-container Claude version defaults to `stable`; pin a specific release with
`--claude-version X.Y.Z`.

## Install

```bash
go install github.com/inovacc/claudebox/cmd/claudebox@latest
```

Or build from source:

```bash
task build      # or: go build ./cmd/claudebox
```

## Usage

```bash
# Run an agent against the current directory
claudebox run -w . -p "fix the failing test"

# Constrain the agent: limited tools, fewer turns, plain text output
claudebox run -w ./myproj \
  -p "summarize the architecture" \
  --max-turns 10 \
  --allowed-tools "Read,Grep,Bash(git log *)" \
  --output-format text

# Use API-key billing instead of your subscription
ANTHROPIC_API_KEY=sk-ant-... claudebox run -w . -p "..."

# Skip copying settings.json (avoids host hooks/plugins/MCP breaking in-container)
claudebox run -w . -p "..." --no-settings
```

### Key flags

| Flag | Default | Purpose |
|------|---------|---------|
| `-w, --workspace` | `.` | Host dir mounted read-write at `/workspace` |
| `-p, --prompt` | _(required)_ | Task prompt for the agent |
| `--max-turns` | `20` | Cap on agent turns |
| `--output-format` | `stream-json` | `text` \| `json` \| `stream-json` |
| `--allowed-tools` | _(none)_ | `--allowedTools` allowlist |
| `--permission-mode` | _(none)_ | e.g. `dontAsk`, `acceptEdits` |
| `--api-key` | env, else OAuth | `ANTHROPIC_API_KEY` override |
| `--no-settings` | `false` | Don't copy `settings.json` into the seed |
| `--claude-version` | `stable` | In-container Claude version (`stable`/`latest`/`X.Y.Z`) |
| `--network` | bridge | Container network mode (`none` to cut egress) |
| `--rebuild` / `--no-cache` | `false` | Force image rebuild |
| `--keep` | `false` | Keep the container after exit (debugging) |

### Hardening flags (default-on)

| Flag | Default | Purpose |
|------|---------|---------|
| `--memory` | `2g` | Container memory limit (`0` = unlimited) |
| `--cpus` | `2` | Container CPU limit (`0` = unlimited) |
| `--pids-limit` | `512` | Max processes (`0` = unlimited) |
| `--cap-add` | _(none)_ | Add a capability back after `cap-drop ALL` (repeatable) |
| `--writable-rootfs` | `false` | Disable the read-only root filesystem |
| `--no-hardening` | `false` | Disable **all** hardening (cap-drop, no-new-privileges, read-only rootfs, limits) |

The process exits with **Claude's own exit code**.

## Architecture

```
cmd/claudebox/         CLI (Cobra) — run command, version, exit-code handling
internal/config/       Config struct + Normalize/Validate
internal/credentials/  Locate ~/.claude, copy scrubbed seed, cleanup
internal/engine/       Docker SDK: build image, run container, stream, remove
internal/sandbox/      Lifecycle orchestration
assets/                Embedded two-stage Dockerfile + entrypoint.sh
```

See [`docs/DESIGN.md`](docs/DESIGN.md) for the full design and the documented
security-hardening backlog (egress allowlist, `--cap-drop ALL`, read-only
rootfs, seccomp, resource limits).

## Security posture (v1)

Enforced today (default-on): non-root container user; **all Linux capabilities
dropped** (`--cap-drop ALL`) with `no-new-privileges`; **read-only root
filesystem** with targeted tmpfs for the writable Claude dirs; **resource
limits** (2g memory / 2 CPU / 512 pids by default); credentials mounted
read-only and only copied (never the host dir directly); host `~/.claude` never
modified; temp seed `0700` / credentials `0600`, deleted on exit;
`--allowed-tools` + `--max-turns` guardrails; container force-removed every run.
Every lockdown has an escape hatch (`--cap-add`, `--writable-rootfs`,
`--memory/--cpus/--pids-limit 0`, `--no-hardening`).

Deferred (tracked in `docs/BACKLOG.md`): network egress allowlist, seccomp/
AppArmor profiles, and moving the API key off the container env.

## License

BSD 3-Clause. See [LICENSE](LICENSE).
