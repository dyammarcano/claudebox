// Package credentials locates the host's Claude credentials/settings and copies
// a minimal, scrubbable subset into an ephemeral seed directory that is
// bind-mounted read-only into the sandbox container.
//
// The host ~/.claude is never modified or mounted directly: OAuth tokens
// refresh at runtime (Claude rewrites .credentials.json) and a read-only mount
// of the live directory would break both refresh and history writes. Instead we
// copy a seed, mount it read-only at /seed, and the container entrypoint copies
// it into a writable ~/.claude inside the throwaway container.
package credentials

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CredentialsFile is the OAuth token file used for subscription auth.
const CredentialsFile = ".credentials.json"

// SettingsFile is the user settings file (model, etc.).
const SettingsFile = "settings.json"

// DefaultClaudeHome returns the host ~/.claude directory.
func DefaultClaudeHome() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// Seed describes a prepared ephemeral credential directory.
type Seed struct {
	// Dir is the temp directory to bind-mount read-only at /seed.
	Dir string
	// Copied lists the seed-relative filenames successfully copied.
	Copied []string
	// HasCredentials reports whether the OAuth credentials file was copied.
	HasCredentials bool
}

// PrepareSeed copies the whitelisted credential/settings files from claudeHome
// into a freshly created 0700 temp directory under baseDir. When includeSettings
// is false, only the OAuth credentials file is copied (settings.json is
// skipped). When baseDir is empty the OS default temp location is used; callers
// on Docker Desktop should pass a base inside the engine's file-sharing scope
// (see sandbox.defaultSeedBase). The returned cleanup function removes the temp
// directory and must always be called (e.g. via defer), even on error paths.
//
// Missing whitelist files are skipped silently; it is the caller's job to
// decide whether the absence of credentials is fatal (it is not, when an API
// key is supplied instead).
func PrepareSeed(claudeHome string, includeSettings bool, baseDir string) (*Seed, func(), error) {
	dir, err := os.MkdirTemp(baseDir, "claudebox-seed-*")
	if err != nil {
		return nil, func() {}, fmt.Errorf("create seed temp dir: %w", err)
	}
	cleanup := func() { _ = os.RemoveAll(dir) }

	if err := os.Chmod(dir, 0o700); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("chmod seed dir: %w", err)
	}

	wanted := []string{CredentialsFile}
	if includeSettings {
		wanted = append(wanted, SettingsFile)
	}

	seed := &Seed{Dir: dir}
	for _, name := range wanted {
		src := filepath.Join(claudeHome, name)
		info, statErr := os.Stat(src)
		if statErr != nil || info.IsDir() {
			continue // not present (or a dir) — skip
		}
		dst := filepath.Join(dir, name)
		if err := copyFile(src, dst, 0o600); err != nil {
			cleanup()
			return nil, func() {}, fmt.Errorf("copy %s into seed: %w", name, err)
		}
		seed.Copied = append(seed.Copied, name)
		if name == CredentialsFile {
			seed.HasCredentials = true
		}
	}

	return seed, cleanup, nil
}

// copyFile copies src to dst, truncating dst, and sets its mode.
func copyFile(src, dst string, mode os.FileMode) (err error) {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := out.Close(); cerr != nil && err == nil {
			err = cerr
		}
	}()

	if _, err = io.Copy(out, in); err != nil {
		return err
	}
	return os.Chmod(dst, mode)
}
