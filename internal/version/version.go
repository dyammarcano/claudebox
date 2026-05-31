// Package version detects the Claude Code version installed on the host so the
// sandbox container can be pinned to the exact same release.
package version

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// semverRE matches a semantic version, optionally with a pre-release suffix,
// anywhere in a string. `claude --version` prints e.g. "2.1.158 (Claude Code)".
var semverRE = regexp.MustCompile(`(\d+\.\d+\.\d+(?:-[0-9A-Za-z.]+)?)`)

// DefaultBinary is the host executable consulted for the version.
const DefaultBinary = "claude"

// Detect runs `<binary> --version` on the host and extracts the semantic
// version. binary may be empty to use DefaultBinary. On Windows, exec.LookPath
// resolves launcher shims (claude.cmd / claude.exe) transparently.
func Detect(ctx context.Context, binary string) (string, error) {
	if binary == "" {
		binary = DefaultBinary
	}
	path, err := exec.LookPath(binary)
	if err != nil {
		return "", fmt.Errorf("locate %q on PATH (pass --claude-version to skip host detection): %w", binary, err)
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, path, "--version")
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("run %q --version: %w (stderr: %s)", path, err, strings.TrimSpace(stderr.String()))
	}

	return Parse(stdout.String())
}

// Parse extracts a semantic version from arbitrary `--version` output.
func Parse(out string) (string, error) {
	m := semverRE.FindString(out)
	if m == "" {
		return "", fmt.Errorf("no semantic version found in %q", strings.TrimSpace(out))
	}
	return m, nil
}
