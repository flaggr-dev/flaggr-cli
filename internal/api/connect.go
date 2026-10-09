package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"connectrpc.com/connect"
	flaggrv1 "github.com/flaggr-dev/flaggr-cli/internal/flaggrv1"
	"github.com/flaggr-dev/flaggr-cli/internal/flaggrv1/flaggrv1connect"
)

// ConnectClient wraps the Flaggr Connect-RPC evaluation service.
type ConnectClient struct {
	eval   flaggrv1connect.EvaluationServiceClient
	stream flaggrv1connect.FlagStreamServiceClient
}

// tokenInterceptor injects the Bearer token into every request.
type tokenInterceptor struct {
	token string
}

func (t *tokenInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if t.token != "" {
			req.Header().Set("Authorization", "Bearer "+t.token)
		}
		return next(ctx, req)
	}
}

func (t *tokenInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return func(ctx context.Context, spec connect.Spec) connect.StreamingClientConn {
		conn := next(ctx, spec)
		if t.token != "" {
			conn.RequestHeader().Set("Authorization", "Bearer "+t.token)
		}
		return conn
	}
}

func (t *tokenInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

func NewConnectClient(baseURL, apiToken string) *ConnectClient {
	baseURL = strings.TrimRight(baseURL, "/")
	// When connecting directly to the dedicated Go API service (api.flaggr.dev or local port 8090),
	// Connect-RPC is mounted directly at the root. Next.js control plane routes via /api/grpc-web.
	if !strings.Contains(baseURL, "api.flaggr.dev") && !strings.Contains(baseURL, ":8090") && !strings.HasSuffix(baseURL, "/api/grpc-web") {
		baseURL += "/api/grpc-web"
	}
	interceptor := &tokenInterceptor{token: apiToken}
	opts := []connect.ClientOption{
		connect.WithInterceptors(interceptor),
		connect.WithProtoJSON(), // Server returns JSON, not binary proto
	}

	return &ConnectClient{
		eval:   flaggrv1connect.NewEvaluationServiceClient(http.DefaultClient, baseURL, opts...),
		stream: flaggrv1connect.NewFlagStreamServiceClient(http.DefaultClient, baseURL, opts...),
	}
}

// ResolveBoolean evaluates a boolean flag via Connect-RPC.
func (c *ConnectClient) ResolveBoolean(ctx context.Context, flagKey, serviceID string, evalCtx map[string]string) (bool, string, error) {
	req := &flaggrv1.ResolveBooleanRequest{
		FlagKey:   flagKey,
		ServiceId: serviceID,
		Context:   buildContext(evalCtx),
	}

	resp, err := c.eval.ResolveBoolean(ctx, connect.NewRequest(req))
	if err != nil {
		return false, "", fmt.Errorf("ResolveBoolean: %w", err)
	}
	return resp.Msg.Value, resp.Msg.Reason.String(), nil
}

// ResolveString evaluates a string flag via Connect-RPC.
func (c *ConnectClient) ResolveString(ctx context.Context, flagKey, serviceID string, evalCtx map[string]string) (string, string, error) {
	req := &flaggrv1.ResolveStringRequest{
		FlagKey:   flagKey,
		ServiceId: serviceID,
		Context:   buildContext(evalCtx),
	}

	resp, err := c.eval.ResolveString(ctx, connect.NewRequest(req))
	if err != nil {
		return "", "", fmt.Errorf("ResolveString: %w", err)
	}
	return resp.Msg.Value, resp.Msg.Reason.String(), nil
}

// ResolveNumber evaluates a number flag via Connect-RPC.
func (c *ConnectClient) ResolveNumber(ctx context.Context, flagKey, serviceID string, evalCtx map[string]string) (float64, string, error) {
	req := &flaggrv1.ResolveNumberRequest{
		FlagKey:   flagKey,
		ServiceId: serviceID,
		Context:   buildContext(evalCtx),
	}

	resp, err := c.eval.ResolveNumber(ctx, connect.NewRequest(req))
	if err != nil {
		return 0, "", fmt.Errorf("ResolveNumber: %w", err)
	}
	return resp.Msg.Value, resp.Msg.Reason.String(), nil
}

// GetConfiguration fetches the current flag configuration via Connect-RPC.
func (c *ConnectClient) GetConfiguration(ctx context.Context, serviceID, environment string) (*flaggrv1.ConfigurationSync, error) {
	req := &flaggrv1.StreamFlagsRequest{
		ServiceId:   serviceID,
		Environment: parseEnvironment(environment),
	}
	resp, err := c.stream.GetConfiguration(ctx, connect.NewRequest(req))
	if err != nil {
		return nil, fmt.Errorf("GetConfiguration: %w", err)
	}
	return resp.Msg, nil
}

// StreamFlags subscribes to real-time flag updates via Connect-RPC server-side streaming.
func (c *ConnectClient) StreamFlags(ctx context.Context, serviceID, environment string) (*connect.ServerStreamForClient[flaggrv1.FlagUpdate], error) {
	req := &flaggrv1.StreamFlagsRequest{
		ServiceId:   serviceID,
		Environment: parseEnvironment(environment),
		ClientId:    fmt.Sprintf("flaggr-cli-%d", time.Now().UnixNano()),
	}
	return c.stream.StreamFlags(ctx, connect.NewRequest(req))
}

func buildContext(attrs map[string]string) *flaggrv1.EvaluationContext {
	if len(attrs) == 0 {
		return nil
	}
	ctx := &flaggrv1.EvaluationContext{
		StringAttributes: make(map[string]string),
	}
	for k, val := range attrs {
		if k == "targetingKey" {
			ctx.TargetingKey = val
		} else {
			ctx.StringAttributes[k] = val
		}
	}
	return ctx
}

func parseEnvironment(env string) flaggrv1.Environment {
	switch strings.ToLower(env) {
	case "development", "dev":
		return flaggrv1.Environment_ENVIRONMENT_DEVELOPMENT
	case "staging":
		return flaggrv1.Environment_ENVIRONMENT_STAGING
	case "production", "prod":
		return flaggrv1.Environment_ENVIRONMENT_PRODUCTION
	default:
		return flaggrv1.Environment_ENVIRONMENT_UNSPECIFIED
	}
}
