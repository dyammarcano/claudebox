#!/usr/bin/env bash
#
# claudebox container entrypoint.
#
# Responsibilities (runs as the non-root `agent` user inside the throwaway
# container):
#   1. Seed an ephemeral, WRITABLE ~/.claude from the read-only /seed mount.
#      OAuth tokens refresh at runtime (Claude rewrites .credentials.json), and
#      session history is written under ~/.claude, so the live dir must be
#      writable. The host's ~/.claude is never touched — only the RO seed copy.
#   2. Exec Claude Code in headless (`-p`) mode with the caller's prompt and
#      guardrail flags. exec replaces PID 1 so signals (docker stop) propagate.
#
# Inputs (environment, set by claudebox):
#   PROMPT             required — the task prompt for the agent
#   MAX_TURNS          optional — cap on agent turns (default: 20)
#   OUTPUT_FORMAT      optional — text | json | stream-json (default: stream-json)
#   ALLOWED_TOOLS      optional — comma/space tool allowlist for --allowedTools
#   PERMISSION_MODE    optional — e.g. dontAsk, acceptEdits
#   EXTRA_ARGS         optional — additional raw flags appended verbatim
#   SEED_DIR           optional — read-only seed mount (default: /seed)
#   ANTHROPIC_API_KEY  optional — if set, Claude uses API billing instead of OAuth
#
# Auth precedence: if ANTHROPIC_API_KEY is present Claude uses it; otherwise the
# seeded OAuth credentials (~/.claude/.credentials.json) are used.

set -euo pipefail

SEED_DIR="${SEED_DIR:-/seed}"
CLAUDE_HOME="${HOME}/.claude"

log() { printf 'claudebox[entrypoint] %s\n' "$*" >&2; }

# --- 1. Seed ephemeral credentials/settings -------------------------------
if [ -d "${SEED_DIR}" ]; then
    mkdir -p "${CLAUDE_HOME}"
    # Copy contents of the seed into the writable home .claude. -a preserves
    # perms; we then tighten the credential file explicitly.
    if [ -n "$(ls -A "${SEED_DIR}" 2>/dev/null || true)" ]; then
        cp -a "${SEED_DIR}/." "${CLAUDE_HOME}/"
    fi
    if [ -f "${CLAUDE_HOME}/.credentials.json" ]; then
        chmod 600 "${CLAUDE_HOME}/.credentials.json"
    fi
    log "seeded credentials into ${CLAUDE_HOME}"
else
    log "no seed dir at ${SEED_DIR}; relying on ANTHROPIC_API_KEY"
fi

if [ -z "${PROMPT:-}" ]; then
    log "ERROR: PROMPT is empty; nothing to do"
    exit 64  # EX_USAGE
fi

# --- 2. Build the Claude headless command ---------------------------------
MAX_TURNS="${MAX_TURNS:-20}"
OUTPUT_FORMAT="${OUTPUT_FORMAT:-stream-json}"

# Assemble args as an array to preserve quoting/whitespace safety.
args=(--print "${PROMPT}" --output-format "${OUTPUT_FORMAT}" --max-turns "${MAX_TURNS}")

# stream-json requires --verbose to emit events.
if [ "${OUTPUT_FORMAT}" = "stream-json" ]; then
    args+=(--verbose)
fi

if [ -n "${ALLOWED_TOOLS:-}" ]; then
    args+=(--allowedTools "${ALLOWED_TOOLS}")
fi

if [ -n "${PERMISSION_MODE:-}" ]; then
    args+=(--permission-mode "${PERMISSION_MODE}")
fi

# EXTRA_ARGS is a raw string of extra flags; split on whitespace intentionally.
# NOTE: because of this split, an individual extra arg cannot contain spaces.
# Pass space-bearing content (e.g. a long prompt) via PROMPT, not EXTRA_ARGS.
if [ -n "${EXTRA_ARGS:-}" ]; then
    # shellcheck disable=SC2206
    extra=(${EXTRA_ARGS})
    args+=("${extra[@]}")
fi

log "exec: claude ${args[*]}"
exec claude "${args[@]}"
