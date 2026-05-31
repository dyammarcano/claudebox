// Package engine wraps the Docker Go SDK to build the sandbox image and run a
// single throwaway Claude Code container, streaming its output back to the
// caller and force-removing it on completion.
package engine

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/docker/docker/client"
)

// Engine performs Docker operations for claudebox.
type Engine struct {
	cli *client.Client
	log *slog.Logger
}

// New constructs an Engine using the ambient Docker environment
// (DOCKER_HOST, etc.) with API-version negotiation so it works across daemon
// versions. The caller must Close it.
func New(log *slog.Logger) (*Engine, error) {
	if log == nil {
		log = slog.Default()
	}
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("create docker client: %w", err)
	}
	return &Engine{cli: cli, log: log}, nil
}

// Ping verifies the Docker daemon is reachable, returning a friendly error if
// not (the most common first-run failure).
func (e *Engine) Ping(ctx context.Context) error {
	if _, err := e.cli.Ping(ctx); err != nil {
		return fmt.Errorf("docker daemon unreachable (is Docker running?): %w", err)
	}
	return nil
}

// Close releases the underlying Docker client.
func (e *Engine) Close() error {
	if e.cli == nil {
		return nil
	}
	return e.cli.Close()
}
