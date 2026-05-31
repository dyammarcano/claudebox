package engine

import (
	"archive/tar"
	"io"
	"testing"
)

func TestBuildContextTar(t *testing.T) {
	t.Parallel()
	df := []byte("FROM scratch\n")
	ep := []byte("#!/bin/sh\necho hi\n")

	r, err := buildContextTar(df, ep)
	if err != nil {
		t.Fatalf("buildContextTar: %v", err)
	}

	want := map[string]struct {
		mode int64
		data string
	}{
		"Dockerfile":    {0o644, string(df)},
		"entrypoint.sh": {0o755, string(ep)},
	}

	tr := tar.NewReader(r)
	seen := map[string]bool{}
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("tar next: %v", err)
		}
		exp, ok := want[hdr.Name]
		if !ok {
			t.Errorf("unexpected tar entry %q", hdr.Name)
			continue
		}
		if hdr.Mode != exp.mode {
			t.Errorf("%s mode = %o, want %o", hdr.Name, hdr.Mode, exp.mode)
		}
		body, _ := io.ReadAll(tr)
		if string(body) != exp.data {
			t.Errorf("%s body = %q, want %q", hdr.Name, body, exp.data)
		}
		seen[hdr.Name] = true
	}
	for name := range want {
		if !seen[name] {
			t.Errorf("missing tar entry %q", name)
		}
	}
}

func TestBuildContextTarRejectsEmpty(t *testing.T) {
	t.Parallel()
	if _, err := buildContextTar(nil, []byte("x")); err == nil {
		t.Error("want error for empty Dockerfile")
	}
	if _, err := buildContextTar([]byte("x"), nil); err == nil {
		t.Error("want error for empty entrypoint")
	}
}
