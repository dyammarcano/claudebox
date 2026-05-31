package engine

import (
	"reflect"
	"testing"

	"github.com/docker/docker/api/types/mount"
)

func TestEnvSliceSorted(t *testing.T) {
	t.Parallel()
	got := envSlice(map[string]string{
		"PROMPT":    "do x",
		"MAX_TURNS": "20",
		"SEED_DIR":  "/seed",
	})
	want := []string{"MAX_TURNS=20", "PROMPT=do x", "SEED_DIR=/seed"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("envSlice = %v, want %v", got, want)
	}
}

func TestEnvSliceEmpty(t *testing.T) {
	t.Parallel()
	if got := envSlice(nil); len(got) != 0 {
		t.Errorf("envSlice(nil) = %v, want empty", got)
	}
}

func TestShort(t *testing.T) {
	t.Parallel()
	tests := []struct {
		in   string
		want string
	}{
		{"abcdef0123456789", "abcdef012345"}, // 16 -> 12
		{"short", "short"},                   // < 12 unchanged
		{"", ""},                             // empty unchanged
		{"123456789012", "123456789012"},     // exactly 12 unchanged
		{"1234567890123", "123456789012"},    // 13 -> 12
	}
	for _, tt := range tests {
		if got := short(tt.in); got != tt.want {
			t.Errorf("short(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestMountsWorkspaceAndSeed(t *testing.T) {
	t.Parallel()
	m := RunOptions{WorkspaceHost: "/host/ws", SeedHost: "/host/seed"}.mounts()
	if len(m) != 2 {
		t.Fatalf("mounts = %d, want 2", len(m))
	}
	if m[0].Source != "/host/ws" || m[0].Target != workspaceTarget || m[0].ReadOnly {
		t.Errorf("workspace mount wrong: %+v", m[0])
	}
	if m[1].Source != "/host/seed" || m[1].Target != seedTarget || !m[1].ReadOnly {
		t.Errorf("seed mount must be read-only: %+v", m[1])
	}
	if m[0].Type != mount.TypeBind {
		t.Errorf("workspace mount type = %v, want bind", m[0].Type)
	}
}

func TestMountsNoSeed(t *testing.T) {
	t.Parallel()
	m := RunOptions{WorkspaceHost: "/host/ws"}.mounts()
	if len(m) != 1 {
		t.Fatalf("mounts = %d, want 1 (no seed)", len(m))
	}
}

func TestToHostConfigHardened(t *testing.T) {
	t.Parallel()
	pids := int64(512)
	hc := RunOptions{
		WorkspaceHost:  "/ws",
		CapDrop:        []string{"ALL"},
		CapAdd:         []string{"NET_ADMIN"},
		SecurityOpt:    []string{"no-new-privileges"},
		ReadonlyRootfs: true,
		Tmpfs:          map[string]string{"/tmp": "rw,size=64m"},
		MemoryBytes:    2 << 30,
		NanoCPUs:       2_000_000_000,
		PidsLimit:      pids,
		Network:        "bridge",
	}.toHostConfig()

	if !hc.ReadonlyRootfs {
		t.Error("ReadonlyRootfs = false, want true")
	}
	if len(hc.CapDrop) != 1 || hc.CapDrop[0] != "ALL" {
		t.Errorf("CapDrop = %v, want [ALL]", hc.CapDrop)
	}
	if len(hc.CapAdd) != 1 || hc.CapAdd[0] != "NET_ADMIN" {
		t.Errorf("CapAdd = %v, want [NET_ADMIN]", hc.CapAdd)
	}
	if len(hc.SecurityOpt) != 1 || hc.SecurityOpt[0] != "no-new-privileges" {
		t.Errorf("SecurityOpt = %v", hc.SecurityOpt)
	}
	if hc.Tmpfs["/tmp"] == "" {
		t.Errorf("Tmpfs missing /tmp: %v", hc.Tmpfs)
	}
	if hc.Resources.Memory != 2<<30 {
		t.Errorf("Memory = %d, want %d", hc.Resources.Memory, int64(2<<30))
	}
	if hc.Resources.NanoCPUs != 2_000_000_000 {
		t.Errorf("NanoCPUs = %d", hc.Resources.NanoCPUs)
	}
	if hc.Resources.PidsLimit == nil || *hc.Resources.PidsLimit != 512 {
		t.Errorf("PidsLimit = %v, want 512", hc.Resources.PidsLimit)
	}
	if hc.NetworkMode != "bridge" {
		t.Errorf("NetworkMode = %q, want bridge", hc.NetworkMode)
	}
}

func TestToHostConfigNoHardening(t *testing.T) {
	t.Parallel()
	// Zero-value hardening fields must produce a permissive config.
	hc := RunOptions{WorkspaceHost: "/ws"}.toHostConfig()
	if hc.ReadonlyRootfs {
		t.Error("ReadonlyRootfs = true, want false when unset")
	}
	if len(hc.CapDrop) != 0 {
		t.Errorf("CapDrop = %v, want empty", hc.CapDrop)
	}
	if len(hc.SecurityOpt) != 0 {
		t.Errorf("SecurityOpt = %v, want empty", hc.SecurityOpt)
	}
	if hc.Resources.Memory != 0 || hc.Resources.NanoCPUs != 0 || hc.Resources.PidsLimit != nil {
		t.Errorf("resource limits set when unset: %+v", hc.Resources)
	}
	if len(hc.Tmpfs) != 0 {
		t.Errorf("Tmpfs = %v, want empty", hc.Tmpfs)
	}
}
