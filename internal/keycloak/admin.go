// Package keycloak provides a thin, concurrency-safe wrapper around the
// gocloak Keycloak client, handling administrator authentication and mapping
// errors to descriptive messages.
package keycloak

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v13"
)

// refreshLead is how long before its expiry a cached admin token is refreshed.
const refreshLead = 30 * time.Second

// Admin performs Keycloak administration through the Admin REST API.
// The zero value is not usable; construct one with NewAdmin.
// All methods are safe for concurrent use.
type Admin struct {
	client   *gocloak.GoCloak
	username string
	password string
	realm    string

	mu           sync.Mutex
	accessToken  string
	tokenExpires time.Time
}

// NewAdmin returns an Admin that authenticates as the given administrator
// against the Keycloak server at baseURL. realm is the realm the account
// lives in, usually "master".
func NewAdmin(baseURL, username, password, realm string) *Admin {
	return &Admin{
		client:   gocloak.NewClient(baseURL),
		username: username,
		password: password,
		realm:    realm,
	}
}

// token returns a cached admin access token, logging in again when the
// cached token is missing or about to expire.
func (a *Admin) token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.accessToken != "" && time.Now().Add(refreshLead).Before(a.tokenExpires) {
		return a.accessToken, nil
	}
	jwt, err := a.client.LoginAdmin(ctx, a.username, a.password, a.realm)
	if err != nil {
		return "", wrapErr("admin login", err)
	}
	a.accessToken = jwt.AccessToken
	a.tokenExpires = time.Now().Add(time.Duration(jwt.ExpiresIn) * time.Second)
	return a.accessToken, nil
}

// wrapErr adds context to an error returned by gocloak, surfacing Keycloak
// API error codes when present.
func wrapErr(op string, err error) error {
	var apiErr *gocloak.APIError
	if errors.As(err, &apiErr) {
		return fmt.Errorf("%s: keycloak API error %d: %s", op, apiErr.Code, apiErr.Message)
	}
	return fmt.Errorf("%s: %w", op, err)
}

// Realms.

// ListRealms returns all realms on the server.
func (a *Admin) ListRealms(ctx context.Context) ([]*gocloak.RealmRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	realms, err := a.client.GetRealms(ctx, tok)
	if err != nil {
		return nil, wrapErr("list realms", err)
	}
	return realms, nil
}

// GetRealm returns the realm with the given name.
func (a *Admin) GetRealm(ctx context.Context, realm string) (*gocloak.RealmRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	rep, err := a.client.GetRealm(ctx, tok, realm)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get realm %q", realm), err)
	}
	return rep, nil
}

