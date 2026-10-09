package main

import (
	"os"
	"runtime/debug"
	"strings"
	"testing"
)

// The release workflow (.github/workflows/release.yml) builds with
// GoReleaser, which stamps the tag into these variables through -X ldflags.
// Renaming either side would ship binaries that report "dev".
func TestGoReleaserStampsTheVersionAndCommitMainPassesOn(t *testing.T) {
	cfg, err := os.ReadFile("../../.goreleaser.yml")
	if err != nil {
		t.Fatalf("read .goreleaser.yml: %v", err)
	}
	for _, ldflag := range []string{
		"-X main.version={{ .Version }}",
		"-X main.commit={{ .ShortCommit }}",
	} {
		if !strings.Contains(string(cfg), ldflag) {
			t.Errorf(".goreleaser.yml ldflags lack %q", ldflag)
		}
	}
	if !strings.Contains(string(cfg), "main: ./cmd/flaggr") {
		t.Errorf(".goreleaser.yml must build ./cmd/flaggr, the package that declares version and commit")
	}
	if !strings.Contains(string(cfg), "binary: flaggr") {
		t.Errorf(".goreleaser.yml must name the binary flaggr")
	}

	// Unstamped builds (go build, go test) keep the defaults.
	if version != "dev" || commit != "none" {
		t.Errorf("unstamped defaults changed: version=%q commit=%q", version, commit)
	}
}

func TestBuildVersion(t *testing.T) {
	revision := []debug.BuildSetting{{Key: "vcs.revision", Value: "0123456789abcdef0123456789abcdef01234567"}}
	tests := []struct {
		name            string
		version, commit string
		info            *debug.BuildInfo
		wantVersion     string
		wantCommit      string
	}{
		{
			name:    "a release build keeps what GoReleaser stamped",
			version: "0.5.0", commit: "abc1234",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v0.5.0"}, Settings: revision},
			wantVersion: "0.5.0", wantCommit: "abc1234",
		},
		{
			name:    "go install of a tagged version reports the module version",
			version: "dev", commit: "none",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v0.5.0"}},
			wantVersion: "0.5.0", wantCommit: "none",
		},
		{
			name:    "a build from a git checkout reports Go's version and commit",
			version: "dev", commit: "none",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "v0.5.1-0.20261009000000-0123456789ab+dirty"}, Settings: revision},
			wantVersion: "0.5.1-0.20261009000000-0123456789ab+dirty", wantCommit: "0123456",
		},
		{
			name:    "go run and go test builds have no module version",
			version: "dev", commit: "none",
			info:        &debug.BuildInfo{Main: debug.Module{Version: "(devel)"}},
			wantVersion: "dev", wantCommit: "none",
		},
		{
			name:    "no build information",
			version: "dev", commit: "none",
			info:        nil,
			wantVersion: "dev", wantCommit: "none",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotVersion, gotCommit := buildVersion(tt.version, tt.commit, tt.info)
			if gotVersion != tt.wantVersion || gotCommit != tt.wantCommit {
				t.Errorf("buildVersion() = (%q, %q), want (%q, %q)", gotVersion, gotCommit, tt.wantVersion, tt.wantCommit)
			}
		})
	}
}
