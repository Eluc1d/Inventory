package main

import (
	"log"
	"os/exec"
	"strings"
)

// version is the build identifier shown next to the logo in the topbar.
// It's normally baked in at build time via -ldflags "-X main.version=...."
// (see the Dockerfile's build stage); left at its default here,
// detectVersion falls back to asking git directly at startup, which covers
// `go run .` and the dev container (whose compose file bind-mounts the
// repo's .git so this resolves correctly there too).
var version = "dev"

// detectVersion resolves the running build's version. A version baked in via
// -ldflags always wins; otherwise it shells out to `git describe` once at
// startup. Failures (no git binary, no .git available, no tags yet) fall
// back to the "dev" default silently — this is a cosmetic label, not
// something worth failing startup over.
func detectVersion() string {
	if version != "dev" {
		return version
	}
	out, err := exec.Command("git", "describe", "--tags", "--always").Output()
	if err != nil {
		log.Printf("detectVersion: git describe unavailable, using %q: %v", version, err)
		return version
	}
	v := strings.TrimSpace(string(out))
	if v == "" {
		return version
	}
	return v
}
