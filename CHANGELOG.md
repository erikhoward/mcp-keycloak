# Changelog

All notable changes to this project are documented here.

## [0.1.0] - 2026-09-05

Initial release.

- Expose Keycloak realm, client, user, group, and realm-role administration
  through MCP stdio.
- Provide client creation, update, secret retrieval, user password management,
  role assignment, and group membership tools.
- Cache Keycloak administrator tokens and return Keycloak API failures as MCP
  tool errors.
- Include unit tests, Keycloak testcontainers integration tests, client setup
  documentation, and cross-platform release builds.
