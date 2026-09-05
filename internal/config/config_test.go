package config

import (
	"strings"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	t.Setenv("KEYCLOAK_URL", "http://localhost:8080/")
	t.Setenv("KEYCLOAK_ADMIN_USERNAME", "admin")
	t.Setenv("KEYCLOAK_ADMIN_PASSWORD", "secret")
	t.Setenv("KEYCLOAK_ADMIN_REALM", "")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.KeycloakURL != "http://localhost:8080" {
		t.Errorf("KeycloakURL = %q, want trailing slash trimmed", cfg.KeycloakURL)
	}
	if cfg.AdminRealm != "master" {
		t.Errorf("AdminRealm = %q, want default %q", cfg.AdminRealm, "master")
	}
}

func TestLoadExplicitRealm(t *testing.T) {
	t.Setenv("KEYCLOAK_URL", "https://sso.example.com")
	t.Setenv("KEYCLOAK_ADMIN_USERNAME", "admin")
	t.Setenv("KEYCLOAK_ADMIN_PASSWORD", "secret")
	t.Setenv("KEYCLOAK_ADMIN_REALM", "custom")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.AdminRealm != "custom" {
		t.Errorf("AdminRealm = %q, want %q", cfg.AdminRealm, "custom")
	}
}

func TestLoadMissingVariables(t *testing.T) {
	t.Setenv("KEYCLOAK_URL", "")
	t.Setenv("KEYCLOAK_ADMIN_USERNAME", "")
	t.Setenv("KEYCLOAK_ADMIN_PASSWORD", "")

	_, err := Load()
	if err == nil {
		t.Fatal("Load: expected error for missing variables")
	}
	for _, want := range []string{"KEYCLOAK_URL", "KEYCLOAK_ADMIN_USERNAME", "KEYCLOAK_ADMIN_PASSWORD"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error %q does not mention %s", err, want)
		}
	}
}
