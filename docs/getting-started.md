# Getting started

This guide connects the server to a Keycloak realm and verifies the
connection. Install the binary first. See [Installation](install.md).

Before you connect a real account, read the
[Security section](../README.md#security).

## Create an administrator account

1. Create a dedicated Keycloak administrator account for the server. The
   account needs the permissions required by the tools you intend to use.
2. Do not use a production super-admin account.
3. For automation, prefer a service account. Its realm-management roles can
   be scoped explicitly. Never grant more roles than the tools require.

## Set the environment variables

Minimum for a first run:

- `KEYCLOAK_URL`: the Keycloak base URL. Use HTTPS, except for localhost or
  loopback addresses.
- `KEYCLOAK_ADMIN_USERNAME` and `KEYCLOAK_ADMIN_PASSWORD`: the administrator
  account.

Optional:

- `KEYCLOAK_ADMIN_REALM`: the realm of the admin account. The default is
  `master`.

For all variables, see the [Configuration section](../README.md#configuration).

Keep these rules:

- The URL is the Keycloak base URL. It is not a realm or token endpoint.
- Keep passwords out of committed files.
- When the server starts from a directory that contains `.env`, it loads that
  file. Existing environment variables always win. See `.env.example`.

## Connect an MCP client

Use the absolute path to the binary in the client configuration. The client
starts the binary as a local process and passes the variables in its
environment. The stdio configuration has this shape:

```json
{
  "mcpServers": {
    "keycloak": {
      "command": "/absolute/path/to/mcp-keycloak",
      "env": {
        "KEYCLOAK_URL": "https://keycloak.example.com",
        "KEYCLOAK_ADMIN_USERNAME": "mcp-admin",
        "KEYCLOAK_ADMIN_PASSWORD": "replace-me",
        "KEYCLOAK_ADMIN_REALM": "master"
      }
    }
  }
}
```

For client-specific configuration, see [Client setup](client-setup.md).

## Verify the connection

1. Ask the client to list the Keycloak realms. The first read-only call is
   `realm_list`.
2. The result must include at least the realm where the admin account lives.
3. Ask for one realm in detail as a second read-only test. Use `realm_get`.

## Troubleshooting

If `realm_list` fails, work through this list in order:

1. Run the binary directly with its absolute path. It must start without a
   configuration error.
2. Check that `KEYCLOAK_URL` is the server base URL. It is not a realm or
   token endpoint.
3. Check that the credentials authenticate in `KEYCLOAK_ADMIN_REALM`.
4. Check the client logs for a malformed JSON configuration or an executable
   path error.
5. Restart or reload the client's MCP connections. Do not reuse an old server
   process.

Diagnostics belong on stderr. Do not pipe diagnostic output into stdout.
Stdout carries the MCP protocol messages.

## Next steps

- [Tools](../README.md#tools): the full tool surface.
- [Security](../README.md#security): credential, HTTPS, and transcript
  handling.
- [Event queries](events.md): audit queries and safe read-only examples.
