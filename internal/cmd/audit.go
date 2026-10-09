package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAuditCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "audit",
		Short: "Audit log and flag change history with diffs",
	}

	cmd.AddCommand(
		newAuditLogCmd(),
		newAuditHistoryCmd(),
	)

	return cmd
}

func newAuditLogCmd() *cobra.Command {
	var projectID, action, resourceType, search, cursor string
	var limit int

	cmd := &cobra.Command{
		Use:   "log",
		Short: "List audit log entries for a project (with change summaries + diffs)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := rest.GetAuditLog(cmd.Context(), projectID, limit, cursor, action, resourceType, search)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(resp)
				return nil
			}

			if len(resp.Logs) == 0 {
				output.Success("No audit events found.")
				return nil
			}

			rows := make([][]string, len(resp.Logs))
			for i, l := range resp.Logs {
				actor := l.UserEmail
				if actor == "" {
					actor = l.UserName
				}
				summary := l.Summary
				if summary == "" && len(l.Changes) > 0 {
					summary = fmt.Sprintf("%d change(s)", len(l.Changes))
				}
				name := l.ResourceName
				if name == "" {
					name = l.ResourceID
				}
				rows[i] = []string{l.Timestamp, l.Action, name, actor, summary}
			}
			output.Table([]string{"TIMESTAMP", "ACTION", "RESOURCE", "ACTOR", "SUMMARY"}, rows)

			// Print diffs compactly below the table
			for _, l := range resp.Logs {
				if len(l.Changes) == 0 {
					continue
				}
				name := l.ResourceName
				if name == "" {
					name = l.ResourceID
				}
				fmt.Printf("\n%s %s (%s):\n", l.Action, name, l.Timestamp)
				for _, c := range l.Changes {
					path := c.Path
					if path == "" {
						path = c.Field
					}
					oldB, _ := json.Marshal(c.OldValue)
					newB, _ := json.Marshal(c.NewValue)
					switch c.Type {
					case "added":
						fmt.Printf("  + %s = %s\n", path, string(newB))
					case "removed":
						fmt.Printf("  - %s was %s\n", path, string(oldB))
					default:
						fmt.Printf("  ~ %s: %s → %s\n", path, string(oldB), string(newB))
					}
				}
			}

			if resp.HasMore && resp.NextCursor != nil {
				fmt.Printf("\n(more available — pass --cursor %d)\n", *resp.NextCursor)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	cmd.Flags().StringVar(&action, "action", "", "Filter by action (e.g. flag.update)")
	cmd.Flags().StringVar(&resourceType, "resource-type", "", "Filter by resource type (flag, service, ...)")
	cmd.Flags().StringVar(&search, "search", "", "Text search across resources/actors")
	cmd.Flags().StringVar(&cursor, "cursor", "", "Pagination cursor from previous output")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max entries (1-100)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func newAuditHistoryCmd() *cobra.Command {
	var key, serviceID, environment string
	var limit, offset int

	cmd := &cobra.Command{
		Use:   "history",
		Short: "Show version history for a flag with per-version diffs",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := rest.GetFlagHistory(cmd.Context(), key, serviceID, environment, limit, offset)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(resp)
				return nil
			}

			if len(resp.Versions) == 0 {
				output.Success("No version history.")
				return nil
			}

			rows := make([][]string, len(resp.Versions))
			for i, v := range resp.Versions {
				actor := v.ChangedByName
				if actor == "" {
					actor = v.ChangedByEmail
				}
				if actor == "" {
					actor = v.ChangedBy
				}
				rows[i] = []string{
					fmt.Sprintf("v%d", v.Version), v.ChangeType, actor, v.Timestamp, v.DiffSummary,
				}
			}
			output.Table([]string{"VERSION", "TYPE", "BY", "AT", "DIFF"}, rows)

			for _, v := range resp.Versions {
				if len(v.Diffs) == 0 {
					continue
				}
				fmt.Printf("\nv%d (%s):\n", v.Version, v.ChangeType)
				for _, c := range v.Diffs {
					path := c.Path
					if path == "" {
						path = c.Field
					}
					oldB, _ := json.Marshal(c.OldValue)
					newB, _ := json.Marshal(c.NewValue)
					switch c.Type {
					case "added":
						fmt.Printf("  + %s = %s\n", path, string(newB))
					case "removed":
						fmt.Printf("  - %s was %s\n", path, string(oldB))
					default:
						fmt.Printf("  ~ %s: %s → %s\n", path, string(oldB), string(newB))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "development", "Environment")
	cmd.Flags().IntVar(&limit, "limit", 20, "Max versions")
	cmd.Flags().IntVar(&offset, "offset", 0, "Offset")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}
