package mcpserver

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listClientScopesInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
}

type clientScopeRefInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
	Name  string `json:"name" jsonschema:"client scope name"`
}

type createClientScopeInput struct {
	Realm       string `json:"realm" jsonschema:"realm to create the client scope in"`
	Name        string `json:"name" jsonschema:"unique client scope name, e.g. \"orders\""`
	Description string `json:"description,omitempty" jsonschema:"description of the claims or permissions represented by this scope"`
	Protocol    string `json:"protocol,omitempty" jsonschema:"protocol for the scope; defaults to \"openid-connect\""`
}

type clientScopeAssignmentInput struct {
	Realm     string `json:"realm" jsonschema:"realm name"`
	ClientID  string `json:"clientId" jsonschema:"client identifier, not the internal UUID"`
	ScopeName string `json:"scopeName" jsonschema:"client scope name"`
	Optional  bool   `json:"optional,omitempty" jsonschema:"when true, add or remove the scope as optional; default false means default scope"`
}

func resolveClientScope(ctx context.Context, admin AdminAPI, realm, name string) (*gocloak.ClientScope, error) {
	scopes, err := admin.ListClientScopes(ctx, realm)
	if err != nil {
		return nil, err
	}
	for _, scope := range scopes {
		if deref(scope.Name) == name {
			return scope, nil
		}
	}
	return nil, fmt.Errorf("client scope %q not found in realm %q", name, realm)
}

func addClientScopeTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_list",
		Title:       "List client scopes",
		Description: "List all client scopes in a realm, including built-in scopes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listClientScopesInput) (*mcp.CallToolResult, any, error) {
		scopes, err := admin.ListClientScopes(ctx, in.Realm)
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(scopes), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_get",
		Title:       "Get client scope",
		Description: "Get a client scope by name, including its protocol and protocol mappers.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientScopeRefInput) (*mcp.CallToolResult, any, error) {
		scope, err := resolveClientScope(ctx, admin, in.Realm, in.Name)
		if err != nil {
			return nil, nil, err
		}
		full, err := admin.GetClientScope(ctx, in.Realm, deref(scope.ID))
		if err != nil {
			return nil, nil, err
		}
		return nil, full, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_create",
		Title:       "Create client scope",
		Description: "Create a client scope in a realm. The scope can then be assigned to clients as a default or optional scope.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createClientScopeInput) (*mcp.CallToolResult, any, error) {
		protocol := in.Protocol
		if protocol == "" {
			protocol = "openid-connect"
		}
		scope := gocloak.ClientScope{
			Name:     gocloak.StringP(in.Name),
			Protocol: gocloak.StringP(protocol),
		}
		if in.Description != "" {
			scope.Description = gocloak.StringP(in.Description)
		}
		created, err := admin.CreateClientScope(ctx, in.Realm, scope)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_delete",
		Title:       "Delete client scope",
		Description: "Delete a client scope by name. Built-in scopes may be protected by Keycloak, and deleting a custom scope removes it from every client.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientScopeRefInput) (*mcp.CallToolResult, any, error) {
		scope, err := resolveClientScope(ctx, admin, in.Realm, in.Name)
		if err != nil {
			return nil, nil, err
		}
		if err := admin.DeleteClientScope(ctx, in.Realm, deref(scope.ID)); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "name": in.Name, "deleted": true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_assign",
		Title:       "Assign client scope",
		Description: "Assign a client scope to a client by client identifier. By default it becomes a default scope; set optional=true to make it optional.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientScopeAssignmentInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		scope, err := resolveClientScope(ctx, admin, in.Realm, in.ScopeName)
		if err != nil {
			return nil, nil, err
		}
		clientID, scopeID := deref(client.ID), deref(scope.ID)
		if clientID == "" || scopeID == "" {
			return nil, nil, fmt.Errorf("client or client scope has no internal ID")
		}
		if in.Optional {
			err = admin.AddOptionalScopeToClient(ctx, in.Realm, clientID, scopeID)
		} else {
			err = admin.AddDefaultScopeToClient(ctx, in.Realm, clientID, scopeID)
		}
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"realm": in.Realm, "clientId": in.ClientID, "scopeName": in.ScopeName,
			"optional": in.Optional, "assigned": true,
		}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "client_scope_unassign",
		Title:       "Unassign client scope",
		Description: "Remove a client scope from a client by client identifier. Set optional=true to remove it from optional scopes; default removes a default scope.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientScopeAssignmentInput) (*mcp.CallToolResult, any, error) {
		client, err := resolveClient(ctx, admin, in.Realm, in.ClientID)
		if err != nil {
			return nil, nil, err
		}
		scope, err := resolveClientScope(ctx, admin, in.Realm, in.ScopeName)
		if err != nil {
			return nil, nil, err
		}
		clientID, scopeID := deref(client.ID), deref(scope.ID)
		if clientID == "" || scopeID == "" {
			return nil, nil, fmt.Errorf("client or client scope has no internal ID")
		}
		if in.Optional {
			err = admin.RemoveOptionalScopeFromClient(ctx, in.Realm, clientID, scopeID)
		} else {
			err = admin.RemoveDefaultScopeFromClient(ctx, in.Realm, clientID, scopeID)
		}
		if err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{
			"realm": in.Realm, "clientId": in.ClientID, "scopeName": in.ScopeName,
			"optional": in.Optional, "unassigned": true,
		}, nil
	})
}
