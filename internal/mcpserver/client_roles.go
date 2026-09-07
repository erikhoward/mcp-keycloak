package mcpserver

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type clientRoleInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	ClientID string `json:"clientId" jsonschema:"public client identifier"`
	Max      int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type createClientRoleInput struct {
	Realm       string `json:"realm" jsonschema:"realm name"`
	ClientID    string `json:"clientId" jsonschema:"public client identifier"`
	Name        string `json:"name" jsonschema:"unique client role name"`
	Description string `json:"description,omitempty"`
}

type clientRoleRefInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	ClientID string `json:"clientId" jsonschema:"public client identifier"`
	Name     string `json:"name" jsonschema:"client role name"`
}

type userClientRolesInput struct {
	Realm    string   `json:"realm" jsonschema:"realm name"`
	ClientID string   `json:"clientId" jsonschema:"public client identifier"`
	UserID   string   `json:"userId" jsonschema:"internal user ID (UUID)"`
	Roles    []string `json:"roles" jsonschema:"client role names"`
}

type userClientRoleListInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	ClientID string `json:"clientId" jsonschema:"public client identifier"`
	UserID   string `json:"userId" jsonschema:"internal user ID (UUID)"`
}

func addClientRoleTools(s *mcp.Server, admin AdminAPI, options Options) {
	mcp.AddTool(s, &mcp.Tool{Name: "client_role_list", Title: "List client roles", Description: "List roles belonging to a client.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientRoleInput) (*mcp.CallToolResult, any, error) {
		roles, err := admin.ListClientRoles(ctx, in.Realm, in.ClientID, resolveMax(in.Max))
		return nil, nonNil(roles), err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "user_client_roles_list", Title: "List user client roles", Description: "List client roles directly assigned to a user.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in userClientRoleListInput) (*mcp.CallToolResult, any, error) {
		roles, err := admin.GetUserClientRoles(ctx, in.Realm, in.ClientID, in.UserID)
		return nil, nonNil(roles), err
	})
	if options.ReadOnly {
		return
	}
	mcp.AddTool(s, &mcp.Tool{Name: "client_role_create", Title: "Create client role", Description: "Create a role belonging to a client.", Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)}}, func(ctx context.Context, _ *mcp.CallToolRequest, in createClientRoleInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.Role{Name: gocloak.StringP(in.Name)}
		if in.Description != "" {
			rep.Description = gocloak.StringP(in.Description)
		}
		role, err := admin.CreateClientRole(ctx, in.Realm, in.ClientID, rep)
		return nil, role, err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "client_role_delete", Title: "Delete client role", Description: "Delete a client role by name.", Annotations: &mcp.ToolAnnotations{IdempotentHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in clientRoleRefInput) (*mcp.CallToolResult, any, error) {
		err := admin.DeleteClientRole(ctx, in.Realm, in.ClientID, in.Name)
		return nil, map[string]any{"clientId": in.ClientID, "name": in.Name, "deleted": true}, err
	})
	assign := func(remove bool) func(context.Context, *mcp.CallToolRequest, userClientRolesInput) (*mcp.CallToolResult, any, error) {
		return func(ctx context.Context, _ *mcp.CallToolRequest, in userClientRolesInput) (*mcp.CallToolResult, any, error) {
			if len(in.Roles) == 0 {
				return nil, nil, fmt.Errorf("roles must contain at least one role name")
			}
			var err error
			if remove {
				err = admin.RemoveClientRolesFromUser(ctx, in.Realm, in.ClientID, in.UserID, in.Roles)
			} else {
				err = admin.AddClientRolesToUser(ctx, in.Realm, in.ClientID, in.UserID, in.Roles)
			}
			return nil, map[string]any{"clientId": in.ClientID, "userId": in.UserID, "roles": in.Roles}, err
		}
	}
	mcp.AddTool(s, &mcp.Tool{Name: "user_add_client_role", Title: "Assign client roles to user", Description: "Assign named client roles to a user.", Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)}}, assign(false))
	mcp.AddTool(s, &mcp.Tool{Name: "user_remove_client_role", Title: "Remove client roles from user", Description: "Remove named client roles from a user.", Annotations: &mcp.ToolAnnotations{IdempotentHint: true}}, assign(true))
}
