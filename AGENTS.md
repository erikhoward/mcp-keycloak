# AGENTS.md

## Project

Go MCP server exposing Keycloak administration over stdio. Single Go module, MIT licensed.

## Layout

- `cmd/mcp-keycloak` — stdio entrypoint
- `internal/config` — env config (`KEYCLOAK_URL`, `KEYCLOAK_ADMIN_USERNAME`, `KEYCLOAK_ADMIN_PASSWORD`, `KEYCLOAK_ADMIN_REALM`)
- `internal/keycloak` — gocloak wrapper with admin-token caching
- `internal/mcpserver` — MCP server, tool definitions/handlers, all tests
- `.github/workflows/ci.yml` — lint, unit, integration jobs
- `.github/workflows/release.yml` — tag-triggered cross-platform release archives

## Binding stack decisions (do not substitute alternatives)

- MCP: official SDK `github.com/modelcontextprotocol/go-sdk`
- Keycloak: `github.com/Nerzal/gocloak/v13` over hand-rolled REST
- Integration tests: `github.com/stillya/testcontainers-keycloak` (community Keycloak module for testcontainers-go); requires Docker
- Lint: `golangci-lint` (config in `.golangci.yml`)

## Adding a tool

Tools live in `internal/mcpserver/<domain>.go`: an input struct with `jsonschema` description tags, then `mcp.AddTool` with a handler that calls the admin. Handlers depend on the `AdminAPI` interface in `server.go`; new operations must be added to `AdminAPI` **and** implemented on `*keycloak.Admin` (`internal/keycloak/admin.go`) — the compile-time assertion in `server.go` keeps them in sync. Unit tests use the `fakeAdmin` in `server_test.go`; add integration coverage in `integration_test.go`.

## Verify

Run in order:

```
gofmt -l .
golangci-lint run
go vet ./...
go test ./...                     # unit; no Docker required
go test -tags integration ./...   # integration; needs Docker, pulls Keycloak 26.7.3
```

Single test: `go test ./internal/mcpserver/ -run TestRealmCreate`. Integration tests are guarded by a `//go:build integration` tag; without it they are excluded entirely (they do not silently skip when Docker is missing — they fail).

## Gotchas

- golangci-lint must be built with a Go toolchain at least as new as the project's; prebuilt binaries that lag fail with a version error or panic in `go/types`. Fix: `go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0`.
- The stdio transport must be `&mcp.StdioTransport{}`, not `&mcp.IOTransport{Reader: os.Stdin, Writer: os.Stdout}` — IOTransport closes its writer on disconnect, which closes `os.Stdout` and kills pending responses.
- For manual stdio testing, drive the binary with an SDK client (e.g. `mcp.CommandTransport`); hand-rolled JSON-RPC with a stale `protocolVersion` is rejected.
- `.env` is gitignored; Keycloak URL/credentials belong there or in the environment, never in code or commits. `.env.example` documents the variables.
- `go.work` is gitignored — keep this a single module; no nested `go.mod`.
- Keycloak API failures are returned as MCP tool errors by design (so the model can see and self-correct); do not convert them into protocol errors.
