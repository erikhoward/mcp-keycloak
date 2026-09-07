package mcpserver

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type serverInfoOutput struct {
	Version *string `json:"version,omitempty"`
}

func addServerInfoTool(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{Name: "server_info", Title: "Get Keycloak server info", Description: "Get the Keycloak server version without exposing host or runtime details.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, any, error) {
		info, err := admin.GetServerInfo(ctx)
		if err != nil {
			return nil, nil, err
		}
		if info == nil || info.SystemInfo == nil {
			return nil, serverInfoOutput{}, nil
		}
		return nil, serverInfoOutput{Version: info.SystemInfo.Version}, nil
	})
}
