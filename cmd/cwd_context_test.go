package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestParseOriginIntoRepo_HTTPS(t *testing.T) {
	hosts := mergeCitadelHosts()
	ns, slug, err := parseOriginIntoRepo("https://src.land/acme/widget.git", hosts)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" || slug != "widget" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestParseOriginIntoRepo_HTTPSNoDotGit(t *testing.T) {
	hosts := mergeCitadelHosts()
	ns, slug, err := parseOriginIntoRepo("https://git.src.land/acme/widget", hosts)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" || slug != "widget" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestParseOriginIntoRepo_SSH(t *testing.T) {
	hosts := mergeCitadelHosts()
	ns, slug, err := parseOriginIntoRepo("git@src.land:acme/widget.git", hosts)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" || slug != "widget" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestParseOrigin_CustomHostViaEnv(t *testing.T) {
	t.Setenv(citadelGitHostsEnv, "github.com")
	hosts := mergeCitadelHosts()
	ns, slug, err := parseOriginIntoRepo("https://github.com/acme/widget.git", hosts)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" || slug != "widget" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestMergeCitadelHostsEnv(t *testing.T) {
	t.Setenv(citadelGitHostsEnv, "git.example.com, Example.ORG ")
	hosts := mergeCitadelHosts()
	if _, ok := hosts["git.example.com"]; !ok {
		t.Fatal("missing env host")
	}
	if _, ok := hosts["example.org"]; !ok {
		t.Fatal("expected lowercased trimmed host")
	}
}

func TestResolveRepoFlag_ExplicitR(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("repo", "ns/slug"); err != nil {
		t.Fatal(err)
	}
	ns, slug, err := resolveRepoFlag(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "ns" || slug != "slug" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestResolveRepoFlag_EnvOnly(t *testing.T) {
	t.Setenv(citadelRepoEnv, "envns/envslug")
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	ns, slug, err := resolveRepoFlag(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "envns" || slug != "envslug" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestResolveRepoFlag_NoCwdRepoOptOut(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("no-cwd-repo", "true"); err != nil {
		t.Fatal(err)
	}
	_, _, err := resolveRepoFlag(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRepoFromPosOrFlag_PositionalWinsOverInference(t *testing.T) {
	// Positional is used when -R empty; does not need git.
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, "posns/posslug")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "posns" || slug != "posslug" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestResolveRepoFromPosOrFlag_RFlagCanonical(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("repo", "rns/rslug"); err != nil {
		t.Fatal(err)
	}
	ns, slug, err := resolveRepoFromPosOrFlag(cmd, "posns/posslug")
	if err != nil {
		t.Fatal(err)
	}
	if ns != "rns" || slug != "rslug" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestGitOriginIntegration(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "remote", "add", "origin", "git@src.land:testns/testslug.git").Run(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "")
	cmd.SetContext(context.Background())

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	ns, slug, err := resolveRepoFlag(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "testns" || slug != "testslug" {
		t.Fatalf("got %s/%s", ns, slug)
	}
}

func TestInferenceHintSkippedWhenStderrNotTTYFile(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not on PATH")
	}
	dir := t.TempDir()
	if err := exec.Command("git", "-C", dir, "init").Run(); err != nil {
		t.Fatal(err)
	}
	if err := exec.Command("git", "-C", dir, "remote", "add", "origin", "https://src.land/hintns/hintslug").Run(); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "")
	cmd.SetContext(context.Background())
	cmd.SetErr(&bytes.Buffer{}) // non-TTY stderr → no hint

	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	var stderr bytes.Buffer
	cmd.SetErr(&stderr)
	_, _, err = resolveRepoFlag(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(stderr.String(), "Inferred") {
		t.Fatalf("unexpected hint on non-terminal stderr: %q", stderr.String())
	}
}

func TestResolveRepoNamespaceForCompletion_RFlagFullPath(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("repo", "acme/widget"); err != nil {
		t.Fatal(err)
	}
	ns, err := ResolveRepoNamespaceForCompletion(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "acme" {
		t.Fatalf("got %q", ns)
	}
}

func TestResolveRepoNamespaceForCompletion_RFlagPartialOrgSlash(t *testing.T) {
	// Typing "-R myorg/" scopes completion to namespace myorg even before repo slug is present.
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("repo", "myorg/"); err != nil {
		t.Fatal(err)
	}
	ns, err := ResolveRepoNamespaceForCompletion(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "myorg" {
		t.Fatalf("got %q", ns)
	}
}

func TestResolveRepoNamespaceForCompletion_RFlagBareOrgFails(t *testing.T) {
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	if err := cmd.Flags().Set("repo", "onlyorg"); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveRepoNamespaceForCompletion(cmd)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveRepoNamespaceForCompletion_EnvRepo(t *testing.T) {
	t.Setenv(citadelRepoEnv, "envns/envslug")
	cmd := &cobra.Command{}
	cmd.Flags().StringP("repo", "R", "", "")
	cmd.Flags().Bool("no-cwd-repo", false, "")
	ns, err := ResolveRepoNamespaceForCompletion(cmd)
	if err != nil {
		t.Fatal(err)
	}
	if ns != "envns" {
		t.Fatalf("got %q", ns)
	}
}
