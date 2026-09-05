package mcpserver

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listClientsInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	ClientID string `json:"clientId,omitempty" jsonschema:"filter by client identifier, e.g. \"account\""`
	Max      int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type clientRefInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	ClientID string `json:"clientId" jsonschema:"client identifier (not the internal UUID), e.g. \"my-app\""`
}

type clientSecretGetInput struct {
	Realm         string `json:"realm" jsonschema:"realm name"`
	ClientID      string `json:"clientId" jsonschema:"client identifier (not the internal UUID)"`
	IncludeSecret bool   `json:"includeSecret,omitempty" jsonschema:"explicitly include the secret in structured output; default false because MCP transcripts may be retained"`
}

type clientSecretOutput struct {
	Realm           string `json:"realm"`
	ClientID        string `json:"clientId"`
	SecretAvailable bool   `json:"secretAvailable"`
	Secret          string `json:"secret,omitempty"`
}

type createClientInput struct {
	Realm                     string   `json:"realm" jsonschema:"realm to create the client in"`
	ClientID                  string   `json:"clientId" jsonschema:"unique client identifier used in OIDC requests, e.g. \"my-app\""`
	Name                      string   `json:"name,omitempty" jsonschema:"display name"`
	Description               string   `json:"description,omitempty"`
	Public                    *bool    `json:"public,omitempty" jsonschema:"true for a public client (no secret); default false, a confidential client whose secret can be fetched with client_secret_get"`
	RedirectURIs              []string `json:"redirectURIs,omitempty" jsonschema:"allowed redirect URIs for the browser login flow, e.g. [\"http://localhost:8080/callback\"]"`
	DirectAccessGrantsEnabled *bool    `json:"directAccessGrantsEnabled,omitempty" jsonschema:"allow the password grant (direct access grants); default false"`
	ServiceAccountsEnabled    *bool    `json:"serviceAccountsEnabled,omitempty" jsonschema:"enable service accounts for the client credentials grant; default false"`
}

// resolveClient finds a client by its clientId (the human-facing identifier)
// within a realm.
func resolveClient(ctx context.Context, admin AdminAPI, realm, clientID string) (*gocloak.Client, error) {
	clients, err := admin.ListClients(ctx, realm, clientID, 0)
	if err != nil {
		return nil, err
	}
	for _, c := range clients {
		if deref(c.ClientID) == clientID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("client %q not found in realm %q", clientID, realm)
}

type updateClientInput struct {
	Realm                     string   `json:"realm" jsonschema:"realm name"`
	ClientID                  string   `json:"clientId" jsonschema:"client identifier (not the internal UUID), e.g. \"my-app\""`
	Name                      *string  `json:"name,omitempty" jsonschema:"new display name; omit to leave unchanged"`
	Description               *string  `json:"description,omitempty" jsonschema:"new description; omit to leave unchanged"`
	Public                    *bool    `json:"public,omitempty" jsonschema:"whether the client is public (no secret); omit to leave unchanged"`
	RedirectURIs              []string `json:"redirectURIs,omitempty" jsonschema:"allowed redirect URIs for the browser login flow; omit to leave unchanged, set [] to clear"`
	DirectAccessGrantsEnabled *bool    `json:"directAccessGrantsEnabled,omitempty" jsonschema:"allow the password grant (direct access grants); omit to leave unchanged"`
	ServiceAccountsEnabled    *bool    `json:"serviceAccountsEnabled,omitempty" jsonschema:"enable service accounts for the client credentials grant; omit to leave unchanged"`
}

func addClientTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_list",
		Title:       "List clients",
		Description: "List the clients of a realm, optionally filtered by client identifier.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listClientsInput) (*mcp.CallToolResult, any, error) {
		clients, err := admin.ListClients(ctx, in.Realm, in.ClientID, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(clients), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_get",
		Title:       "Get client",
		Description: "Get a client by its client identifier, including its internal ID (needed by other Keycloak APIs).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientRefInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		return nil, client, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_create",
		Title:       "Create client",
		Description: "Create a client in a realm. Confidential clients (the default) get an auto-generated secret, retrievable with client_secret_get. The standard login flow is enabled automatically when redirectURIs are provided. Returns the created client.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createClientInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.Client{
			ClientID:                  gocloak.StringP(in.ClientID),
			PublicClient:              gocloak.BoolP(in.Public != nil && *in.Public),
			StandardFlowEnabled:       gocloak.BoolP(len(in.RedirectURIs) > 0),
			RedirectURIs:              in.RedirectURIs,
			DirectAccessGrantsEnabled: gocloak.BoolP(in.DirectAccessGrantsEnabled != nil && *in.DirectAccessGrantsEnabled),
			ServiceAccountsEnabled:    gocloak.BoolP(in.ServiceAccountsEnabled != nil && *in.ServiceAccountsEnabled),
		}
		if in.Name != "" {
			rep.Name = gocloak.StringP(in.Name)
		}
		if in.Description != "" {
			rep.Description = gocloak.StringP(in.Description)
		}
		created, err := admin.CreateClient(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_update",
		Title:       "Update client",
		Description: "Update a client by its client identifier. Only the provided fields are changed. Returns the updated client.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateClientInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		rep := gocloak.Client{
			ID:                        client.ID,
			Name:                      in.Name,
			Description:               in.Description,
			PublicClient:              in.Public,
			DirectAccessGrantsEnabled: in.DirectAccessGrantsEnabled,
			ServiceAccountsEnabled:    in.ServiceAccountsEnabled,
		}
		if in.RedirectURIs != nil {
			rep.RedirectURIs = in.RedirectURIs
		}
		updated, err := admin.UpdateClient(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_secret_get",
		Title:       "Get client secret",
		Description: "Inspect whether a confidential client has a secret. The secret is omitted by default; set includeSecret=true only when explicitly needed because MCP clients may retain tool transcripts.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientSecretGetInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		secretAvailable := client.PublicClient == nil || !*client.PublicClient
		output := clientSecretOutput{
			Realm:           in.Realm,
			ClientID:        in.ClientID,
			SecretAvailable: secretAvailable,
		}
		message := "Client secret was not returned. Set includeSecret=true only when the secret is explicitly needed and can be handled securely."
		if in.IncludeSecret {
			secret, err := admin.GetClientSecret(ctx, in.Realm, deref(client.ID))
			if err != nil {
				return nil, nil, err
			}
			output.Secret = deref(secret.Value)
			output.SecretAvailable = output.Secret != ""
			message = "Client secret retrieved in structured output; treat it as sensitive and do not repeat or store it."
		}
		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: message}}}, output, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_delete",
		Title:       "Delete client",
		Description: "Delete a client by its client identifier. This cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientRefInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		if err := admin.DeleteClient(ctx, in.Realm, deref(client.ID)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "clientId": in.ClientID, "deleted": true}, nil
	})
}
