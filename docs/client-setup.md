# MCP Client Setup

This server uses MCP stdio transport. Each client starts the binary as a local
process and passes the Keycloak settings in its environment.

## Prerequisites

Build the binary at an absolute path:

```sh
go build -o ./mcp-keycloak ./cmd/mcp-keycloak
```

Create a dedicated Keycloak administrator account for the server. It needs the
permissions required by the tools you intend to use. Do not use a production
super-admin account unless that level of access is intentional.

Use the following values in the examples below:

```text
KEYCLOAK_URL=https://keycloak.example.com
KEYCLOAK_ADMIN_USERNAME=mcp-admin
KEYCLOAK_ADMIN_PASSWORD=replace-me
KEYCLOAK_ADMIN_REALM=master
```

The URL is the Keycloak base URL, not a realm or token endpoint. Keep the
password out of committed configuration files. The examples use a placeholder
only. Replace it in your local client configuration. Use an environment
variable reference where the client supports one.

## Claude Desktop

Open Claude Desktop's **Settings → Developer → Edit Config**, or edit the
configuration file directly:

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%/Claude/claude_desktop_config.json`

Add the `keycloak` entry under the existing `mcpServers` object:

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

Use a platform-appropriate absolute path. Restart Claude Desktop after saving
the file.

## Claude Code

Register the local stdio server with the CLI. The `--` separates Claude Code
options from the command that starts the server:

```sh
claude mcp add --transport stdio \
  --env KEYCLOAK_URL=https://keycloak.example.com \
  --env KEYCLOAK_ADMIN_USERNAME=mcp-admin \
  --env KEYCLOAK_ADMIN_PASSWORD=replace-me \
  --env KEYCLOAK_ADMIN_REALM=master \
  keycloak -- "$PWD/mcp-keycloak"
```

Check the registration with:

```sh
claude mcp get keycloak
```

The command stores the environment values in Claude Code configuration. Avoid
using this form on shared shell history or shared configuration when the
password is sensitive. Claude Code also supports the same stdio JSON shape in
`.mcp.json` or `~/.claude.json`. See its [MCP documentation](https://code.claude.com/docs/en/mcp).

## Cursor

Create `.cursor/mcp.json` for a project-specific server or
`~/.cursor/mcp.json` for a global server. Keep a password-bearing project file
out of version control.

```json
{
  "mcpServers": {
    "keycloak": {
      "type": "stdio",
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

Cursor also exposes MCP setup from **Customize → MCP**. Its [MCP
documentation](https://cursor.com/docs/mcp) describes project and global
configuration locations.

## OpenCode

Add the server to `opencode.json` using a local command array. OpenCode
supports `{env:NAME}` references, which avoids putting the password directly in
the config. Export the four variables in the shell that starts OpenCode first.

```sh
export KEYCLOAK_URL=https://keycloak.example.com
export KEYCLOAK_ADMIN_USERNAME=mcp-admin
export KEYCLOAK_ADMIN_PASSWORD=replace-me
export KEYCLOAK_ADMIN_REALM=master
```

Then add this entry under `mcp`:

```json
{
  "$schema": "https://opencode.ai/config.json",
  "mcp": {
    "keycloak": {
      "type": "local",
      "command": ["/absolute/path/to/mcp-keycloak"],
      "environment": {
        "KEYCLOAK_URL": "{env:KEYCLOAK_URL}",
        "KEYCLOAK_ADMIN_USERNAME": "{env:KEYCLOAK_ADMIN_USERNAME}",
        "KEYCLOAK_ADMIN_PASSWORD": "{env:KEYCLOAK_ADMIN_PASSWORD}",
        "KEYCLOAK_ADMIN_REALM": "{env:KEYCLOAK_ADMIN_REALM}"
      }
    }
  }
}
```

OpenCode's [MCP server documentation](https://opencode.ai/docs/mcp-servers)
covers local server options and configuration precedence.

## Verify

After configuring a client, ask it to list the Keycloak realms. A successful
connection calls `realm_list` and returns at least the realm where the admin
account lives. Then try a read-only request such as getting that realm.

If the client does not show the server, follow the
[troubleshooting steps](getting-started.md#troubleshooting).
