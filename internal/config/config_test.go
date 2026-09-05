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

func TestLoadKeycloakURLValidation(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{name: "remote HTTPS", url: "https://sso.example.com", wantErr: false},
		{name: "HTTPS with context path", url: "https://sso.example.com/auth", wantErr: false},
		{name: "localhost HTTP", url: "http://localhost:8080", wantErr: false},
		{name: "IPv4 loopback HTTP", url: "http://127.0.0.1:8080", wantErr: false},
		{name: "IPv6 loopback HTTP", url: "http://[::1]:8080", wantErr: false},
		{name: "remote HTTP", url: "http://sso.example.com", wantErr: true},
		{name: "unsupported scheme", url: "ftp://sso.example.com", wantErr: true},
		{name: "embedded credentials", url: "https://admin:secret@sso.example.com", wantErr: true},
		{name: "query string", url: "https://sso.example.com?realm=master", wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("KEYCLOAK_URL", test.url)
			t.Setenv("KEYCLOAK_ADMIN_USERNAME", "admin")
			t.Setenv("KEYCLOAK_ADMIN_PASSWORD", "secret")
			t.Setenv("KEYCLOAK_ADMIN_REALM", "master")

			_, err := Load()
			if test.wantErr && err == nil {
				t.Fatalf("Load(%q): expected an error", test.url)
			}
			if !test.wantErr && err != nil {
				t.Fatalf("Load(%q): unexpected error: %v", test.url, err)
			}
		})
	}
}
