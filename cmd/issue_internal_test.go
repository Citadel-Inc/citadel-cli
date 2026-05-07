package cmd

import "testing"

func TestIssueBrowserURL(t *testing.T) {
	got := issueBrowserURL("https://mcp.src.land/", "acme/demo", 7)
	want := "https://mcp.src.land/acme/demo/issues/7"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeNamespacePath(t *testing.T) {
	got, err := normalizeNamespacePath("/acme/platform/demo/")
	if err != nil {
		t.Fatal(err)
	}
	if got != "acme/platform/demo" {
		t.Fatalf("got %q", got)
	}
}
