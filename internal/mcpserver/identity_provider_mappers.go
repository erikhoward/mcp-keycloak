package mcpserver

import (
	"context"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type identityProviderMapperRefInput struct {
	Realm    string `json:"realm" jsonschema:"realm name"`
	Alias    string `json:"alias" jsonschema:"identity provider alias"`
	MapperID string `json:"mapperId" jsonschema:"internal mapper UUID"`
}

type createIdentityProviderMapperInput struct {
	Realm      string            `json:"realm" jsonschema:"realm name"`
	Alias      string            `json:"alias" jsonschema:"identity provider alias"`
	Name       string            `json:"name" jsonschema:"mapper display name"`
	MapperType string            `json:"mapperType" jsonschema:"Keycloak identity provider mapper type"`
	Config     map[string]string `json:"config,omitempty" jsonschema:"mapper-specific configuration"`
}

type updateIdentityProviderMapperInput struct {
	Realm      string            `json:"realm" jsonschema:"realm name"`
	Alias      string            `json:"alias" jsonschema:"identity provider alias"`
	MapperID   string            `json:"mapperId" jsonschema:"internal mapper UUID"`
	Name       *string           `json:"name,omitempty" jsonschema:"new mapper display name; omit to leave unchanged"`
	MapperType *string           `json:"mapperType,omitempty" jsonschema:"new mapper type; omit to leave unchanged"`
	Config     map[string]string `json:"config,omitempty" jsonschema:"replacement mapper configuration; omit to leave unchanged"`
}

func addIdentityProviderMapperTools(s *mcp.Server, admin AdminAPI, options Options) {
	mcp.AddTool(s, &mcp.Tool{Name: "identity_provider_mapper_list", Title: "List identity provider mappers", Description: "List mappers configured for an identity provider. Sensitive config values are redacted.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderRefInput) (*mcp.CallToolResult, any, error) {
		mappers, err := admin.ListIdentityProviderMappers(ctx, in.Realm, in.Alias)
		return nil, redactIdentityProviderMappers(mappers), err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "identity_provider_mapper_get", Title: "Get identity provider mapper", Description: "Get an identity provider mapper by UUID. Sensitive config values are redacted.", Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderMapperRefInput) (*mcp.CallToolResult, any, error) {
		mapper, err := admin.GetIdentityProviderMapper(ctx, in.Realm, in.Alias, in.MapperID)
		return nil, redactIdentityProviderMapper(mapper), err
	})
	if options.ReadOnly {
		return
	}
	mcp.AddTool(s, &mcp.Tool{Name: "identity_provider_mapper_create", Title: "Create identity provider mapper", Description: "Create a mapper for an identity provider. Sensitive config values are redacted from the result.", Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)}}, func(ctx context.Context, _ *mcp.CallToolRequest, in createIdentityProviderMapperInput) (*mcp.CallToolResult, any, error) {
		mapper := gocloak.IdentityProviderMapper{Name: gocloak.StringP(in.Name), IdentityProviderMapper: gocloak.StringP(in.MapperType), IdentityProviderAlias: gocloak.StringP(in.Alias), Config: in.Config}
		created, err := admin.CreateIdentityProviderMapper(ctx, in.Realm, in.Alias, mapper)
		return nil, redactIdentityProviderMapper(created), err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "identity_provider_mapper_update", Title: "Update identity provider mapper", Description: "Partially update an identity provider mapper. Omitted fields remain unchanged.", Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)}}, func(ctx context.Context, _ *mcp.CallToolRequest, in updateIdentityProviderMapperInput) (*mcp.CallToolResult, any, error) {
		mapper := gocloak.IdentityProviderMapper{Name: in.Name, IdentityProviderMapper: in.MapperType, Config: in.Config}
		updated, err := admin.UpdateIdentityProviderMapper(ctx, in.Realm, in.Alias, in.MapperID, mapper)
		return nil, redactIdentityProviderMapper(updated), err
	})
	mcp.AddTool(s, &mcp.Tool{Name: "identity_provider_mapper_delete", Title: "Delete identity provider mapper", Description: "Delete an identity provider mapper by UUID.", Annotations: &mcp.ToolAnnotations{IdempotentHint: true}}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderMapperRefInput) (*mcp.CallToolResult, any, error) {
		err := admin.DeleteIdentityProviderMapper(ctx, in.Realm, in.Alias, in.MapperID)
		return nil, map[string]any{"alias": in.Alias, "mapperId": in.MapperID, "deleted": true}, err
	})
}

func redactIdentityProviderMappers(mappers []*gocloak.IdentityProviderMapper) []*gocloak.IdentityProviderMapper {
	redacted := make([]*gocloak.IdentityProviderMapper, len(mappers))
	for i, mapper := range mappers {
		redacted[i] = redactIdentityProviderMapper(mapper)
	}
	return redacted
}

func redactIdentityProviderMapper(mapper *gocloak.IdentityProviderMapper) *gocloak.IdentityProviderMapper {
	if mapper == nil {
		return nil
	}
	copy := *mapper
	copy.Config = make(map[string]string, len(mapper.Config))
	for key, value := range mapper.Config {
		if isSensitiveKey(key) {
			copy.Config[key] = redactedSecret
		} else {
			copy.Config[key] = value
		}
	}
	return &copy
}
