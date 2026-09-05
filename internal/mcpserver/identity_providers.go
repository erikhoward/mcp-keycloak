package mcpserver

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/Nerzal/gocloak/v14"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const redactedSecret = "[REDACTED]"

type identityProviderRefInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
	Alias string `json:"alias" jsonschema:"identity provider alias"`
}

type identityProviderInput struct {
	Realm                string  `json:"realm" jsonschema:"realm name"`
	Alias                string  `json:"alias" jsonschema:"unique provider alias, e.g. \"corporate-oidc\""`
	DisplayName          *string `json:"displayName,omitempty" jsonschema:"display name shown to users; omit to leave unchanged"`
	Enabled              *bool   `json:"enabled,omitempty" jsonschema:"whether the provider can be used; omit to leave unchanged"`
	Issuer               *string `json:"issuer,omitempty" jsonschema:"OIDC issuer URL; use HTTPS in production"`
	AuthorizationURL     *string `json:"authorizationUrl,omitempty" jsonschema:"OIDC authorization endpoint URL"`
	TokenURL             *string `json:"tokenUrl,omitempty" jsonschema:"OIDC token endpoint URL"`
	UserInfoURL          *string `json:"userInfoUrl,omitempty" jsonschema:"OIDC user-info endpoint URL"`
	JWKSURL              *string `json:"jwksUrl,omitempty" jsonschema:"OIDC JWKS endpoint URL"`
	ClientID             *string `json:"clientId,omitempty" jsonschema:"OIDC client ID registered at the external provider"`
	ClientSecret         *string `json:"clientSecret,omitempty" jsonschema:"OIDC client secret; sensitive and never returned by this tool"`
	DefaultScope         *string `json:"defaultScope,omitempty" jsonschema:"OIDC default scope, commonly \"openid profile email\""`
	TrustEmail           *bool   `json:"trustEmail,omitempty" jsonschema:"trust the provider's email verification"`
	LinkOnly             *bool   `json:"linkOnly,omitempty" jsonschema:"only allow linking existing accounts, not automatic login account creation"`
	StoreToken           *bool   `json:"storeToken,omitempty" jsonschema:"store provider tokens for the linked user"`
	FirstBrokerLoginFlow *string `json:"firstBrokerLoginFlowAlias,omitempty" jsonschema:"first-broker-login flow alias"`
	PostBrokerLoginFlow  *string `json:"postBrokerLoginFlowAlias,omitempty" jsonschema:"post-broker-login flow alias"`
}

func addIdentityProviderTools(s *mcp.Server, admin AdminAPI) {
	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_provider_list",
		Title:       "List identity providers",
		Description: "List OIDC and other identity providers configured in a realm. Client secrets are redacted.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in listIdentityProvidersInput) (*mcp.CallToolResult, any, error) {
		providers, err := admin.ListIdentityProviders(ctx, in.Realm)
		if err != nil {
			return nil, nil, err
		}
		return nil, redactIdentityProviders(providers), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_provider_get",
		Title:       "Get identity provider",
		Description: "Get an identity provider by alias. Client secrets are redacted.",
		Annotations: &mcp.ToolAnnotations{ReadOnlyHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderRefInput) (*mcp.CallToolResult, any, error) {
		provider, err := admin.GetIdentityProvider(ctx, in.Realm, in.Alias)
		if err != nil {
			return nil, nil, err
		}
		return nil, redactIdentityProvider(provider), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_provider_create",
		Title:       "Create OIDC identity provider",
		Description: "Create an OIDC identity provider. This changes how users authenticate in the realm; use HTTPS endpoints in production. Client secrets are never returned.",
		Annotations: &mcp.ToolAnnotations{DestructiveHint: gocloak.BoolP(false)},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderInput) (*mcp.CallToolResult, any, error) {
		provider, err := buildIdentityProvider(in, true)
		if err != nil {
			return nil, nil, err
		}
		created, err := admin.CreateIdentityProvider(ctx, in.Realm, provider)
		if err != nil {
			return nil, nil, err
		}
		return nil, redactIdentityProvider(created), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_provider_update",
		Title:       "Update OIDC identity provider",
		Description: "Partially update an OIDC identity provider by alias. Omitted fields remain unchanged. This can affect realm login behavior; client secrets are never returned.",
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderInput) (*mcp.CallToolResult, any, error) {
		provider, err := buildIdentityProvider(in, false)
		if err != nil {
			return nil, nil, err
		}
		updated, err := admin.UpdateIdentityProvider(ctx, in.Realm, in.Alias, provider)
		if err != nil {
			return nil, nil, err
		}
		return nil, redactIdentityProvider(updated), nil
	})

	mcp.AddTool(s, &mcp.Tool{
		Name:        "identity_provider_delete",
		Title:       "Delete identity provider",
		Description: "Delete an identity provider by alias. This can break login for applications and users relying on the provider and cannot be undone.",
		Annotations: &mcp.ToolAnnotations{IdempotentHint: true},
	}, func(ctx context.Context, _ *mcp.CallToolRequest, in identityProviderRefInput) (*mcp.CallToolResult, any, error) {
		if err := admin.DeleteIdentityProvider(ctx, in.Realm, in.Alias); err != nil {
			return nil, nil, err
		}
		return nil, map[string]any{"realm": in.Realm, "alias": in.Alias, "deleted": true}, nil
	})
}

