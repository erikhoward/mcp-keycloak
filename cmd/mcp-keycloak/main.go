// Command mcp-keycloak runs a Model Context Protocol server that exposes
// Keycloak administration tools over stdio.
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/erikhoward/mcp-keycloak/internal/config"
	"github.com/erikhoward/mcp-keycloak/internal/keycloak"
	"github.com/erikhoward/mcp-keycloak/internal/mcpserver"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "mcp-keycloak:", err)
		os.Exit(1)
	}
}

func run() error {
	// A .env file is a development convenience; real environment variables
	// always take precedence (godotenv does not override existing values).
	if err := godotenv.Load(); err != nil && !os.IsNotExist(err) {
		fmt.Fprintln(os.Stderr, "mcp-keycloak: warning: loading .env:", err)
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	admin, err := keycloak.NewAdminWithOptions(
		cfg.KeycloakURL,
		cfg.AdminUsername,
		cfg.AdminPassword,
		cfg.AdminRealm,
		keycloak.AdminOptions{
			HTTPTimeout: cfg.KeycloakTimeout,
			CACertFile:  cfg.KeycloakCACertFile,
		},
	)
	if err != nil {
		return err
	}
	srv := mcpserver.NewWithOptions(admin, mcpserver.Options{ReadOnly: cfg.ReadOnly})
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	// StdioTransport communicates over stdin/stdout and (unlike IOTransport
	// with os.Stdout) does not close stdout when the client disconnects.
	return srv.Run(ctx, &mcp.StdioTransport{})
}
