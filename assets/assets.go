// Package assets embeds the container build context used by claudebox.
//
// The Dockerfile and entrypoint script are compiled into the binary via the
// embed directive so claudebox is a single self-contained executable: no
// external files are required at runtime to build the sandbox image.
package assets

import _ "embed"

// Dockerfile is the embedded two-stage Dockerfile. Stage 1 downloads the
// Claude Code binary pinned to the host version (via the CLAUDE_VERSION build
// arg); stage 2 is the minimal non-root runtime image.
//
//go:embed Dockerfile
var Dockerfile []byte

// Entrypoint is the embedded container entrypoint script. It seeds ephemeral
// credentials from the read-only /seed mount and execs Claude in headless mode.
//
//go:embed entrypoint.sh
var Entrypoint []byte
