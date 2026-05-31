// Package engine wraps the Docker Go SDK to build the sandbox image and run a
// single throwaway Claude Code container, streaming its output back to the
// caller and force-removing it on completion.
package engine
