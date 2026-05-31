# ADR-0002: Ephemeral credential seed (copy, don't mount ~/.claude)

## Status
Accepted

## Context
The container must authenticate as the user (subscription OAuth or API key)
without polluting or corrupting the host's Claude install. Two naive options:

1. **Mount `~/.claude` read-write** — the container could write history, refresh
   tokens, and otherwise mutate the host's real directory. Pollution; the host's
   "clean instance" guarantee is lost.
2. **Mount `~/.claude` read-only** — OAuth tokens refresh at runtime (Claude
   rewrites `.credentials.json`) and session history is written under
   `~/.claude`; a read-only mount breaks both, so Claude fails mid-run.

## Decision
Copy a **whitelisted, scrubbed subset** (`.credentials.json`, optionally
`settings.json`) into an ephemeral temp dir (0700, files 0600), bind-mount it
**read-only at `/seed`**, and have the entrypoint `cp -a /seed/. ~/.claude` into
a **writable** in-container `~/.claude`. The temp seed is deleted on exit
(`defer cleanup()`); the host `~/.claude` is read from but never written.

The seed base dir defaults to `~/.claudebox/seeds` (inside Docker Desktop's
default file-sharing scope on Windows/macOS), overridable via `--seed-base`.

`--no-settings` copies only credentials, because a host `settings.json` often
references hooks/plugins/MCP servers absent in the container that would break a
headless run.

## Consequences
### Positive
- Host `~/.claude` is never modified; every run is a clean instance.
- Token refresh and history work inside the container (writable copy).
- Minimal whitelist limits what the container can see (no history/projects leak).
### Negative
- A copy of credentials briefly exists on disk in the seed dir (0700/0600,
  deleted on exit) and inside the container (dies with it).
- API-key path still passes the key as an env var (visible via `docker inspect`
  while running) — tracked separately in BACKLOG (P2).
