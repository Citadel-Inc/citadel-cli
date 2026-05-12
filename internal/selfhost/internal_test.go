package selfhost

import "testing"

func TestTruncate_Short(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Fatalf("truncate short: got %q want %q", got, "hello")
	}
}

func TestTruncate_Exact(t *testing.T) {
	if got := truncate("hello", 5); got != "hello" {
		t.Fatalf("truncate exact: got %q want %q", got, "hello")
	}
}

func TestTruncate_Long(t *testing.T) {
	got := truncate("hello world", 5)
	if got != "hello..." {
		t.Fatalf("truncate long: got %q want %q", got, "hello...")
	}
}

func TestDerivePostgresURL(t *testing.T) {
	if got := derivePostgresURL("https://abc.supabase.co", ""); got != "" {
		t.Fatalf("derivePostgresURL: got %q want empty", got)
	}
	if got := derivePostgresURL("https://self.hosted.example.com", "pass"); got != "" {
		t.Fatalf("derivePostgresURL self-hosted: got %q want empty", got)
	}
}

func TestCountApplied(t *testing.T) {
	out := "Applying migration 20260101000000_init.sql\nApplying migration 20260102000000_users.sql\nSome other line"
	if n := countApplied(out); n != 2 {
		t.Fatalf("countApplied: got %d want 2", n)
	}
	if n := countApplied("nothing here"); n != 0 {
		t.Fatalf("countApplied empty: got %d want 0", n)
	}
}

func TestRedactedAdminKey_Long(t *testing.T) {
	c := Config{AdminKey: "abcdefgh1234"}
	got := c.redactedAdminKey()
	if got == c.AdminKey {
		t.Fatal("redactedAdminKey should not return raw key")
	}
	if got == "***" {
		t.Fatal("long key should not be fully masked")
	}
}

func TestRedactedAdminKey_Short(t *testing.T) {
	c := Config{AdminKey: "short"}
	if got := c.redactedAdminKey(); got != "***" {
		t.Fatalf("short key: got %q want ***", got)
	}
}

func TestRedactedJWTSecret_Set(t *testing.T) {
	c := Config{JWTSecret: "super-secret"}
	if got := c.redactedJWTSecret(); got != "***" {
		t.Fatalf("set jwt: got %q want ***", got)
	}
}

func TestRedactedJWTSecret_Empty(t *testing.T) {
	c := Config{}
	if got := c.redactedJWTSecret(); got != "(not set)" {
		t.Fatalf("empty jwt: got %q want (not set)", got)
	}
}
