package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listRealmsInput struct{}

type realmRefInput struct {
	Realm string `json:"realm" jsonschema:"realm name, e.g. \"master\""`
}

type createRealmInput struct {
	Realm       string `json:"realm" jsonschema:"name of the realm to create; unique identifier, lowercase with no spaces, e.g. \"acme-prod\""`
	DisplayName string `json:"displayName,omitempty" jsonschema:"human-friendly display name shown in the Keycloak console"`
	Enabled     *bool  `json:"enabled,omitempty" jsonschema:"whether the realm is enabled; default true"`
}

type updateRealmInput struct {
	Realm       string  `json:"realm" jsonschema:"name of the realm to update"`
	DisplayName *string `json:"displayName,omitempty" jsonschema:"new display name; omit to leave unchanged"`
	Enabled     *bool   `json:"enabled,omitempty" jsonschema:"whether the realm is enabled; omit to leave unchanged"`
}

func addRealmTools(s *mcp.Server, admin AdminAPI, options Options) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_list",
		Title:       "List realms",
		Description: "List all realms on the Keycloak server.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, _ listRealmsInput) (*mcp.CallToolResult, any, error) {
		realms, err := admin.ListRealms(ctx)
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(realms), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_get",
		Title:       "Get realm",
		Description: "Get a realm's settings by name.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in realmRefInput) (*mcp.CallToolResult, any, error) {
		realm, err := admin.GetRealm(ctx, in.Realm)
		if err != nil {
			return nil, nil, err
		}
		return nil, realm, nil
	})
	if options.ReadOnly {
		return
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_create",
		Title:       "Create realm",
		Description: "Create a new realm. The realm name is the unique identifier used in URLs and API calls; prefer lowercase with dashes. Returns the created realm.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createRealmInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.RealmRepresentation{
			Realm:   gocloak.StringP(in.Realm),
			Enabled: gocloak.BoolP(in.Enabled == nil || *in.Enabled),
		}
		if in.DisplayName != "" {
			rep.DisplayName = gocloak.StringP(in.DisplayName)
		}
		created, err := admin.CreateRealm(ctx, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_update",
		Title:       "Update realm",
		Description: "Update a realm's settings. Only the provided fields are changed. Returns the updated realm.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateRealmInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.RealmRepresentation{
			Realm:       gocloak.StringP(in.Realm),
			DisplayName: in.DisplayName,
			Enabled:     in.Enabled,
		}
		updated, err := admin.UpdateRealm(ctx, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "realm_delete",
		Title:       "Delete realm",
		Description: "Permanently delete a realm and everything in it (clients, users, groups, roles). This cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in realmRefInput) (*mcp.CallToolResult, any, error) {
		if err := admin.DeleteRealm(ctx, in.Realm); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "deleted": true}, nil
	})
}
