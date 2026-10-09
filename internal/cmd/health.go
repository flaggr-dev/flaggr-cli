package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newHealthCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "health",
		Short: "Health scores for flags and projects",
	}

	cmd.AddCommand(
		newHealthFlagCmd(),
		newHealthProjectCmd(),
	)

	return cmd
}

func newHealthFlagCmd() *cobra.Command {
	var key, serviceID string

	cmd := &cobra.Command{
		Use:   "flag",
		Short: "Show health score for a flag (error rate, latency, volume, staleness)",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := rest.GetFlagHealth(cmd.Context(), key, serviceID)
			if err != nil {
				return err
			}

			if jsonOut {
				var v any
				_ = json.Unmarshal(data, &v)
				output.JSON(v)
				return nil
			}

			var h struct {
				Score  float64 `json:"overallScore"`
				Status string  `json:"status"`
			}
			_ = json.Unmarshal(data, &h)
			// Full payload varies by scorer version — print compact + full JSON fallback
			if h.Status != "" {
				fmt.Printf("Flag %q: score %.0f (%s)\n", key, h.Score, h.Status)
			}
			var v any
			_ = json.Unmarshal(data, &v)
			output.JSON(v)
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newHealthProjectCmd() *cobra.Command {
	var projectID string

	cmd := &cobra.Command{
		Use:   "project",
		Short: "Show aggregated project health (services, top issues, volume)",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := rest.GetProjectHealth(cmd.Context(), projectID)
			if err != nil {
				return err
			}

			if jsonOut {
				var v any
				_ = json.Unmarshal(data, &v)
				output.JSON(v)
				return nil
			}

			var h struct {
				OverallScore     float64 `json:"overallScore"`
				Status           string  `json:"status"`
				TotalFlags       int     `json:"totalFlags"`
				TotalEvaluations int     `json:"totalEvaluations"`
				Services         []struct {
					ServiceID    string  `json:"serviceId"`
					ServiceName  string  `json:"serviceName"`
					OverallScore float64 `json:"overallScore"`
					Status       string  `json:"status"`
					FlagCount    int     `json:"flagCount"`
				} `json:"services"`
				TopIssues []struct {
					FlagKey        string  `json:"flagKey"`
					ServiceName    string  `json:"serviceName"`
					Score          float64 `json:"score"`
					Status         string  `json:"status"`
					WorstDimension string  `json:"worstDimension"`
				} `json:"topIssues"`
			}
			if err := json.Unmarshal(data, &h); err != nil {
				return err
			}

			fmt.Printf("Project health: %.0f (%s) — %d flags, %d evals\n\n",
				h.OverallScore, h.Status, h.TotalFlags, h.TotalEvaluations)

			if len(h.Services) > 0 {
				fmt.Println("Services:")
				rows := make([][]string, len(h.Services))
				for i, s := range h.Services {
					rows[i] = []string{s.ServiceName, fmt.Sprintf("%.0f", s.OverallScore), s.Status, fmt.Sprintf("%d", s.FlagCount)}
				}
				output.Table([]string{"SERVICE", "SCORE", "STATUS", "FLAGS"}, rows)
			}

			if len(h.TopIssues) > 0 {
				fmt.Println("\nTop issues:")
				rows := make([][]string, len(h.TopIssues))
				for i, t := range h.TopIssues {
					rows[i] = []string{t.FlagKey, t.ServiceName, fmt.Sprintf("%.0f", t.Score), t.Status, t.WorstDimension}
				}
				output.Table([]string{"FLAG", "SERVICE", "SCORE", "STATUS", "WORST"}, rows)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}
