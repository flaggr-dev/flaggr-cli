package cmd

import (
	"encoding/json"
	"os"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newExportCmd() *cobra.Command {
	var projectID, outputFile string

	cmd := &cobra.Command{
		Use:   "export",
		Short: "Export project configuration as JSON",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := rest.Export(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			payload, err := json.MarshalIndent(data, "", "  ")
			if err != nil {
				return err
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, append(payload, '\n'), 0644); err != nil {
					return err
				}
				output.Success("Exported to " + outputFile)
			} else {
				output.JSON(data)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (defaults to stdout)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}
