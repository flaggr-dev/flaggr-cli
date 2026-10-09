package cmd

import (
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "projects",
		Short: "List projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := rest.ListProjects(cmd.Context())
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(projects)
				return nil
			}

			rows := make([][]string, len(projects))
			for i, p := range projects {
				rows[i] = []string{p.ID, p.Name, p.Slug, p.Role}
			}
			output.Table([]string{"ID", "NAME", "SLUG", "ROLE"}, rows)
			return nil
		},
	}
	return cmd
}
