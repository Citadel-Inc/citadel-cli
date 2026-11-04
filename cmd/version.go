package cmd

import "runtime/debug"

// resolveVersion lets `go install`, which sets no -ldflags, report the module
// version it installed. A linker-set version and a plain local build are left alone.
func resolveVersion(linked string, info func() (*debug.BuildInfo, bool)) string {
	if linked != "dev" {
		return linked
	}
	if bi, ok := info(); ok && bi.Main.Version != "" && bi.Main.Version != "(devel)" {
		return bi.Main.Version
	}
	return linked
}
