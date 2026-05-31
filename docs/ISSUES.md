# Known Issues

## Open Issues / Limitations

| Issue | Impact | Workaround |
|-------|--------|------------|
| Seed temp dir must be in Docker Desktop's file-sharing scope | On Windows/macOS, a seed outside shared paths fails the RO mount | Default base is `~/.claudebox/seeds` (home is shared by default); override with `--seed-base` |
| `--api-key` value is visible via `docker inspect` while the container runs | Local Docker users could read the key during the (brief) container lifetime | Prefer the default OAuth path (file mount, not env); use ephemeral keys if env is required. Tracked in BACKLOG (P2) |
| Full outbound network by default | A compromised agent could exfiltrate data | `--network none` blocks egress but also the API; egress allowlist tracked in BACKLOG (P1) |
| Read-only rootfs may block runs that write outside `/workspace` and the standard tmpfs paths (`~/.claude`, `~/.cache`, `~/.config`, `/tmp`) | A tool that writes elsewhere in the container fails | Pass `--writable-rootfs` (or `--no-hardening`) for that run |
| Linux containers only | Windows-container targets unsupported | Use Docker Desktop in Linux-container mode |
| `EXTRA_ARGS` are whitespace-split in the entrypoint | An individual `--extra-arg` value cannot contain spaces | Pass space-bearing content via `--prompt` |

## Resolved Issues

| Issue | Resolution | Date |
|-------|------------|------|
| No container resource/capability limits | Default-on hardening: cap-drop ALL, no-new-privileges, read-only rootfs + tmpfs, memory/cpu/pids limits (with escape hatches) | 2026-05-30 |
| `docker/docker/client@latest` resolved to `moby/moby/client` and broke the build | Pin monolithic module `github.com/docker/docker@v28.5.2+incompatible` | 2026-05-30 |
| Seed dir created in OS temp (not Docker-shared) — Windows mount failure risk | `PrepareSeed` takes a base dir; default `~/.claudebox/seeds` | 2026-05-30 |
| Missing runtime libs would only surface at container run time | Added `RUN claude --version` smoke test to the runtime stage | 2026-05-30 |