type listIdentityProvidersInput struct {
	Realm string `json:"realm" jsonschema:"realm name"`
}

func buildIdentityProvider(in identityProviderInput, includeAlias bool) (gocloak.IdentityProviderRepresentation, error) {
	if includeAlias && in.Alias == "" {
		return gocloak.IdentityProviderRepresentation{}, fmt.Errorf("identity provider alias is required")
	}
	for name, value := range map[string]*string{
		"issuer": in.Issuer, "authorizationUrl": in.AuthorizationURL, "tokenUrl": in.TokenURL,
		"userInfoUrl": in.UserInfoURL, "jwksUrl": in.JWKSURL,
	} {
		if err := validateIdentityProviderURL(name, value); err != nil {
			return gocloak.IdentityProviderRepresentation{}, err
		}
	}

	provider := gocloak.IdentityProviderRepresentation{
		DisplayName:               in.DisplayName,
		Enabled:                   in.Enabled,
		FirstBrokerLoginFlowAlias: in.FirstBrokerLoginFlow,
		PostBrokerLoginFlowAlias:  in.PostBrokerLoginFlow,
		TrustEmail:                in.TrustEmail,
		LinkOnly:                  in.LinkOnly,
		StoreToken:                in.StoreToken,
	}
	if includeAlias {
		provider.Alias = gocloak.StringP(in.Alias)
		provider.ProviderID = gocloak.StringP("oidc")
	}
	provider.Config = identityProviderConfig(in)
	return provider, nil
}

func identityProviderConfig(in identityProviderInput) map[string]string {
	config := make(map[string]string)
	add := func(key string, value *string) {
		if value != nil {
			config[key] = *value
		}
	}
	add("issuer", in.Issuer)
	add("authorizationUrl", in.AuthorizationURL)
	add("tokenUrl", in.TokenURL)
	add("userInfoUrl", in.UserInfoURL)
	add("jwksUrl", in.JWKSURL)
	add("clientId", in.ClientID)
	add("clientSecret", in.ClientSecret)
	add("defaultScope", in.DefaultScope)
	return config
}

func validateIdentityProviderURL(field string, value *string) error {
	if value == nil || *value == "" {
		return nil
	}
	parsed, err := url.Parse(*value)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return fmt.Errorf("identity provider %s must be an absolute URL", field)
	}
	if parsed.User != nil {
		return fmt.Errorf("identity provider %s must not include URL credentials", field)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLocalIdentityProviderHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("identity provider %s must use HTTPS; HTTP is only allowed for localhost development", field)
}

func isLocalIdentityProviderHost(host string) bool {
	host = strings.ToLower(host)
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func redactIdentityProviders(providers []*gocloak.IdentityProviderRepresentation) []*gocloak.IdentityProviderRepresentation {
	redacted := make([]*gocloak.IdentityProviderRepresentation, len(providers))
	for i, provider := range providers {
		redacted[i] = redactIdentityProvider(provider)
	}
	return redacted
}

func redactIdentityProvider(provider *gocloak.IdentityProviderRepresentation) *gocloak.IdentityProviderRepresentation {
	if provider == nil {
		return nil
	}
	copy := *provider
	copy.Config = make(map[string]string, len(provider.Config))
	for key, value := range provider.Config {
		normalized := strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(key))
		if strings.Contains(normalized, "secret") || strings.Contains(normalized, "password") {
			copy.Config[key] = redactedSecret
		} else {
			copy.Config[key] = value
		}
	}
	return &copy
}
