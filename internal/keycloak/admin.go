// Package keycloak provides a thin, concurrency-safe wrapper around the
// gocloak Keycloak client, handling administrator authentication and mapping
// errors to descriptive messages.
package keycloak

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/Nerzal/gocloak/v14"
)

// refreshLead is how long before its expiry a cached admin token is refreshed.
const refreshLead = 30 * time.Second

// Admin performs Keycloak administration through the Admin REST API.
// The zero value is not usable; construct one with NewAdmin.
// All methods are safe for concurrent use.
type Admin struct {
	client       *gocloak.GoCloak
	baseURL      string
	username     string
	password     string
	clientID     string
	clientSecret string
	realm        string

	mu           sync.Mutex
	accessToken  string
	tokenExpires time.Time
}

// AdminOptions configures the HTTP client used for Keycloak administration.
type AdminOptions struct {
	HTTPTimeout                time.Duration
	CACertFile                 string
	ServiceAccountClientID     string
	ServiceAccountClientSecret string
}

const defaultHTTPTimeout = 30 * time.Second

// NewAdmin returns an Admin that authenticates as the given administrator
// against the Keycloak server at baseURL. realm is the realm the account
// lives in, usually "master".
func NewAdmin(baseURL, username, password, realm string) *Admin {
	admin, err := NewAdminWithOptions(baseURL, username, password, realm, AdminOptions{})
	if err != nil {
		panic(err)
	}
	return admin
}

// NewAdminWithOptions returns an Admin with bounded HTTP requests and an
// optional additional trusted CA bundle. It never disables TLS verification.
func NewAdminWithOptions(baseURL, username, password, realm string, options AdminOptions) (*Admin, error) {
	timeout := options.HTTPTimeout
	if timeout == 0 {
		timeout = defaultHTTPTimeout
	}
	if timeout < 0 {
		return nil, fmt.Errorf("keycloak HTTP timeout must be positive")
	}
	if (options.ServiceAccountClientID == "") != (options.ServiceAccountClientSecret == "") {
		return nil, fmt.Errorf("keycloak service-account client ID and secret must be configured together")
	}
	if username == "" && options.ServiceAccountClientID == "" {
		return nil, fmt.Errorf("keycloak administrator credentials are required")
	}
	if options.ServiceAccountClientID == "" && password == "" {
		return nil, fmt.Errorf("keycloak administrator password is required")
	}
	client := gocloak.NewClient(baseURL)
	client.RestyClient().SetTimeout(timeout)
	if options.CACertFile != "" {
		if err := addTrustedCA(client, options.CACertFile); err != nil {
			return nil, err
		}
	}
	return &Admin{
		client:       client,
		baseURL:      strings.TrimRight(baseURL, "/"),
		username:     username,
		password:     password,
		clientID:     options.ServiceAccountClientID,
		clientSecret: options.ServiceAccountClientSecret,
		realm:        realm,
	}, nil
}

func addTrustedCA(client *gocloak.GoCloak, fileName string) error {
	pemBytes, err := os.ReadFile(fileName)
	if err != nil {
		return fmt.Errorf("read Keycloak CA certificate %q: %w", fileName, err)
	}
	pool, err := x509.SystemCertPool()
	if err != nil || pool == nil {
		pool = x509.NewCertPool()
	}
	if !pool.AppendCertsFromPEM(pemBytes) {
		return fmt.Errorf("read Keycloak CA certificate %q: no PEM certificates found", fileName)
	}
	client.RestyClient().SetTLSClientConfig(&tls.Config{
		MinVersion: tls.VersionTLS12,
		RootCAs:    pool,
	})
	return nil
}

// token returns a cached admin access token, logging in again when the
// cached token is missing or about to expire.
func (a *Admin) token(ctx context.Context) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.accessToken != "" && time.Now().Add(refreshLead).Before(a.tokenExpires) {
		return a.accessToken, nil
	}
	var jwt *gocloak.JWT
	var err error
	if a.clientID != "" {
		jwt, err = a.client.LoginClient(ctx, a.clientID, a.clientSecret, a.realm)
	} else {
		jwt, err = a.client.LoginAdmin(ctx, a.username, a.password, a.realm)
	}
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

