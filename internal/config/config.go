// Package config loads mcp-keycloak runtime configuration from the
// environment.
package config

import (
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
)

// Config holds the settings needed to administer a Keycloak server.
type Config struct {
	// KeycloakURL is the base URL of the Keycloak server,
	// e.g. "https://sso.example.com" (env KEYCLOAK_URL).
	KeycloakURL string
	// AdminUsername is the administrator account username
	// (env KEYCLOAK_ADMIN_USERNAME).
	AdminUsername string
	// AdminPassword is the administrator account password
	// (env KEYCLOAK_ADMIN_PASSWORD).
	AdminPassword string
	// AdminRealm is the realm the administrator logs into
	// (env KEYCLOAK_ADMIN_REALM, default "master").
	AdminRealm string
}

// Load reads configuration from environment variables, applying defaults and
// validating required values. A .env file can supply the variables; it is
// loaded by the main command before Load is called.
func Load() (Config, error) {
	cfg := Config{
		KeycloakURL:   strings.TrimRight(strings.TrimSpace(os.Getenv("KEYCLOAK_URL")), "/"),
		AdminUsername: os.Getenv("KEYCLOAK_ADMIN_USERNAME"),
		AdminPassword: os.Getenv("KEYCLOAK_ADMIN_PASSWORD"),
		AdminRealm:    os.Getenv("KEYCLOAK_ADMIN_REALM"),
	}
	if cfg.AdminRealm == "" {
		cfg.AdminRealm = "master"
	}

	var missing []string
	if cfg.KeycloakURL == "" {
		missing = append(missing, "KEYCLOAK_URL")
	}
	if cfg.AdminUsername == "" {
		missing = append(missing, "KEYCLOAK_ADMIN_USERNAME")
	}
	if cfg.AdminPassword == "" {
		missing = append(missing, "KEYCLOAK_ADMIN_PASSWORD")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables (set them in the environment or a .env file): %s", strings.Join(missing, ", "))
	}
	if err := validateKeycloakURL(cfg.KeycloakURL); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func validateKeycloakURL(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Hostname() == "" {
		return fmt.Errorf("invalid KEYCLOAK_URL: use an absolute HTTPS URL (HTTP is allowed only for localhost development)")
	}
	if parsed.User != nil {
		return fmt.Errorf("invalid KEYCLOAK_URL: embedded URL credentials are not allowed")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("invalid KEYCLOAK_URL: query strings and fragments are not allowed")
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("invalid KEYCLOAK_URL: use HTTPS; HTTP is allowed only for localhost or loopback development")
}

func isLoopbackHost(host string) bool {
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
