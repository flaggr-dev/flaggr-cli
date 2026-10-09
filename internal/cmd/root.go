package cmd

import (
	"fmt"
	"os"

	"github.com/flaggr-dev/flaggr-cli/internal/api"
	"github.com/flaggr-dev/flaggr-cli/internal/config"
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
	root.SetVersionTemplate(fmt.Sprintf("flaggr version %s (commit %s)\n", CLIVersion, CLICommit))

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
	)

	return root
}

func Execute(version, commit string) {
	CLIVersion = version
	CLICommit = commit
	if err := NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
