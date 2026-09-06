# mcp-keycloak

A [Model Context Protocol](https://modelcontextprotocol.io/) server that lets
AI agents administer a [Keycloak](https://www.keycloak.org/) identity server:
realms, clients, users, groups and realm roles.

Built with the official [Go MCP SDK](https://github.com/modelcontextprotocol/go-sdk)
over [gocloak](https://github.com/Nerzal/gocloak) and the Keycloak Admin REST
API. Speaks MCP over stdio. MIT licensed.

Before connecting an account, review [Security](#security) for credential,
HTTPS, and transcript handling requirements.

## Tools

| Group | Tools |
| --- | --- |
| Realms | `realm_list`, `realm_get`, `realm_create`, `realm_update`, `realm_delete` |
| Clients | `client_list`, `client_get`, `client_create`, `client_update`, `client_secret_get`, `client_delete` |
| Client scopes | `client_scope_list`, `client_scope_get`, `client_scope_create`, `client_scope_delete`, `client_scope_assign`, `client_scope_unassign` |
| Audit events | `event_admin_list`, `event_login_list` |
| Identity providers | `identity_provider_list`, `identity_provider_get`, `identity_provider_create`, `identity_provider_update`, `identity_provider_delete` |
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
- `client_secret_get` omits the secret by default. Set `includeSecret: true`
  only when explicitly needed; the value is returned in structured output and
  may be retained by the MCP client's transcript or model context.
- List tools default to 100 results; use a smaller `max` to limit disclosure.
- `event_admin_list` requires Admin Events enabled in the realm; `event_login_list`
  requires user events enabled. Event type and admin operation/resource filters
  are supported.
- See [event filters, date formats, and troubleshooting](docs/events.md) for
  audit queries.
- Identity-provider tools configure realm login brokering, not MCP
  authentication. OIDC provider client secrets are redacted from every tool
  response; provider changes can affect realm-wide login.
- Failures surface as MCP tool errors with the Keycloak API status and
  message, so the model can see and correct them.

## Security

Use HTTPS for remote Keycloak connections. HTTP is accepted only for
`localhost` (case-insensitive) or literal loopback IPs, such as
`http://127.0.0.1:8080` and `http://[::1]:8080`. Private LAN addresses,
container hostnames, and DNS aliases that resolve to loopback do not qualify.
URLs must not contain embedded credentials, query strings, or fragments.
For a private CA, set `KEYCLOAK_CA_CERT_FILE` to a PEM bundle; TLS certificate
verification remains enabled.

- **Administrator credentials:** use a dedicated, least-privileged account or
  service account. Configure exactly one authentication mode. Keep passwords
  and `KEYCLOAK_ADMIN_CLIENT_SECRET` in the process environment or a protected,
  gitignored `.env`, never in prompts, commits, or shared client configuration.
  The server acts with the configured account's Keycloak permissions.
- **Client secrets:** `client_secret_get` omits the value unless
  `includeSecret: true`. Opting in places the secret in structured tool output,
  which the MCP client and model may retain. Do not request it for routine
  inspection or copy it into tickets or chat.
- **OIDC provider secrets:** provider tools redact sensitive configuration keys
  in responses, including `clientSecret`. Secrets supplied to create/update
  tools still enter tool arguments and may be retained in transcripts.
- **Event data:** audit representations and login details can contain user
  identifiers, IP addresses, email addresses, resource paths, and configuration
  data. Sensitive keys containing `secret`, `password`, `token`, or `credential`
  are redacted case-insensitively. Admin JSON representations are sanitized
  recursively; invalid JSON is replaced with `[REDACTED]`. This is key-based
  redaction, not comprehensive anonymization: personal data, free text, URLs,
  and values under other keys may remain sensitive.
- **Client/model transcripts:** both tool inputs and outputs may be logged,
  shared, or retained by the client or model provider. Check those retention
  settings, restrict access, and review/redact diagnostics before sharing.
  Rotate any credential accidentally exposed.

Set `KEYCLOAK_READ_ONLY=true` to omit mutating tools at startup. This does not
reduce the Keycloak account's permissions or prevent data disclosure:
`client_secret_get` remains available, including its explicit secret opt-in.
Combine read-only mode with scoped Keycloak permissions and narrow queries.

See [event filters and safe read-only examples](docs/events.md).

## Configuration

| Variable | Required | Description |
| --- | --- | --- |
| `KEYCLOAK_URL` | yes | Base URL of the Keycloak server |
| `KEYCLOAK_ADMIN_USERNAME` | conditional | Administrator account username |
| `KEYCLOAK_ADMIN_PASSWORD` | conditional | Administrator account password |
| `KEYCLOAK_ADMIN_CLIENT_ID` | conditional | Confidential service-account client ID |
| `KEYCLOAK_ADMIN_CLIENT_SECRET` | conditional | Confidential service-account client secret |
| `KEYCLOAK_ADMIN_REALM` | no | Realm holding the admin account (default `master`) |
| `KEYCLOAK_TIMEOUT` | no | Per-request HTTP timeout (default `30s`) |
| `KEYCLOAK_CA_CERT_FILE` | no | PEM CA bundle for private Keycloak TLS certificates |
| `KEYCLOAK_READ_ONLY` | no | Advertise only non-mutating tools (default `false`) |

`KEYCLOAK_URL` must use HTTPS for remote Keycloak servers. HTTP is accepted
only for localhost or loopback development environments.

Configure exactly one authentication mode: administrator username/password or
a confidential client ID/secret with a service account. Service accounts are
preferred for automation because their realm-management roles can be scoped
explicitly; never grant more roles than the tools require.

`KEYCLOAK_TIMEOUT` uses Go duration syntax such as `30s` or `2m`. The optional
CA bundle extends the system trust store; certificate verification is never
disabled. Set `KEYCLOAK_READ_ONLY=true` to omit all mutating tools from MCP
discovery; read-only mode is enforced before tools are advertised.

Variables can live in the process environment. When the server is started
from a working directory containing `.env`, it loads that file as a local
development convenience (see `.env.example`); existing environment variables
always win. MCP clients should pass the variables explicitly in their server
configuration.

The server uses the permissions granted to its configured account. Start with
a dedicated account and, where possible, a non-production realm.

## Install

For the v0.2.0 release, download the archive for your platform from
[GitHub Releases](https://github.com/erikhoward/mcp-keycloak/releases), extract
`mcp-keycloak`, and verify it against `checksums.txt`.

With Go 1.25 or newer, install the tagged command directly:

```sh
go install github.com/erikhoward/mcp-keycloak/cmd/mcp-keycloak@v0.2.0
```

The release workflow builds these targets without GoReleaser:

- Linux amd64 and arm64
- macOS amd64 and arm64
- Windows amd64

Maintainers publish a release by pushing an annotated `vX.Y.Z` tag. The
tag-triggered workflow injects that version into the MCP server metadata,
builds the archives, and publishes checksums.

## Quickstart

Build the binary in the repository:

```sh
go build -o ./mcp-keycloak ./cmd/mcp-keycloak
```

Create a dedicated Keycloak admin account, then configure the binary with
`KEYCLOAK_URL`, `KEYCLOAK_ADMIN_USERNAME`, `KEYCLOAK_ADMIN_PASSWORD`, and
optionally `KEYCLOAK_ADMIN_REALM`. The account needs the permissions required
by the tools you intend to use. Keep the password out of committed files.

Choose a client-specific setup:

- [Claude Desktop, Claude Code, Cursor, and OpenCode setup](docs/client-setup.md)

After connecting, ask the client to list the Keycloak realms. The first
read-only request should call `realm_list`. If it fails, verify the absolute
binary path, the Keycloak base URL, and the administrator credentials.

Use the absolute path to `mcp-keycloak` in the client configuration. The
underlying stdio configuration has this shape:

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

Install the same pinned vulnerability scanner used by CI, and ensure
`$(go env GOPATH)/bin` (or your custom `GOBIN`) is on `PATH`:

```sh
go install golang.org/x/vuln/cmd/govulncheck@v1.7.0
```

```sh
gofmt -l .                                        # format check
golangci-lint run                                 # lint (config: .golangci.yml)
go vet ./...
go test ./...                                     # unit tests (no Docker needed)
go test -tags integration ./...                   # integration tests (needs Docker)
govulncheck ./...                                 # known vulnerabilities (needs network)
go test ./internal/mcpserver/ -run TestRealmCreate   # single test
```

CI runs the vulnerability scan in the existing `test` job on pull requests
and pushes to `main`. Findings and scanner errors fail the check rather than
being reported as advisory. Require the `test` check in branch protection to
block merging failing pull requests.

Integration tests start a disposable Keycloak 26.7 container via
testcontainers (image pinned in `internal/mcpserver/integration_test.go`)
and drive the full tool surface over an in-memory MCP transport.

golangci-lint must be built with a Go toolchain at least as new as the one
building the project; release binaries can lag. If `golangci-lint run` fails
with a Go-version complaint, install from source:

```sh
go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.8.0
```

GitHub Actions are pinned to immutable commits. Maintainers review updates
monthly and promptly for security advisories using the
[action update process](docs/action-updates.md).

## Layout

- `cmd/mcp-keycloak` — stdio entrypoint
- `internal/config` — environment configuration
- `internal/keycloak` — gocloak wrapper with admin-token caching
- `internal/mcpserver` — MCP server, tool definitions and tests
