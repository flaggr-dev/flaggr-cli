// Command flaggr is the Flaggr command-line interface.
package main

import (
	"regexp"
	"runtime/debug"
	"strings"

	"github.com/flaggr-dev/flaggr-cli/internal/cmd"
)

// Set via ldflags at build time, as GoReleaser does (.goreleaser.yml):
//
//	go build -ldflags "-X main.version=0.5.0 -X main.commit=abc1234" ./cmd/flaggr
var (
	version = "dev"
	commit  = "none"
)

func main() {
	info, _ := debug.ReadBuildInfo()
	cmd.Execute(buildVersion(version, commit, info))
}

// pseudoVersionRevision matches a Go pseudo-version, such as
// v0.0.0-20261009005306-9bc23b6b9415, and captures its commit hash.
var pseudoVersionRevision = regexp.MustCompile(`^v\d+\.\d+\.\d+-(?:.*\.)?\d{14}-([0-9a-f]{12})(?:\+[0-9A-Za-z.-]+)?$`)

// buildVersion returns the version and commit flaggr reports. Release builds
// have both stamped by ldflags. A build that isn't stamped, such as
// `go install github.com/flaggr-dev/flaggr-cli/cmd/flaggr@v0.5.0`, reports the
// module version Go records in the binary ("v0.5.0" as "0.5.0") and the
// commit Go records for a build from a git checkout, or else the one a
// pseudo-version (`@latest` before a release, or `@main`) names. A tagged
// `go install` build has no commit: it stays "none", which `flaggr --version`
// leaves out.
func buildVersion(version, commit string, info *debug.BuildInfo) (string, string) {
	if info == nil {
		return version, commit
	}
	if version == "dev" {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			version = strings.TrimPrefix(v, "v")
		}
	}
	if commit == "none" {
		for _, setting := range info.Settings {
			if setting.Key == "vcs.revision" && setting.Value != "" {
				commit = setting.Value
			}
		}
		if commit == "none" {
			if m := pseudoVersionRevision.FindStringSubmatch(info.Main.Version); m != nil {
				commit = m[1]
			}
		}
		if commit != "none" && len(commit) > 7 {
			commit = commit[:7]
		}
	}
	return version, commit
}
