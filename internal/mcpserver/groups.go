package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listGroupsInput struct {
	Realm  string `json:"realm" jsonschema:"realm name"`
	Search string `json:"search,omitempty" jsonschema:"substring matched against group names and paths"`
	Max    int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type createGroupInput struct {
	Realm string `json:"realm" jsonschema:"realm to create the group in"`
	Name  string `json:"name" jsonschema:"unique group name"`
}

type groupRefInput struct {
	Realm   string `json:"realm" jsonschema:"realm name"`
	GroupID string `json:"groupId" jsonschema:"internal group ID (UUID) as returned by group_list or group_create"`
}

type listGroupMembersInput struct {
	Realm   string `json:"realm" jsonschema:"realm name"`
	GroupID string `json:"groupId" jsonschema:"internal group ID (UUID) as returned by group_list or group_create"`
	Max     int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

func addGroupTools(s *mcp.Server, admin AdminAPI, options Options) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_list",
		Title:       "List groups",
		Description: "List the top-level groups of a realm, optionally filtered by a search substring. Returns internal group IDs needed by other group tools.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listGroupsInput) (*mcp.CallToolResult, any, error) {
		groups, err := admin.ListGroups(ctx, in.Realm, in.Search, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(groups), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_members_list",
		Title:       "List group members",
		Description: "List the users in a group by internal group ID. Find the group ID with group_list.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listGroupMembersInput) (*mcp.CallToolResult, any, error) {
		users, err := admin.ListGroupMembers(ctx, in.Realm, in.GroupID, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(users), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_roles_list",
		Title:       "List group realm roles",
		Description: "List the realm roles of a group by internal group ID. Returns directly assigned roles and the effective set after composite expansion. Find the group ID with group_list.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in groupRefInput) (*mcp.CallToolResult, any, error) {
		direct, err := admin.GetGroupRealmRoles(ctx, in.Realm, in.GroupID)
		if err != nil {
			return nil, nil, err
		}
		effective, err := admin.GetCompositeGroupRealmRoles(ctx, in.Realm, in.GroupID)
		if err != nil {
			return nil, nil, err
		}
		return nil, realmRoleMappingsOutput{Direct: nonNil(direct), Effective: nonNil(effective)}, nil
	})
	if options.ReadOnly {
		return
	}

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_create",
		Title:       "Create group",
		Description: "Create a top-level group in a realm. Returns the created group including its internal ID.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createGroupInput) (*mcp.CallToolResult, any, error) {
		created, err := admin.CreateGroup(ctx, in.Realm, in.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_delete",
		Title:       "Delete group",
		Description: "Delete a group by internal group ID. This cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in groupRefInput) (*mcp.CallToolResult, any, error) {
		if err := admin.DeleteGroup(ctx, in.Realm, in.GroupID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "groupId": in.GroupID, "deleted": true}, nil
	})
}