// UpdateClient updates the client described by rep (identified by rep.ID)
// and returns the updated representation. Only fields set on rep are
// changed.
func (a *Admin) UpdateClient(ctx context.Context, realm string, rep gocloak.Client) (*gocloak.Client, error) {
	if rep.ID == nil || *rep.ID == "" {
		return nil, fmt.Errorf("update client in realm %q: empty internal ID", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if err := a.client.UpdateClient(ctx, tok, realm, rep); err != nil {
		return nil, wrapErr(fmt.Sprintf("update client %q in realm %q", *rep.ID, realm), err)
	}
	return a.GetClient(ctx, realm, *rep.ID)
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

// ListClientScopes returns all client scopes in realm.
func (a *Admin) ListClientScopes(ctx context.Context, realm string) ([]*gocloak.ClientScope, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	scopes, err := a.client.GetClientScopes(ctx, tok, realm)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list client scopes in realm %q", realm), err)
	}
	return scopes, nil
}

// GetClientScope returns the client scope with the internal ID id.
func (a *Admin) GetClientScope(ctx context.Context, realm, id string) (*gocloak.ClientScope, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := a.client.GetClientScope(ctx, tok, realm, id)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get client scope %q in realm %q", id, realm), err)
	}
	return scope, nil
}

// CreateClientScope creates the client scope described by rep and returns the
// created representation.
func (a *Admin) CreateClientScope(ctx context.Context, realm string, rep gocloak.ClientScope) (*gocloak.ClientScope, error) {
	if rep.Name == nil || *rep.Name == "" {
		return nil, fmt.Errorf("create client scope in realm %q: empty name", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	id, err := a.client.CreateClientScope(ctx, tok, realm, rep)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("create client scope %q in realm %q", *rep.Name, realm), err)
	}
	return a.GetClientScope(ctx, realm, id)
}

// DeleteClientScope deletes the client scope with the internal ID id.
func (a *Admin) DeleteClientScope(ctx context.Context, realm, id string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteClientScope(ctx, tok, realm, id); err != nil {
		return wrapErr(fmt.Sprintf("delete client scope %q in realm %q", id, realm), err)
	}
	return nil
}

// AddDefaultScopeToClient adds the scope with scopeID to the client's default
// scopes.
func (a *Admin) AddDefaultScopeToClient(ctx context.Context, realm, clientID, scopeID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.AddDefaultScopeToClient(ctx, tok, realm, clientID, scopeID); err != nil {
		return wrapErr(fmt.Sprintf("add default client scope %q to client %q in realm %q", scopeID, clientID, realm), err)
	}
	return nil
}

// AddOptionalScopeToClient adds the scope with scopeID to the client's
// optional scopes.
func (a *Admin) AddOptionalScopeToClient(ctx context.Context, realm, clientID, scopeID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.AddOptionalScopeToClient(ctx, tok, realm, clientID, scopeID); err != nil {
		return wrapErr(fmt.Sprintf("add optional client scope %q to client %q in realm %q", scopeID, clientID, realm), err)
	}
	return nil
}

// RemoveDefaultScopeFromClient removes the scope with scopeID from the
// client's default scopes.
func (a *Admin) RemoveDefaultScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.RemoveDefaultScopeFromClient(ctx, tok, realm, clientID, scopeID); err != nil {
		return wrapErr(fmt.Sprintf("remove default client scope %q from client %q in realm %q", scopeID, clientID, realm), err)
	}
	return nil
}

