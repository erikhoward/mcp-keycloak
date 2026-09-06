package mcpserver

import (
	"context"
	"fmt"

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

type getGroupInput struct {
	Realm   string `json:"realm" jsonschema:"realm name"`
	GroupID string `json:"groupId,omitempty" jsonschema:"internal group ID (UUID) as returned by group_list or group_create; set this or path, not both"`
	Path    string `json:"path,omitempty" jsonschema:"group path such as parent/child; set this or groupId, not both"`
}

type updateGroupInput struct {
	Realm      string              `json:"realm" jsonschema:"realm name"`
	GroupID    string              `json:"groupId" jsonschema:"internal group ID (UUID) as returned by group_list or group_create"`
	Name       *string             `json:"name,omitempty" jsonschema:"new group name; omit to leave unchanged"`
	Attributes map[string][]string `json:"attributes,omitempty" jsonschema:"new attributes; replaces all attributes when provided; omit to leave unchanged"`
}

type listChildGroupsInput struct {
	Realm   string `json:"realm" jsonschema:"realm name"`
	GroupID string `json:"groupId" jsonschema:"internal group ID (UUID) of the parent group, as returned by group_list or group_create"`
	Max     int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type createChildGroupInput struct {
	Realm    string `json:"realm" jsonschema:"realm to create the subgroup in"`
	ParentID string `json:"parentId" jsonschema:"internal group ID (UUID) of the parent group, as returned by group_list or group_create"`
	Name     string `json:"name" jsonschema:"unique subgroup name"`
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

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_get",
		Title:       "Get group",
		Description: "Get one group by internal group ID or by path such as parent/child. Set exactly one of groupId and path. Returns the ID, name, path, and attributes.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in getGroupInput) (*mcp.CallToolResult, any, error) {
		switch {
		case in.GroupID != "" && in.Path != "":
			return nil, nil, fmt.Errorf("set groupId or path, not both")
		case in.GroupID == "" && in.Path == "":
			return nil, nil, fmt.Errorf("set groupId or path")
		}
		if in.GroupID != "" {
			group, err := admin.GetGroup(ctx, in.Realm, in.GroupID)
			if err != nil {
				return nil, nil, err
			}
			return nil, group, nil
		}
		group, err := admin.GetGroupByPath(ctx, in.Realm, in.Path)
		if err != nil {
			return nil, nil, err
		}
		return nil, group, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_children_list",
		Title:       "List child groups",
		Description: "List the direct child groups of a group by internal group ID. Returns group IDs, names, and paths.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listChildGroupsInput) (*mcp.CallToolResult, any, error) {
		children, err := admin.ListChildGroups(ctx, in.Realm, in.GroupID, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(children), nil
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
		Name:        "group_child_create",
		Title:       "Create child group",
		Description: "Create a subgroup under a parent group by internal group ID. Returns the created group including its internal ID and path.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createChildGroupInput) (*mcp.CallToolResult, any, error) {
		created, err := admin.CreateChildGroup(ctx, in.Realm, in.ParentID, in.Name)
		if err != nil {
			return nil, nil, err
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "group_update",
		Title:       "Update group",
		Description: "Update a group by internal group ID. Only the provided fields are changed. Attributes replace all existing attributes when provided. Returns the updated group.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateGroupInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.Group{ID: gocloak.StringP(in.GroupID)}
		if in.Name != nil {
			rep.Name = in.Name
		}
		if in.Attributes != nil {
			rep.Attributes = in.Attributes
		}
		updated, err := admin.UpdateGroup(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
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
