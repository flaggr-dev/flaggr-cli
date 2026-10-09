package cmd

import (
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newServicesCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "services",
		Short: "List services for a project",
		RunE: func(cmd *cobra.Command, args []string) error {
			services, err := rest.ListServices(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(services)
				return nil
			}

			rows := make([][]string, len(services))
			for i, s := range services {
				rows[i] = []string{s.ID, s.Name, s.ProjectID}
			}
			output.Table([]string{"ID", "NAME", "PROJECT"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}
