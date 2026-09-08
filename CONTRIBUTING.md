# Contributing

Thanks for helping improve mcp-keycloak. Changes should keep the server safe
for administrators and predictable for MCP clients.

## Before opening an issue or pull request

- Search existing issues and pull requests first.
- For security vulnerabilities, follow [SECURITY.md](SECURITY.md) instead of
  opening a public issue.
- Do not include credentials, tokens, client secrets, or private Keycloak data
  in issues, pull requests, tests, logs, or examples.

## Development

1. Fork the repository and create a focused branch from `main`.
2. Make the smallest change that addresses the issue.
3. Add or update tests for behavior changes. Integration tests require Docker.
4. Update user-facing documentation when configuration or tool behavior
   changes.
5. Run the relevant checks before opening a pull request:

```sh
gofmt -l .
golangci-lint run
go vet ./...
go test ./...
go test -tags integration ./...
```

The integration test command requires Docker. See [AGENTS.md](AGENTS.md) for
the complete verification sequence, including vulnerability scanning.

## Pull requests

Describe the problem, the approach, and any compatibility or security impact.
Keep unrelated refactoring out of the pull request. Include test results and
call out checks that could not be run.

GitHub Actions are pinned to full commit SHAs. Follow
[`docs/action-updates.md`](docs/action-updates.md) when changing workflow
actions.

By participating, you agree to follow the project's [Code of
Conduct](CODE_OF_CONDUCT.md).
