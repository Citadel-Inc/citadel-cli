package selfhost

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// defaultTokenDuration is the default validity window for a bootstrap token.
const defaultTokenDuration = 7 * 24 * time.Hour

// GenerateBootstrapToken mints a Supabase-compatible admin JWT signed with the
// JWT secret from cfg.  The token carries:
//   - role: "service_role"   (grants full DB access via RLS bypass)
//   - iss:  "supabase"
//   - sub:  "citadel-bootstrap"
//   - exp:  now + duration
//
// Q6 decision: token is returned as a string only; callers write it to
// stdout.  This function never persists the token to disk.
func GenerateBootstrapToken(cfg Config, duration time.Duration) (string, error) {
	if cfg.JWTSecret == "" {
		return "", errors.New("jwt_secret is not configured; run `citadel self-host init` to set it")
	}
	if duration <= 0 {
		duration = defaultTokenDuration
	}

	now := time.Now().UTC()
	claims := jwt.MapClaims{
		"role": "service_role",
		"iss":  "supabase",
		"sub":  "citadel-bootstrap",
		"iat":  now.Unix(),
		"exp":  now.Add(duration).Unix(),
	}

	tok := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := tok.SignedString([]byte(cfg.JWTSecret))
	if err != nil {
		return "", fmt.Errorf("sign bootstrap token: %w", err)
	}
	return signed, nil
}

// ValidateBootstrapToken parses and validates a bootstrap token against the
// JWT secret.  Useful in tests and for the `health` probe.
func ValidateBootstrapToken(tokenStr, jwtSecret string) (jwt.MapClaims, error) {
	if jwtSecret == "" {
		return nil, errors.New("jwt_secret is empty")
	}
	tok, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse bootstrap token: %w", err)
	}
	claims, ok := tok.Claims.(jwt.MapClaims)
	if !ok || !tok.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}
