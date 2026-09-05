package mcpserver

import (
	"context"
	"fmt"

	"github.com/Nerzal/gocloak/v13"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listUsersInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	Search   string `json:"search,omitempty" jsonschema:"substring matched against username, email, first and last name"`
	Username string `json:"username,omitempty" jsonschema:"exact username match"`
	Max      int    `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type userRefInput struct {
	Realm  string `json:"realm" jsonschema:"realm name"`
	UserID string `json:"userId" jsonschema:"internal user ID (UUID) as returned by user_list or user_create"`
}

type createUserInput struct {
	Realm                    string `json:"realm" jsonschema:"realm to create the user in"`
	Username                 string `json:"username" jsonschema:"unique username"`
	Email                    string `json:"email,omitempty"`
	FirstName                string `json:"firstName,omitempty"`
	LastName                 string `json:"lastName,omitempty"`
	Enabled                  *bool  `json:"enabled,omitempty" jsonschema:"whether the user can log in; default true"`
	EmailVerified            *bool  `json:"emailVerified,omitempty" jsonschema:"mark the email address as verified; default false"`
	InitialPassword          string `json:"initialPassword,omitempty" jsonschema:"optional initial password; by default the user must change it at first login (see initialPasswordTemporary)"`
	InitialPasswordTemporary *bool  `json:"initialPasswordTemporary,omitempty" jsonschema:"when true (the default) the user must change the initial password at first login"`
}

type updateUserInput struct {
	Realm         string  `json:"realm" jsonschema:"realm name"`
	UserID        string  `json:"userId" jsonschema:"internal user ID (UUID) as returned by user_list or user_create"`
	Email         *string `json:"email,omitempty" jsonschema:"new email address; omit to leave unchanged"`
	FirstName     *string `json:"firstName,omitempty" jsonschema:"new first name; omit to leave unchanged"`
	LastName      *string `json:"lastName,omitempty" jsonschema:"new last name; omit to leave unchanged"`
	Enabled       *bool   `json:"enabled,omitempty" jsonschema:"whether the user can log in; omit to leave unchanged"`
	EmailVerified *bool   `json:"emailVerified,omitempty" jsonschema:"whether the email address is verified; omit to leave unchanged"`
}

type setUserPasswordInput struct {
	Realm     string `json:"realm" jsonschema:"realm name"`
	UserID    string `json:"userId" jsonschema:"internal user ID (UUID) as returned by user_list or user_create"`
	Password  string `json:"password" jsonschema:"the new password"`
	Temporary *bool  `json:"temporary,omitempty" jsonschema:"when true the user must change the password at next login; default false"`
}

type userRolesInput struct {
	Realm  string   `json:"realm" jsonschema:"realm name"`
	UserID string   `json:"userId" jsonschema:"internal user ID (UUID) as returned by user_list or user_create"`
	Roles  []string `json:"roles" jsonschema:"realm role names, e.g. [\"auditor\"]"`
}

type userGroupInput struct {
	Realm   string `json:"realm" jsonschema:"realm name"`
	UserID  string `json:"userId" jsonschema:"internal user ID (UUID) as returned by user_list or user_create"`
	GroupID string `json:"groupId" jsonschema:"internal group ID (UUID) as returned by group_list or group_create"`
}

func addUserTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_list",
		Title:       "List users",
		Description: "List the users of a realm, optionally filtered by a search substring or an exact username. Returns internal user IDs needed by other user tools.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listUsersInput) (*mcp.CallToolResult, any, error) {
		users, err := admin.ListUsers(ctx, in.Realm, in.Search, in.Username, resolveMax(in.Max))
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(users), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_get",
		Title:       "Get user",
		Description: "Get a single user by internal user ID. Find the ID with user_list.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userRefInput) (*mcp.CallToolResult, any, error) {
		user, err := admin.GetUser(ctx, in.Realm, in.UserID)
		if err != nil {
			return nil, nil, err
		}
		return nil, user, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_create",
		Title:       "Create user",
		Description: "Create a user in a realm, optionally with an initial password (temporary by default, so the user must change it at first login). Returns the created user including its internal ID.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in createUserInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.User{
			Username:      gocloak.StringP(in.Username),
			Enabled:       gocloak.BoolP(in.Enabled == nil || *in.Enabled),
			EmailVerified: in.EmailVerified,
		}
		if in.Email != "" {
			rep.Email = gocloak.StringP(in.Email)
		}
		if in.FirstName != "" {
			rep.FirstName = gocloak.StringP(in.FirstName)
		}
		if in.LastName != "" {
			rep.LastName = gocloak.StringP(in.LastName)
		}
		created, err := admin.CreateUser(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		if in.InitialPassword != "" {
			userID := deref(created.ID)
			if userID == "" {
				return nil, nil, fmt.Errorf("user %q created, but its ID is missing; set the password with user_set_password", in.Username)
			}
			temporary := in.InitialPasswordTemporary == nil || *in.InitialPasswordTemporary
			if err := admin.SetUserPassword(ctx, in.Realm, userID, in.InitialPassword, temporary); err != nil {
				return nil, nil, fmt.Errorf("user %q created, but setting its initial password failed: %w", in.Username, err)
			}
		}
		return nil, created, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_update",
		Title:       "Update user",
		Description: "Update a user's profile. Only the provided fields are changed. Returns the updated user.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateUserInput) (*mcp.CallToolResult, any, error) {
		rep := gocloak.User{
			ID:            gocloak.StringP(in.UserID),
			Email:         in.Email,
			FirstName:     in.FirstName,
			LastName:      in.LastName,
			Enabled:       in.Enabled,
			EmailVerified: in.EmailVerified,
		}
		updated, err := admin.UpdateUser(ctx, in.Realm, rep)
		if err != nil {
			return nil, nil, err
		}
		return nil, updated, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_set_password",
		Title:       "Set user password",
		Description: "Set or reset a user's password by internal user ID. With temporary=true the user must change the password at next login.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in setUserPasswordInput) (*mcp.CallToolResult, any, error) {
		temporary := in.Temporary != nil && *in.Temporary
		if err := admin.SetUserPassword(ctx, in.Realm, in.UserID, in.Password, temporary); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "passwordSet": true, "temporary": temporary}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_delete",
		Title:       "Delete user",
		Description: "Delete a user by internal user ID. This cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userRefInput) (*mcp.CallToolResult, any, error) {
		if err := admin.DeleteUser(ctx, in.Realm, in.UserID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "deleted": true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_add_realm_role",
		Title:       "Add realm roles to user",
		Description: "Assign realm roles to a user by internal user ID; roles are matched by name, e.g. [\"auditor\"]. Returns the roles added.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userRolesInput) (*mcp.CallToolResult, any, error) {
		if len(in.Roles) == 0 {
			return nil, nil, fmt.Errorf("roles must list at least one realm role name")
		}
		if err := admin.AddRealmRolesToUser(ctx, in.Realm, in.UserID, in.Roles); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "rolesAdded": in.Roles}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_remove_realm_role",
		Title:       "Remove realm roles from user",
		Description: "Remove realm roles from a user by internal user ID; roles are matched by name. Returns the roles removed.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userRolesInput) (*mcp.CallToolResult, any, error) {
		if len(in.Roles) == 0 {
			return nil, nil, fmt.Errorf("roles must list at least one realm role name")
		}
		if err := admin.RemoveRealmRolesFromUser(ctx, in.Realm, in.UserID, in.Roles); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "rolesRemoved": in.Roles}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_add_to_group",
		Title:       "Add user to group",
		Description: "Add a user to a group by internal user ID and group ID.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userGroupInput) (*mcp.CallToolResult, any, error) {
		if err := admin.AddUserToGroup(ctx, in.Realm, in.UserID, in.GroupID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "groupId": in.GroupID, "added": true}, nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "user_remove_from_group",
		Title:       "Remove user from group",
		Description: "Remove a user from a group by internal user ID and group ID.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in userGroupInput) (*mcp.CallToolResult, any, error) {
		if err := admin.RemoveUserFromGroup(ctx, in.Realm, in.UserID, in.GroupID); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "userId": in.UserID, "groupId": in.GroupID, "removed": true}, nil
	})
}
