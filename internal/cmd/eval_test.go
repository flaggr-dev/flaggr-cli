package cmd

import (
	"testing"
)

func TestEvalCommandTree(t *testing.T) {
	evalCmd := newEvalCmd()
	if evalCmd == nil {
		t.Fatal("expected non-nil evalCmd")
	}

	expectedSubcommands := []string{"bool", "string", "number", "config", "stream"}
	subcommands := make(map[string]bool)
	for _, sub := range evalCmd.Commands() {
		subcommands[sub.Name()] = true
	}

	for _, expected := range expectedSubcommands {
		if !subcommands[expected] {
			t.Errorf("expected eval subcommand %q to be registered", expected)
		}
	}
}

func TestEvalStreamCmdFlags(t *testing.T) {
	cmd := newEvalStreamCmd()
	if cmd == nil {
		t.Fatal("expected non-nil stream command")
	}

	serviceFlag := cmd.Flag("service")
	if serviceFlag == nil {
		t.Error("expected --service flag to exist")
	}

	envFlag := cmd.Flag("environment")
	if envFlag == nil {
		t.Error("expected --environment flag to exist")
	}
	if envFlag.DefValue != "production" {
		t.Errorf("expected default environment 'production', got %q", envFlag.DefValue)
	}
}

func TestEvalBoolCmdFlags(t *testing.T) {
	cmd := newEvalBoolCmd()
	if cmd == nil {
		t.Fatal("expected non-nil bool command")
	}

	for _, flagName := range []string{"key", "service", "environment", "context"} {
		if cmd.Flag(flagName) == nil {
			t.Errorf("expected --%s flag to exist on bool command", flagName)
		}
	}
}

func TestFlagsToggleCmdFlags(t *testing.T) {
	cmd := newFlagsToggleCmd()
	if cmd == nil {
		t.Fatal("expected non-nil toggle command")
	}

	for _, flagName := range []string{"key", "service", "environment"} {
		if cmd.Flag(flagName) == nil {
			t.Errorf("expected --%s flag to exist on toggle command", flagName)
		}
	}
}

func TestParseContextJSON(t *testing.T) {
	t.Run("empty string returns nil", func(t *testing.T) {
		if got := parseContextJSON(""); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("invalid json returns nil", func(t *testing.T) {
		if got := parseContextJSON("not a json"); got != nil {
			t.Errorf("expected nil, got %v", got)
		}
	})

	t.Run("valid json parses into map", func(t *testing.T) {
		input := `{"userId":"123","plan":"pro"}`
		got := parseContextJSON(input)
		if got == nil {
			t.Fatal("expected non-nil map")
		}
		if got["userId"] != "123" || got["plan"] != "pro" {
			t.Errorf("unexpected map contents: %+v", got)
		}
	})
}
