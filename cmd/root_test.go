package cmd

import (
	"bytes"
	"strings"
	"testing"

	"clawproxy/internal/auth"
)

func TestTokenCommand(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--jwt-secret", "test-secret", "token", "--device-id", "dev-1", "--expires-in", "2d"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	tokenString := bytes.TrimSpace(buf.Bytes())
	if len(tokenString) == 0 {
		t.Fatal("expected token output")
	}

	parts := strings.Split(string(tokenString), ".")
	if len(parts) != 3 {
		t.Fatalf("expected jwt with 3 segments, got %d", len(parts))
	}

	if err := auth.ValidateToken([]byte("test-secret"), string(tokenString)); err != nil {
		t.Fatalf("validate token: %v", err)
	}
}

func TestTokenCommand_DefaultNoExpire(t *testing.T) {
	buf := &bytes.Buffer{}
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"--jwt-secret", "test-secret", "token", "--device-id", "dev-1"})

	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute root command: %v", err)
	}

	tokenString := bytes.TrimSpace(buf.Bytes())
	if err := auth.ValidateToken([]byte("test-secret"), string(tokenString)); err != nil {
		t.Fatalf("validate token: %v", err)
	}
}

func TestParseExpiresInDays(t *testing.T) {
	d, err := parseExpiresInDays("1d")
	if err != nil {
		t.Fatalf("parse expires-in: %v", err)
	}
	if d.Hours() != 24 {
		t.Fatalf("expected 24h, got %s", d)
	}

	if _, err := parseExpiresInDays("2h"); err == nil {
		t.Fatalf("expected parse error for invalid format")
	}
}
