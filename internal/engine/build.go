package engine

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/docker/docker/api/types/build"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/pkg/jsonmessage"
)

// BuildOptions configures an image build.
type BuildOptions struct {
	// Tag is the image tag to produce, e.g. "claudebox:2.1.158".
	Tag string
	// ClaudeVersion is passed as the CLAUDE_VERSION build arg so the in-image
	// Claude matches the host (one of stable|latest|X.Y.Z).
	ClaudeVersion string
	// NoCache forces a clean rebuild.
	NoCache bool
	// Dockerfile is the embedded Dockerfile bytes.
	Dockerfile []byte
	// Entrypoint is the embedded entrypoint.sh bytes.
	Entrypoint []byte
}

// ImageExists reports whether an image with the given tag is present locally.
func (e *Engine) ImageExists(ctx context.Context, tag string) (bool, error) {
	images, err := e.cli.ImageList(ctx, image.ListOptions{})
	if err != nil {
		return false, fmt.Errorf("list images: %w", err)
	}
	for _, img := range images {
		if slices.Contains(img.RepoTags, tag) {
			return true, nil
		}
	}
	return false, nil
}

// BuildImage builds the sandbox image from the embedded build context, streaming
// build progress to stderr. It uses the classic builder via the engine API,
// which ignores the `# syntax=` directive (BuildKit-session builds are a
// backlog item).
func (e *Engine) BuildImage(ctx context.Context, opts BuildOptions) error {
	if opts.Tag == "" {
		return fmt.Errorf("build: tag is required")
	}
	tarCtx, err := buildContextTar(opts.Dockerfile, opts.Entrypoint)
	if err != nil {
		return fmt.Errorf("build: assemble context: %w", err)
	}

	version := opts.ClaudeVersion
	if version == "" {
		version = "stable"
	}

	e.log.Info("building sandbox image", "tag", opts.Tag, "claude_version", version, "no_cache", opts.NoCache)

	resp, err := e.cli.ImageBuild(ctx, tarCtx, build.ImageBuildOptions{
		Tags:       []string{opts.Tag},
		Dockerfile: "Dockerfile",
		BuildArgs:  map[string]*string{"CLAUDE_VERSION": &version},
		Remove:     true,
		NoCache:    opts.NoCache,
		PullParent: false,
	})
	if err != nil {
		return fmt.Errorf("build: image build: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Render the build log and surface any in-stream build error.
	if err := jsonmessage.DisplayJSONMessagesStream(resp.Body, os.Stderr, 0, false, nil); err != nil {
		return fmt.Errorf("build failed: %w", err)
	}
	e.log.Info("sandbox image ready", "tag", opts.Tag)
	return nil
}

// buildContextTar produces an in-memory tar containing the Dockerfile and the
// entrypoint script at the context root.
func buildContextTar(dockerfile, entrypoint []byte) (io.Reader, error) {
	if len(dockerfile) == 0 {
		return nil, fmt.Errorf("empty Dockerfile")
	}
	if len(entrypoint) == 0 {
		return nil, fmt.Errorf("empty entrypoint.sh")
	}

	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	files := []struct {
		name string
		data []byte
		mode int64
	}{
		{"Dockerfile", dockerfile, 0o644},
		{"entrypoint.sh", entrypoint, 0o755},
	}
	for _, f := range files {
		hdr := &tar.Header{
			Name:     f.name,
			Mode:     f.mode,
			Size:     int64(len(f.data)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return nil, fmt.Errorf("tar header %s: %w", f.name, err)
		}
		if _, err := tw.Write(f.data); err != nil {
			return nil, fmt.Errorf("tar write %s: %w", f.name, err)
		}
	}
	if err := tw.Close(); err != nil {
		return nil, fmt.Errorf("close tar: %w", err)
	}
	return &buf, nil
}
