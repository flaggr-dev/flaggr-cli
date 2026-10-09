package api

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	flaggrv1 "github.com/flaggr-dev/flaggr-cli/internal/flaggrv1"
)

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		input    string
		expected flaggrv1.Environment
	}{
		{"development", flaggrv1.Environment_ENVIRONMENT_DEVELOPMENT},
		{"dev", flaggrv1.Environment_ENVIRONMENT_DEVELOPMENT},
		{"DEV", flaggrv1.Environment_ENVIRONMENT_DEVELOPMENT},
		{"staging", flaggrv1.Environment_ENVIRONMENT_STAGING},
		{"STAGING", flaggrv1.Environment_ENVIRONMENT_STAGING},
		{"production", flaggrv1.Environment_ENVIRONMENT_PRODUCTION},
		{"prod", flaggrv1.Environment_ENVIRONMENT_PRODUCTION},
		{"PROD", flaggrv1.Environment_ENVIRONMENT_PRODUCTION},
		{"unknown", flaggrv1.Environment_ENVIRONMENT_UNSPECIFIED},
		{"", flaggrv1.Environment_ENVIRONMENT_UNSPECIFIED},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseEnvironment(tt.input)
			if got != tt.expected {
				t.Errorf("parseEnvironment(%q) = %v; want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestBuildContext(t *testing.T) {
	t.Run("empty attributes returns nil", func(t *testing.T) {
		if got := buildContext(nil); got != nil {
			t.Errorf("expected nil for empty map, got %+v", got)
		}
		if got := buildContext(map[string]string{}); got != nil {
			t.Errorf("expected nil for empty map, got %+v", got)
		}
	})

	t.Run("extracts targetingKey and attributes", func(t *testing.T) {
		attrs := map[string]string{
			"targetingKey": "user-42",
			"tier":         "enterprise",
			"country":      "AU",
		}
		ctx := buildContext(attrs)
		if ctx == nil {
			t.Fatal("expected non-nil context")
		}
		if ctx.TargetingKey != "user-42" {
			t.Errorf("expected targetingKey 'user-42', got %q", ctx.TargetingKey)
		}
		if ctx.StringAttributes["tier"] != "enterprise" {
			t.Errorf("expected tier 'enterprise', got %q", ctx.StringAttributes["tier"])
		}
		if ctx.StringAttributes["country"] != "AU" {
			t.Errorf("expected country 'AU', got %q", ctx.StringAttributes["country"])
		}
		if _, exists := ctx.StringAttributes["targetingKey"]; exists {
			t.Error("targetingKey should not be duplicated in StringAttributes")
		}
	})
}

func TestTokenInterceptor(t *testing.T) {
	interceptor := &tokenInterceptor{token: "fgr_test_secret_token"}

	unary := interceptor.WrapUnary(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		auth := req.Header().Get("Authorization")
		expected := "Bearer fgr_test_secret_token"
		if auth != expected {
			t.Errorf("expected Authorization header %q, got %q", expected, auth)
		}
		return nil, nil
	})

	dummyReq := connect.NewRequest(&flaggrv1.StreamFlagsRequest{})
	_, _ = unary(context.Background(), dummyReq)
}

func TestNewConnectClient(t *testing.T) {
	client := NewConnectClient("https://flaggr.dev", "fgr_token_test")
	if client == nil {
		t.Fatal("expected non-nil ConnectClient")
	}
	if client.eval == nil {
		t.Error("expected initialized eval client")
	}
	if client.stream == nil {
		t.Error("expected initialized stream client")
	}
}
