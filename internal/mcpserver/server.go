// Package mcpserver exposes Keycloak administration as Model Context
// Protocol tools.
package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/erikhoward/mcp-keycloak/internal/keycloak"
)

// serverVersion is the local development fallback. Release builds override it
// with the tag version through the linker.
var serverVersion = "0.1.0"

// Options controls which MCP tools are advertised.
type Options struct {
	ReadOnly bool
}

// AdminAPI is the subset of Keycloak administration the tools rely on.
// *keycloak.Admin implements it; tests use fakes.
type AdminAPI interface {
	ListRealms(ctx context.Context) ([]*gocloak.RealmRepresentation, error)
	GetRealm(ctx context.Context, realm string) (*gocloak.RealmRepresentation, error)
	CreateRealm(ctx context.Context, realm gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error)
	UpdateRealm(ctx context.Context, realm gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error)
	DeleteRealm(ctx context.Context, realm string) error

	ListClients(ctx context.Context, realm, clientID string, max int) ([]*gocloak.Client, error)
	GetClient(ctx context.Context, realm, id string) (*gocloak.Client, error)
	CreateClient(ctx context.Context, realm string, client gocloak.Client) (*gocloak.Client, error)
	UpdateClient(ctx context.Context, realm string, client gocloak.Client) (*gocloak.Client, error)
	GetClientSecret(ctx context.Context, realm, id string) (*gocloak.CredentialRepresentation, error)
	DeleteClient(ctx context.Context, realm, id string) error

	ListClientScopes(ctx context.Context, realm string) ([]*gocloak.ClientScope, error)
	GetClientScope(ctx context.Context, realm, id string) (*gocloak.ClientScope, error)
	CreateClientScope(ctx context.Context, realm string, scope gocloak.ClientScope) (*gocloak.ClientScope, error)
	DeleteClientScope(ctx context.Context, realm, id string) error
	AddDefaultScopeToClient(ctx context.Context, realm, clientID, scopeID string) error
	AddOptionalScopeToClient(ctx context.Context, realm, clientID, scopeID string) error
	RemoveDefaultScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error
	RemoveOptionalScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error
	ListEvents(ctx context.Context, realm string, params gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error)
	ListAdminEvents(ctx context.Context, realm string, params gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error)
	ListIdentityProviders(ctx context.Context, realm string) ([]*gocloak.IdentityProviderRepresentation, error)
	GetIdentityProvider(ctx context.Context, realm, alias string) (*gocloak.IdentityProviderRepresentation, error)
	CreateIdentityProvider(ctx context.Context, realm string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error)
	UpdateIdentityProvider(ctx context.Context, realm, alias string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error)
	DeleteIdentityProvider(ctx context.Context, realm, alias string) error

	ListUsers(ctx context.Context, realm, search, username string, max int) ([]*gocloak.User, error)
	GetUser(ctx context.Context, realm, userID string) (*gocloak.User, error)
	CreateUser(ctx context.Context, realm string, user gocloak.User) (*gocloak.User, error)
	UpdateUser(ctx context.Context, realm string, user gocloak.User) (*gocloak.User, error)
	SetUserPassword(ctx context.Context, realm, userID, password string, temporary bool) error
	DeleteUser(ctx context.Context, realm, userID string) error
	AddRealmRolesToUser(ctx context.Context, realm, userID string, roleNames []string) error
	RemoveRealmRolesFromUser(ctx context.Context, realm, userID string, roleNames []string) error
	AddUserToGroup(ctx context.Context, realm, userID, groupID string) error
	RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) error
	ListUserSessions(ctx context.Context, realm, userID string) ([]*gocloak.UserSessionRepresentation, error)
	LogoutAllUserSessions(ctx context.Context, realm, userID string) error
	LogoutUserSession(ctx context.Context, realm, sessionID string) error
	ListUserGroups(ctx context.Context, realm, userID string, max int) ([]*gocloak.Group, error)
	ListGroupMembers(ctx context.Context, realm, groupID string, max int) ([]*gocloak.User, error)
	GetUserRealmRoles(ctx context.Context, realm, userID string) ([]*gocloak.Role, error)
	GetCompositeUserRealmRoles(ctx context.Context, realm, userID string) ([]*gocloak.Role, error)
	GetGroupRealmRoles(ctx context.Context, realm, groupID string) ([]*gocloak.Role, error)
	GetCompositeGroupRealmRoles(ctx context.Context, realm, groupID string) ([]*gocloak.Role, error)

	ListGroups(ctx context.Context, realm, search string, max int) ([]*gocloak.Group, error)
	CreateGroup(ctx context.Context, realm, name string) (*gocloak.Group, error)
	DeleteGroup(ctx context.Context, realm, groupID string) error

	ListRealmRoles(ctx context.Context, realm string, max int) ([]*gocloak.Role, error)
	CreateRealmRole(ctx context.Context, realm string, role gocloak.Role) (*gocloak.Role, error)
	DeleteRealmRole(ctx context.Context, realm, name string) error
}

// Compile-time check that *keycloak.Admin satisfies AdminAPI.
var _ AdminAPI = (*keycloak.Admin)(nil)

// New returns an MCP server exposing Keycloak administration tools backed by
// the given admin client.
func New(admin AdminAPI) *mcp.Server {
	return NewWithOptions(admin, Options{})
}

// NewWithOptions returns an MCP server backed by admin with the requested
// safety settings.
func NewWithOptions(admin AdminAPI, options Options) *mcp.Server {
	s := mcp.NewServer(&mcp.Implementation{
		Name:    "mcp-keycloak",
		Version: serverVersion,
	}, nil)
	addRealmTools(s, admin, options)
	addClientTools(s, admin, options)
	addClientScopeTools(s, admin, options)
	addEventTools(s, admin)
	addIdentityProviderTools(s, admin, options)
	addUserTools(s, admin, options)
	addGroupTools(s, admin, options)
	addRealmRoleTools(s, admin, options)
	return s
}
