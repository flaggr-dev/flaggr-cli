package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newEvalCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "eval",
		Short: "Evaluate flags via Connect-RPC (typed proto client)",
	}

	cmd.AddCommand(
		newEvalBoolCmd(),
		newEvalStringCmd(),
		newEvalNumberCmd(),
		newEvalConfigCmd(),
		newEvalStreamCmd(),
	)

	return cmd
}

func newEvalBoolCmd() *cobra.Command {
	var key, serviceID, environment, contextJSON string

	cmd := &cobra.Command{
		Use:   "bool",
		Short: "Evaluate a boolean flag via Connect-RPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			if environment != "" && environment != "development" {
				res, err := rest.EvaluateFlag(cmd.Context(), key, serviceID, environment)
				if err != nil {
					return err
				}
				if jsonOut {
					var v any
					_ = json.Unmarshal(res, &v)
					output.JSON(v)
					return nil
				}
				var parsed struct {
					Value  bool   `json:"value"`
					Reason string `json:"reason"`
				}
				_ = json.Unmarshal(res, &parsed)
				fmt.Printf("%s = %v (reason: %s)\n", key, parsed.Value, parsed.Reason)
				return nil
			}

			ctx := parseContextJSON(contextJSON)
			value, reason, err := conn.ResolveBoolean(cmd.Context(), key, serviceID, ctx)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(map[string]any{"value": value, "reason": reason})
			} else {
				fmt.Printf("%s = %v (reason: %s)\n", key, value, reason)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().StringVarP(&contextJSON, "context", "c", "", "Evaluation context as JSON")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newEvalStringCmd() *cobra.Command {
	var key, serviceID, environment, contextJSON string

	cmd := &cobra.Command{
		Use:   "string",
		Short: "Evaluate a string flag via Connect-RPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := parseContextJSON(contextJSON)
			value, reason, err := conn.ResolveString(cmd.Context(), key, serviceID, ctx)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(map[string]any{"value": value, "reason": reason})
			} else {
				fmt.Printf("%s = %q (reason: %s)\n", key, value, reason)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().StringVarP(&contextJSON, "context", "c", "", "Evaluation context as JSON")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newEvalNumberCmd() *cobra.Command {
	var key, serviceID, environment, contextJSON string

	cmd := &cobra.Command{
		Use:   "number",
		Short: "Evaluate a number flag via Connect-RPC",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := parseContextJSON(contextJSON)
			value, reason, err := conn.ResolveNumber(cmd.Context(), key, serviceID, ctx)
			if err != nil {
				return err
			}

			if jsonOut {
				output.JSON(map[string]any{"value": value, "reason": reason})
			} else {
				fmt.Printf("%s = %g (reason: %s)\n", key, value, reason)
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&key, "key", "k", "", "Flag key (required)")
	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	cmd.Flags().StringVarP(&contextJSON, "context", "c", "", "Evaluation context as JSON")
	_ = cmd.MarkFlagRequired("key")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newEvalConfigCmd() *cobra.Command {
	var serviceID, environment string

	cmd := &cobra.Command{
		Use:   "config",
		Short: "Fetch full flag configuration via Connect-RPC GetConfiguration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := conn.GetConfiguration(cmd.Context(), serviceID, environment)
			if err != nil {
				return err
			}

			output.JSON(cfg)
			return nil
		},
	}

	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "", "Environment")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func newEvalStreamCmd() *cobra.Command {
	var serviceID, environment string

	cmd := &cobra.Command{
		Use:   "stream",
		Short: "Subscribe to real-time flag updates via Connect-RPC gRPC streaming",
		RunE: func(cmd *cobra.Command, args []string) error {
			stream, err := conn.StreamFlags(cmd.Context(), serviceID, environment)
			if err != nil {
				return err
			}
			defer stream.Close()

			if !jsonOut {
				output.Success(fmt.Sprintf("Connected to Connect-RPC stream for service %q (%s)", serviceID, environment))
			}

			for stream.Receive() {
				msg := stream.Msg()
				if jsonOut {
					output.JSON(msg)
				} else {
					flagName := msg.FlagKey
					if msg.Flag != nil && msg.Flag.Name != "" {
						flagName = fmt.Sprintf("%s (%s)", msg.FlagKey, msg.Flag.Name)
					}
					fmt.Printf("[gRPC Stream] %-12s | flag: %-25s | version: %s\n",
						msg.EventType.String(), flagName, msg.ConfigVersion)
				}
			}
			return stream.Err()
		},
	}

	cmd.Flags().StringVarP(&serviceID, "service", "s", "", "Service ID (required)")
	cmd.Flags().StringVarP(&environment, "environment", "e", "production", "Environment")
	_ = cmd.MarkFlagRequired("service")

	return cmd
}

func parseContextJSON(raw string) map[string]string {
	if raw == "" {
		return nil
	}
	var m map[string]string
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil
	}
	return m
}
