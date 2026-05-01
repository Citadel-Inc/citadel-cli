package cmd

import (
	"reflect"
	"testing"
)

func TestCoerceArg(t *testing.T) {
	cases := []struct {
		in   string
		want any
	}{
		// Strings.
		{"hello", "hello"},
		{"", ""},
		{"damon", "damon"},

		// Booleans.
		{"true", true},
		{"false", false},
		{"True", "True"}, // canonical form only
		{"yes", "yes"},

		// Integers.
		{"5", int64(5)},
		{"0", int64(0)},
		{"-7", int64(-7)},
		{"100", int64(100)},

		// Leading-zero IDs / zip codes stay as strings (don't clobber).
		{"07823", "07823"},
		{"00", "00"},

		// Floats.
		{"1.5", 1.5},
		{"-3.14", -3.14},

		// Float-shaped that should NOT parse.
		{".5", ".5"},
		{"5.", "5."},
		{"1.2.3", "1.2.3"},

		// CSV → array, with each element coerced.
		{"a,b,c", []any{"a", "b", "c"}},
		{"1,2,3", []any{int64(1), int64(2), int64(3)}},
		{"1,foo,3", []any{int64(1), "foo", int64(3)}},
		{"true,false", []any{true, false}},

		// Single-element trailing comma still produces array.
		{"x,", []any{"x", ""}},
	}
	for _, tc := range cases {
		got := coerceArg(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("coerceArg(%q) = %#v want %#v", tc.in, got, tc.want)
		}
	}
}

func TestPickToken(t *testing.T) {
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("flag", "jwt"); got != "flag" {
		t.Errorf("flag wins: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "envtok")
	if got := pickToken("", "jwt"); got != "envtok" {
		t.Errorf("env beats jwt: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("", "jwt"); got != "jwt" {
		t.Errorf("jwt fallback: got %q", got)
	}
	t.Setenv("CITADEL_AGENT_TOKEN", "")
	if got := pickToken("", ""); got != "" {
		t.Errorf("nothing: got %q", got)
	}
}

func TestResolveMCPURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"https://api.src.land", "https://mcp.src.land/mcp"},
		{"https://api.src.land/", "https://mcp.src.land/mcp"},
		{"https://mcp.src.land", "https://mcp.src.land/mcp"},
		{"http://localhost:8080", "http://localhost:8080/mcp"},
		{"http://localhost:8080/", "http://localhost:8080/mcp"},
	}
	for _, tc := range cases {
		got := resolveMCPURL(tc.in)
		if got != tc.want {
			t.Errorf("resolveMCPURL(%q) = %q want %q", tc.in, got, tc.want)
		}
	}
}
