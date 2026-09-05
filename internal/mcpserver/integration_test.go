//go:build integration

package mcpserver

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/Nerzal/gocloak/v13"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	keycloaktc "github.com/stillya/testcontainers-keycloak"

	"github.com/erikhoward/mcp-keycloak/internal/keycloak"
)

// keycloakImage is pinned so integration results are reproducible; bump it
// deliberately when testing against newer Keycloak releases.
const keycloakImage = "quay.io/keycloak/keycloak:26.7.3"

var (
	testAdmin   *keycloak.Admin
	testBaseURL string
)

// TestMain starts a disposable Keycloak container for the whole integration
// run. It requires a working Docker daemon; without the integration build
// tag these tests are skipped entirely.
func TestMain(m *testing.M) {
	ctx := context.Background()
	kc, err := keycloaktc.Run(ctx, keycloakImage,
		keycloaktc.WithAdminUsername("admin"),
		keycloaktc.WithAdminPassword("admin"),
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-keycloak integration: starting Keycloak container: %v\n", err)
		os.Exit(1)
	}
	baseURL, err := kc.GetAuthServerURL(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "mcp-keycloak integration: getting Keycloak URL: %v\n", err)
		_ = kc.Terminate(ctx)
		os.Exit(1)
	}
	testBaseURL = baseURL
	testAdmin = keycloak.NewAdmin(baseURL, "admin", "admin", "master")

	code := m.Run()
	if err := kc.Terminate(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "mcp-keycloak integration: terminating container: %v\n", err)
	}
	os.Exit(code)
}

