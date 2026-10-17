// Package crash reports the CLI's own crashes (panics) to the Flaggr team,
// when the user has opted in.
//
// Reports go to https://flaggr.dev/api/ingest/cli-crashes, which takes no
// credential and stores them only in Flaggr's own project. A report carries
// the panic's message and stack, the CLI's version, the subcommand that ran
// (such as "flags list", never its arguments or flags), and the OS,
// architecture and Go version. Never the API token, URLs, flag keys or
// output. Flaggr scrubs tokens and addresses from the message and stack too.
//
// Reporting is off unless the user turns it on (`flaggr crash-reports on`,
// or FLAGGR_CRASH_REPORTS=1). DO_NOT_TRACK=1 and a CI environment always
// turn it off. Errors a command returns (a bad token, a network failure)
// are not crashes and are never reported.
package crash

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
	"github.com/flaggr-dev/flaggr-go/errorreport"
)

// DefaultIngestURL receives the CLI's crash reports.
const DefaultIngestURL = "https://flaggr.dev/api/ingest/cli-crashes"

// Enabled reports whether crash reports are on, and why, from the
// environment and the config. getenv is os.Getenv outside tests.
func Enabled(cfg *config.Config, getenv func(string) string) (bool, string) {
	if v := getenv("DO_NOT_TRACK"); v != "" && v != "0" && !strings.EqualFold(v, "false") {
		return false, "DO_NOT_TRACK is set"
	}
	if v := getenv("CI"); v != "" && v != "0" && !strings.EqualFold(v, "false") {
		return false, "running in CI"
	}
	switch strings.ToLower(strings.TrimSpace(getenv("FLAGGR_CRASH_REPORTS"))) {
	case "1", "true", "on", "yes":
		return true, "FLAGGR_CRASH_REPORTS is on"
	case "0", "false", "off", "no":
		return false, "FLAGGR_CRASH_REPORTS is off"
	}
	if cfg != nil && cfg.CrashReports != nil {
		if *cfg.CrashReports {
			return true, "turned on with `flaggr crash-reports on`"
		}
		return false, "turned off with `flaggr crash-reports off`"
	}
	return false, "off by default"
}

// Guard runs at the top of the CLI: deferred, it recovers a panic, reports
// it when crash reports are on (or says how to turn them on when they're
// off), and re-panics so the CLI still fails with Go's panic output and exit
// code. command names the subcommand that ran.
type Guard struct {
	Version string
	Command func() string
	Stderr  io.Writer
	Getenv  func(string) string
	// IngestURL overrides DefaultIngestURL (FLAGGR_CRASH_REPORTS_URL too).
	IngestURL string
}

// Recover must be deferred directly: defer guard.Recover().
func (g Guard) Recover() {
	value := recover()
	if value == nil {
		return
	}
	getenv := g.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	stderr := g.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	if on, _ := Enabled(config.Load(), getenv); on {
		Report(value, g.Version, g.command(), g.ingestURL(getenv))
		fmt.Fprintln(stderr, "flaggr crashed. A crash report was sent to the Flaggr team; thank you.")
	} else {
		fmt.Fprintln(stderr, "flaggr crashed. Please report it at https://github.com/flaggr-dev/flaggr-cli/issues, or run `flaggr crash-reports on` to send crash reports automatically.")
	}
	panic(value)
}

func (g Guard) command() string {
	if g.Command == nil {
		return ""
	}
	return g.Command()
}

func (g Guard) ingestURL(getenv func(string) string) string {
	if v := getenv("FLAGGR_CRASH_REPORTS_URL"); v != "" {
		return v
	}
	if g.IngestURL != "" {
		return g.IngestURL
	}
	return DefaultIngestURL
}

// Report sends one panic and waits up to 3 s for it to go out.
func Report(value any, version, command, ingestURL string) {
	reporter := errorreport.New(errorreport.Options{
		IngestURL:   ingestURL,
		Release:     "flaggr-cli@" + strings.TrimPrefix(version, "v"),
		Environment: "cli",
	})
	reporter.CapturePanic(value, errorreport.WithExtra(map[string]any{
		"os":        runtime.GOOS,
		"arch":      runtime.GOARCH,
		"goVersion": runtime.Version(),
		"command":   command,
	}))
	reporter.Flush(3 * time.Second)
	reporter.Close()
}
