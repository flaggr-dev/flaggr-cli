package cmd

import (
	"io"
	"os"
	"testing"
)

// main passes the GoReleaser-stamped version and commit (the tag without its
// leading "v", and the short commit) to Execute; `flaggr --version` must
// print exactly those.
func TestExecuteReportsTheStampedVersion(t *testing.T) {
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
	Execute("0.5.0", "abc1234")
	os.Stdout = oldStdout
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	out, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "flaggr version 0.5.0 (commit abc1234)\n"; got != want {
		t.Fatalf("flaggr --version printed %q, want %q", got, want)
	}
}
