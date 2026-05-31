package version

import "testing"

func TestParse(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		in      string
		want    string
		wantErr bool
	}{
		{"claude format", "2.1.158 (Claude Code)", "2.1.158", false},
		{"bare semver", "1.0.0", "1.0.0", false},
		{"with newline", "\n2.0.76 (Claude Code)\n", "2.0.76", false},
		{"prerelease", "3.2.1-beta.4 (Claude Code)", "3.2.1-beta.4", false},
		{"prefixed text", "claude version 2.1.0", "2.1.0", false},
		{"no version", "Claude Code", "", true},
		{"empty", "", "", true},
		{"two-part not matched", "v2.1 something", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := Parse(tt.in)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Parse(%q) err = %v, wantErr %v", tt.in, err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("Parse(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}