// RemoveOptionalScopeFromClient removes the scope with scopeID from the
// client's optional scopes.
func (a *Admin) RemoveOptionalScopeFromClient(ctx context.Context, realm, clientID, scopeID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.RemoveOptionalScopeFromClient(ctx, tok, realm, clientID, scopeID); err != nil {
		return wrapErr(fmt.Sprintf("remove optional client scope %q from client %q in realm %q", scopeID, clientID, realm), err)
	}
	return nil
}

// ListEvents returns login and user events matching params. Results are
// fetched in bounded pages because gocloak's generic query serializer cannot
// encode the Type slice reliably.
func (a *Admin) ListEvents(ctx context.Context, realm string, params gocloak.GetEventsParams) ([]*gocloak.EventRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	max := eventResultLimit(params.Max)
	query := eventQuery(params)
	return a.fetchEventPages(ctx, tok, realm, max, query)
}

// ListAdminEvents returns administrative events matching params. Results are
// fetched in bounded pages because gocloak v14's generic query serializer
// cannot encode some numeric and slice admin-event parameters.
func (a *Admin) ListAdminEvents(ctx context.Context, realm string, params gocloak.GetAdminEventsParams) ([]*gocloak.AdminEventRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	max := eventResultLimit(params.Max)
	query := adminEventQuery(params)
	return a.fetchAdminEventPages(ctx, tok, realm, max, query)
}

const (
	eventPageSize = 100
	eventFetchCap = 10000
)

func eventResultLimit(value *int32) int {
	if value == nil || *value <= 0 {
		return eventPageSize
	}
	return int(*value)
}

func eventQuery(params gocloak.GetEventsParams) url.Values {
	query := url.Values{}
	addString(query, "client", params.Client)
	addString(query, "dateFrom", params.DateFrom)
	addString(query, "dateTo", params.DateTo)
	addString(query, "ipAddress", params.IPAddress)
	addString(query, "user", params.UserID)
	for _, value := range params.Type {
		query.Add("type", value)
	}
	return query
}

func adminEventQuery(params gocloak.GetAdminEventsParams) url.Values {
	query := url.Values{}
	addString(query, "authClient", params.AuthClient)
	addString(query, "authIpAddress", params.AuthIPAddress)
	addString(query, "authRealm", params.AuthRealm)
	addString(query, "authUser", params.AuthUser)
	addString(query, "dateFrom", params.DateFrom)
	addString(query, "dateTo", params.DateTo)
	addString(query, "resourcePath", params.ResourcePath)
	for _, value := range params.OperationTypes {
		query.Add("operationTypes", value)
	}
	for _, value := range params.ResourceTypes {
		query.Add("resourceTypes", value)
	}
	return query
}

func addString(query url.Values, key string, value *string) {
	if value != nil && *value != "" {
		query.Set(key, *value)
	}
}

func (a *Admin) fetchEventPages(ctx context.Context, token, realm string, max int, query url.Values) ([]*gocloak.EventRepresentation, error) {
	var events []*gocloak.EventRepresentation
	for first := 0; first < eventFetchCap && len(events) < max; first += eventPageSize {
		page := make([]*gocloak.EventRepresentation, 0, eventPageSize)
		pageQuery := cloneURLValues(query)
		requested := minInt(eventPageSize, max-len(events))
		pageQuery.Set("first", strconv.Itoa(first))
		pageQuery.Set("max", strconv.Itoa(requested))
		response, err := a.client.GetRequestWithBearerAuth(ctx, token).
			SetResult(&page).
			SetQueryParamsFromValues(pageQuery).
			Get(a.eventEndpoint(realm, "events"))
		if err != nil {
			return nil, wrapErr(fmt.Sprintf("list events in realm %q", realm), err)
		}
		if response.IsError() {
			return nil, fmt.Errorf("list events in realm %q: keycloak API error %d: %s", realm, response.StatusCode(), response.String())
		}
		events = append(events, page...)
		if len(events) >= max {
			return events[:max], nil
		}
		if len(page) < requested {
			break
		}
	}
	return events, nil
}