// TestKeycloakToolsIntegration exercises the tool surface end to end against
// a real Keycloak: the MCP protocol layer (via an in-memory transport) plus
// gocloak against the container.
func TestKeycloakToolsIntegration(t *testing.T) {
	cs := newTestClient(t, testAdmin)
	realm := fmt.Sprintf("it-%d", time.Now().UnixNano())

	res := callTool(t, cs, "realm_create", map[string]any{"realm": realm, "displayName": "Integration"})
	created := decodeResult[gocloak.RealmRepresentation](t, res)
	if deref(created.Realm) != realm {
		t.Fatalf("realm_create returned realm %q, want %q", deref(created.Realm), realm)
	}

	userID := createUser(t, cs, realm)
	clientID := createClient(t, cs, realm)
	scopeID := createClientScope(t, cs, realm)
	groupID := createGroup(t, cs, realm)
	_ = createRealmRole(t, cs, realm)

	t.Run("realm settings round trip", func(t *testing.T) {
		res := callTool(t, cs, "realm_get", map[string]any{"realm": realm})
		got := decodeResult[gocloak.RealmRepresentation](t, res)
		if deref(got.Realm) != realm {
			t.Errorf("got realm %q, want %q", deref(got.Realm), realm)
		}

		res = callTool(t, cs, "realm_update", map[string]any{
			"realm": realm, "displayName": "Integration v2",
		})
		got = decodeResult[gocloak.RealmRepresentation](t, res)
		if deref(got.DisplayName) != "Integration v2" {
			t.Errorf("displayName = %q, want %q", deref(got.DisplayName), "Integration v2")
		}
	})

	t.Run("client secret", func(t *testing.T) {
		res := callTool(t, cs, "client_secret_get", map[string]any{"realm": realm, "clientId": "web-app"})
		secret := decodeResult[gocloak.CredentialRepresentation](t, res)
		if deref(secret.Value) == "" {
			t.Error("expected a non-empty client secret")
		}
	})

	t.Run("client update", func(t *testing.T) {
		res := callTool(t, cs, "client_update", map[string]any{
			"realm": realm, "clientId": "web-app",
			"name":                   "Web App v2",
			"redirectURIs":           []string{"https://app.example.com/callback"},
			"serviceAccountsEnabled": true,
		})
		client := decodeResult[gocloak.Client](t, res)
		if got := deref(client.Name); got != "Web App v2" {
			t.Errorf("name = %q, want %q", got, "Web App v2")
		}
		if client.RedirectURIs == nil || len(*client.RedirectURIs) != 1 || (*client.RedirectURIs)[0] != "https://app.example.com/callback" {
			t.Errorf("redirectURIs = %v, want the updated URI", client.RedirectURIs)
		}
		if client.ServiceAccountsEnabled == nil || !*client.ServiceAccountsEnabled {
			t.Error("serviceAccountsEnabled should be true after client_update")
		}
		// Fields not sent must keep their original values.
		if client.DirectAccessGrantsEnabled == nil || *client.DirectAccessGrantsEnabled {
			t.Error("directAccessGrantsEnabled should be unchanged (false)")
		}
		if client.PublicClient == nil || *client.PublicClient {
			t.Error("publicClient should be unchanged (false)")
		}
	})

	t.Run("client scope lifecycle", func(t *testing.T) {
		res := callTool(t, cs, "client_scope_list", map[string]any{"realm": realm})
		scopes := decodeResult[[]*gocloak.ClientScope](t, res)
		if !hasClientScopeID(scopes, scopeID) {
			t.Fatal("client_scope_list does not include the created orders scope")
		}

		res = callTool(t, cs, "client_scope_get", map[string]any{"realm": realm, "name": "orders"})
		got := decodeResult[gocloak.ClientScope](t, res)
		if deref(got.ID) != scopeID || deref(got.Protocol) != "openid-connect" {
			t.Errorf("client_scope_get returned ID=%q protocol=%q, want %q/openid-connect", deref(got.ID), deref(got.Protocol), scopeID)
		}

		callTool(t, cs, "client_scope_assign", map[string]any{
			"realm": realm, "clientId": "web-app", "scopeName": "orders",
		})
		gc := gocloak.NewClient(testBaseURL)
		tok := verifyToken(t)
		defaults, err := gc.GetClientsDefaultScopes(t.Context(), tok, realm, clientID)
		if err != nil {
			t.Fatalf("checking default client scopes: %v", err)
		}
		if !hasClientScopeID(defaults, scopeID) {
			t.Error("orders scope was not added as a default client scope")
		}

		callTool(t, cs, "client_scope_unassign", map[string]any{
			"realm": realm, "clientId": "web-app", "scopeName": "orders",
		})
		defaults, err = gc.GetClientsDefaultScopes(t.Context(), tok, realm, clientID)
		if err != nil {
			t.Fatalf("checking removed default client scope: %v", err)
		}
		if hasClientScopeID(defaults, scopeID) {
			t.Error("orders scope is still a default client scope after unassign")
		}

		callTool(t, cs, "client_scope_assign", map[string]any{
			"realm": realm, "clientId": "web-app", "scopeName": "orders", "optional": true,
		})
		optional, err := gc.GetClientsOptionalScopes(t.Context(), tok, realm, clientID)
		if err != nil {
			t.Fatalf("checking optional client scopes: %v", err)
		}
		if !hasClientScopeID(optional, scopeID) {
			t.Error("orders scope was not added as an optional client scope")
		}

		callTool(t, cs, "client_scope_unassign", map[string]any{
			"realm": realm, "clientId": "web-app", "scopeName": "orders", "optional": true,
		})
		callTool(t, cs, "client_scope_delete", map[string]any{"realm": realm, "name": "orders"})
	})

	t.Run("user list and get", func(t *testing.T) {
		res := callTool(t, cs, "user_list", map[string]any{"realm": realm, "username": "alice"})
		users := decodeResult[[]gocloak.User](t, res)
		if len(users) != 1 || deref(users[0].ID) != userID {
			t.Fatalf("user_list returned %v, want exactly alice (%s)", users, userID)
		}

		res = callTool(t, cs, "user_get", map[string]any{"realm": realm, "userId": userID})
		user := decodeResult[gocloak.User](t, res)
		if deref(user.Email) != "alice@example.com" {
			t.Errorf("email = %q, want %q", deref(user.Email), "alice@example.com")
		}
	})

	t.Run("user update", func(t *testing.T) {
		res := callTool(t, cs, "user_update", map[string]any{
			"realm": realm, "userId": userID, "enabled": false, "firstName": "Alicia",
		})
		user := decodeResult[gocloak.User](t, res)
		if user.Enabled == nil || *user.Enabled {
			t.Error("user should be disabled after user_update")
		}
		if deref(user.FirstName) != "Alicia" {
			t.Errorf("firstName = %q, want %q", deref(user.FirstName), "Alicia")
		}
	})

	t.Run("user role and group assignment", func(t *testing.T) {
		callTool(t, cs, "user_add_realm_role", map[string]any{
			"realm": realm, "userId": userID, "roles": []string{"auditor"},
		})
		callTool(t, cs, "user_add_to_group", map[string]any{
			"realm": realm, "userId": userID, "groupId": groupID,
		})

		// Verify effective grants straight through Keycloak, bypassing
		// the tool layer.
		gc := gocloak.NewClient(testBaseURL)
		tok := verifyToken(t)
		roles, err := gc.GetRealmRolesByUserID(t.Context(), tok, realm, userID)
		if err != nil {
			t.Fatalf("verifying user roles: %v", err)
		}
		if !hasRoleName(roles, "auditor") {
			t.Error("user does not have the auditor role after user_add_realm_role")
		}
		groups, err := gc.GetUserGroups(t.Context(), tok, realm, userID, gocloak.GetGroupsParams{})
		if err != nil {
			t.Fatalf("verifying user groups: %v", err)
		}
		if !hasGroupID(groups, groupID) {
			t.Error("user is not a member of the engineers group after user_add_to_group")
		}

		callTool(t, cs, "user_remove_realm_role", map[string]any{
			"realm": realm, "userId": userID, "roles": []string{"auditor"},
		})
		callTool(t, cs, "user_remove_from_group", map[string]any{
			"realm": realm, "userId": userID, "groupId": groupID,
		})

		roles, err = gc.GetRealmRolesByUserID(t.Context(), tok, realm, userID)
		if err != nil {
			t.Fatalf("re-verifying user roles: %v", err)
		}
		if hasRoleName(roles, "auditor") {
			t.Error("user still has the auditor role after user_remove_realm_role")
		}
		groups, err = gc.GetUserGroups(t.Context(), tok, realm, userID, gocloak.GetGroupsParams{})
		if err != nil {
			t.Fatalf("re-verifying user groups: %v", err)
		}
		if hasGroupID(groups, groupID) {
			t.Error("user is still a member of the engineers group after user_remove_from_group")
		}
	})

	t.Run("group list", func(t *testing.T) {
		res := callTool(t, cs, "group_list", map[string]any{"realm": realm, "search": "engineers"})
		groups := decodeResult[[]gocloak.Group](t, res)
		if len(groups) != 1 || deref(groups[0].ID) != groupID {
			t.Fatalf("group_list returned %v, want the created engineers group", groups)
		}
	})

	t.Run("realm role list", func(t *testing.T) {
		res := callTool(t, cs, "realm_role_list", map[string]any{"realm": realm})
		roles := decodeResult[[]gocloak.Role](t, res)
		found := false
		for _, r := range roles {
			if deref(r.Name) == "auditor" {
				found = true
			}
		}
		if !found {
			t.Error("realm_role_list does not include the created auditor role")
		}
	})

	t.Run("missing realm yields tool error", func(t *testing.T) {
		res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
			Name:      "realm_get",
			Arguments: map[string]any{"realm": "no-such-realm"},
		})
		if err != nil {
			t.Fatalf("CallTool protocol error: %v", err)
		}
		if !res.IsError {
			t.Fatal("expected a tool error for a missing realm")
		}
	})

	t.Run("cleanup", func(t *testing.T) {
		callTool(t, cs, "user_delete", map[string]any{"realm": realm, "userId": userID})
		callTool(t, cs, "client_delete", map[string]any{"realm": realm, "clientId": "web-app"})
		callTool(t, cs, "group_delete", map[string]any{"realm": realm, "groupId": groupID})
		callTool(t, cs, "realm_role_delete", map[string]any{"realm": realm, "name": "auditor"})
		callTool(t, cs, "realm_delete", map[string]any{"realm": realm})

		res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
			Name:      "realm_get",
			Arguments: map[string]any{"realm": realm},
		})
		if err != nil {
			t.Fatalf("CallTool protocol error: %v", err)
		}
		if !res.IsError {
			t.Fatal("realm_get should fail after realm_delete")
		}
		if text := resultText(t, res); !strings.Contains(text, "404") {
			t.Errorf("error text %q does not mention the 404 status", text)
		}
	})
}

