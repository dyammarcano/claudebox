// Package credentials locates the host's Claude credentials and settings and
// copies a minimal, scrubbable subset into an ephemeral seed directory that is
// bind-mounted read-only into the sandbox container. The host ~/.claude is
// never modified or mounted directly.
package credentials
