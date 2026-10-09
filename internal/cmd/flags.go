package cmd

import (
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newFlagsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "flags",
		Short: "Manage feature flags",
	}

	cmd.AddCommand(
		newFlagsListCmd(),
		newFlagsCreateCmd(),
		newFlagsToggleCmd(),
		newFlagsDeleteCmd(),
		newFlagsStaleCmd(),
	)

	return cmd
}

func newFlagsListCmd() *cobra.Command {
	var projectID, serviceID, environment string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List flags",
		RunE: func(cmd *cobra.Command, args []string) error {
			flags, err := rest.ListFlags(cmd.Context(), projectID, serviceID, environment)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(flags)
				return nil
			}

			rows := make([][]string, len(flags))
			for i, f := range flags {
				enabled := "off"
				if f.Enabled {
					enabled = "on"
				}
				rows[i] = []string{f.Key, f.Name, f.Type, enabled, f.Environment}
			}
			output.Table([]string{"KEY", "NAME", "TYPE", "ENABLED", "ENV"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func newFlagsCreateCmd() *cobra.Command {
	var (
		key, name, serviceID, flagType, defaultVal, environment string
		enabled                                                 bool
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a flag",
		RunE: func(cmd *cobra.Command, args []string) error {
			var dv any
			switch flagType {
			case "boolean":
				dv = defaultVal == "true"
			case "number":
				v, err := strconv.ParseFloat(defaultVal, 64)
				if err != nil {
					return fmt.Errorf("invalid number default: %s", defaultVal)
				}
				dv = v
			case "object":
				if err := json.Unmarshal([]byte(defaultVal), &dv); err != nil {
					return fmt.Errorf("invalid JSON default: %w", err)
				}
			default:
				dv = defaultVal
			}

			data := map[string]any{
				"key":          key,
				"name":         name,
				"serviceId":    serviceID,
				"type":         flagType,
				"defaultValue": dv,
				"enabled":      enabled,
			}
			if environment != "" {
				data["environment"] = environment
			}

			result, err := rest.CreateFlag(cmd.Context(), data)
			if err != nil {
				return err
			}

			if jsonOut {
				var v any
				_ = json.Unmarshal(result, &v)
				output.JSON(v)
			} else {
				output.Success(fmt.Sprintf("Flag %q created", key))
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&name, "name", "n", "", "Flag name (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&flagType, "type", "t", "boolean", "Flag type (boolean|string|number|object)")
	cmd.Flags().StringVarP(&defaultVal, "default", "d", "false", "Default value")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Create enabled")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newFlagsToggleCmd() *cobra.Command {
	var key, serviceID, environment string

	cmd := &cobra.Command{
		Use:   "toggle",
		Short: "Toggle a flag on/off",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := rest.ToggleFlag(cmd.Context(), key, serviceID, environment)
			if err != nil {
				return err
			}

			if jsonOut {
				var v any
				_ = json.Unmarshal(result, &v)
				output.JSON(v)
				return nil
			}

			var flag struct{ Enabled bool }
			_ = json.Unmarshal(result, &flag)
			state := "OFF"
			if flag.Enabled {
				state = "ON"
			}
			output.Success(fmt.Sprintf("Flag %q is now %s", key, state))
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newFlagsDeleteCmd() *cobra.Command {
	var key, serviceID, environment string

	cmd := &cobra.Command{
		Use:   "delete",
		Short: "Delete a flag",
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := rest.DeleteFlag(cmd.Context(), key, serviceID, environment); err != nil {
				return err
			}
			output.Success(fmt.Sprintf("Flag %q deleted", key))
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newFlagsStaleCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "stale",
		Short: "List stale and inactive flags",
		Long:  "Show flags that have not been evaluated recently and may be candidates for cleanup or archival.",
		RunE: func(cmd *cobra.Command, args []string) error {
			result, err := rest.GetStaleFlags(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(result)
				return nil
			}

			if result.Total == 0 {
				output.Success("No stale or inactive flags found.")
				return nil
			}

			fmt.Printf("Found %d flag(s) needing attention (%d stale, %d inactive)\n\n",
				result.Total, result.StaleCount, result.InactiveCount)

			rows := make([][]string, len(result.Flags))
			for i, f := range result.Flags {
				days := "never"
				if f.DaysSinceLastEvaluation >= 0 {
					days = fmt.Sprintf("%.0f", f.DaysSinceLastEvaluation)
				}

				rec := "Monitor or archive"
				if f.Status == "stale" || f.DaysSinceLastEvaluation >= 30 {
					rec = "Remove flag, hard-code value"
				}

				rows[i] = []string{f.FlagKey, f.Status, days, rec}
			}

			output.Table([]string{"FLAG KEY", "STATUS", "DAYS INACTIVE", "RECOMMENDATION"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}
