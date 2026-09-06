# Install

mcp-keycloak ships as one binary. Install it from a release archive, with
Go, or from source. Then continue with [Getting started](getting-started.md).

The release archive and the source build need no extra tools. The Go install
needs Go 1.25 or newer.

## Install from a release archive

1. Download the archive for your platform from
   [GitHub Releases](https://github.com/erikhoward/mcp-keycloak/releases).
   The current release is v0.2.0.
2. Extract the `mcp-keycloak` binary from the archive.
3. Verify the binary against the `checksums.txt` file from the same release.

Built targets:

- Linux amd64 and arm64
- macOS amd64 and arm64
- Windows amd64

## Install with Go

With Go 1.25 or newer, install the tagged command directly:

```sh
go install github.com/erikhoward/mcp-keycloak/cmd/mcp-keycloak@v0.2.0
```

## Build from source

```sh
git clone https://github.com/erikhoward/mcp-keycloak.git
cd mcp-keycloak
go build -o ./mcp-keycloak ./cmd/mcp-keycloak
```

## Verify

Run the binary without configuration:

```sh
./mcp-keycloak
```

The binary must print a configuration error and exit with status 1. The
error names the missing environment variables. This proves that the binary
runs on your platform.

## Next steps

- [Getting started](getting-started.md): connect the server to Keycloak and
  verify the connection.
- [Client setup](client-setup.md): configure Claude Desktop, Claude Code,
  Cursor, or OpenCode.
