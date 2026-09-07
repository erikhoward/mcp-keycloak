package mcpserver

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listRealmRolesInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
	First int    `json:"first,omitempty" jsonschema:"zero-based index of the first result; default 0"`
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

type updateRealmRoleInput struct {
	Realm       string              `json:"realm" jsonschema:"realm name"`
	Name        string              `json:"name" jsonschema:"current role name"`
	NewName     *string             `json:"newName,omitempty" jsonschema:"new role name; omit to leave unchanged"`
	Description *string             `json:"description,omitempty" jsonschema:"new description; omit to leave unchanged"`
	Attributes  map[string][]string `json:"attributes,omitempty" jsonschema:"replacement role attributes; omit to leave unchanged"`
}

type realmRoleCompositesInput struct {
	Realm string   `json:"realm" jsonschema:"realm name"`
	Name  string   `json:"name" jsonschema:"composite parent role name"`
	Roles []string `json:"roles" jsonschema:"child realm role names"`
}

func addRealmRoleTools(s *mcp.Server, admin AdminAPI, options Options) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_role_list",
		Title:       "List realm roles",
		Description: "List the realm-level roles of a realm (excluding built-in ones only if the server hides them).",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listRealmRolesInput) (*mcp.CallToolResult, any, error) {
		if err := validateFirst(in.First); err != nil {
			return nil, nil, err
		}
		roles, err := admin.ListRealmRoles(ctx, in.Realm, in.First, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(roles), nil
	})
	mcp.AddTool(s, &mcp.Tool{
		Name: "realm_role_get", Title: "Get realm role",
		Description: "Get a realm-level role by name.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in realmRoleRefInput) (*mcp.CallToolResult, any, error) {
		role, err := admin.GetRealmRole(ctx, in.Realm, in.Name)
		return nil, role, err
	})
	if options.ReadOnly {
		return
	}

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
		Name: "realm_role_update", Title: "Update realm role",
		Description: "Rename a realm role or replace its description or attributes.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateRealmRoleInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.Role{Name: in.NewName, Description: in.Description, Attributes: in.Attributes}
		role, err := admin.UpdateRealmRole(ctx, in.Realm, in.Name, rep)
		return nil, role, err
	})

	addComposite := func(remove bool) func(context.Context, *mcp.CallToolRequest, realmRoleCompositesInput) (*mcp.CallToolResult, any, error) {
		return func(ctx context.Context, _ *mcp.CallToolRequest, in realmRoleCompositesInput) (*mcp.CallToolResult, any, error) {
			if len(in.Roles) == 0 {
				return nil, nil, fmt.Errorf("roles must contain at least one role name")
			}
			var err error
			if remove {
				err = admin.RemoveRealmRoleComposites(ctx, in.Realm, in.Name, in.Roles)
			} else {
				err = admin.AddRealmRoleComposites(ctx, in.Realm, in.Name, in.Roles)
			}
			return nil, map[string]any{"realm": in.Realm, "name": in.Name, "roles": in.Roles}, err
		}
	}
	mcp.AddTool(s, &mcp.Tool{Name: "realm_role_composite_add", Title: "Add realm role composites", Description: "Attach child realm roles to a composite role.", Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)}}, addComposite(false))
	mcp.AddTool(s, &mcp.Tool{Name: "realm_role_composite_remove", Title: "Remove realm role composites", Description: "Detach child realm roles from a composite role.", Annotations: &mcp.ToolAnnotations{IdempotentHint: true}}, addComposite(true))

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
