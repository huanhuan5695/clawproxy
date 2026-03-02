package auth

import (
	"strings"
	"testing"
	"time"
)

func TestGenerateAndValidateToken(t *testing.T) {
	token, err := GenerateToken([]byte("secret"), "dev-1", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if err := ValidateToken([]byte("secret"), token); err != nil {
		t.Fatalf("validate token: %v", err)
	}
}

func TestValidateTokenInvalid(t *testing.T) {
	token, err := GenerateToken([]byte("secret"), "dev-1", time.Hour)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if err := ValidateToken([]byte("wrong"), token); err == nil {
		t.Fatal("expected validation error")
	}
}

func TestValidateTokenExpired(t *testing.T) {
	token, err := GenerateToken([]byte("secret"), "dev-1", time.Second)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}
	time.Sleep(1100 * time.Millisecond)

	err = ValidateToken([]byte("secret"), token)
	if err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expected expired token error, got %v", err)
	}
}

func TestGenerateTokenWithoutExpiration(t *testing.T) {
	token, err := GenerateToken([]byte("secret"), "dev-1", 0)
	if err != nil {
		t.Fatalf("generate token: %v", err)
	}

	if err := ValidateToken([]byte("secret"), token); err != nil {
		t.Fatalf("validate token without exp: %v", err)
	}
}
