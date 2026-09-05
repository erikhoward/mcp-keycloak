package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// fakeAdmin implements AdminAPI for tests. Each test overrides only the
// methods it exercises via the function fields; the embedded nil AdminAPI
// panics if any other method is called, which fails the test loudly.
type fakeAdmin struct {
	AdminAPI

	listRealms               func(ctx context.Context) ([]*gocloak.RealmRepresentation, error)
	createRealm              func(ctx context.Context, rep gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error)
	deleteRealm              func(ctx context.Context, realm string) error
	listClients              func(ctx context.Context, realm, clientID string, max int) ([]*gocloak.Client, error)
	updateClient             func(ctx context.Context, realm string, rep gocloak.Client) (*gocloak.Client, error)
	getClientSecret          func(ctx context.Context, realm, id string) (*gocloak.CredentialRepresentation, error)
	listClientScopes         func(ctx context.Context, realm string) ([]*gocloak.ClientScope, error)
	getClientScope           func(ctx context.Context, realm, id string) (*gocloak.ClientScope, error)
	createClientScope        func(ctx context.Context, realm string, scope gocloak.ClientScope) (*gocloak.ClientScope, error)
	deleteClientScope        func(ctx context.Context, realm, id string) error
	addDefaultScope          func(ctx context.Context, realm, clientID, scopeID string) error
	addOptionalScope         func(ctx context.Context, realm, clientID, scopeID string) error
	removeDefaultScope       func(ctx context.Context, realm, clientID, scopeID string) error
	removeOptionalScope      func(ctx context.Context, realm, clientID, scopeID string) error
	listEvents               func(ctx context.Context, realm string, params gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error)
	listAdminEvents          func(ctx context.Context, realm string, params gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error)
	listIdentityProviders    func(ctx context.Context, realm string) ([]*gocloak.IdentityProviderRepresentation, error)
	getIdentityProvider      func(ctx context.Context, realm, alias string) (*gocloak.IdentityProviderRepresentation, error)
	createIdentityProvider   func(ctx context.Context, realm string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error)
	updateIdentityProvider   func(ctx context.Context, realm, alias string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error)
	deleteIdentityProvider   func(ctx context.Context, realm, alias string) error
	createUser               func(ctx context.Context, realm string, rep gocloak.User) (*gocloak.User, error)
	setUserPassword          func(ctx context.Context, realm, userID, password string, temporary bool) error
	addRealmRolesToUser      func(ctx context.Context, realm, userID string, roleNames []string) error
	removeRealmRolesFromUser func(ctx context.Context, realm, userID string, roleNames []string) error
	addUserToGroup           func(ctx context.Context, realm, userID, groupID string) error
	removeUserFromGroup      func(ctx context.Context, realm, userID, groupID string) error
}

func (f fakeAdmin) ListRealms(ctx context.Context) ([]*gocloak.RealmRepresentation, error) {
	return f.listRealms(ctx)
}

func (f fakeAdmin) CreateRealm(ctx context.Context, rep gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error) {
	return f.createRealm(ctx, rep)
}

func (f fakeAdmin) DeleteRealm(ctx context.Context, realm string) error {
	return f.deleteRealm(ctx, realm)
}

func (f fakeAdmin) ListClients(ctx context.Context, realm, clientID string, max int) ([]*gocloak.Client, error) {
	return f.listClients(ctx, realm, clientID, max)
}

func (f fakeAdmin) UpdateClient(ctx context.Context, realm string, rep gocloak.Client) (*gocloak.Client, error) {
	return f.updateClient(ctx, realm, rep)
}

func (f fakeAdmin) GetClientSecret(ctx context.Context, realm, id string) (*gocloak.CredentialRepresentation, error) {
	return f.getClientSecret(ctx, realm, id)
}

func (f fakeAdmin) ListClientScopes(ctx context.Context, realm string) ([]*gocloak.ClientScope, error) {
	return f.listClientScopes(ctx, realm)
}

func (f fakeAdmin) GetClientScope(ctx context.Context, realm, id string) (*gocloak.ClientScope, error) {
	return f.getClientScope(ctx, realm, id)
}

func (f fakeAdmin) CreateClientScope(ctx context.Context, realm string, scope gocloak.ClientScope) (*gocloak.ClientScope, error) {
	return f.createClientScope(ctx, realm, scope)
}

func (f fakeAdmin) DeleteClientScope(ctx context.Context, realm, id string) error {
	return f.deleteClientScope(ctx, realm, id)
}

func (f fakeAdmin) AddDefaultScopeToClient(ctx context.Context, realm, clientID, scopeID string) error {
	return f.addDefaultScope(ctx, realm, clientID, scopeID)
}

func (f fakeAdmin) AddOptionalScopeToClient(ctx context.Context, realm, clientID, scopeID string) error {
	return f.addOptionalScope(ctx, realm, clientID, scopeID)
}

