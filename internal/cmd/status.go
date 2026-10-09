package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show CLI and server version, check API health",
		RunE: func(cmd *cobra.Command, args []string) error {
			fmt.Printf("CLI version:    %s (%s)\n", CLIVersion, CLICommit)

			data, err := rest.Health(cmd.Context())
			if err != nil {
				fmt.Println()
				output.Error("Cannot reach Flaggr API: " + err.Error())
				return err
			}

			var health struct {
				Status  string `json:"status"`
				Version struct {
					App         string `json:"app"`
					Commit      string `json:"commit"`
					Environment string `json:"environment"`
				} `json:"version"`
			}
			_ = json.Unmarshal(data, &health)

			if jsonOut {
				combined := map[string]any{
					"cli": map[string]string{
						"version": CLIVersion,
						"commit":  CLICommit,
					},
				}
				var raw map[string]any
				_ = json.Unmarshal(data, &raw)
				combined["server"] = raw
				output.JSON(combined)
				return nil
			}

			fmt.Printf("Server version: %s (%s)\n", health.Version.App, health.Version.Commit)
			fmt.Printf("Environment:    %s\n", health.Version.Environment)
			fmt.Println()

			if health.Status == "healthy" || health.Status == "ok" {
				output.Success("Flaggr is healthy")
			} else {
				output.Error("Status: " + health.Status)
			}
			return nil
		},
	}
}
