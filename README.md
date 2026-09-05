# mcp-keycloak

A [Model Context Protocol](https://modelcontextprotocol.io/) server that lets
AI agents administer a [Keycloak](https://www.keycloak.org/) identity server:
realms, clients, users, groups and realm roles.

Built with the official [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
over [gocloak](https://github.com/Nerzal/gocloak) and the Keycloak Admin REST
API. Speaks MCP over stdio. MIT licensed.

## Tools

| Group | Tools |
| --- | --- |
| Realms | `realm_list`, `realm_get`, `realm_create`, `realm_update`, `realm_delete` |
| Clients | `client_list`, `client_get`, `client_create`, `client_secret_get`, `client_delete` |
| Users | `user_list`, `user_get`, `user_create`, `user_update`, `user_set_password`, `user_delete`, `user_add_realm_role`, `user_remove_realm_role`, `user_add_to_group`, `user_remove_from_group` |
| Groups | `group_list`, `group_create`, `group_delete` |
| Realm roles | `realm_role_list`, `realm_role_create`, `realm_role_delete` |

Notes for agents calling the tools:

- Realms are addressed by name; clients by their `clientId` (resolved
  internally); users and groups by their internal UUID, which every create
  and list tool returns. Role assignment matches realm roles by name.
- `user_create` accepts an optional initial password, temporary by default so
  the user must change it at first login.
- Created clients are confidential (with an auto-generated secret, fetchable
  via `client_secret_get`) unless `public: true` is passed.
- List tools return at most 100 results unless a smaller `max` is given.
- Failures surface as MCP tool errors with the Keycloak API status and
  message, so the model can see and correct them.

## Configuration

| Variable | Required | Description |
| --- | --- | --- |
| `KEYCLOAK_URL` | yes | Base URL of the Keycloak server |
| `KEYCLOAK_ADMIN_USERNAME` | yes | Administrator account username |
| `KEYCLOAK_ADMIN_PASSWORD` | yes | Administrator account password |
| `KEYCLOAK_ADMIN_REALM` | no | Realm holding the admin account (default `master`) |

Variables can live in the process environment or a `.env` file next to the
binary (see `.env.example`); existing environment variables always win.

The server authenticates with full administrator rights. Point it at a
dedicated admin account and, where possible, a non-production realm.

## Running

```sh
go build -o mcp-keycloak ./cmd/mcp-keycloak
```

Register with any stdio MCP client, e.g.:

```json
{
  "mcpServers": {
    "keycloak": {
      "command": "/path/to/mcp-keycloak",
      "env": {
        "KEYCLOAK_URL": "https://keycloak.example.com",
        "KEYCLOAK_ADMIN_USERNAME": "admin",
        "KEYCLOAK_ADMIN_PASSWORD": "secret"
      }
    }
  }
}
```

## Development

```sh
gofmt -l .                                        # format check
golangci-lint run                                 # lint (config: .golangci.yml)
go vet ./...
go test ./...                                     # unit tests (no Docker needed)
go test -tags integration ./...                   # integration tests (needs Docker)
go test ./internal/mcpserver/ -run TestRealmCreate   # single test
```

Integration tests start a disposable Keycloak 26.7 container via
testcontainers (image pinned in `internal/mcpserver/integration_test.go`)
and drive the full tool surface over an in-memory MCP transport.

golangci-lint must be built with a Go toolchain at least as new as the one
building the project; release binaries can lag. If `golangci-lint run` fails
with a Go-version complaint, install from source:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
```

## Layout

- `cmd/mcp-keycloak` — stdio entrypoint
- `internal/config` — environment configuration
- `internal/keycloak` — gocloak wrapper with admin-token caching
- `internal/mcpserver` — MCP server, tool definitions and tests