func (a *Admin) fetchAdminEventPages(ctx context.Context, token, realm string, max int, query url.Values) ([]*gocloak.AdminEventRepresentation, error) {
	var events []*gocloak.AdminEventRepresentation
	for first := 0; first < eventFetchCap && len(events) < max; first += eventPageSize {
		page := make([]*gocloak.AdminEventRepresentation, 0, eventPageSize)
		pageQuery := cloneURLValues(query)
		requested := minInt(eventPageSize, max-len(events))
		pageQuery.Set("first", strconv.Itoa(first))
		pageQuery.Set("max", strconv.Itoa(requested))
		response, err := a.client.GetRequestWithBearerAuth(ctx, token).
			SetResult(&page).
			SetQueryParamsFromValues(pageQuery).
			Get(a.eventEndpoint(realm, "admin-events"))
		if err != nil {
			return nil, wrapErr(fmt.Sprintf("list admin events in realm %q", realm), err)
		}
		if response.IsError() {
			return nil, fmt.Errorf("list admin events in realm %q: keycloak API error %d: %s", realm, response.StatusCode(), response.String())
		}
		events = append(events, page...)
		if len(events) >= max {
			return events[:max], nil
		}
		if len(page) < requested {
			break
		}
	}
	return events, nil
}

func cloneURLValues(source url.Values) url.Values {
	clone := make(url.Values, len(source))
	for key, values := range source {
		clone[key] = append([]string(nil), values...)
	}
	return clone
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func (a *Admin) eventEndpoint(realm, endpoint string) string {
	return a.baseURL + "/admin/realms/" + url.PathEscape(realm) + "/" + endpoint
}

// ListIdentityProviders returns all identity providers configured in realm.
func (a *Admin) ListIdentityProviders(ctx context.Context, realm string) ([]*gocloak.IdentityProviderRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	providers, err := a.client.GetIdentityProviders(ctx, tok, realm)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list identity providers in realm %q", realm), err)
	}
	return providers, nil
}

// GetIdentityProvider returns the provider with the given alias.
func (a *Admin) GetIdentityProvider(ctx context.Context, realm, alias string) (*gocloak.IdentityProviderRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	provider, err := a.client.GetIdentityProvider(ctx, tok, realm, alias)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get identity provider %q in realm %q", alias, realm), err)
	}
	return provider, nil
}

