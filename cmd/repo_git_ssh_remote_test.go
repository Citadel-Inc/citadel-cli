package cmd

import "testing"

func TestGitSSHRemotePreservesMultiSegmentNamespace(t *testing.T) {
	got, err := gitSSHRemote("http://127.0.0.1", "acme/project", "demo")
	if err != nil {
		t.Fatal(err)
	}
	want := "git@127.0.0.1:acme/project/demo.git"
	if got != want {
		t.Fatalf("gitSSHRemote() = %q, want %q", got, want)
	}
}

func TestGitSSHRemoteFlatNamespace(t *testing.T) {
	got, err := gitSSHRemote("https://src.land", "acme", "demo")
	if err != nil {
		t.Fatal(err)
	}
	want := "git@src.land:acme/demo.git"
	if got != want {
		t.Fatalf("gitSSHRemote() = %q, want %q", got, want)
	}
}
