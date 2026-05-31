package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalizeDefaults(t *testing.T) {
	t.Parallel()
	c := Config{Workspace: ".", Prompt: "x"}
	if err := c.Normalize(); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if c.MaxTurns != 20 {
		t.Errorf("MaxTurns default = %d, want 20", c.MaxTurns)
	}
	if c.OutputFormat != "stream-json" {
		t.Errorf("OutputFormat default = %q, want stream-json", c.OutputFormat)
	}
	if !filepath.IsAbs(c.Workspace) {
		t.Errorf("Workspace = %q, want absolute", c.Workspace)
	}
}

func TestNormalizeLowercasesFormat(t *testing.T) {
	t.Parallel()
	c := Config{Workspace: ".", Prompt: "x", OutputFormat: " STREAM-JSON "}
	if err := c.Normalize(); err != nil {
		t.Fatalf("Normalize: %v", err)
	}
	if c.OutputFormat != "stream-json" {
		t.Errorf("OutputFormat = %q, want stream-json", c.OutputFormat)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	file := filepath.Join(dir, "afile")
	if err := os.WriteFile(file, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{"valid", Config{Workspace: dir, Prompt: "do x", MaxTurns: 5, OutputFormat: "text"}, false},
		{"missing prompt", Config{Workspace: dir, MaxTurns: 5, OutputFormat: "text"}, true},
		{"blank prompt", Config{Workspace: dir, Prompt: "   ", MaxTurns: 5, OutputFormat: "text"}, true},
		{"missing workspace", Config{Prompt: "x", MaxTurns: 5, OutputFormat: "text"}, true},
		{"workspace not found", Config{Workspace: filepath.Join(dir, "nope"), Prompt: "x", MaxTurns: 5, OutputFormat: "text"}, true},
		{"workspace is file", Config{Workspace: file, Prompt: "x", MaxTurns: 5, OutputFormat: "text"}, true},
		{"zero turns", Config{Workspace: dir, Prompt: "x", MaxTurns: 0, OutputFormat: "text"}, true},
		{"negative turns", Config{Workspace: dir, Prompt: "x", MaxTurns: -1, OutputFormat: "text"}, true},
		{"bad format", Config{Workspace: dir, Prompt: "x", MaxTurns: 5, OutputFormat: "yaml"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			err := tt.cfg.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestResolvedImageTag(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		cfg     Config
		version string
		want    string
	}{
		{"explicit tag wins", Config{ImageTag: "custom:1"}, "2.1.158", "custom:1"},
		{"derived from version", Config{}, "2.1.158", "claudebox:2.1.158"},
		{"empty version falls back", Config{}, "", "claudebox:latest"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.cfg.ResolvedImageTag(tt.version); got != tt.want {
				t.Errorf("ResolvedImageTag(%q) = %q, want %q", tt.version, got, tt.want)
			}
		})
	}
}

func TestNormalizeMemoryParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		memory  string
		want    int64
		wantErr bool
	}{
		{"2g", "2g", 2 << 30, false},
		{"512m", "512m", 512 << 20, false},
		{"empty unlimited", "", 0, false},
		{"zero unlimited", "0", 0, false},
		{"whitespace", "  1g  ", 1 << 30, false},
		{"garbage", "lots", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := Config{Workspace: ".", Prompt: "x", Memory: tt.memory}
			err := c.Normalize()
			if (err != nil) != tt.wantErr {
				t.Fatalf("Normalize() err = %v, wantErr %v", err, tt.wantErr)
			}
			if err == nil && c.MemoryBytes != tt.want {
				t.Errorf("MemoryBytes = %d, want %d", c.MemoryBytes, tt.want)
			}
		})
	}
}

func TestValidateHardeningRanges(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	base := func() Config {
		return Config{Workspace: dir, Prompt: "x", MaxTurns: 5, OutputFormat: "text"}
	}
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr bool
	}{
		{"negative cpus", func(c *Config) { c.CPUs = -1 }, true},
		{"zero cpus ok", func(c *Config) { c.CPUs = 0 }, false},
		{"negative pids", func(c *Config) { c.PidsLimit = -1 }, true},
		{"memory too low", func(c *Config) { c.MemoryBytes = 1024 }, true},
		{"memory ok", func(c *Config) { c.MemoryBytes = 256 << 20 }, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := base()
			tt.mutate(&c)
			if err := c.Validate(); (err != nil) != tt.wantErr {
				t.Errorf("Validate() err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
