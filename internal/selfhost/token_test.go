package selfhost_test

import (
	"testing"
	"time"

	"github.com/Rethunk-Tech/citadel-cli/internal/selfhost"
)

func TestGenerateBootstrapToken(t *testing.T) {
	cfg := selfhost.Config{
		APIEndpoint: "https://citadel.example.com",
		SupabaseURL: "https://abc.supabase.co",
		AdminKey:    "service_role_key",
		JWTSecret:   "my-very-secret-jwt-signing-key",
	}

	token, err := selfhost.GenerateBootstrapToken(cfg, 24*time.Hour)
	if err != nil {
		t.Fatalf("GenerateBootstrapToken: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}

	// Validate using the same secret.
	claims, err := selfhost.ValidateBootstrapToken(token, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("ValidateBootstrapToken: %v", err)
	}

	if role, ok := claims["role"].(string); !ok || role != "service_role" {
		t.Errorf("claims[role] = %v; want service_role", claims["role"])
	}
	if iss, ok := claims["iss"].(string); !ok || iss != "supabase" {
		t.Errorf("claims[iss] = %v; want supabase", claims["iss"])
	}
	if sub, ok := claims["sub"].(string); !ok || sub != "citadel-bootstrap" {
		t.Errorf("claims[sub] = %v; want citadel-bootstrap", claims["sub"])
	}
}

func TestGenerateBootstrapTokenNoSecret(t *testing.T) {
	cfg := selfhost.Config{}
	_, err := selfhost.GenerateBootstrapToken(cfg, 0)
	if err == nil {
		t.Fatal("expected error for empty jwt_secret")
	}
}

func TestValidateBootstrapTokenWrongSecret(t *testing.T) {
	cfg := selfhost.Config{JWTSecret: "correct-secret"}
	token, err := selfhost.GenerateBootstrapToken(cfg, time.Hour)
	if err != nil {
		t.Fatalf("GenerateBootstrapToken: %v", err)
	}
	_, err = selfhost.ValidateBootstrapToken(token, "wrong-secret")
	if err == nil {
		t.Fatal("expected error for wrong jwt_secret")
	}
}

func TestValidateBootstrapToken_EmptySecret(t *testing.T) {
	_, err := selfhost.ValidateBootstrapToken("any.token.string", "")
	if err == nil {
		t.Fatal("expected error for empty jwtSecret")
	}
}

func TestGenerateBootstrapTokenDefaultDuration(t *testing.T) {
	cfg := selfhost.Config{JWTSecret: "test-secret"}

	// duration=0 should use default (7 days)
	token, err := selfhost.GenerateBootstrapToken(cfg, 0)
	if err != nil {
		t.Fatalf("GenerateBootstrapToken with zero duration: %v", err)
	}
	claims, err := selfhost.ValidateBootstrapToken(token, cfg.JWTSecret)
	if err != nil {
		t.Fatalf("ValidateBootstrapToken: %v", err)
	}

	expF, ok := claims["exp"].(float64)
	if !ok {
		t.Fatal("exp claim not a float64")
	}
	iatF, ok := claims["iat"].(float64)
	if !ok {
		t.Fatal("iat claim not a float64")
	}
	delta := expF - iatF
	wantDelta := float64((7 * 24 * time.Hour).Seconds())
	if delta != wantDelta {
		t.Errorf("token validity = %v seconds; want %v (7 days)", delta, wantDelta)
	}
}
