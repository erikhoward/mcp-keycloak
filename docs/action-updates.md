# Updating GitHub Actions

Every external `uses:` reference in `.github/workflows` must use a full
40-character commit SHA, followed by a comment with the exact release version,
for example `# v5.1.0`. Tags are useful labels but must not be executable refs.
Keep matching action pins consistent across CI and release workflows.

## Review cadence

Maintainers review upstream action releases monthly and promptly when a
security advisory affects an action. Updates are deliberate pull requests;
do not automatically merge pin changes. A version comment alone is not proof
that the commit belongs to the intended upstream release.

## Update procedure

1. Read the release notes and compare the current and proposed commits in the
   action's upstream repository. Check runtime and runner requirements,
   permissions, input/output changes, and artifact compatibility. Treat major
   upgrades separately from routine patch updates.
2. Resolve the exact release tag through GitHub's API. For example:

   ```sh
   gh api repos/actions/checkout/git/ref/tags/v5.1.0
   ```

   If `object.type` is `commit`, use `object.sha`. If it is `tag`, retrieve
   `repos/actions/checkout/git/tags/<object.sha>` and follow the referenced
   object until it is a commit. Pin the commit, not an annotated tag object's
   SHA. Verify that the resolved commit is in the intended upstream repository.
3. Replace every occurrence of that action's SHA and version comment in both
   workflows. Preserve existing inputs and permissions unless the reviewed
   release explicitly requires a change.
4. Run `actionlint .github/workflows/ci.yml .github/workflows/release.yml`,
   `git diff --check`, and the repository's [verification commands](../AGENTS.md#verify).
   Confirm every external action still has a 40-character SHA and matching
   version comment, and that workflow triggers, job dependencies, permissions,
   and the five release platform targets are preserved.
5. Open a PR describing the old/new versions and SHAs, upstream release and
   compare links, compatibility considerations, and verification results.
   Require review and passing CI before merging.

## Release verification

CI runs on pull requests and pushes to `main`. The release workflow runs only
on `v*.*.*` tag pushes. Before merging a pin update, validate release workflow
syntax and cross-build the five matrix targets (Linux amd64/arm64, macOS
amd64/arm64, Windows amd64). Check the archive commands and upload/download
artifact inputs together, including `merge-multiple` and unique artifact names.

Local linting and builds cannot exercise GitHub-hosted artifact transfer or
release publication. Record that limitation in the PR. At the next intended
release, verify all build jobs, artifact downloads, checksums, and published
archives. Do not create a production release tag merely to test an action pin.
