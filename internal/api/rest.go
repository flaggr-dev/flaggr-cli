// Package api provides HTTP clients for the Flaggr REST and Connect APIs.
package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// RESTClient wraps the Flaggr REST API for management operations.
type RESTClient struct {
	BaseURL    string
	APIToken   string
	HTTPClient *http.Client
}

// NewRESTClient creates a REST client for management endpoints.
func NewRESTClient(baseURL, apiToken string) *RESTClient {
	return &RESTClient{
		BaseURL:  strings.TrimRight(baseURL, "/"),
		APIToken: apiToken,
		HTTPClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *RESTClient) do(ctx context.Context, method, path string, body any) ([]byte, error) {
	u := c.BaseURL + path

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, u, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
	}
	return respBody, nil
}

// ── Projects ──────────────────────────────────────────────────

type Project struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Slug        string `json:"slug"`
	Description string `json:"description,omitempty"`
	Role        string `json:"role,omitempty"`
}

func (c *RESTClient) ListProjects(ctx context.Context) ([]Project, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/projects", nil)
	if err != nil {
		return nil, err
	}
	var resp struct{ Projects []Project }
	return resp.Projects, json.Unmarshal(data, &resp)
}

// ── Services ──────────────────────────────────────────────────

type Service struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	ProjectID string `json:"projectId"`
}

func (c *RESTClient) ListServices(ctx context.Context, projectID string) ([]Service, error) {
	data, err := c.do(ctx, http.MethodGet, "/api/services?projectId="+url.QueryEscape(projectID), nil)
	if err != nil {
		return nil, err
	}
	var resp struct{ Services []Service }
	return resp.Services, json.Unmarshal(data, &resp)
}

// ── Flags ─────────────────────────────────────────────────────