// CreateRealm creates the realm described by rep and returns the created
// representation.
func (a *Admin) CreateRealm(ctx context.Context, rep gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error) {
	if rep.Realm == nil || *rep.Realm == "" {
		return nil, fmt.Errorf("create realm: empty realm name")
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := a.client.CreateRealm(ctx, tok, rep); err != nil {
		return nil, wrapErr(fmt.Sprintf("create realm %q", *rep.Realm), err)
	}
	return a.GetRealm(ctx, *rep.Realm)
}

// UpdateRealm updates the realm described by rep (identified by rep.Realm)
// and returns the updated representation. Only fields set on rep are
// changed.
func (a *Admin) UpdateRealm(ctx context.Context, rep gocloak.RealmRepresentation) (*gocloak.RealmRepresentation, error) {
	if rep.Realm == nil || *rep.Realm == "" {
		return nil, fmt.Errorf("update realm: empty realm name")
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.client.UpdateRealm(ctx, tok, rep); err != nil {
		return nil, wrapErr(fmt.Sprintf("update realm %q", *rep.Realm), err)
	}
	return a.GetRealm(ctx, *rep.Realm)
}

// DeleteRealm permanently deletes the named realm and everything in it.
func (a *Admin) DeleteRealm(ctx context.Context, realm string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteRealm(ctx, tok, realm); err != nil {
		return wrapErr(fmt.Sprintf("delete realm %q", realm), err)
	}
	return nil
}

// Clients.

// ListClients returns the clients in realm. If clientID is non-empty, only
// clients with that client identifier are returned. max <= 0 means no
// explicit limit.
func (a *Admin) ListClients(ctx context.Context, realm, clientID string, max int) ([]*gocloak.Client, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetClientsParams{}
	if clientID != "" {
		params.ClientID = gocloak.StringP(clientID)
	}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	clients, err := a.client.GetClients(ctx, tok, realm, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list clients in realm %q", realm), err)
	}
	return clients, nil
}

// GetClient returns the client with the internal ID id.
func (a *Admin) GetClient(ctx context.Context, realm, id string) (*gocloak.Client, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	client, err := a.client.GetClient(ctx, tok, realm, id)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get client %q in realm %q", id, realm), err)
	}
	return client, nil
}

// CreateClient creates the client described by rep in realm and returns the
// created representation.
func (a *Admin) CreateClient(ctx context.Context, realm string, rep gocloak.Client) (*gocloak.Client, error) {
	if rep.ClientID == nil || *rep.ClientID == "" {
		return nil, fmt.Errorf("create client in realm %q: empty clientId", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	id, err := a.client.CreateClient(ctx, tok, realm, rep)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("create client %q in realm %q", *rep.ClientID, realm), err)
	}
	return a.GetClient(ctx, realm, id)
}

// GetClientSecret returns the current secret of the client with the internal
// ID id.
func (a *Admin) GetClientSecret(ctx context.Context, realm, id string) (*gocloak.CredentialRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	secret, err := a.client.GetClientSecret(ctx, tok, realm, id)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get secret of client %q in realm %q", id, realm), err)
	}
	return secret, nil
}

// DeleteClient deletes the client with the internal ID id.
func (a *Admin) DeleteClient(ctx context.Context, realm, id string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteClient(ctx, tok, realm, id); err != nil {
		return wrapErr(fmt.Sprintf("delete client %q in realm %q", id, realm), err)
	}
	return nil
}

// Users.

// ListUsers returns users in realm matching the optional search (substring
// against username, email and names) or exact username. max <= 0 means no
// explicit limit.
func (a *Admin) ListUsers(ctx context.Context, realm, search, username string, max int) ([]*gocloak.User, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetUsersParams{BriefRepresentation: gocloak.BoolP(true)}
	if search != "" {
		params.Search = gocloak.StringP(search)
	}
	if username != "" {
		params.Username = gocloak.StringP(username)
		params.Exact = gocloak.BoolP(true)
	}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	users, err := a.client.GetUsers(ctx, tok, realm, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list users in realm %q", realm), err)
	}
	return users, nil
}

// GetUser returns the user with the internal ID userID.
func (a *Admin) GetUser(ctx context.Context, realm, userID string) (*gocloak.User, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	user, err := a.client.GetUserByID(ctx, tok, realm, userID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get user %q in realm %q", userID, realm), err)
	}
	return user, nil
}

// CreateUser creates the user described by rep in realm and returns the
// created representation.
func (a *Admin) CreateUser(ctx context.Context, realm string, rep gocloak.User) (*gocloak.User, error) {
	if rep.Username == nil || *rep.Username == "" {
		return nil, fmt.Errorf("create user in realm %q: empty username", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	id, err := a.client.CreateUser(ctx, tok, realm, rep)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("create user %q in realm %q", *rep.Username, realm), err)
	}
	return a.GetUser(ctx, realm, id)
}

