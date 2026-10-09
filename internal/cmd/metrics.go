package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newMetricsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "metrics",
		Short: "Flag evaluation metrics and analytics",
	}

	cmd.AddCommand(
		newMetricsFlagCmd(),
		newMetricsWatchCmd(),
		newMetricsHeatmapCmd(),
	)

	return cmd
}

func newMetricsFlagCmd() *cobra.Command {
	var key, serviceID, environment, rng string

	cmd := &cobra.Command{
		Use:   "flag",
		Short: "Show evaluation analytics for a flag (volume, errors, latency, breakdowns)",
		RunE: func(cmd *cobra.Command, args []string) error {
			data, err := rest.GetFlagMetrics(cmd.Context(), key, serviceID, environment, rng)
			if err != nil {
				return err
			}

			if jsonOut {
				var v any
				_ = json.Unmarshal(data, &v)
				output.JSON(v)
				return nil
			}

			var m struct {
				Source  string `json:"source"`
				Summary struct {
					TotalEvaluations     int     `json:"totalEvaluations"`
					TotalErrors          int     `json:"totalErrors"`
					ErrorRate            float64 `json:"errorRate"`
					AvgLatencyMs         float64 `json:"avgLatencyMs"`
					P50LatencyMs         float64 `json:"p50LatencyMs"`
					P95LatencyMs         float64 `json:"p95LatencyMs"`
					P99LatencyMs         float64 `json:"p99LatencyMs"`
					EvaluationsPerMinute float64 `json:"evaluationsPerMinute"`
					EvaluationsPerSecond float64 `json:"evaluationsPerSecond"`
				} `json:"summary"`
				Realtime struct {
					EvaluationsPerSecond float64 `json:"evaluationsPerSecond"`
					EvaluationsPerMinute float64 `json:"evaluationsPerMinute"`
					Trend                string  `json:"trend"`
				} `json:"realtime"`
				ReasonBreakdown map[string]int `json:"reasonBreakdown"`
				ValueBreakdown  map[string]int `json:"valueBreakdown"`
			}
			if err := json.Unmarshal(data, &m); err != nil {
				return err
			}

			fmt.Printf("Flag %q (source: %s)\n", key, m.Source)
			fmt.Printf("  Total: %d evals, %d errors (%.2f%%)\n",
				m.Summary.TotalEvaluations, m.Summary.TotalErrors, m.Summary.ErrorRate*100)
			fmt.Printf("  Rate: %.2f/min (%.2f/sec) — live %.2f/sec (%s)\n",
				m.Summary.EvaluationsPerMinute, m.Summary.EvaluationsPerSecond,
				m.Realtime.EvaluationsPerSecond, m.Realtime.Trend)
			fmt.Printf("  Latency: avg %.2fms, p50 %.2fms, p95 %.2fms, p99 %.2fms\n",
				m.Summary.AvgLatencyMs, m.Summary.P50LatencyMs, m.Summary.P95LatencyMs, m.Summary.P99LatencyMs)

			if len(m.ReasonBreakdown) > 0 {
				fmt.Println("\nBy reason:")
				rows := breakdownRows(m.ReasonBreakdown)
				output.Table([]string{"REASON", "EVALUATIONS"}, rows)
			}
			if len(m.ValueBreakdown) > 0 {
				fmt.Println("\nBy value/variant:")
				rows := breakdownRows(m.ValueBreakdown)
				output.Table([]string{"VALUE", "EVALUATIONS"}, rows)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().StringVar(&rng, "range", "24h", "Range: 1h, 24h, 7d, 30d")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newMetricsWatchCmd() *cobra.Command {
	var key, serviceID, environment string
	var interval time.Duration
	var count int

	cmd := &cobra.Command{
		Use:   "watch",
		Short: "Watch realtime evaluation rate for a flag (polls live metrics)",
		RunE: func(cmd *cobra.Command, args []string) error {
			iterations := count
			for {
				data, err := rest.GetFlagMetrics(cmd.Context(), key, serviceID, environment, "1h")
				if err != nil {
					return err
				}
				var m struct {
					Realtime struct {
						EvaluationsPerSecond  float64 `json:"evaluationsPerSecond"`
						EvaluationsPerMinute  float64 `json:"evaluationsPerMinute"`
						EvaluationsLastMinute int     `json:"evaluationsLastMinute"`
						ErrorRateLastMinute   float64 `json:"errorRateLastMinute"`
						Trend                 string  `json:"trend"`
						TopReason             string  `json:"topReason"`
					} `json:"realtime"`
				}
				if err := json.Unmarshal(data, &m); err != nil {
					return err
				}
				if jsonOut {
					var v any
					_ = json.Unmarshal(data, &v)
					output.JSON(v)
				} else {
					fmt.Printf("%s  %s: %.2f evals/sec (%d last min, trend %s, err %.2f%%, top %s)\n",
						time.Now().Format("15:04:05"), key,
						m.Realtime.EvaluationsPerSecond, m.Realtime.EvaluationsLastMinute,
						m.Realtime.Trend, m.Realtime.ErrorRateLastMinute*100, m.Realtime.TopReason)
				}
				if iterations > 0 {
					iterations--
					if iterations == 0 {
						break
					}
				}
				select {
				case <-cmd.Context().Done():
					return nil
				case <-time.After(interval):
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().DurationVar(&interval, "interval", 2*time.Second, "Poll interval")
	cmd.Flags().IntVarP(&count, "count", "n", 0, "Number of samples (0 = infinite)")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newMetricsHeatmapCmd() *cobra.Command {
	var projectID, serviceID, environment, rng string

	cmd := &cobra.Command{
		Use:   "heatmap",
		Short: "Project-wide evaluation heatmap (usage across all flags)",
		RunE: func(cmd *cobra.Command, args []string) error {
			resp, err := rest.GetEvaluationHeatmap(cmd.Context(), projectID, rng, serviceID, environment)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(resp)
				return nil
			}

			fmt.Printf("Project %s: %d evals (%d flags, %d active, %.2f/min)\n\n",
				projectID, resp.Totals.Evaluations, resp.Totals.Flags,
				resp.Totals.ActiveFlags, resp.Totals.EvaluationsPerMinute)

			rows := make([][]string, len(resp.Cells))
			for i, c := range resp.Cells {
				last := "never"
				if c.LastEvaluatedAt != nil {
					last = time.UnixMilli(*c.LastEvaluatedAt).Format("15:04:05")
				}
				rows[i] = []string{
					c.ServiceID, c.FlagKey,
					fmt.Sprintf("%d", c.Evaluations),
					fmt.Sprintf("%.2f", c.EvaluationsPerMinute),
					fmt.Sprintf("%.2f%%", c.ErrorRate*100),
					last,
				}
			}
			output.Table([]string{"SERVICE", "FLAG", "EVALS", "PER MIN", "ERR", "LAST EVAL"}, rows)
			return nil
		},
	}

	cmd.Flags().StringVarP(&projectID, "project", "p", "", "Project ID (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Filter to a single service")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().StringVar(&rng, "range", "1h", "Range: 15m, 1h, 24h, 7d")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

func breakdownRows(m map[string]int) [][]string {
	type kv struct {
		k string
		v int
	}
	var pairs []kv
	for k, v := range m {
		pairs = append(pairs, kv{k, v})
	}
	// sort desc by count (simple insertion for small maps)
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0 && pairs[j].v > pairs[j-1].v; j-- {
			pairs[j], pairs[j-1] = pairs[j-1], pairs[j]
		}
	}
	rows := make([][]string, 0, len(pairs))
	for _, p := range pairs {
		if len(rows) >= 8 {
			break
		}
		rows = append(rows, []string{p.k, fmt.Sprintf("%d", p.v)})
	}
	return rows
}
