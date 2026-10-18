package cmd

import (
	"fmt"
	"os"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
	"github.com/flaggr-dev/flaggr-cli/internal/crash"
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newCrashReportsCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "crash-reports [on|off|status]",
		Short:     "Send the CLI's crashes to the Flaggr team (off by default)",
		ValidArgs: []string{"on", "off", "status"},
		Args:      cobra.MatchAll(cobra.MaximumNArgs(1), cobra.OnlyValidArgs),
		Long: `Turn crash reports on or off, or show whether they are on.

When the CLI crashes (a panic, not an error such as a bad token or a network
failure), a report goes to the Flaggr team: the panic's message and stack,
the CLI's version, the subcommand that ran (never its arguments or flags),
and your OS, architecture and Go version. Never your token, URLs, flag keys
or output.

Reports are off unless you turn them on. FLAGGR_CRASH_REPORTS=1 or 0
overrides this setting; DO_NOT_TRACK=1 and CI environments always turn
reports off.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			action := "status"
			if len(args) == 1 {
				action = args[0]
			}
			c := config.Load()
			if action == "on" || action == "off" {
				on := action == "on"
				c.CrashReports = &on
				if err := config.Save(c); err != nil {
					return fmt.Errorf("failed to save config: %w", err)
				}
			}
			on, reason := crash.Enabled(c, os.Getenv)
			state := "off"
			if on {
				state = "on"
			}
			if jsonOut {
				output.JSON(map[string]any{"crashReports": on, "reason": reason})
				return nil
			}
			if action != "status" {
				output.Success("Crash reports " + action)
			}
			fmt.Printf("Crash reports are %s (%s).\n", state, reason)
			return nil
		},
	}
}
