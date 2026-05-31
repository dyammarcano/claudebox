# Contributors

## Maintainers

| Name | GitHub | Role |
|------|--------|------|
| dyammarcano | [@dyammarcano](https://github.com/dyammarcano) | Owner |

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'feat: add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Commit Convention

Use [Conventional Commits](https://www.conventionalcommits.org/):
`feat:`, `fix:`, `docs:`, `refactor:`, `test:`, `chore:`.

### Code Standards

- Run `task check` (build + vet + lint + test) before submitting.
- Keep `go build ./...`, `go vet ./...`, and `go test ./...` green.
- Use `errors.Is`/`errors.As` and `%w` wrapping (never `==` on errors).
- Table-driven tests; target 80%+ coverage.
- Never run a binary you "compiled then ran" — use `go run` / `go test`.
- Build commands:
  - Build: `go build ./...` (or `task build`)
  - Test: `go test ./...` (or `task test`)
  - Lint: `golangci-lint run --fix ./... --timeout=5m` (or `task lint`)
