// Package sandbox orchestrates the full claudebox lifecycle: detect the host
// Claude version, build the pinned image, prepare ephemeral credentials, run
// the throwaway container, and clean everything up.
package sandbox
