package credentials

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestPrepareSeedCopiesWhitelist(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeFile(t, filepath.Join(home, CredentialsFile), `{"token":"secret"}`)
	writeFile(t, filepath.Join(home, SettingsFile), `{"model":"opus"}`)
	// A non-whitelisted file must NOT be copied.
	writeFile(t, filepath.Join(home, "history.jsonl"), "noisy")

	seed, cleanup, err := PrepareSeed(home, true, t.TempDir())
	defer cleanup()
	if err != nil {
		t.Fatalf("PrepareSeed: %v", err)
	}

	if !seed.HasCredentials {
		t.Error("HasCredentials = false, want true")
	}
	if len(seed.Copied) != 2 {
		t.Errorf("copied %v, want 2 files", seed.Copied)
	}
	if _, err := os.Stat(filepath.Join(seed.Dir, CredentialsFile)); err != nil {
		t.Errorf("credentials not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(seed.Dir, SettingsFile)); err != nil {
		t.Errorf("settings not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(seed.Dir, "history.jsonl")); !os.IsNotExist(err) {
		t.Errorf("non-whitelisted file leaked into seed (err=%v)", err)
	}
}

func TestPrepareSeedSkipsSettings(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeFile(t, filepath.Join(home, CredentialsFile), `{"token":"x"}`)
	writeFile(t, filepath.Join(home, SettingsFile), `{"model":"opus"}`)

	seed, cleanup, err := PrepareSeed(home, false, t.TempDir()) // includeSettings=false
	defer cleanup()
	if err != nil {
		t.Fatalf("PrepareSeed: %v", err)
	}
	if !seed.HasCredentials {
		t.Error("HasCredentials = false, want true")
	}
	if _, err := os.Stat(filepath.Join(seed.Dir, SettingsFile)); !os.IsNotExist(err) {
		t.Errorf("settings.json should be skipped (err=%v)", err)
	}
	if len(seed.Copied) != 1 {
		t.Errorf("copied %v, want only credentials", seed.Copied)
	}
}

func TestPrepareSeedNoCredentials(t *testing.T) {
	t.Parallel()
	home := t.TempDir() // empty: no creds, no settings

	seed, cleanup, err := PrepareSeed(home, true, t.TempDir())
	defer cleanup()
	if err != nil {
		t.Fatalf("PrepareSeed should not error on missing files: %v", err)
	}
	if seed.HasCredentials {
		t.Error("HasCredentials = true, want false for empty home")
	}
	if len(seed.Copied) != 0 {
		t.Errorf("copied %v, want none", seed.Copied)
	}
}

func TestPrepareSeedCleanupRemovesDir(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	writeFile(t, filepath.Join(home, CredentialsFile), "x")

	seed, cleanup, err := PrepareSeed(home, true, t.TempDir())
	if err != nil {
		t.Fatalf("PrepareSeed: %v", err)
	}
	dir := seed.Dir
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("seed dir should exist before cleanup: %v", err)
	}
	cleanup()
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Errorf("seed dir should be gone after cleanup (err=%v)", err)
	}
}

func TestPrepareSeedSkipsDirectories(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	// A directory named like a whitelist entry must be skipped, not copied.
	if err := os.Mkdir(filepath.Join(home, SettingsFile), 0o700); err != nil {
		t.Fatal(err)
	}
	seed, cleanup, err := PrepareSeed(home, true, t.TempDir())
	defer cleanup()
	if err != nil {
		t.Fatalf("PrepareSeed: %v", err)
	}
	if len(seed.Copied) != 0 {
		t.Errorf("copied %v, want none (dir should be skipped)", seed.Copied)
	}
}
