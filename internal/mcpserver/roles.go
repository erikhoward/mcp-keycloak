package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listRealmRolesInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
	Max   int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type createRealmRoleInput struct {
	Realm       string `json:"realm" jsonschema:"realm to create the role in"`
	Name        string `json:"name" jsonschema:"unique role name"`
	Description string `json:"description,omitempty"`
}

type realmRoleRefInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
	Name  string `json:"name" jsonschema:"role name"`
}

func addRealmRoleTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_role_list",
		Title:       "List realm roles",
		Description: "List the realm-level roles of a realm (excluding built-in ones only if the server hides them).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listRealmRolesInput) (*mcp.CallToolResult, any, error) {
		roles, err := admin.ListRealmRoles(ctx, in.Realm, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(roles), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_role_create",
		Title:       "Create realm role",
		Description: "Create a realm-level role. Returns the created role.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createRealmRoleInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.Role{Name: gocloak.StringP(in.Name)}
		if in.Description != "" {
			rep.Description = gocloak.StringP(in.Description)
		}
		created, err := admin.CreateRealmRole(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_role_delete",
		Title:       "Delete realm role",
		Description: "Delete a realm-level role by name. This cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in realmRoleRefInput) (*mcp.CallToolResult, any, error) {
		if err := admin.DeleteRealmRole(ctx, in.Realm, in.Name); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "name": in.Name, "deleted": true}, nil
	})
}
