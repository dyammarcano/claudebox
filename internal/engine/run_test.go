package engine

import (
	"reflect"
	"testing"
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
