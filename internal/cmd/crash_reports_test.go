package cmd

import "testing"

func TestCommandPathNamesTheSubcommandOnly(t *testing.T) {
	root := NewRootCmd()
	cases := map[string][]string{
		"flags list":    {"flags", "list", "--project", "secret-project", "--json"},
		"crash-reports": {"crash-reports", "on"},
		"":              {"--version"},
	}
	for want, args := range cases {
		if got := commandPath(root, args); got != want {
			t.Errorf("commandPath(%v) = %q, want %q", args, got, want)
		}
	}
}