// UpdateUser updates the user described by rep (identified by rep.ID) and
// returns the updated representation. Only fields set on rep are changed.
func (a *Admin) UpdateUser(ctx context.Context, realm string, rep gocloak.User) (*gocloak.User, error) {
	if rep.ID == nil || *rep.ID == "" {
		return nil, fmt.Errorf("update user in realm %q: empty userId", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.client.UpdateUser(ctx, tok, realm, rep); err != nil {
		return nil, wrapErr(fmt.Sprintf("update user %q in realm %q", *rep.ID, realm), err)
	}
	return a.GetUser(ctx, realm, *rep.ID)
}

// SetUserPassword sets userID's password in realm. If temporary is true, the
// user must change the password at next login.
func (a *Admin) SetUserPassword(ctx context.Context, realm, userID, password string, temporary bool) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.SetPassword(ctx, tok, userID, realm, password, temporary); err != nil {
		return wrapErr(fmt.Sprintf("set password of user %q in realm %q", userID, realm), err)
	}
	return nil
}

// DeleteUser deletes the user with the internal ID userID.
func (a *Admin) DeleteUser(ctx context.Context, realm, userID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteUser(ctx, tok, realm, userID); err != nil {
		return wrapErr(fmt.Sprintf("delete user %q in realm %q", userID, realm), err)
	}
	return nil
}

// Groups.

// ListGroups returns groups in realm matching the optional search substring.
// max <= 0 means no explicit limit.
func (a *Admin) ListGroups(ctx context.Context, realm, search string, max int) ([]*gocloak.Group, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetGroupsParams{BriefRepresentation: gocloak.BoolP(true)}
	if search != "" {
		params.Search = gocloak.StringP(search)
	}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	groups, err := a.client.GetGroups(ctx, tok, realm, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list groups in realm %q", realm), err)
	}
	return groups, nil
}

// CreateGroup creates a top-level group with the given name in realm and
// returns the created representation.
func (a *Admin) CreateGroup(ctx context.Context, realm, name string) (*gocloak.Group, error) {
	if name == "" {
		return nil, fmt.Errorf("create group in realm %q: empty name", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	id, err := a.client.CreateGroup(ctx, tok, realm, gocloak.Group{Name: gocloak.StringP(name)})
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("create group %q in realm %q", name, realm), err)
	}
	group, err := a.client.GetGroup(ctx, tok, realm, id)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get created group %q in realm %q", name, realm), err)
	}
	return group, nil
}

// DeleteGroup deletes the group with the internal ID groupID.
func (a *Admin) DeleteGroup(ctx context.Context, realm, groupID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteGroup(ctx, tok, realm, groupID); err != nil {
		return wrapErr(fmt.Sprintf("delete group %q in realm %q", groupID, realm), err)
	}
	return nil
}

// Realm roles.

// ListRealmRoles returns the realm roles of realm. max <= 0 means no
// explicit limit.
func (a *Admin) ListRealmRoles(ctx context.Context, realm string, max int) ([]*gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetRoleParams{BriefRepresentation: gocloak.BoolP(true)}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	roles, err := a.client.GetRealmRoles(ctx, tok, realm, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list realm roles in realm %q", realm), err)
	}
	return roles, nil
}

// CreateRealmRole creates the role described by rep in realm and returns the
// created representation.
func (a *Admin) CreateRealmRole(ctx context.Context, realm string, rep gocloak.Role) (*gocloak.Role, error) {
	if rep.Name == nil || *rep.Name == "" {
		return nil, fmt.Errorf("create realm role in realm %q: empty name", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := a.client.CreateRealmRole(ctx, tok, realm, rep); err != nil {
		return nil, wrapErr(fmt.Sprintf("create realm role %q in realm %q", *rep.Name, realm), err)
	}
	role, err := a.client.GetRealmRole(ctx, tok, realm, *rep.Name)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get created realm role %q in realm %q", *rep.Name, realm), err)
	}
	return role, nil
}

// DeleteRealmRole deletes the named realm role.
func (a *Admin) DeleteRealmRole(ctx context.Context, realm, name string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteRealmRole(ctx, tok, realm, name); err != nil {
		return wrapErr(fmt.Sprintf("delete realm role %q in realm %q", name, realm), err)
	}
	return nil
}