func createUser(t *testing.T, cs *mcp.ClientSession, realm string) string {
	t.Helper()
	res := callTool(t, cs, "user_create", map[string]any{
		"realm":           realm,
		"username":        "alice",
		"email":           "alice@example.com",
		"firstName":       "Alice",
		"lastName":        "Doe",
		"initialPassword": "correct horse battery staple",
	})
	user := decodeResult[gocloak.User](t, res)
	id := deref(user.ID)
	if id == "" {
		t.Fatal("user_create returned no ID")
	}
	return id
}

func createClient(t *testing.T, cs *mcp.ClientSession, realm string) string {
	t.Helper()
	res := callTool(t, cs, "client_create", map[string]any{
		"realm":        realm,
		"clientId":     "web-app",
		"name":         "Web App",
		"redirectURIs": []string{"http://localhost:8080/callback"},
	})
	client := decodeResult[gocloak.Client](t, res)
	id := deref(client.ID)
	if id == "" {
		t.Fatal("client_create returned no ID")
	}
	if client.PublicClient == nil || *client.PublicClient {
		t.Error("client should be confidential (public=false) by default")
	}
	return id
}

func createClientScope(t *testing.T, cs *mcp.ClientSession, realm string) string {
	t.Helper()
	res := callTool(t, cs, "client_scope_create", map[string]any{
		"realm": realm, "name": "orders", "description": "Order claims",
	})
	scope := decodeResult[gocloak.ClientScope](t, res)
	id := deref(scope.ID)
	if id == "" {
		t.Fatal("client_scope_create returned no ID")
	}
	return id
}

