# Changelog

All notable changes to this project are documented here.

## [0.3.0] - 2026-09-07

- Add user session inspection and logout tools.
- Add user and group membership, role-mapping, hierarchy, and update tools.
- Add realm-role updates and composites plus client-role lifecycle and user
  assignment tools.
- Add user counts, Keycloak server information, and brute-force status.
- Add offset pagination to bounded user, group, client, and realm-role lists.
- Add client service-account discovery, identity-provider mapper CRUD, and
  user broker-identity inspection and unlinking.
- Restructure installation, setup, event-query, and operational documentation.

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
