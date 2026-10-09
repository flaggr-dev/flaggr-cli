// Command flaggr is the Flaggr command-line interface.
package main

import (
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

// buildVersion returns the version and commit flaggr reports. Release builds
// have both stamped by ldflags. A build that isn't stamped, such as
// `go install github.com/flaggr-dev/flaggr-cli/cmd/flaggr@v0.5.0`, reports the
// module version Go records in the binary ("v0.5.0" as "0.5.0") and, when the
// build came from a git checkout, the commit Go records.
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
				if len(commit) > 7 {
					commit = commit[:7]
				}
			}
		}
	}
	return version, commit
}
