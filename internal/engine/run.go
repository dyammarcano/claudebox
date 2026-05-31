package engine

import (
	"context"
	"errors"
	"fmt"
	"io"
	"sort"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/pkg/stdcopy"
)

// RunOptions configures a single container run.
type RunOptions struct {
	// Image is the tag to run.
	Image string
	// Env are container environment variables (KEY -> value).
	Env map[string]string
	// WorkspaceHost is the host path bind-mounted read-write at /workspace.
	WorkspaceHost string
	// SeedHost is the host path bind-mounted read-only at /seed (may be empty
	// when authenticating purely via ANTHROPIC_API_KEY).
	SeedHost string
	// Network is the container network mode; empty uses the default bridge.
	Network string
	// Stdout receives the de-multiplexed container stdout.
	Stdout io.Writer
	// Stderr receives the de-multiplexed container stderr.
	Stderr io.Writer
	// Keep, when true, leaves the container in place after exit instead of
	// force-removing it.
	Keep bool
}

const (
	workspaceTarget = "/workspace"
	seedTarget      = "/seed"
)

// Run creates, starts, and streams a throwaway container, returning its exit
// code. Unless opts.Keep is set, the container is force-removed before Run
// returns (success or failure).
func (e *Engine) Run(ctx context.Context, opts RunOptions) (int, error) {
	if opts.Image == "" {
		return -1, fmt.Errorf("run: image is required")
	}
	if opts.WorkspaceHost == "" {
		return -1, fmt.Errorf("run: workspace host path is required")
	}

	mounts := []mount.Mount{{
		Type:   mount.TypeBind,
		Source: opts.WorkspaceHost,
		Target: workspaceTarget,
	}}
	if opts.SeedHost != "" {
		mounts = append(mounts, mount.Mount{
			Type:     mount.TypeBind,
			Source:   opts.SeedHost,
			Target:   seedTarget,
			ReadOnly: true,
		})
	}

	hostCfg := &container.HostConfig{Mounts: mounts}
	if opts.Network != "" {
		hostCfg.NetworkMode = container.NetworkMode(opts.Network)
	}

	containerCfg := &container.Config{
		Image:      opts.Image,
		Env:        envSlice(opts.Env),
		WorkingDir: workspaceTarget,
		Tty:        false, // keep stdout/stderr separable for stdcopy
	}

	created, err := e.cli.ContainerCreate(ctx, containerCfg, hostCfg, nil, nil, "")
	if err != nil {
		return -1, fmt.Errorf("run: create container: %w", err)
	}
	id := created.ID
	e.log.Info("container created", "id", short(id), "image", opts.Image)

	if !opts.Keep {
		// Cleanup must run even if ctx was cancelled (e.g. Ctrl-C), so detach
		// the cancellation for the remove call.
		defer e.remove(context.WithoutCancel(ctx), id)
	}

	// Register the wait BEFORE starting so a fast exit is never missed.
	statusCh, errCh := e.cli.ContainerWait(ctx, id, container.WaitConditionNotRunning)

	if err := e.cli.ContainerStart(ctx, id, container.StartOptions{}); err != nil {
		return -1, fmt.Errorf("run: start container: %w", err)
	}
	e.log.Info("container started", "id", short(id))

	logs, err := e.cli.ContainerLogs(ctx, id, container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})
	if err != nil {
		return -1, fmt.Errorf("run: attach logs: %w", err)
	}
	defer func() { _ = logs.Close() }()

	stdout := opts.Stdout
	stderr := opts.Stderr
	if stdout == nil {
		stdout = io.Discard
	}
	if stderr == nil {
		stderr = io.Discard
	}

	// stdcopy de-multiplexes the Docker log stream into stdout/stderr. It
	// blocks until the stream closes (when the container stops).
	if _, copyErr := stdcopy.StdCopy(stdout, stderr, logs); copyErr != nil && !errors.Is(copyErr, io.EOF) {
		return -1, fmt.Errorf("run: stream logs: %w", copyErr)
	}

	select {
	case werr := <-errCh:
		if werr != nil {
			return -1, fmt.Errorf("run: wait container: %w", werr)
		}
		return -1, fmt.Errorf("run: wait returned no status")
	case st := <-statusCh:
		if st.Error != nil {
			return int(st.StatusCode), fmt.Errorf("run: container error: %s", st.Error.Message)
		}
		e.log.Info("container exited", "id", short(id), "code", st.StatusCode)
		return int(st.StatusCode), nil
	case <-ctx.Done():
		return -1, ctx.Err()
	}
}

// remove force-removes a container, logging (not returning) any error since it
// runs in a deferred cleanup path.
func (e *Engine) remove(ctx context.Context, id string) {
	if err := e.cli.ContainerRemove(ctx, id, container.RemoveOptions{Force: true}); err != nil {
		e.log.Warn("failed to remove container", "id", short(id), "err", err)
		return
	}
	e.log.Info("container removed", "id", short(id))
}

// envSlice converts an env map to a sorted KEY=value slice for deterministic
// ordering.
func envSlice(env map[string]string) []string {
	out := make([]string, 0, len(env))
	for k, v := range env {
		out = append(out, k+"="+v)
	}
	sort.Strings(out)
	return out
}

// short truncates a container/image ID for log readability.
func short(id string) string {
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
