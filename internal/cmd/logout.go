package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/flaggr-dev/flaggr-cli/internal/config"
	"github.com/flaggr-dev/flaggr-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLogoutCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials",
		Long: `Remove the credentials stored in ~/.flaggr/config.json.

Logging out doesn't revoke the token: it stays valid until it expires or you
revoke it in the dashboard (Profile → Personal access tokens).`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Read before removing: the notice names the token kind and the dashboard URL.
			cfg := config.Load()
			apiURL, _ := config.Resolve(cfg)

			path := config.Path()
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("failed to remove config: %w", err)
			}
			output.Success("Logged out. Credentials removed from " + path)
			for _, line := range logoutNotice(cfg.APIToken, apiURL, os.Getenv("FLAGGR_API_TOKEN")) {
				fmt.Println(line)
			}
			return nil
		},
	}
}

// logoutNotice says what logging out leaves behind. Removing the config only
// forgets the token on this machine; the token keeps working until it
// expires or is revoked, so the user is told where to revoke it (never the
// token itself). A FLAGGR_API_TOKEN in the environment keeps authenticating
// every command, so that is named too.
func logoutNotice(token, apiURL, envToken string) []string {
	base := strings.TrimRight(apiURL, "/")
	if base == "" {
		base = "https://flaggr.dev"
	}
	var lines []string
	switch {
	case strings.HasPrefix(token, "fgp_"):
		lines = append(lines, "The personal access token itself stays valid until it expires or you revoke it in the dashboard: "+
			"Profile → Personal access tokens ("+base+"/console/profile#personal-access-tokens).")
	case strings.HasPrefix(token, "fgr_"):
		lines = append(lines, "The project API token itself stays valid until you revoke it in the dashboard: "+
			"the project's Settings → API tokens ("+base+"/console).")
	case token != "":
		lines = append(lines, "The token itself stays valid until it expires or you revoke it in the dashboard "+
			"(personal access tokens: Profile → Personal access tokens, "+base+"/console/profile#personal-access-tokens).")
	}
	if envToken != "" {
		lines = append(lines, "FLAGGR_API_TOKEN is still set in your environment, so commands keep using that token; unset it to stay logged out.")
	}
	return lines
}
