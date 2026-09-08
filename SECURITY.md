# Security Policy

## Supported versions

Only the latest release receives security fixes. Older releases should be
updated before reporting a problem that is already fixed in a newer release.

## Reporting a vulnerability

Please report suspected vulnerabilities privately through [GitHub private
vulnerability reporting](https://github.com/erikhoward/mcp-keycloak/security/advisories/new).
Do not open a public issue or include credentials, tokens, client secrets, or
private Keycloak data in a report.

Include enough detail to reproduce the problem safely, including:

- The affected release or commit.
- The relevant configuration and operating environment, with secrets removed.
- Reproduction steps or a proof of concept that does not expose real data.
- The impact you believe the issue has.

The maintainer will acknowledge a report when practical, investigate it, and
coordinate a fix and disclosure timeline with the reporter. Please do not
publicly disclose the issue until a fix or mitigation is available.

If private vulnerability reporting is unavailable, contact the maintainer
privately through GitHub rather than using a public issue.

## Scope

Reports are especially useful for authentication or authorization bypasses,
credential or secret disclosure, TLS validation failures, unsafe handling of
MCP tool input, and vulnerabilities in the release or CI process.

Keycloak itself is a separate project. Vulnerabilities in Keycloak should be
reported to the [Keycloak security team](https://www.keycloak.org/security).