type Flag struct {
	Key          string `json:"key"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Enabled      bool   `json:"enabled"`
	Environment  string `json:"environment"`
	ServiceID    string `json:"serviceId"`
	ProjectID    string `json:"projectId"`
	DefaultValue any    `json:"defaultValue"`
}

func (c *RESTClient) ListFlags(ctx context.Context, projectID string, serviceID, environment string) ([]Flag, error) {
	params := url.Values{"projectId": {projectID}}
	if serviceID != "" {
		params.Set("serviceId", serviceID)
	}
	if environment != "" {
		params.Set("environment", environment)
	}
	data, err := c.do(ctx, http.MethodGet, "/api/flags?"+params.Encode(), nil)
	if err != nil {
		return nil, err
	}
	var resp struct{ Flags []Flag }
	return resp.Flags, json.Unmarshal(data, &resp)
}

func (c *RESTClient) CreateFlag(ctx context.Context, flag map[string]any) (json.RawMessage, error) {
	data, err := c.do(ctx, http.MethodPost, "/api/flags", flag)
	return data, err
}

func (c *RESTClient) ToggleFlag(ctx context.Context, key, serviceID, environment string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/flags/%s/toggle?serviceId=%s", url.PathEscape(key), url.QueryEscape(serviceID))
	if environment != "" {
		path += fmt.Sprintf("&environment=%s", url.QueryEscape(environment))
	}
	return c.do(ctx, http.MethodPost, path, nil)
}

func (c *RESTClient) EvaluateFlag(ctx context.Context, flagKey, serviceID, environment string) (json.RawMessage, error) {
	body := map[string]any{
		"flagKey":   flagKey,
		"serviceId": serviceID,
	}
	if environment != "" {
		body["environment"] = environment
	}
	return c.do(ctx, http.MethodPost, "/api/flags/evaluate", body)
}

func (c *RESTClient) DeleteFlag(ctx context.Context, key, serviceID, environment string) error {
	params := url.Values{"serviceId": {serviceID}}
	if environment != "" {
		params.Set("environment", environment)
	}
	_, err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/flags/%s?%s", url.PathEscape(key), params.Encode()), nil)
	return err
}

// ── Stale Flags ──────────────────────────────────────────────

type StaleFlag struct {
	FlagKey                 string  `json:"flagKey"`
	ServiceID               string  `json:"serviceId"`
	Environment             string  `json:"environment"`
	Status                  string  `json:"status"`
	DaysSinceLastEvaluation float64 `json:"daysSinceLastEvaluation"`
	EvaluationVolume7d      int     `json:"evaluationVolume7d"`
	StatusChangedAt         string  `json:"statusChangedAt"`
}

type StaleFlagsResponse struct {
	Flags         []StaleFlag `json:"flags"`
	StaleCount    int         `json:"staleCount"`
	InactiveCount int         `json:"inactiveCount"`
	Total         int         `json:"total"`
}

func (c *RESTClient) GetStaleFlags(ctx context.Context, projectID string) (*StaleFlagsResponse, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/projects/%s/stale-flags", url.PathEscape(projectID)), nil)
	if err != nil {
		return nil, err
	}
	var resp StaleFlagsResponse
	return &resp, json.Unmarshal(data, &resp)
}

// ── Health ────────────────────────────────────────────────────

func (c *RESTClient) Health(ctx context.Context) (json.RawMessage, error) {
	return c.do(ctx, http.MethodGet, "/api/health", nil)
}

func (c *RESTClient) GetFlagHealth(ctx context.Context, key, serviceID string) (json.RawMessage, error) {
	path := fmt.Sprintf("/api/flags/%s/health?serviceId=%s", url.PathEscape(key), url.QueryEscape(serviceID))
	data, err := c.do(ctx, http.MethodGet, path, nil)
	return data, err
}

func (c *RESTClient) GetProjectHealth(ctx context.Context, projectID string) (json.RawMessage, error) {
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/projects/%s/health", url.PathEscape(projectID)), nil)
	return data, err
}

// ── Metrics & analytics ─────────────────────────────────────────

func (c *RESTClient) GetFlagMetrics(ctx context.Context, key, serviceID, environment, rng string) (json.RawMessage, error) {
	params := url.Values{"serviceId": {serviceID}}
	if environment != "" {
		params.Set("environment", environment)
	}
	if rng != "" {
		params.Set("range", rng)
	}
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/flags/%s/metrics?%s", url.PathEscape(key), params.Encode()), nil)
	return data, err
}

func (c *RESTClient) GetFlagImpact(ctx context.Context, key, serviceID, environment string) (json.RawMessage, error) {
	params := url.Values{"serviceId": {serviceID}}
	if environment != "" {
		params.Set("environment", environment)
	}
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/flags/%s/impact?%s", url.PathEscape(key), params.Encode()), nil)
	return data, err
}

type HeatmapCell struct {
	FlagKey              string  `json:"flagKey"`
	ServiceID            string  `json:"serviceId"`
	Environment          string  `json:"environment"`
	Evaluations          int     `json:"evaluations"`
	EvaluationsPerMinute float64 `json:"evaluationsPerMinute"`
	ErrorRate            float64 `json:"errorRate"`
	AvgLatencyMs         float64 `json:"avgLatencyMs"`
	P99LatencyMs         float64 `json:"p99LatencyMs"`
	LastEvaluatedAt      *int64  `json:"lastEvaluatedAt"`
}

type HeatmapResponse struct {
	ProjectID string `json:"projectId"`
	Services  []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	} `json:"services"`
	Cells  []HeatmapCell `json:"cells"`
	Totals struct {
		Evaluations          int     `json:"evaluations"`
		Errors               int     `json:"errors"`
		Flags                int     `json:"flags"`
		ActiveFlags          int     `json:"activeFlags"`
		EvaluationsPerMinute float64 `json:"evaluationsPerMinute"`
	} `json:"totals"`
}

func (c *RESTClient) GetEvaluationHeatmap(ctx context.Context, projectID, rng, serviceID, environment string) (*HeatmapResponse, error) {
	params := url.Values{}
	if rng != "" {
		params.Set("range", rng)
	}
	if serviceID != "" {
		params.Set("serviceId", serviceID)
	}
	if environment != "" {
		params.Set("environment", environment)
	}
	path := fmt.Sprintf("/api/projects/%s/evaluation-heatmap", url.PathEscape(projectID))
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp HeatmapResponse
	return &resp, json.Unmarshal(data, &resp)
}

// ── Audit log & history ─────────────────────────────────────────

type AuditChange struct {
	Field    string `json:"field"`
	Path     string `json:"path"`
	Type     string `json:"type"`
	OldValue any    `json:"oldValue"`
	NewValue any    `json:"newValue"`
	Summary  string `json:"summary"`
}

type AuditLogEntry struct {
	ID           string        `json:"id"`
	Action       string        `json:"action"`
	ResourceType string        `json:"resourceType"`
	ResourceID   string        `json:"resourceId"`
	ResourceName string        `json:"resourceName"`
	UserEmail    string        `json:"userEmail"`
	UserName     string        `json:"userName"`
	Timestamp    string        `json:"timestamp"`
	Summary      string        `json:"summary"`
	Changes      []AuditChange `json:"changes"`
}

type AuditLogResponse struct {
	Logs       []AuditLogEntry `json:"logs"`
	NextCursor *int64          `json:"nextCursor"`
	HasMore    bool            `json:"hasMore"`
}

func (c *RESTClient) GetAuditLog(ctx context.Context, projectID string, limit int, cursor, action, resourceType, search string) (*AuditLogResponse, error) {
	params := url.Values{}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if action != "" {
		params.Set("action", action)
	}
	if resourceType != "" {
		params.Set("resourceType", resourceType)
	}
	if search != "" {
		params.Set("search", search)
	}
	path := fmt.Sprintf("/api/projects/%s/audit-log", url.PathEscape(projectID))
	if len(params) > 0 {
		path += "?" + params.Encode()
	}
	data, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	var resp AuditLogResponse
	return &resp, json.Unmarshal(data, &resp)
}

type FlagVersion struct {
	Version        int           `json:"version"`
	ChangeType     string        `json:"changeType"`
	ChangeSummary  string        `json:"changeSummary"`
	ChangedBy      string        `json:"changedBy"`
	ChangedByEmail string        `json:"changedByEmail"`
	ChangedByName  string        `json:"changedByName"`
	Timestamp      string        `json:"timestamp"`
	DiffSummary    string        `json:"diffSummary"`
	Diffs          []AuditChange `json:"diffs"`
}

type FlagHistoryResponse struct {
	Versions []FlagVersion `json:"versions"`
	Total    int           `json:"total"`
	HasMore  bool          `json:"hasMore"`
}

func (c *RESTClient) GetFlagHistory(ctx context.Context, key, serviceID, environment string, limit, offset int) (*FlagHistoryResponse, error) {
	params := url.Values{"serviceId": {serviceID}}
	if environment != "" {
		params.Set("environment", environment)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprintf("%d", limit))
	}
	if offset > 0 {
		params.Set("offset", fmt.Sprintf("%d", offset))
	}
	data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/flags/%s/history?%s", url.PathEscape(key), params.Encode()), nil)
	if err != nil {
		return nil, err
	}
	var resp FlagHistoryResponse
	return &resp, json.Unmarshal(data, &resp)
}

// ── Export ─────────────────────────────────────────────────────

type ExportData struct {
	Projects []json.RawMessage `json:"projects"`
	Services []json.RawMessage `json:"services"`
	Flags    []json.RawMessage `json:"flags"`
}

func (c *RESTClient) Export(ctx context.Context, projectID string) (*ExportData, error) {
	// Fetch services and flags in parallel
	type svcResult struct {
		data []byte
		err  error
	}
	type flagResult struct {
		data []byte
		err  error
	}

	svcCh := make(chan svcResult, 1)
	flagCh := make(chan flagResult, 1)

	go func() {
		d, e := c.do(ctx, http.MethodGet, "/api/services?projectId="+url.QueryEscape(projectID), nil)
		svcCh <- svcResult{d, e}
	}()
	go func() {
		d, e := c.do(ctx, http.MethodGet, "/api/flags?projectId="+url.QueryEscape(projectID), nil)
		flagCh <- flagResult{d, e}
	}()

	sr := <-svcCh
	if sr.err != nil {
		return nil, sr.err
	}
	fr := <-flagCh
	if fr.err != nil {
		return nil, fr.err
	}

	var svcResp struct{ Services []json.RawMessage }
	var flagResp struct{ Flags []json.RawMessage }
	if err := json.Unmarshal(sr.data, &svcResp); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(fr.data, &flagResp); err != nil {
		return nil, err
	}

	return &ExportData{
		Services: svcResp.Services,
		Flags:    flagResp.Flags,
	}, nil
}
