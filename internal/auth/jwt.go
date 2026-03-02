package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type claims struct {
	Subject   string `json:"sub"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt *int64 `json:"exp,omitempty"`
}

func GenerateToken(secret []byte, subject string, expiresIn time.Duration) (string, error) {
	now := time.Now().Unix()
	jwtClaims := claims{Subject: subject, IssuedAt: now}
	if expiresIn > 0 {
		expiresAt := now + int64(expiresIn.Seconds())
		jwtClaims.ExpiresAt = &expiresAt
	}

	payload, err := json.Marshal(jwtClaims)
	if err != nil {
		return "", fmt.Errorf("marshal jwt payload: %w", err)
	}

	header := []byte(`{"alg":"HS256","typ":"JWT"}`)
	headerEncoded := base64.RawURLEncoding.EncodeToString(header)
	payloadEncoded := base64.RawURLEncoding.EncodeToString(payload)
	unsigned := headerEncoded + "." + payloadEncoded

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(unsigned)); err != nil {
		return "", fmt.Errorf("sign jwt: %w", err)
	}
	signatureEncoded := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return unsigned + "." + signatureEncoded, nil
}

func ValidateToken(secret []byte, token string) error {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return fmt.Errorf("invalid jwt format")
	}

	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("decode jwt header: %w", err)
	}
	if string(headerRaw) != `{"alg":"HS256","typ":"JWT"}` {
		return fmt.Errorf("unsupported jwt header")
	}

	mac := hmac.New(sha256.New, secret)
	if _, err := mac.Write([]byte(parts[0] + "." + parts[1])); err != nil {
		return fmt.Errorf("calculate signature: %w", err)
	}
	expectedSig := mac.Sum(nil)
	actualSig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return fmt.Errorf("decode jwt signature: %w", err)
	}
	if !hmac.Equal(actualSig, expectedSig) {
		return fmt.Errorf("jwt signature mismatch")
	}

	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return fmt.Errorf("decode jwt payload: %w", err)
	}

	var c claims
	if err := json.Unmarshal(payloadRaw, &c); err != nil {
		return fmt.Errorf("unmarshal jwt payload: %w", err)
	}

	if c.ExpiresAt != nil && *c.ExpiresAt <= time.Now().Unix() {
		return fmt.Errorf("jwt token expired")
	}

	return nil
}