func (f fakeAdmin) RemoveDefaultScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error {
	return f.removeDefaultScope(ctx, realm, clientID, scopeID)
}

func (f fakeAdmin) RemoveOptionalScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error {
	return f.removeOptionalScope(ctx, realm, clientID, scopeID)
}

func (f fakeAdmin) ListEvents(ctx context.Context, realm string, params gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error) {
	return f.listEvents(ctx, realm, params)
}

func (f fakeAdmin) ListAdminEvents(ctx context.Context, realm string, params gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error) {
	return f.listAdminEvents(ctx, realm, params)
}

func (f fakeAdmin) ListIdentityProviders(ctx context.Context, realm string) ([]*gocloak.IdentityProviderRepresentation, error) {
	return f.listIdentityProviders(ctx, realm)
}

func (f fakeAdmin) GetIdentityProvider(ctx context.Context, realm, alias string) (*gocloak.IdentityProviderRepresentation, error) {
	return f.getIdentityProvider(ctx, realm, alias)
}

func (f fakeAdmin) CreateIdentityProvider(ctx context.Context, realm string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
	return f.createIdentityProvider(ctx, realm, provider)
}

func (f fakeAdmin) UpdateIdentityProvider(ctx context.Context, realm, alias string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
	return f.updateIdentityProvider(ctx, realm, alias, provider)
}

func (f fakeAdmin) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	return f.deleteIdentityProvider(ctx, realm, alias)
}

func (f fakeAdmin) CreateUser(ctx context.Context, realm string, rep gocloak.User) (*gocloak.User, error) {
	return f.createUser(ctx, realm, rep)
}

func (f fakeAdmin) SetUserPassword(ctx context.Context, realm, userID, password string, temporary bool) error {
	return f.setUserPassword(ctx, realm, userID, password, temporary)
}

func (f fakeAdmin) AddRealmRolesToUser(ctx context.Context, realm, userID string, roleNames []string) error {
	return f.addRealmRolesToUser(ctx, realm, userID, roleNames)
}

func (f fakeAdmin) RemoveRealmRolesFromUser(ctx context.Context, realm, userID string, roleNames []string) error {
	return f.removeRealmRolesFromUser(ctx, realm, userID, roleNames)
}

func (f fakeAdmin) AddUserToGroup(ctx context.Context, realm, userID, groupID string) error {
	return f.addUserToGroup(ctx, realm, userID, groupID)
}

func (f fakeAdmin) RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) error {
	return f.removeUserFromGroup(ctx, realm, userID, groupID)
}

// newTestClient connects an MCP client session to a server built from admin
// over an in-memory transport.
func newTestClient(t *testing.T, admin AdminAPI) *mcp.ClientSession {
	return newTestClientWithOptions(t, admin, Options{})
}

func newTestClientWithOptions(t *testing.T, admin AdminAPI, options Options) *mcp.ClientSession {
	t.Helper()
	srv := NewWithOptions(admin, options)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	go func() {
		_ = srv.Run(t.Context(), serverTransport)
	}()
	client := mcp.NewClient(&mcp.Implementation{Name: "mcp-keycloak-test", Version: "0.0.0"}, nil)
	cs, err := client.Connect(t.Context(), clientTransport, nil)
	if err != nil {
		t.Fatalf("connecting MCP client: %v", err)
	}
	t.Cleanup(func() {
		if err := cs.Close(); err != nil {
			t.Errorf("closing session: %v", err)
		}
	})
	return cs
}

func TestServerAdvertisesVersion(t *testing.T) {
	cs := newTestClient(t, &fakeAdmin{})
	if got := cs.InitializeResult().ServerInfo.Version; got != serverVersion {
		t.Errorf("server version = %q, want %q", got, serverVersion)
	}
}

func TestReadOnlyOmitsMutatingTools(t *testing.T) {
	cs := newTestClientWithOptions(t, &fakeAdmin{}, Options{ReadOnly: true})
	result, err := cs.ListTools(t.Context(), &mcp.ListToolsParams{})
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	tools := make(map[string]bool, len(result.Tools))
	for _, tool := range result.Tools {
		tools[tool.Name] = true
	}
	for _, name := range []string{
		"realm_create", "realm_update", "realm_delete",
		"client_create", "client_update", "client_delete",
		"client_scope_create", "client_scope_delete", "client_scope_assign", "client_scope_unassign",
		"identity_provider_create", "identity_provider_update", "identity_provider_delete",
		"user_create", "user_update", "user_set_password", "user_delete",
		"user_add_realm_role", "user_remove_realm_role", "user_add_to_group", "user_remove_from_group",
		"group_create", "group_delete", "realm_role_create", "realm_role_delete",
	} {
		if tools[name] {
			t.Errorf("read-only server advertises mutating tool %q", name)
		}
	}
	for _, name := range []string{
		"realm_list", "realm_get", "client_list", "client_get", "client_secret_get",
		"client_scope_list", "client_scope_get", "event_admin_list", "event_login_list",
		"identity_provider_list", "identity_provider_get", "user_list", "user_get",
		"group_list", "realm_role_list",
	} {
		if !tools[name] {
			t.Errorf("read-only server omitted read tool %q", name)
		}
	}
}

