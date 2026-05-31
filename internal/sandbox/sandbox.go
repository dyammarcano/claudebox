// Package sandbox orchestrates the full claudebox lifecycle: detect the host
// Claude version, build the pinned image, prepare ephemeral credentials, run
// the throwaway container, and clean everything up.
package sandbox

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/inovacc/claudebox/assets"
	"github.com/inovacc/claudebox/internal/config"
	"github.com/inovacc/claudebox/internal/credentials"
	"github.com/inovacc/claudebox/internal/engine"
	"github.com/inovacc/claudebox/internal/version"
)

// defaultSeedBase returns a directory for the ephemeral credential seed that is
// inside Docker Desktop's default file-sharing scope. The user home is shared
// by default on Windows (C:\Users\...) and macOS (/Users/...); the OS temp dir
// is not guaranteed to be. Returns "" (OS default temp) if home is unavailable
// or the base cannot be created.
func defaultSeedBase() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	base := filepath.Join(home, ".claudebox", "seeds")
	if err := os.MkdirAll(base, 0o700); err != nil {
		return ""
	}
	return base
}

// Sandbox wires the pieces of a run together.
type Sandbox struct {
	cfg    config.Config
	eng    *engine.Engine
	log    *slog.Logger
	stdout io.Writer
	stderr io.Writer
}

// New constructs a Sandbox. stdout/stderr receive the container's output
// streams; pass os.Stdout/os.Stderr in normal use.
func New(cfg config.Config, eng *engine.Engine, log *slog.Logger, stdout, stderr io.Writer) *Sandbox {
	if log == nil {
		log = slog.Default()
	}
	return &Sandbox{cfg: cfg, eng: eng, log: log, stdout: stdout, stderr: stderr}
}

// Run executes the lifecycle and returns the container's exit code. A non-nil
// error indicates a claudebox-level failure (build/setup/Docker); a non-zero
// exit code with a nil error means Claude itself exited non-zero.
func (s *Sandbox) Run(ctx context.Context) (int, error) {
	if err := s.eng.Ping(ctx); err != nil {
		return -1, err
	}

	// 1. Resolve the Claude version to pin.
	claudeVersion := s.cfg.ClaudeVersion
	if claudeVersion == "" {
		detected, err := version.Detect(ctx, "")
		if err != nil {
			return -1, fmt.Errorf("detect host claude version (pass --claude-version to override): %w", err)
		}
		claudeVersion = detected
	}
	s.log.Info("resolved claude version", "version", claudeVersion)

	tag := s.cfg.ResolvedImageTag(claudeVersion)

	// 2. Build the image unless it already exists (and no rebuild requested).
	if err := s.ensureImage(ctx, tag, claudeVersion); err != nil {
		return -1, err
	}

	// 3. Prepare ephemeral credentials seed.
	claudeHome := s.cfg.ClaudeHome
	if claudeHome == "" {
		h, err := credentials.DefaultClaudeHome()
		if err != nil {
			return -1, err
		}
		claudeHome = h
	}
	// PrepareSeed always returns a non-nil cleanup, even on error, so register
	// it immediately to remove the temp dir on every path. seed is only safe to
	// dereference after the error check below. The base dir is chosen to fall
	// inside Docker Desktop's default file-sharing scope (the user home), since
	// the OS temp dir may not be shared on Windows/macOS.
	seedBase := s.cfg.SeedBaseDir
	if seedBase == "" {
		seedBase = defaultSeedBase()
	}
	seed, cleanup, err := credentials.PrepareSeed(claudeHome, !s.cfg.SkipSettings, seedBase)
	defer cleanup()
	if err != nil {
		return -1, fmt.Errorf("prepare credentials seed: %w", err)
	}

	// Fail fast on the common misconfiguration: no API key AND no OAuth creds.
	if s.cfg.APIKey == "" && !seed.HasCredentials {
		return -1, fmt.Errorf(
			"no ANTHROPIC_API_KEY and no %s found in %s; "+
				"log in with `claude` first or pass --api-key",
			credentials.CredentialsFile, claudeHome)
	}
	s.log.Info("prepared credential seed", "files", seed.Copied, "oauth", seed.HasCredentials)

	// 4. Run the throwaway container.
	exitCode, err := s.eng.Run(ctx, engine.RunOptions{
		Image:         tag,
		Env:           s.buildEnv(),
		WorkspaceHost: s.cfg.Workspace,
		SeedHost:      seed.Dir,
		Network:       s.cfg.Network,
		Stdout:        s.stdout,
		Stderr:        s.stderr,
		Keep:          s.cfg.Keep,
	})
	if err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

// ensureImage builds the image when missing or when a rebuild is requested.
func (s *Sandbox) ensureImage(ctx context.Context, tag, claudeVersion string) error {
	if !s.cfg.Rebuild && !s.cfg.NoCache {
		exists, err := s.eng.ImageExists(ctx, tag)
		if err != nil {
			return err
		}
		if exists {
			s.log.Info("reusing existing image", "tag", tag)
			return nil
		}
	}
	return s.eng.BuildImage(ctx, engine.BuildOptions{
		Tag:           tag,
		ClaudeVersion: claudeVersion,
		NoCache:       s.cfg.NoCache,
		Dockerfile:    assets.Dockerfile,
		Entrypoint:    assets.Entrypoint,
	})
}

// buildEnv assembles the container environment from the config.
func (s *Sandbox) buildEnv() map[string]string {
	env := map[string]string{
		"PROMPT":        s.cfg.Prompt,
		"MAX_TURNS":     strconv.Itoa(s.cfg.MaxTurns),
		"OUTPUT_FORMAT": s.cfg.OutputFormat,
		"SEED_DIR":      "/seed",
	}
	if s.cfg.AllowedTools != "" {
		env["ALLOWED_TOOLS"] = s.cfg.AllowedTools
	}
	if s.cfg.PermissionMode != "" {
		env["PERMISSION_MODE"] = s.cfg.PermissionMode
	}
	if len(s.cfg.ExtraArgs) > 0 {
		env["EXTRA_ARGS"] = strings.Join(s.cfg.ExtraArgs, " ")
	}
	if s.cfg.APIKey != "" {
		env["ANTHROPIC_API_KEY"] = s.cfg.APIKey
	}
	return env
}
