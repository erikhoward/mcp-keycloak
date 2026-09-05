package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/Nerzal/gocloak/v13"
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
	t.Helper()
	srv := New(admin)
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
		"client_get", "client_delete", "client_update",
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
	if captured.RedirectURIs == nil || len(*captured.RedirectURIs) != 1 || (*captured.RedirectURIs)[0] != "https://app.example.com/cb" {
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