// callTool invokes a tool and fails the test unless it succeeds.
func callTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool(%s): %v", name, err)
	}
	if res.IsError {
		t.Fatalf("CallTool(%s) failed: %s", name, resultText(t, res))
	}
	return res
}

// resultText returns the JSON text content of a result.
func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	for _, c := range res.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			return tc.Text
		}
	}
	t.Fatal("result has no text content")
	return ""
}

// decodeResult decodes a result's JSON text content into v.
func decodeResult[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var v T
	if err := json.Unmarshal([]byte(resultText(t, res)), &v); err != nil {
		t.Fatalf("decoding result into %T: %v", v, err)
	}
	return v
}

func decodeStructuredResult[T any](t *testing.T, res *mcp.CallToolResult) T {
	t.Helper()
	var v T
	encoded, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("encoding structured result: %v", err)
	}
	if err := json.Unmarshal(encoded, &v); err != nil {
		t.Fatalf("decoding structured result into %T: %v", v, err)
	}
	return v
}

func TestRealmList(t *testing.T) {
	admin := &fakeAdmin{listRealms: func(context.Context) ([]*gocloak.RealmRepresentation, error) {
		return []*gocloak.RealmRepresentation{
			{ID: gocloak.StringP("r1"), Realm: gocloak.StringP("master")},
			{ID: gocloak.StringP("r2"), Realm: gocloak.StringP("acme")},
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "realm_list", nil)
	realms := decodeResult[[]map[string]any](t, res)
	if len(realms) != 2 {
		t.Fatalf("got %d realms, want 2", len(realms))
	}
	if res.StructuredContent == nil {
		t.Error("expected structured content alongside text")
	}
}

func TestRealmListEmptyMarshalsAsArray(t *testing.T) {
	admin := &fakeAdmin{listRealms: func(context.Context) ([]*gocloak.RealmRepresentation, error) {
		return nil, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "realm_list", nil)
	if got := resultText(t, res); got != "[]" {
		t.Errorf("got %s, want []", got)
	}
}

func TestRealmCreate(t *testing.T) {
	var captured gocloak.RealmRepresentation
	admin := &fakeAdmin{createRealm: func(_ context.Context, rep gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error) {
		captured = rep
		return &gocloak.RealmRepresentation{
			ID: gocloak.StringP("r-1"), Realm: rep.Realm, DisplayName: rep.DisplayName, Enabled: rep.Enabled,
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "realm_create", map[string]any{
		"realm": "acme", "displayName": "ACME Corp",
	})
	if got := deref(captured.Realm); got != "acme" {
		t.Errorf("captured realm = %q, want %q", got, "acme")
	}
	if got := deref(captured.DisplayName); got != "ACME Corp" {
		t.Errorf("captured displayName = %q, want %q", got, "ACME Corp")
	}
	if captured.Enabled == nil || !*captured.Enabled {
		t.Error("enabled should default to true")
	}
	created := decodeResult[gocloak.RealmRepresentation](t, res)
	if got := deref(created.DisplayName); got != "ACME Corp" {
		t.Errorf("result displayName = %q, want %q", got, "ACME Corp")
	}
}

func TestRealmCreateErrorBecomesToolError(t *testing.T) {
	admin := &fakeAdmin{createRealm: func(context.Context, gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error) {
		return nil, errors.New("boom: conflict")
	}}
	cs := newTestClient(t, admin)

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "realm_create",
		Arguments: map[string]any{"realm": "acme"},
	})
	if err != nil {
		t.Fatalf("CallTool protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error result")
	}
	if text := resultText(t, res); !strings.Contains(text, "boom: conflict") {
		t.Errorf("error text %q does not contain the cause", text)
	}
}

func TestRealmDeleteOutput(t *testing.T) {
	var deleted string
	admin := &fakeAdmin{deleteRealm: func(_ context.Context, realm string) error {
		deleted = realm
		return nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "realm_delete", map[string]any{"realm": "acme"})
	if deleted != "acme" {
		t.Errorf("deleted realm = %q, want %q", deleted, "acme")
	}
	out := decodeResult[map[string]any](t, res)
	if out["deleted"] != true {
		t.Errorf("output = %v, want deleted=true", out)
	}
}

func TestRequiredArgumentsValidated(t *testing.T) {
	cs := newTestClient(t, &fakeAdmin{})

	for _, name := range []string{
		"realm_get", "realm_delete",
		"client_get", "client_delete", "client_update", "client_secret_get",
		"client_scope_get", "client_scope_delete",
		"client_scope_assign", "client_scope_unassign",
		"event_admin_list", "event_login_list",
		"identity_provider_get", "identity_provider_update", "identity_provider_delete",
		"user_get", "user_set_password", "user_delete",
		"user_add_realm_role", "user_remove_realm_role",
		"user_add_to_group", "user_remove_from_group",
		"group_delete",
		"realm_role_delete",
	} {
		res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: map[string]any{}})
		if err == nil && !res.IsError {
			t.Errorf("%s: expected an error when required arguments are missing", name)
		}
	}
}

func TestIdentityProviderCreateRedactsSecret(t *testing.T) {
	var captured gocloak.IdentityProviderRepresentation
	admin := &fakeAdmin{createIdentityProvider: func(_ context.Context, _ string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
		captured = provider
		return &gocloak.IdentityProviderRepresentation{
			Alias: provider.Alias, ProviderID: provider.ProviderID, DisplayName: provider.DisplayName,
			Config: map[string]string{"clientId": "client-1", "clientSecret": "super-secret", "issuer": "https://idp.example.com"},
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "identity_provider_create", map[string]any{
		"realm": "acme", "alias": "corporate", "displayName": "Corporate SSO",
		"issuer": "https://idp.example.com", "clientId": "client-1", "clientSecret": "super-secret",
	})
	if deref(captured.ProviderID) != "oidc" {
		t.Errorf("providerID = %q, want oidc", deref(captured.ProviderID))
	}
	if captured.Config["clientSecret"] != "super-secret" {
		t.Errorf("captured client secret = %q, want original input", captured.Config["clientSecret"])
	}
	text := resultText(t, res)
	if strings.Contains(text, "super-secret") || !strings.Contains(text, redactedSecret) {
		t.Errorf("result leaked or failed to redact client secret: %s", text)
	}
}

func TestIdentityProviderUpdateIsPartialAndRedacted(t *testing.T) {
	var captured gocloak.IdentityProviderRepresentation
	admin := &fakeAdmin{updateIdentityProvider: func(_ context.Context, _, _ string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
		captured = provider
		return &gocloak.IdentityProviderRepresentation{
			Alias: gocloak.StringP("corporate"), DisplayName: provider.DisplayName,
			Config: map[string]string{"clientSecret": "super-secret", "defaultScope": "openid email"},
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "identity_provider_update", map[string]any{
		"realm": "acme", "alias": "corporate", "displayName": "Corporate v2", "defaultScope": "openid email",
	})
	if deref(captured.DisplayName) != "Corporate v2" || captured.Config["defaultScope"] != "openid email" {
		t.Errorf("captured update = display=%q config=%v", deref(captured.DisplayName), captured.Config)
	}
	if _, ok := captured.Config["clientSecret"]; ok {
		t.Error("omitted client secret should not be sent in a partial update")
	}
	text := resultText(t, res)
	if strings.Contains(text, "super-secret") || !strings.Contains(text, redactedSecret) {
		t.Errorf("result leaked or failed to redact client secret: %s", text)
	}
}

func TestIdentityProviderRejectsInsecureRemoteURL(t *testing.T) {
	called := false
	admin := &fakeAdmin{createIdentityProvider: func(context.Context, string, gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
		called = true
		return nil, nil
	}}
	cs := newTestClient(t, admin)

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "identity_provider_create",
		Arguments: map[string]any{"realm": "acme", "alias": "corporate", "issuer": "http://idp.example.com"},
	})
	if err != nil {
		t.Fatalf("CallTool protocol error: %v", err)
	}
	if !res.IsError || called {
		t.Errorf("expected URL validation tool error before admin call; result=%v called=%v", res.IsError, called)
	}
}

func TestAdminEventListMapsFilters(t *testing.T) {
	var captured gocloak.GetAdminEventsParams
	admin := &fakeAdmin{
		listAdminEvents: func(_ context.Context, _ string, params gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error) {
			captured = params
			return []*gocloak.AdminEventRepresentation{{OperationType: gocloak.StringP("CREATE")}}, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "event_admin_list", map[string]any{
		"realm": "acme", "authClient": "admin-cli", "authUser": "u-1",
		"operationTypes": []string{"CREATE", "UPDATE"}, "resourceTypes": []string{"USER"}, "max": 25,
	})
	if deref(captured.AuthClient) != "admin-cli" || deref(captured.AuthUser) != "u-1" {
		t.Errorf("auth filters = %q/%q, want admin-cli/u-1", deref(captured.AuthClient), deref(captured.AuthUser))
	}
	if len(captured.OperationTypes) != 2 || captured.OperationTypes[0] != "CREATE" || len(captured.ResourceTypes) != 1 || captured.ResourceTypes[0] != "USER" {
		t.Errorf("event filters = operations %v/resources %v", captured.OperationTypes, captured.ResourceTypes)
	}
	if captured.Max == nil || *captured.Max != 25 {
		t.Errorf("max = %v, want 25", captured.Max)
	}
	if len(decodeResult[[]gocloak.AdminEventRepresentation](t, res)) != 1 {
		t.Error("expected one admin event")
	}
}

func TestLoginEventListMapsFiltersAndDefaultsMax(t *testing.T) {
	var captured gocloak.GetEventsParams
	admin := &fakeAdmin{
		listEvents: func(_ context.Context, _ string, params gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error) {
			captured = params
			return []*gocloak.EventRepresentation{{Type: gocloak.StringP("LOGIN")}}, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "event_login_list", map[string]any{
		"realm": "acme", "clientId": "web-app", "userId": "u-1",
		"ipAddress": "127.0.0.1", "types": []string{"LOGIN", "LOGIN_ERROR"},
	})
	if deref(captured.Client) != "web-app" || deref(captured.UserID) != "u-1" || deref(captured.IPAddress) != "127.0.0.1" {
		t.Errorf("login filters = client=%q user=%q ip=%q", deref(captured.Client), deref(captured.UserID), deref(captured.IPAddress))
	}
	if len(captured.Type) != 2 || captured.Type[0] != "LOGIN" || captured.Type[1] != "LOGIN_ERROR" {
		t.Errorf("types = %v, want LOGIN/LOGIN_ERROR", captured.Type)
	}
	if captured.Max == nil || *captured.Max != int32(defaultMax) {
		t.Errorf("max = %v, want default %d", captured.Max, defaultMax)
	}
	if len(decodeResult[[]gocloak.EventRepresentation](t, res)) != 1 {
		t.Error("expected one login event")
	}
}

func TestAuditEventPayloadsRedactSensitiveValues(t *testing.T) {
	admin := &fakeAdmin{
		listAdminEvents: func(context.Context, string, gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error) {
			return []*gocloak.AdminEventRepresentation{{
				Representation: gocloak.StringP(`{"clientSecret":"secret-value","nested":{"password":"password-value"},"name":"safe"}`),
			}}, nil
		},
		listEvents: func(context.Context, string, gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error) {
			return []*gocloak.EventRepresentation{{
				Type:    gocloak.StringP("LOGIN"),
				Details: map[string]string{"access_token": "token-value", "username": "alice"},
			}}, nil
		},
	}
	cs := newTestClient(t, admin)

	adminResult := callTool(t, cs, "event_admin_list", map[string]any{"realm": "acme"})
	adminText := resultText(t, adminResult)
	if strings.Contains(adminText, "secret-value") || strings.Contains(adminText, "password-value") || !strings.Contains(adminText, redactedSecret) || !strings.Contains(adminText, "safe") {
		t.Errorf("admin event output was not safely redacted: %s", adminText)
	}

	loginResult := callTool(t, cs, "event_login_list", map[string]any{"realm": "acme"})
	loginText := resultText(t, loginResult)
	if strings.Contains(loginText, "token-value") || !strings.Contains(loginText, redactedSecret) || !strings.Contains(loginText, "alice") {
		t.Errorf("login event output was not safely redacted: %s", loginText)
	}
}

func TestClientScopeCreateDefaultsProtocol(t *testing.T) {
	var captured gocloak.ClientScope
	admin := &fakeAdmin{createClientScope: func(_ context.Context, _ string, scope gocloak.ClientScope) (*gocloak.ClientScope, error) {
		captured = scope
		return &gocloak.ClientScope{
			ID: gocloak.StringP("scope-1"), Name: scope.Name, Description: scope.Description, Protocol: scope.Protocol,
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_scope_create", map[string]any{
		"realm": "acme", "name": "orders", "description": "Order claims",
	})
	if got := deref(captured.Name); got != "orders" {
		t.Errorf("captured name = %q, want orders", got)
	}
	if got := deref(captured.Protocol); got != "openid-connect" {
		t.Errorf("captured protocol = %q, want openid-connect", got)
	}
	if got := deref(captured.Description); got != "Order claims" {
		t.Errorf("captured description = %q, want Order claims", got)
	}
	created := decodeResult[gocloak.ClientScope](t, res)
	if got := deref(created.ID); got != "scope-1" {
		t.Errorf("created ID = %q, want scope-1", got)
	}
}

func TestClientScopeGetResolvesName(t *testing.T) {
	admin := &fakeAdmin{
		listClientScopes: func(context.Context, string) ([]*gocloak.ClientScope, error) {
			return []*gocloak.ClientScope{{ID: gocloak.StringP("scope-1"), Name: gocloak.StringP("orders")}}, nil
		},
		getClientScope: func(_ context.Context, _ string, id string) (*gocloak.ClientScope, error) {
			if id != "scope-1" {
				return nil, fmt.Errorf("unexpected scope ID %q", id)
			}
			return &gocloak.ClientScope{ID: gocloak.StringP(id), Name: gocloak.StringP("orders"), Protocol: gocloak.StringP("openid-connect")}, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_scope_get", map[string]any{"realm": "acme", "name": "orders"})
	scope := decodeResult[gocloak.ClientScope](t, res)
	if got := deref(scope.ID); got != "scope-1" {
		t.Errorf("scope ID = %q, want scope-1", got)
	}
}

func TestClientScopeAssignmentUsesInternalIDs(t *testing.T) {
	var added, removed [2]string
	admin := &fakeAdmin{
		listClients: func(context.Context, string, string, int) ([]*gocloak.Client, error) {
			return []*gocloak.Client{{ID: gocloak.StringP("client-1"), ClientID: gocloak.StringP("web-app")}}, nil
		},
		listClientScopes: func(context.Context, string) ([]*gocloak.ClientScope, error) {
			return []*gocloak.ClientScope{{ID: gocloak.StringP("scope-1"), Name: gocloak.StringP("orders")}}, nil
		},
		addDefaultScope: func(_ context.Context, _, clientID, scopeID string) error {
			added = [2]string{clientID, scopeID}
			return nil
		},
		removeOptionalScope: func(_ context.Context, _, clientID, scopeID string) error {
			removed = [2]string{clientID, scopeID}
			return nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_scope_assign", map[string]any{
		"realm": "acme", "clientId": "web-app", "scopeName": "orders",
	})
	if added != [2]string{"client-1", "scope-1"} {
		t.Errorf("default assignment IDs = %v, want client-1 / scope-1", added)
	}
	assigned := decodeResult[map[string]any](t, res)
	if assigned["assigned"] != true || assigned["optional"] != false {
		t.Errorf("assignment result = %v, want assigned=true and optional=false", assigned)
	}

	res = callTool(t, cs, "client_scope_unassign", map[string]any{
		"realm": "acme", "clientId": "web-app", "scopeName": "orders", "optional": true,
	})
	if removed != [2]string{"client-1", "scope-1"} {
		t.Errorf("optional removal IDs = %v, want client-1 / scope-1", removed)
	}
	unassigned := decodeResult[map[string]any](t, res)
	if unassigned["unassigned"] != true || unassigned["optional"] != true {
		t.Errorf("unassignment result = %v, want unassigned=true and optional=true", unassigned)
	}
}

func TestClientSecretGetOmitsSecretByDefault(t *testing.T) {
	called := false
	admin := &fakeAdmin{
		listClients: func(context.Context, string, string, int) ([]*gocloak.Client, error) {
			return []*gocloak.Client{{ID: gocloak.StringP("client-1"), ClientID: gocloak.StringP("app"), PublicClient: gocloak.BoolP(false)}}, nil
		},
		getClientSecret: func(context.Context, string, string) (*gocloak.CredentialRepresentation, error) {
			called = true
			return &gocloak.CredentialRepresentation{Value: gocloak.StringP("secret-value")}, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_secret_get", map[string]any{"realm": "acme", "clientId": "app"})
	if called {
		t.Error("secret lookup should not run when includeSecret is false")
	}
	if strings.Contains(resultText(t, res), "secret-value") {
		t.Error("default client_secret_get output contains a secret")
	}
	out := decodeStructuredResult[clientSecretOutput](t, res)
	if !out.SecretAvailable || out.Secret != "" {
		t.Errorf("safe output = %+v, want available=true and empty secret", out)
	}
}

func TestClientSecretGetExplicitlyIncludesSecretWithoutTextDuplication(t *testing.T) {
	admin := &fakeAdmin{
		listClients: func(context.Context, string, string, int) ([]*gocloak.Client, error) {
			return []*gocloak.Client{{ID: gocloak.StringP("client-1"), ClientID: gocloak.StringP("app"), PublicClient: gocloak.BoolP(false)}}, nil
		},
		getClientSecret: func(context.Context, string, string) (*gocloak.CredentialRepresentation, error) {
			return &gocloak.CredentialRepresentation{Value: gocloak.StringP("secret-value")}, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_secret_get", map[string]any{
		"realm": "acme", "clientId": "app", "includeSecret": true,
	})
	if strings.Contains(resultText(t, res), "secret-value") {
		t.Error("human-readable client_secret_get content contains the secret")
	}
	out := decodeStructuredResult[clientSecretOutput](t, res)
	if out.Secret != "secret-value" || !out.SecretAvailable {
		t.Errorf("explicit output = %+v, want secret-value and available=true", out)
	}
}

func TestClientGetResolvesExactClientID(t *testing.T) {
	admin := &fakeAdmin{listClients: func(_ context.Context, _, clientID string, _ int) ([]*gocloak.Client, error) {
		return []*gocloak.Client{
			{ID: gocloak.StringP("id-2"), ClientID: gocloak.StringP("app2")},
			{ID: gocloak.StringP("id-1"), ClientID: gocloak.StringP("app")},
		}, nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_get", map[string]any{"realm": "acme", "clientId": "app"})
	client := decodeResult[gocloak.Client](t, res)
	if got := deref(client.ID); got != "id-1" {
		t.Errorf("resolved client ID = %q, want %q", got, "id-1")
	}
}

func TestClientGetNotFound(t *testing.T) {
	admin := &fakeAdmin{listClients: func(context.Context, string, string, int) ([]*gocloak.Client, error) {
		return nil, nil
	}}
	cs := newTestClient(t, admin)

	res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
		Name:      "client_get",
		Arguments: map[string]any{"realm": "acme", "clientId": "ghost"},
	})
	if err != nil {
		t.Fatalf("CallTool protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatal("expected tool error for missing client")
	}
	if text := resultText(t, res); !strings.Contains(text, "ghost") {
		t.Errorf("error text %q does not name the missing client", text)
	}
}

func TestUserCreateWithInitialPassword(t *testing.T) {
	var gotUserID, gotPassword string
	var gotTemporary bool
	admin := &fakeAdmin{
		createUser: func(_ context.Context, _ string, rep gocloak.User) (*gocloak.User, error) {
			return &gocloak.User{ID: gocloak.StringP("u-1"), Username: rep.Username}, nil
		},
		setUserPassword: func(_ context.Context, _, userID, password string, temporary bool) error {
			gotUserID, gotPassword, gotTemporary = userID, password, temporary
			return nil
		},
	}
	cs := newTestClient(t, admin)

	callTool(t, cs, "user_create", map[string]any{
		"realm": "acme", "username": "alice", "initialPassword": "s3cret",
	})
	if gotUserID != "u-1" || gotPassword != "s3cret" {
		t.Errorf("setUserPassword(%q, %q), want (u-1, s3cret)", gotUserID, gotPassword)
	}
	if !gotTemporary {
		t.Error("initial password should be temporary by default")
	}

	callTool(t, cs, "user_create", map[string]any{
		"realm": "acme", "username": "bob",
		"initialPassword": "s3cret", "initialPasswordTemporary": false,
	})
	if gotTemporary {
		t.Error("initialPasswordTemporary=false should be honored")
	}
}

func TestUserSetPasswordDefaultsToPermanent(t *testing.T) {
	var gotTemporary bool
	admin := &fakeAdmin{setUserPassword: func(_ context.Context, _, _, _ string, temporary bool) error {
		gotTemporary = temporary
		return nil
	}}
	cs := newTestClient(t, admin)

	callTool(t, cs, "user_set_password", map[string]any{
		"realm": "acme", "userId": "u-1", "password": "pw",
	})
	if gotTemporary {
		t.Error("password should be permanent (temporary=false) by default")
	}
}

func TestClientUpdatePartialChanges(t *testing.T) {
	resolve := func(context.Context, string, string, int) ([]*gocloak.Client, error) {
		return []*gocloak.Client{{ID: gocloak.StringP("id-1"), ClientID: gocloak.StringP("app")}}, nil
	}
	var captured gocloak.Client
	admin := &fakeAdmin{
		listClients: resolve,
		updateClient: func(_ context.Context, _ string, rep gocloak.Client) (*gocloak.Client, error) {
			captured = rep
			echo := rep
			echo.ClientID = gocloak.StringP("app")
			return &echo, nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "client_update", map[string]any{
		"realm": "acme", "clientId": "app",
		"name": "App v2", "redirectURIs": []string{"https://app.example.com/cb"},
		"serviceAccountsEnabled": true,
	})
	if got := deref(captured.ID); got != "id-1" {
		t.Errorf("captured internal ID = %q, want id-1", got)
	}
	if got := deref(captured.Name); got != "App v2" {
		t.Errorf("captured name = %q, want %q", got, "App v2")
	}
	if len(captured.RedirectURIs) != 1 || captured.RedirectURIs[0] != "https://app.example.com/cb" {
		t.Errorf("captured redirectURIs = %v, want [https://app.example.com/cb]", captured.RedirectURIs)
	}
	if captured.ServiceAccountsEnabled == nil || !*captured.ServiceAccountsEnabled {
		t.Error("serviceAccountsEnabled should be set to true")
	}
	// Fields not provided must stay nil so Keycloak leaves them unchanged.
	if captured.Description != nil {
		t.Errorf("description = %v, want nil (unchanged)", captured.Description)
	}
	if captured.DirectAccessGrantsEnabled != nil {
		t.Errorf("directAccessGrantsEnabled = %v, want nil (unchanged)", captured.DirectAccessGrantsEnabled)
	}
	if captured.PublicClient != nil {
		t.Errorf("publicClient = %v, want nil (unchanged)", captured.PublicClient)
	}
	client := decodeResult[gocloak.Client](t, res)
	if got := deref(client.Name); got != "App v2" {
		t.Errorf("result name = %q, want %q", got, "App v2")
	}
}

func TestClientUpdateOmittedRedirectURIsUnchanged(t *testing.T) {
	resolve := func(context.Context, string, string, int) ([]*gocloak.Client, error) {
		return []*gocloak.Client{{ID: gocloak.StringP("id-1"), ClientID: gocloak.StringP("app")}}, nil
	}
	var captured gocloak.Client
	admin := &fakeAdmin{
		listClients: resolve,
		updateClient: func(_ context.Context, _ string, rep gocloak.Client) (*gocloak.Client, error) {
			captured = rep
			return &rep, nil
		},
	}
	cs := newTestClient(t, admin)

	callTool(t, cs, "client_update", map[string]any{"realm": "acme", "clientId": "app", "name": "App v3"})
	if captured.RedirectURIs != nil {
		t.Errorf("redirectURIs = %v, want nil when omitted", captured.RedirectURIs)
	}
}

func TestUserRealmRoleAssignment(t *testing.T) {
	var gotRealm, gotUserID string
	var gotNames []string
	admin := &fakeAdmin{addRealmRolesToUser: func(_ context.Context, realm, userID string, roleNames []string) error {
		gotRealm, gotUserID, gotNames = realm, userID, roleNames
		return nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "user_add_realm_role", map[string]any{
		"realm": "acme", "userId": "u-1", "roles": []string{"admin", "auditor"},
	})
	if gotRealm != "acme" || gotUserID != "u-1" {
		t.Errorf("called with (%q, %q), want (acme, u-1)", gotRealm, gotUserID)
	}
	if len(gotNames) != 2 || gotNames[0] != "admin" || gotNames[1] != "auditor" {
		t.Errorf("role names = %v, want [admin auditor]", gotNames)
	}
	out := decodeResult[map[string]any](t, res)
	if fmt.Sprint(out["rolesAdded"]) != "[admin auditor]" {
		t.Errorf("rolesAdded = %v, want [admin auditor]", out["rolesAdded"])
	}
}

func TestUserRealmRoleRemoval(t *testing.T) {
	var gotNames []string
	admin := &fakeAdmin{removeRealmRolesFromUser: func(_ context.Context, _, _ string, roleNames []string) error {
		gotNames = roleNames
		return nil
	}}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "user_remove_realm_role", map[string]any{
		"realm": "acme", "userId": "u-1", "roles": []string{"auditor"},
	})
	if len(gotNames) != 1 || gotNames[0] != "auditor" {
		t.Errorf("role names = %v, want [auditor]", gotNames)
	}
	out := decodeResult[map[string]any](t, res)
	if fmt.Sprint(out["rolesRemoved"]) != "[auditor]" {
		t.Errorf("rolesRemoved = %v, want [auditor]", out["rolesRemoved"])
	}
}

func TestUserRealmRoleEmptyListRejected(t *testing.T) {
	cs := newTestClient(t, &fakeAdmin{})
	for _, name := range []string{"user_add_realm_role", "user_remove_realm_role"} {
		res, err := cs.CallTool(t.Context(), &mcp.CallToolParams{
			Name:      name,
			Arguments: map[string]any{"realm": "acme", "userId": "u-1", "roles": []string{}},
		})
		if err != nil {
			t.Fatalf("%s: protocol error: %v", name, err)
		}
		if !res.IsError {
			t.Errorf("%s: expected tool error for empty roles list", name)
		}
	}
}

func TestUserGroupMembership(t *testing.T) {
	var added, removed [2]string
	admin := &fakeAdmin{
		addUserToGroup: func(_ context.Context, realm, userID, groupID string) error {
			added = [2]string{realm + ":" + userID, groupID}
			return nil
		},
		removeUserFromGroup: func(_ context.Context, realm, userID, groupID string) error {
			removed = [2]string{realm + ":" + userID, groupID}
			return nil
		},
	}
	cs := newTestClient(t, admin)

	res := callTool(t, cs, "user_add_to_group", map[string]any{
		"realm": "acme", "userId": "u-1", "groupId": "g-1",
	})
	if added != [2]string{"acme:u-1", "g-1"} {
		t.Errorf("addUserToGroup args = %v, want acme:u-1 / g-1", added)
	}
	out := decodeResult[map[string]any](t, res)
	if out["added"] != true {
		t.Errorf("add output = %v, want added=true", out)
	}

	res = callTool(t, cs, "user_remove_from_group", map[string]any{
		"realm": "acme", "userId": "u-1", "groupId": "g-1",
	})
	if removed != [2]string{"acme:u-1", "g-1"} {
		t.Errorf("removeUserFromGroup args = %v, want acme:u-1 / g-1", removed)
	}
	out = decodeResult[map[string]any](t, res)
	if out["removed"] != true {
		t.Errorf("remove output = %v, want removed=true", out)
	}
}
