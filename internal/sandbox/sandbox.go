// Package sandbox orchestrates the full claudebox lifecycle: resolve the Claude
// version to pin, build the image, prepare ephemeral credentials, run the
// throwaway container, and clean everything up.
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
)

// defaultClaudeVersion is the version target installed in the container when the
// caller does not pin one with --claude-version. It matches the Dockerfile's
// CLAUDE_VERSION default and the install script's accepted targets
// (stable | latest | X.Y.Z).
const defaultClaudeVersion = "stable"

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

	// 1. Resolve the Claude version to pin. Defaults to "stable"; override with
	//    --claude-version (one of stable|latest|X.Y.Z).
	claudeVersion := s.cfg.ClaudeVersion
	if claudeVersion == "" {
		claudeVersion = defaultClaudeVersion
	}
	s.log.Info("using claude version", "version", claudeVersion)

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
	exitCode, err := s.eng.Run(ctx, s.runOptions(tag, seed.Dir))
	if err != nil {
		return exitCode, err
	}
	return exitCode, nil
}

// containerHome is the in-container HOME (matches the Dockerfile HOME_DIR). The
// Claude install lives at $containerHome/.local, so the read-only-rootfs tmpfs
// mounts target only the writable subdirectories — never $containerHome itself,
// which would mask the binary.
const containerHome = "/home/agent"

// hardenedTmpfs returns the tmpfs mounts that provide the writable paths Claude
// needs when the root filesystem is read-only: a scratch /tmp plus the Claude
// state, cache, and config directories under HOME. /workspace is a separate
// writable bind mount.
func hardenedTmpfs() map[string]string {
	return map[string]string{
		"/tmp":                     "rw,nosuid,nodev,size=256m",
		containerHome + "/.claude": "rw,nosuid,nodev,uid=1000,gid=1000,mode=0700,size=128m",
		containerHome + "/.cache":  "rw,nosuid,nodev,uid=1000,gid=1000,size=256m",
		containerHome + "/.config": "rw,nosuid,nodev,uid=1000,gid=1000,size=64m",
	}
}

// runOptions assembles the engine RunOptions, applying default-on hardening
// unless the caller opted out via --no-hardening / --writable-rootfs.
func (s *Sandbox) runOptions(tag, seedDir string) engine.RunOptions {
	opts := engine.RunOptions{
		Image:         tag,
		Env:           s.buildEnv(),
		WorkspaceHost: s.cfg.Workspace,
		SeedHost:      seedDir,
		Network:       s.cfg.Network,
		Stdout:        s.stdout,
		Stderr:        s.stderr,
		Keep:          s.cfg.Keep,
	}

	if s.cfg.NoHardening {
		s.log.Warn("container hardening disabled (--no-hardening)")
		return opts
	}

	// Capabilities + privilege escalation lockdown.
	opts.CapDrop = []string{"ALL"}
	opts.CapAdd = s.cfg.CapAdd
	opts.SecurityOpt = []string{"no-new-privileges"}

	// Resource limits (0 = unlimited, left unset).
	opts.MemoryBytes = s.cfg.MemoryBytes
	if s.cfg.CPUs > 0 {
		opts.NanoCPUs = int64(s.cfg.CPUs * 1e9)
	}
	opts.PidsLimit = s.cfg.PidsLimit

	// Read-only root filesystem with targeted tmpfs for writable paths.
	if !s.cfg.WritableRootfs {
		opts.ReadonlyRootfs = true
		opts.Tmpfs = hardenedTmpfs()
	}

	s.log.Info("container hardening enabled",
		"readonly_rootfs", opts.ReadonlyRootfs,
		"memory_bytes", opts.MemoryBytes,
		"nano_cpus", opts.NanoCPUs,
		"pids_limit", opts.PidsLimit,
		"cap_add", opts.CapAdd)
	return opts
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