func createGroup(t *testing.T, cs *mcp.ClientSession, realm string) string {
	t.Helper()
	res := callTool(t, cs, "group_create", map[string]any{"realm": realm, "name": "engineers"})
	group := decodeResult[gocloak.Group](t, res)
	id := deref(group.ID)
	if id == "" {
		t.Fatal("group_create returned no ID")
	}
	return id
}

func createRealmRole(t *testing.T, cs *mcp.ClientSession, realm string) string {
	t.Helper()
	res := callTool(t, cs, "realm_role_create", map[string]any{
		"realm": realm, "name": "auditor", "description": "Read-only auditor access",
	})
	role := decodeResult[gocloak.Role](t, res)
	return deref(role.Name)
}

func hasClientScopeID(scopes []*gocloak.ClientScope, id string) bool {
	for _, scope := range scopes {
		if deref(scope.ID) == id {
			return true
		}
	}
	return false
}

// verifyToken returns a fresh admin token for verifying effects straight
// through Keycloak, bypassing the tool layer under test.
func verifyToken(t *testing.T) string {
	t.Helper()
	gc := gocloak.NewClient(testBaseURL)
	jwt, err := gc.LoginAdmin(t.Context(), "admin", "admin", "master")
	if err != nil {
		t.Fatalf("verification admin login: %v", err)
	}
	return jwt.AccessToken
}

func hasRoleName(roles []*gocloak.Role, name string) bool {
	for _, r := range roles {
		if deref(r.Name) == name {
			return true
		}
	}
	return false
}

func hasGroupID(groups []*gocloak.Group, id string) bool {
	for _, g := range groups {
		if deref(g.ID) == id {
			return true
		}
	}
	return false
}