// CreateIdentityProvider creates an identity provider and returns the stored
// representation.
func (a *Admin) CreateIdentityProvider(ctx context.Context, realm string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
	if provider.Alias == nil || *provider.Alias == "" {
		return nil, fmt.Errorf("create identity provider in realm %q: empty alias", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	if _, err := a.client.CreateIdentityProvider(ctx, tok, realm, provider); err != nil {
		return nil, wrapErr(fmt.Sprintf("create identity provider %q in realm %q", *provider.Alias, realm), err)
	}
	return a.GetIdentityProvider(ctx, realm, *provider.Alias)
}

// UpdateIdentityProvider partially updates the provider with alias. Existing
// fields and config keys not present in provider are preserved.
func (a *Admin) UpdateIdentityProvider(ctx context.Context, realm, alias string, provider gocloak.IdentityProviderRepresentation) (*gocloak.IdentityProviderRepresentation, error) {
	if alias == "" {
		return nil, fmt.Errorf("update identity provider in realm %q: empty alias", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := a.client.GetIdentityProvider(ctx, tok, realm, alias)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get identity provider %q before update in realm %q", alias, realm), err)
	}
	merged := *existing
	if provider.DisplayName != nil {
		merged.DisplayName = provider.DisplayName
	}
	if provider.Enabled != nil {
		merged.Enabled = provider.Enabled
	}
	if provider.FirstBrokerLoginFlowAlias != nil {
		merged.FirstBrokerLoginFlowAlias = provider.FirstBrokerLoginFlowAlias
	}
	if provider.PostBrokerLoginFlowAlias != nil {
		merged.PostBrokerLoginFlowAlias = provider.PostBrokerLoginFlowAlias
	}
	if provider.LinkOnly != nil {
		merged.LinkOnly = provider.LinkOnly
	}
	if provider.StoreToken != nil {
		merged.StoreToken = provider.StoreToken
	}
	if provider.TrustEmail != nil {
		merged.TrustEmail = provider.TrustEmail
	}
	if provider.HideOnLogin != nil {
		merged.HideOnLogin = provider.HideOnLogin
	}
	if provider.AddReadTokenRoleOnCreate != nil {
		merged.AddReadTokenRoleOnCreate = provider.AddReadTokenRoleOnCreate
	}
	if provider.Config != nil {
		merged.Config = cloneStringMap(existing.Config)
		for key, value := range provider.Config {
			merged.Config[key] = value
		}
	}
	if err := a.client.UpdateIdentityProvider(ctx, tok, realm, alias, merged); err != nil {
		return nil, wrapErr(fmt.Sprintf("update identity provider %q in realm %q", alias, realm), err)
	}
	return a.GetIdentityProvider(ctx, realm, alias)
}

// DeleteIdentityProvider deletes the provider with alias.
func (a *Admin) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteIdentityProvider(ctx, tok, realm, alias); err != nil {
		return wrapErr(fmt.Sprintf("delete identity provider %q in realm %q", alias, realm), err)
	}
	return nil
}

func cloneStringMap(source map[string]string) map[string]string {
	clone := make(map[string]string, len(source))
	for key, value := range source {
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
		if strings.Contains(normalized, "secret") || strings.Contains(normalized, "password") {
			// Keycloak returns secrets as ********** and expects that masked
			// value to preserve the existing secret during an update.
			if value != "" {
				clone[key] = value
			}
			continue
		}
		clone[key] = value
	}
	return clone
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

// AddRealmRolesToUser assigns the named realm roles to the user with the
// internal ID userID. Names are resolved to roles first, so an unknown name
// fails before any mapping is changed.
func (a *Admin) AddRealmRolesToUser(ctx context.Context, realm, userID string, roleNames []string) error {
	roles, err := a.resolveRealmRoles(ctx, realm, roleNames)
	if err != nil {
		return err
	}
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.AddRealmRoleToUser(ctx, tok, realm, userID, roles); err != nil {
		return wrapErr(fmt.Sprintf("add realm roles to user %q in realm %q", userID, realm), err)
	}
	return nil
}

// RemoveRealmRolesFromUser removes the named realm roles from the user with
// the internal ID userID. Names are resolved to roles first, so an unknown
// name fails before any mapping is changed.
func (a *Admin) RemoveRealmRolesFromUser(ctx context.Context, realm, userID string, roleNames []string) error {
	roles, err := a.resolveRealmRoles(ctx, realm, roleNames)
	if err != nil {
		return err
	}
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteRealmRoleFromUser(ctx, tok, realm, userID, roles); err != nil {
		return wrapErr(fmt.Sprintf("remove realm roles from user %q in realm %q", userID, realm), err)
	}
	return nil
}

// resolveRealmRoles looks up full role representations for the given names.
// Keycloak's role-mapping endpoints look up roles by name and then verify the
// ID, so the request body needs the full representation; name-only or
// id-only representations are rejected with 404s.
func (a *Admin) resolveRealmRoles(ctx context.Context, realm string, roleNames []string) ([]gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	roles := make([]gocloak.Role, 0, len(roleNames))
	for _, name := range roleNames {
		role, err := a.client.GetRealmRole(ctx, tok, realm, name)
		if err != nil {
			return nil, wrapErr(fmt.Sprintf("look up realm role %q in realm %q", name, realm), err)
		}
		roles = append(roles, *role)
	}
	return roles, nil
}

// AddUserToGroup adds the user with the internal ID userID to the group with
// the internal ID groupID.
func (a *Admin) AddUserToGroup(ctx context.Context, realm, userID, groupID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.AddUserToGroup(ctx, tok, realm, userID, groupID); err != nil {
		return wrapErr(fmt.Sprintf("add user %q to group %q in realm %q", userID, groupID, realm), err)
	}
	return nil
}

// RemoveUserFromGroup removes the user with the internal ID userID from the
// group with the internal ID groupID.
func (a *Admin) RemoveUserFromGroup(ctx context.Context, realm, userID, groupID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.DeleteUserFromGroup(ctx, tok, realm, userID, groupID); err != nil {
		return wrapErr(fmt.Sprintf("remove user %q from group %q in realm %q", userID, groupID, realm), err)
	}
	return nil
}

// ListUserSessions returns the active sessions of the user with the internal
// ID userID.
func (a *Admin) ListUserSessions(ctx context.Context, realm, userID string) ([]*gocloak.UserSessionRepresentation, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	sessions, err := a.client.GetUserSessions(ctx, tok, realm, userID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list sessions of user %q in realm %q", userID, realm), err)
	}
	return sessions, nil
}

// LogoutAllUserSessions ends all active sessions of the user with the
// internal ID userID.
func (a *Admin) LogoutAllUserSessions(ctx context.Context, realm, userID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.LogoutAllSessions(ctx, tok, realm, userID); err != nil {
		return wrapErr(fmt.Sprintf("log out all sessions of user %q in realm %q", userID, realm), err)
	}
	return nil
}

// LogoutUserSession ends the session with the internal ID sessionID.
func (a *Admin) LogoutUserSession(ctx context.Context, realm, sessionID string) error {
	tok, err := a.token(ctx)
	if err != nil {
		return err
	}
	if err := a.client.LogoutUserSession(ctx, tok, realm, sessionID); err != nil {
		return wrapErr(fmt.Sprintf("log out session %q in realm %q", sessionID, realm), err)
	}
	return nil
}

// ListUserGroups returns the groups of the user with the internal ID userID,
// including parents of member subgroups. max <= 0 means no explicit limit.
func (a *Admin) ListUserGroups(ctx context.Context, realm, userID string, max int) ([]*gocloak.Group, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetGroupsParams{BriefRepresentation: gocloak.BoolP(true)}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	groups, err := a.client.GetUserGroups(ctx, tok, realm, userID, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list groups of user %q in realm %q", userID, realm), err)
	}
	return groups, nil
}

// GetUserRealmRoles returns the realm roles directly assigned to the user
// with the internal ID userID.
func (a *Admin) GetUserRealmRoles(ctx context.Context, realm, userID string) ([]*gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := a.client.GetRealmRolesByUserID(ctx, tok, realm, userID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get realm roles of user %q in realm %q", userID, realm), err)
	}
	return roles, nil
}

// GetCompositeUserRealmRoles returns the effective realm roles of the user
// with the internal ID userID, including composite expansion.
func (a *Admin) GetCompositeUserRealmRoles(ctx context.Context, realm, userID string) ([]*gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := a.client.GetCompositeRealmRolesByUserID(ctx, tok, realm, userID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get effective realm roles of user %q in realm %q", userID, realm), err)
	}
	return roles, nil
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

// ListGroupMembers returns the users in the group with the internal ID
// groupID. max <= 0 means no explicit limit.
func (a *Admin) ListGroupMembers(ctx context.Context, realm, groupID string, max int) ([]*gocloak.User, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetGroupsParams{BriefRepresentation: gocloak.BoolP(true)}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	users, err := a.client.GetGroupMembers(ctx, tok, realm, groupID, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list members of group %q in realm %q", groupID, realm), err)
	}
	return users, nil
}

// GetGroupRealmRoles returns the realm roles directly assigned to the group
// with the internal ID groupID.
func (a *Admin) GetGroupRealmRoles(ctx context.Context, realm, groupID string) ([]*gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := a.client.GetRealmRolesByGroupID(ctx, tok, realm, groupID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get realm roles of group %q in realm %q", groupID, realm), err)
	}
	return roles, nil
}

// GetCompositeGroupRealmRoles returns the effective realm roles of the group
// with the internal ID groupID, including composite expansion.
func (a *Admin) GetCompositeGroupRealmRoles(ctx context.Context, realm, groupID string) ([]*gocloak.Role, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	roles, err := a.client.GetCompositeRealmRolesByGroupID(ctx, tok, realm, groupID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get effective realm roles of group %q in realm %q", groupID, realm), err)
	}
	return roles, nil
}

// GetGroup returns the group with the internal ID groupID.
func (a *Admin) GetGroup(ctx context.Context, realm, groupID string) (*gocloak.Group, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	group, err := a.client.GetGroup(ctx, tok, realm, groupID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get group %q in realm %q", groupID, realm), err)
	}
	return group, nil
}

// GetGroupByPath returns the group at path, such as "parent/child". A
// leading slash is accepted and removed.
func (a *Admin) GetGroupByPath(ctx context.Context, realm, path string) (*gocloak.Group, error) {
	trimmed := strings.TrimPrefix(strings.TrimSpace(path), "/")
	if trimmed == "" {
		return nil, fmt.Errorf("get group by path in realm %q: empty path", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	group, err := a.client.GetGroupByPath(ctx, tok, realm, trimmed)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get group by path %q in realm %q", trimmed, realm), err)
	}
	return group, nil
}

// UpdateGroup partially updates the group with the internal ID rep.ID and
// returns the updated representation. Name and attributes are changed only
// when set on rep; other fields are preserved. Keycloak requires a name on
// every group update, so the existing name is kept when rep.Name is nil.
func (a *Admin) UpdateGroup(ctx context.Context, realm string, rep gocloak.Group) (*gocloak.Group, error) {
	if rep.ID == nil || *rep.ID == "" {
		return nil, fmt.Errorf("update group in realm %q: empty groupId", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	existing, err := a.client.GetGroup(ctx, tok, realm, *rep.ID)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get group %q before update in realm %q", *rep.ID, realm), err)
	}
	merged := *existing
	merged.SubGroups = nil
	if rep.Name != nil {
		merged.Name = rep.Name
	}
	if rep.Attributes != nil {
		merged.Attributes = rep.Attributes
	}
	if err := a.client.UpdateGroup(ctx, tok, realm, merged); err != nil {
		return nil, wrapErr(fmt.Sprintf("update group %q in realm %q", *rep.ID, realm), err)
	}
	return a.GetGroup(ctx, realm, *rep.ID)
}

// ListChildGroups returns the direct child groups of the group with the
// internal ID groupID. max <= 0 means no explicit limit.
func (a *Admin) ListChildGroups(ctx context.Context, realm, groupID string, max int) ([]*gocloak.Group, error) {
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	params := gocloak.GetChildGroupsParams{BriefRepresentation: gocloak.BoolP(true)}
	if max > 0 {
		params.Max = gocloak.IntP(max)
	}
	children, err := a.client.GetChildGroups(ctx, tok, realm, groupID, params)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("list child groups of group %q in realm %q", groupID, realm), err)
	}
	return children, nil
}

// CreateChildGroup creates the named subgroup under the group with the
// internal ID parentID and returns the created representation.
func (a *Admin) CreateChildGroup(ctx context.Context, realm, parentID, name string) (*gocloak.Group, error) {
	if name == "" {
		return nil, fmt.Errorf("create child group in realm %q: empty name", realm)
	}
	tok, err := a.token(ctx)
	if err != nil {
		return nil, err
	}
	id, err := a.client.CreateChildGroup(ctx, tok, realm, parentID, gocloak.Group{Name: gocloak.StringP(name)})
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("create child group %q under group %q in realm %q", name, parentID, realm), err)
	}
	group, err := a.client.GetGroup(ctx, tok, realm, id)
	if err != nil {
		return nil, wrapErr(fmt.Sprintf("get created child group %q in realm %q", name, realm), err)
	}
	return group, nil
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
