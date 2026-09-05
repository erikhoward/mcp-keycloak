package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type listAdminEventsInput struct {
	Realm          string   `json:"realm" jsonschema:"realm name"`
	AuthClient     string   `json:"authClient,omitempty" jsonschema:"administrator client ID filter"`
	AuthUser       string   `json:"authUser,omitempty" jsonschema:"administrator user ID filter"`
	DateFrom       string   `json:"dateFrom,omitempty" jsonschema:"start date/time filter in Keycloak's accepted format"`
	DateTo         string   `json:"dateTo,omitempty" jsonschema:"end date/time filter in Keycloak's accepted format"`
	OperationTypes []string `json:"operationTypes,omitempty" jsonschema:"operation type filters such as CREATE, UPDATE, or DELETE"`
	ResourcePath   string   `json:"resourcePath,omitempty" jsonschema:"resource path filter"`
	ResourceTypes  []string `json:"resourceTypes,omitempty" jsonschema:"resource type filters such as USER, CLIENT, or REALM"`
	Max            int      `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

type listEventsInput struct {
	Realm     string   `json:"realm" jsonschema:"realm name"`
	ClientID  string   `json:"clientId,omitempty" jsonschema:"client identifier filter"`
	DateFrom  string   `json:"dateFrom,omitempty" jsonschema:"start date/time filter in Keycloak's accepted format"`
	DateTo    string   `json:"dateTo,omitempty" jsonschema:"end date/time filter in Keycloak's accepted format"`
	IPAddress string   `json:"ipAddress,omitempty" jsonschema:"source IP address filter"`
	Types     []string `json:"types,omitempty" jsonschema:"event type filters such as LOGIN or LOGIN_ERROR"`
	UserID    string   `json:"userId,omitempty" jsonschema:"internal user ID filter"`
	Max       int      `json:"max,omitempty" jsonschema:"maximum number of results; default 100"`
}

func addEventTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "event_admin_list",
		Title:       "List admin events",
		Description: "List Keycloak administrative audit events. Enable Admin Events in the realm's Events settings first; failures from a disabled event store are returned as tool errors.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listAdminEventsInput) (*mcp.CallToolResult, any, error) {
		params := gocloak.GetAdminEventsParams{
			AuthClient:     optionalString(in.AuthClient),
			AuthUser:       optionalString(in.AuthUser),
			DateFrom:       optionalString(in.DateFrom),
			DateTo:         optionalString(in.DateTo),
			OperationTypes: in.OperationTypes,
			ResourcePath:   optionalString(in.ResourcePath),
			ResourceTypes:  in.ResourceTypes,
			Max:            gocloak.Int32P(int32(resolveMax(in.Max))),
		}
		events, err := admin.ListAdminEvents(ctx, in.Realm, params)
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(events), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "event_login_list",
		Title:       "List login events",
		Description: "List Keycloak login and user events with optional client, user, IP, date, and event-type filters. Enable user events in the realm's Events settings first.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listEventsInput) (*mcp.CallToolResult, any, error) {
		params := gocloak.GetEventsParams{
			Client:    optionalString(in.ClientID),
			DateFrom:  optionalString(in.DateFrom),
			DateTo:    optionalString(in.DateTo),
			IPAddress: optionalString(in.IPAddress),
			Type:      in.Types,
			UserID:    optionalString(in.UserID),
			Max:       gocloak.Int32P(int32(resolveMax(in.Max))),
		}
		events, err := admin.ListEvents(ctx, in.Realm, params)
		if err != nil {
			return nil, nil, err
		}
		return nil, nonNil(events), nil
	})
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	return gocloak.StringP(value)
}
