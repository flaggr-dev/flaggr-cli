package cmd

import (
	"os"
	"strings"

	"github.com/flaggr-dev/flaggr-cli/internal/api"
	"github.com/flaggr-dev/flaggr-cli/internal/config"
	"github.com/flaggr-dev/flaggr-cli/internal/crash"
	"github.com/spf13/cobra"
)

var (
	cfg     *config.Config
	rest    *api.RESTClient
	conn    *api.ConnectClient
	jsonOut bool

	// CLIVersion and CLICommit are set from main.go
	CLIVersion = "dev"
	CLICommit  = "none"
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:     "flaggr",
		Short:   "Flaggr CLI — manage feature flags from the terminal",
		Long:    "Flaggr CLI provides flag management, evaluation via Connect-RPC, realtime metrics, health, audit log, and project export.",
		Version: CLIVersion,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			cfg = config.Load()
			apiURL, apiToken := config.Resolve(cfg)
			evalURL := config.ResolveEvalURL(cfg)
			rest = api.NewRESTClient(apiURL, apiToken)
			conn = api.NewConnectClient(evalURL, apiToken)
		},
	}

	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "Output as JSON")

	// Override default version template
	versionLine := "flaggr version " + CLIVersion
	if commitKnown(CLICommit) {
		versionLine += " (commit " + CLICommit + ")"
	}
	root.SetVersionTemplate(versionLine + "\n")

	root.AddCommand(
		newLoginCmd(),
		newLogoutCmd(),
		newStatusCmd(),
		newProjectsCmd(),
		newServicesCmd(),
		newFlagsCmd(),
		newEvalCmd(),
		newMetricsCmd(),
		newHealthCmd(),
		newAuditCmd(),
		newExportCmd(),
		newCrashReportsCmd(),
		newSourceMapsCmd(),
	)

	return root
}

// commitKnown reports whether commit names one: a `go install` of a tagged
// version records none, so flaggr reports only the version.
func commitKnown(commit string) bool {
	return commit != "" && commit != "none"
}

func Execute(version, commit string) {
	CLIVersion = version
	CLICommit = commit
	root := NewRootCmd()
	// A panic is reported to the Flaggr team when the user opted in, then
	// re-panics (internal/crash).
	defer crash.Guard{Version: version, Command: func() string { return commandPath(root, os.Args[1:]) }}.Recover()
	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// commandPath is the subcommand args select, such as "flags list": the
// command names only, never their arguments or flags.
func commandPath(root *cobra.Command, args []string) string {
	found, _, err := root.Find(args)
	if err != nil || found == nil || found == root {
		return ""
	}
	return strings.TrimPrefix(found.CommandPath(), root.Name()+" ")
}
