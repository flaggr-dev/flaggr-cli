package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// SourceMapUpload is what the API answers for one uploaded map.
type SourceMapUpload struct {
	Release  string `json:"release"`
	File     string `json:"file"`
	Bytes    int    `json:"bytes"`
	Replaced bool   `json:"replaced"`
}

// SourceMapRelease is a release with uploaded maps.
type SourceMapRelease struct {
	Release        string `json:"release"`
	Files          int    `json:"files"`
	Bytes          int64  `json:"bytes"`
	LastUploadedAt string `json:"lastUploadedAt"`
}

// SourceMapFile is one uploaded map of a release.
type SourceMapFile struct {
	File       string `json:"file"`
	Bytes      int64  `json:"bytes"`
	UploadedBy string `json:"uploadedBy"`
	UploadedAt string `json:"uploadedAt"`
}

func sourceMapsPath(projectID string, query url.Values) string {
	path := "/api/projects/" + url.PathEscape(projectID) + "/sourcemaps"
	if len(query) > 0 {
		path += "?" + query.Encode()
	}
	return path
}

// UploadSourceMap sends one gzipped source map for the minified file `file`
// (its path in stack traces, such as /_next/static/chunks/app/page-3f9a.js).
func (c *RESTClient) UploadSourceMap(ctx context.Context, projectID, release, file string, gzipped []byte) (*SourceMapUpload, error) {
	query := url.Values{"file": {file}}
	if release != "" {
		query.Set("release", release)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.BaseURL+sourceMapsPath(projectID, query), bytes.NewReader(gzipped))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	if c.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.APIToken)
	}
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	var out SourceMapUpload
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &out, nil
}

// ListSourceMapReleases lists a project's releases with maps, newest first.
func (c *RESTClient) ListSourceMapReleases(ctx context.Context, projectID string) ([]SourceMapRelease, error) {
	data, err := c.do(ctx, http.MethodGet, sourceMapsPath(projectID, nil), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Releases []SourceMapRelease `json:"releases"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return out.Releases, nil
}

// ListSourceMapFiles lists one release's maps.
func (c *RESTClient) ListSourceMapFiles(ctx context.Context, projectID, release string) ([]SourceMapFile, error) {
	data, err := c.do(ctx, http.MethodGet, sourceMapsPath(projectID, url.Values{"release": {release}}), nil)
	if err != nil {
		return nil, err
	}
	var out struct {
		Files []SourceMapFile `json:"files"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return out.Files, nil
}

// DeleteSourceMaps deletes a release's maps, or one file's when file isn't empty.
func (c *RESTClient) DeleteSourceMaps(ctx context.Context, projectID, release, file string) (int, error) {
	query := url.Values{"release": {release}}
	if file != "" {
		query.Set("file", file)
	}
	data, err := c.do(ctx, http.MethodDelete, sourceMapsPath(projectID, query), nil)
	if err != nil {
		return 0, err
	}
	var out struct {
		Deleted int `json:"deleted"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return 0, fmt.Errorf("decode: %w", err)
	}
	return out.Deleted, nil
}
