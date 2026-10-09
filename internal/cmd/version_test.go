package cmd

import (
	"io"
	"os"
	"testing"
)

// versionOutput runs `flaggr --version` with the version and commit main
// passes to Execute, and returns what it printed.
func versionOutput(t *testing.T, version, commit string) string {
	t.Helper()
	oldArgs, oldStdout := os.Args, os.Stdout
	oldVersion, oldCommit := CLIVersion, CLICommit
	t.Cleanup(func() {
		os.Args, os.Stdout = oldArgs, oldStdout
		CLIVersion, CLICommit = oldVersion, oldCommit
	})

	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	os.Args = []string{"flaggr", "--version"}
	os.Stdout = w
	Execute(version, commit)
	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(out)
}

// main passes the GoReleaser-stamped version and commit (the tag without its
// leading "v", and the short commit) to Execute; `flaggr --version` must
// print exactly those.
func TestExecuteReportsTheStampedVersion(t *testing.T) {
	if got, want := versionOutput(t, "0.5.0", "abc1234"), "flaggr version 0.5.0 (commit abc1234)\n"; got != want {
		t.Fatalf("flaggr --version printed %q, want %q", got, want)
	}
}

// A `go install` of a tagged version records no commit: main passes "none",
// which isn't printed.
func TestExecuteLeavesOutAnUnknownCommit(t *testing.T) {
	for _, commit := range []string{"none", ""} {
		if got, want := versionOutput(t, "0.5.0", commit), "flaggr version 0.5.0\n"; got != want {
			t.Errorf("commit %q: flaggr --version printed %q, want %q", commit, got, want)
		}
	}
}
