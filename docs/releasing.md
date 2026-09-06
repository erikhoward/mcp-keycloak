# Releasing

This guide is for maintainers. It describes how to publish a release.

The release workflow lives in `.github/workflows/release.yml`. It runs on
push of a `v*.*.*` tag. It does not use GoReleaser.

## Publish a release

1. Update the version references in `docs/install.md` to the new version.
2. Push an annotated tag:

   ```sh
   git tag -a vX.Y.Z -m "vX.Y.Z"
   git push origin vX.Y.Z
   ```

3. Wait for the workflow. It verifies the commit, builds the archives, and
   publishes the release.

## What the workflow does

- Verifies the commit with `go vet ./...` and `go test ./...`.
- Builds one archive per target: Linux amd64 and arm64, macOS amd64 and
  arm64, Windows amd64.
- Injects the tag version into the MCP server metadata through linker flags.
- Packs the binary, `LICENSE`, and `README.md` into each archive. Windows
  gets a zip archive. The other targets get tar.gz archives.
- Writes `checksums.txt` with SHA-256 hashes of all archives.
- Creates the GitHub release with generated notes.
