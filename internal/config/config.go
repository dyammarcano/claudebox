// Package config defines the runtime configuration for a claudebox sandbox run
// and the validation rules that guard it.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// validOutputFormats is the set of --output-format values claude accepts.
var validOutputFormats = map[string]struct{}{
	"text":        {},
	"json":        {},
	"stream-json": {},
}

// Config holds everything needed to build the image and run a single
// throwaway Claude Code container. It is populated from CLI flags and
// validated before any Docker work begins.
type Config struct {
	// Workspace is the host directory bind-mounted read-write at /workspace.
	// Claude operates here; this is the only path that persists after the run.
	Workspace string

	// Prompt is the task handed to Claude in headless mode. Required.
	Prompt string

	// MaxTurns caps the agent loop. Must be > 0.
	MaxTurns int

	// OutputFormat is one of text|json|stream-json.
	OutputFormat string

	// AllowedTools is the --allowedTools allowlist (permission-rule syntax),
	// e.g. "Read,Edit,Bash(git diff *)". Empty means none pre-approved.
	AllowedTools string

	// PermissionMode optionally sets a baseline permission mode for the run,
	// e.g. "dontAsk" or "acceptEdits".
	PermissionMode string

	// ExtraArgs are additional raw claude flags appended verbatim.
	ExtraArgs []string

	// APIKey, when non-empty, is injected as ANTHROPIC_API_KEY (API billing).
	// When empty, the run falls back to the seeded OAuth credentials.
	APIKey string

	// ClaudeHome is the host ~/.claude directory whose credentials/settings
	// are copied into the ephemeral seed. Defaults to the user's ~/.claude.
	ClaudeHome string

	// SeedBaseDir overrides the base directory under which the ephemeral
	// credential seed is created. Empty selects a default inside Docker
	// Desktop's file-sharing scope (~/.claudebox/seeds). Set this if your
	// Docker file-sharing config requires a specific location.
	SeedBaseDir string

	// SkipSettings, when true, copies ONLY credentials into the seed (not
	// settings.json). Useful because a host settings.json often references
	// hooks/plugins/MCP servers that do not exist inside the container and
	// could break a headless run.
	SkipSettings bool

	// ClaudeVersion pins the in-container Claude version (CLAUDE_VERSION build
	// arg). Empty means "detect from the host `claude --version`".
	ClaudeVersion string

	// ImageTag is the built image tag. Empty means a tag derived from the
	// resolved Claude version (claudebox:<version>).
	ImageTag string

	// Network is the container network mode (e.g. "bridge", "none"). Empty
	// uses Docker's default bridge — full outbound, required for the API.
	Network string

	// NoCache forces a clean image rebuild.
	NoCache bool

	// Rebuild forces a build even if the target image tag already exists.
	Rebuild bool

	// Keep leaves the container in place after exit (debugging). Default is
	// force-remove for a clean instance every time.
	Keep bool
}

// Normalize fills defaults and resolves the workspace to an absolute path.
func (c *Config) Normalize() error {
	if c.MaxTurns == 0 {
		c.MaxTurns = 20
	}
	if c.OutputFormat == "" {
		c.OutputFormat = "stream-json"
	}
	c.OutputFormat = strings.ToLower(strings.TrimSpace(c.OutputFormat))
	if c.Workspace != "" {
		abs, err := filepath.Abs(c.Workspace)
		if err != nil {
			return fmt.Errorf("resolve workspace path: %w", err)
		}
		c.Workspace = abs
	}
	return nil
}

// Validate checks the configuration is internally consistent and that
// referenced host paths exist. Call Normalize first.
func (c *Config) Validate() error {
	if strings.TrimSpace(c.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	if c.Workspace == "" {
		return fmt.Errorf("workspace is required")
	}
	info, err := os.Stat(c.Workspace)
	if err != nil {
		return fmt.Errorf("workspace %q: %w", c.Workspace, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("workspace %q is not a directory", c.Workspace)
	}
	if c.MaxTurns <= 0 {
		return fmt.Errorf("max-turns must be > 0, got %d", c.MaxTurns)
	}
	if _, ok := validOutputFormats[c.OutputFormat]; !ok {
		return fmt.Errorf("invalid output-format %q (want text|json|stream-json)", c.OutputFormat)
	}
	return nil
}

// ResolvedImageTag returns the image tag to build/run, deriving one from the
// Claude version when ImageTag is unset.
func (c *Config) ResolvedImageTag(claudeVersion string) string {
	if c.ImageTag != "" {
		return c.ImageTag
	}
	v := claudeVersion
	if v == "" {
		v = "latest"
	}
	return "claudebox:" + v
}
