# syntax=docker/dockerfile:1
#
# Two-stage build for claudebox.
#
# Stage 1 (builder) downloads the Claude Code native binary pinned to the EXACT
# version detected on the host, using the official install script. The version
# is passed as a build arg by the Go wrapper (claudebox), so the container
# always matches the host's `claude --version`.
#
# Stage 2 (runtime) is a minimal image containing only the runtime deps, a
# non-root `agent` user, the copied Claude install, and the entrypoint that
# seeds ephemeral credentials before exec'ing Claude in headless mode.
#
# IMPORTANT: HOME is kept identical (/home/agent) in BOTH stages. The Claude
# launcher under ~/.local can contain absolute path references; copying the
# install to a different path would break them. Same HOME => paths resolve.

ARG HOME_DIR=/home/agent

# ---------------------------------------------------------------------------
# Stage 1: builder — fetch the pinned Claude Code binary into $HOME_DIR/.local
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim AS builder

ARG HOME_DIR
# Exact Claude Code version to install (e.g. 2.1.158). Injected by claudebox via
# --build-arg CLAUDE_VERSION=$(claude --version). Defaults to "stable" so the
# image is still buildable by hand.
ARG CLAUDE_VERSION=stable

# Install into the SAME home path the runtime stage will use.
ENV HOME=${HOME_DIR}

RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        curl \
        bash \
    && rm -rf /var/lib/apt/lists/* \
    && mkdir -p "${HOME}"

# Install Claude Code at the pinned version. The install script accepts a
# positional target of `stable | latest | X.Y.Z` and installs the native binary
# into $HOME/.local (launcher at ~/.local/bin/claude, runtime under
# ~/.local/share/claude). On glibc Debian it pulls the linux-x64 build.
RUN curl -fsSL https://claude.ai/install.sh | bash -s -- "${CLAUDE_VERSION}"

# Sanity check: ensure the binary exists and is runnable in the builder.
RUN "${HOME}/.local/bin/claude" --version

# ---------------------------------------------------------------------------
# Stage 2: runtime — minimal image that runs Claude as a non-root user
# ---------------------------------------------------------------------------
FROM debian:bookworm-slim AS runtime

ARG HOME_DIR

# Runtime dependencies only. git is included because agents commonly operate on
# git workspaces; ca-certificates for TLS to the Anthropic API.
RUN apt-get update \
    && apt-get install -y --no-install-recommends \
        ca-certificates \
        git \
    && rm -rf /var/lib/apt/lists/* \
    && update-ca-certificates

# Non-root user with home at the SAME path used by the builder. uid/gid 1000
# matches the common host default so files Claude writes into the RW workspace
# carry a sane owner.
RUN groupadd --gid 1000 agent \
    && useradd --uid 1000 --gid 1000 --create-home --home-dir "${HOME_DIR}" --shell /bin/bash agent

# Copy the pinned Claude install from the builder at the identical path so any
# absolute references inside the launcher remain valid.
COPY --from=builder --chown=agent:agent ${HOME_DIR}/.local ${HOME_DIR}/.local

# Entrypoint seeds credentials then execs Claude headless.
COPY entrypoint.sh /usr/local/bin/entrypoint.sh
RUN chmod 0755 /usr/local/bin/entrypoint.sh

USER agent
ENV HOME=${HOME_DIR}
ENV PATH=${HOME_DIR}/.local/bin:$PATH

# Smoke test: fail the build early if the copied native binary is missing a
# runtime library present in the builder but not in this minimal runtime image.
# (COPY does not execute the binary, so without this a missing lib would only
# surface at container run time.)
RUN claude --version

WORKDIR /workspace

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
