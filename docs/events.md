# Event queries

`event_login_list` reads login and other user events. `event_admin_list` reads
administrative audit events. Both require a `realm` name and permission to view
events in that realm. Review the [security guidance](../README.md#security)
before sending event data to a model or sharing a transcript.

## Event-store prerequisites

In the target realm's Events settings, enable saving user events and select
the event types to retain for `event_login_list`. Enable Admin Events for
`event_admin_list`. A working Keycloak event-store provider is required. A
logging listener alone does not make past logs queryable here. Retention and
expiration determine how far back queries can reach. Storage that you enable
now does not reconstruct earlier activity.

Admin event representations also depend on Keycloak's Include
Representation setting. Enable it only when payload details are needed.
Representations can contain sensitive configuration and personal data.
These two MCP tools only read events. Configure storage through Keycloak.

## Dates

Both tools accept `dateFrom` and `dateTo` as **JSON strings** in either format:

| Format | Example | Meaning |
| --- | --- | --- |
| `yyyy-MM-dd` | `"2026-09-05"` | UTC calendar date: start of day for `dateFrom`, end of day for `dateTo` |
| Unix epoch milliseconds | `"1788566400000"` | Exact instant: September 5, 2026 at 00:00:00 UTC |

Both bounds are inclusive. Omit a bound to leave that end unrestricted. Use
milliseconds for sub-day windows. Seconds are interpreted as milliseconds,
not converted. RFC 3339 timestamps such as `2026-09-05T12:00:00Z` are not
accepted. The MCP server forwards dates unchanged. Keycloak rejects malformed
dates. Set `dateFrom` no later than `dateTo`.

These rules follow the pinned Keycloak 26.7.3
[date parser](https://github.com/keycloak/keycloak/blob/26.7.3/services/src/main/java/org/keycloak/services/util/DateUtil.java)
and [event endpoints](https://github.com/keycloak/keycloak/blob/26.7.3/services/src/main/java/org/keycloak/services/resources/admin/RealmAdminResource.java).

## Filters and limits

All exposed filters are sent to Keycloak and applied **server-side**, including
type arrays. There is no client-side filtering of event matches. Locally, the
MCP server fetches bounded pages, limits the returned count, and redacts
sensitive fields. Different filters combine with AND. Values within a type
array are alternatives (OR). Omitted or empty filters impose no restriction.

| Tool | Filter | Meaning |
| --- | --- | --- |
| Both | `realm` | Required target realm name, not its internal UUID |
| Both | `dateFrom`, `dateTo` | Inclusive time bounds described above |
| Both | `max` | Result count: omitted, zero, or negative defaults to 100 |
| `event_login_list` | `clientId` | Client identifier stored on the event, such as `my-app`. This is not the client's internal UUID |
| `event_login_list` | `userId` | Internal user ID, not username |
| `event_login_list` | `ipAddress` | Source IP value, not a CIDR range |
| `event_login_list` | `types` | Case-sensitive event names, for example `["LOGIN", "LOGIN_ERROR"]`. The tool can query other user-event types too |
| `event_admin_list` | `authClient` | Administrator's authenticating client identifier stored in `authDetails.clientId`. This is not the client being changed |
| `event_admin_list` | `authUser` | Administrator's internal user ID, not the affected user |
| `event_admin_list` | `operationTypes` | Case-sensitive operations, for example `["CREATE", "UPDATE", "DELETE", "ACTION"]` |
| `event_admin_list` | `resourceTypes` | Case-sensitive resource names, for example `["USER", "CLIENT", "REALM"]` |
| `event_admin_list` | `resourcePath` | Stored relative resource path, for example `users/<user-id>`. `users/*` matches paths below users |

With Keycloak's default JPA store, `resourcePath` uses SQL LIKE matching with
`*` converted to `%`. `%` and `_` also act as wildcards. It is not a regular
expression. See the [admin event query implementation](https://github.com/keycloak/keycloak/blob/26.7.3/model/jpa/src/main/java/org/keycloak/events/jpa/JpaAdminEventQuery.java).

Results follow Keycloak's ordering (newest first with the default JPA store).
The tools expose no offset, cursor, or sort option. Internal pages contain at
most 100 events, with a 10,000-event fetch cap per call even if `max` is larger.
A full result set may be truncated. Narrow the dates or filters instead of
treating a single call as a complete audit export.

## Safe read-only examples

Start the server with scoped credentials already supplied securely in its
environment. Add these non-secret settings to the MCP client's environment:

```text
KEYCLOAK_URL=https://keycloak.example.com
KEYCLOAK_READ_ONLY=true
```

Restart the MCP connection after changing startup settings. Read-only mode
omits mutations, but event data and explicitly requested client secrets can
still be disclosed. Avoid `includeSecret: true` during routine inspection.

Ask: “In realm demo, list up to 10 failed login events for my-app on September
5, 2026 UTC. Summarize the counts without repeating personal details.”
The `event_login_list` arguments are:

```json
{
  "realm": "demo",
  "clientId": "my-app",
  "types": ["LOGIN_ERROR"],
  "dateFrom": "2026-09-05",
  "dateTo": "2026-09-05",
  "max": 10
}
```

For `event_admin_list`, inspect up to five user updates in a one-hour window:

```json
{
  "realm": "demo",
  "operationTypes": ["UPDATE"],
  "resourceTypes": ["USER"],
  "resourcePath": "users/*",
  "dateFrom": "1788566400000",
  "dateTo": "1788569999999",
  "max": 5
}
```

Asking for a summary does not prevent the underlying tool result from entering
the model context. Query only the data needed for the task.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Empty array | Correct realm, event saving and selected types, retention, UTC bounds, and actor versus target IDs. Generate a relevant event after enabling storage. Earlier activity is not backfilled. |
| Invalid date or type error | Use the exact formats above, millisecond strings, and uppercase Keycloak enum names supported by your version. |
| 401 or 403 | Authentication realm, chosen credential mode, and the account's permission to view events in the target realm. Read-only mode does not grant permissions. |
| Event-store error | Ask the Keycloak operator to check the event-store provider and server logs. API failures appear as MCP tool errors. An empty array and an API failure are different outcomes. |
| Missing representation or `[REDACTED]` | Check Include Representation for future events. Redaction is intentional. Do not disable it to share credentials. |
| HTTPS validation or certificate error | Remote HTTP is rejected. Use HTTPS and, for a private CA, `KEYCLOAK_CA_CERT_FILE`. Only localhost and literal loopback IPs allow HTTP. |
| Startup authentication error | Supply one complete username/password pair or one complete service-account client ID/secret pair, not both. |
| Timeout or too many results | Narrow the query and use a small positive `max`. Check connectivity. `KEYCLOAK_TIMEOUT` accepts positive durations such as `30s`. |

For connection setup, see [MCP client troubleshooting](client-setup.md#verify).
Review and redact logs before sharing. Keep diagnostics off MCP stdout.
